package updater

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

type Event struct {
	EventID string
	Date    time.Time
	Link    string
}

type ScrapeFile struct {
	EventID     string
	FileName    string
	FilePath    string
	Status      string // pending, downloaded, processed
	LastChecked time.Time
}

const (
	BaseURL     = "https://archives.amasupercross.com"
	PDFNameMask = `(?i)href\s*=\s*['"]([^'" ]+\.pdf)['"]` // Маска для имени файла PDF
)

type SXChecker struct {
	log      *slog.Logger
	pdfRegex *regexp.Regexp
}

// Check fetches and parses event pages for PDF links.
func (c *SXChecker) Check(events []Event, db *sql.DB) error {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	for _, event := range events {
		c.log.Info("Checking event link", slog.String("link", event.Link))

		var pageContent string

		err := chromedp.Run(ctx,
			chromedp.Navigate(event.Link),
			chromedp.Sleep(5*time.Second), // Wait for loading dynamic content
			chromedp.OuterHTML("html", &pageContent),
		)

		if err != nil {
			c.log.Error("Couldn't visit event page", slog.String("link", event.Link), slog.String("error", err.Error()))
			continue
		}

		matches := c.pdfRegex.FindAllStringSubmatch(pageContent, -1)

		for _, match := range matches {
			if len(match) > 1 {
				pdfLink := match[1]

				if !strings.HasPrefix(pdfLink, "http") {
					pdfLink = toAbsoluteURL(pdfLink, event.Link)
				}

				fileName := pdfLink[strings.LastIndex(pdfLink, "/")+1:]
				file := ScrapeFile{
					EventID:     event.EventID,
					FileName:    fileName,
					FilePath:    pdfLink,
					Status:      "pending",
					LastChecked: time.Now(),
				}
				if err := c.insertScrapeFile(db, file); err != nil {
					c.log.Error("Failed to insert scrape file", slog.String("event_id", file.EventID), slog.String("file_name", file.FileName), slog.String("error", err.Error()))
				}
			}
		}
	}
	return nil
}

// LoadEventsFromDB loads events and their details from the database.
func (c *SXChecker) LoadEventsFromDB(db *sql.DB, baseID string) ([]Event, error) {
	query := `
		SELECT 
			CONCAT(?, LPAD(CAST(round_number AS TEXT), 2, '0')) AS event_id,
			event_date,
			CONCAT(?, '/', YEAR(event_date), '/index.html?EventID=', CONCAT(?, LPAD(CAST(round_number AS TEXT), 2, '0'))) AS link
		FROM events
		WHERE championship_id = 1
	`
	rows, err := db.Query(query, baseID, BaseURL, baseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]Event, 0)
	for rows.Next() {
		var event Event
		if err := rows.Scan(&event.EventID, &event.Date, &event.Link); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, nil
}

// InsertScrapeFile inserts or updates scrape file details in the database.
func (c *SXChecker) insertScrapeFile(db *sql.DB, scrapeFile ScrapeFile) error {
	query := `
		INSERT INTO event_scrape (event_id, file_name, file_path, status, last_checked)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (event_id, file_name) DO UPDATE
		SET file_path = EXCLUDED.file_path, status = EXCLUDED.status, last_checked = EXCLUDED.last_checked
	`
	_, err := db.Exec(query, scrapeFile.EventID, scrapeFile.FileName, scrapeFile.FilePath, scrapeFile.Status, scrapeFile.LastChecked)
	return err
}

// ToAbsoluteURL converts a relative URL to an absolute URL based on the base URL.
func toAbsoluteURL(relative, base string) string {
	if strings.HasPrefix(relative, "/") {
		base = strings.TrimRight(base, "/")
	}
	return fmt.Sprintf("%s%s", base, relative)
}

// expectedFiles := []string{"S1F1RES.pdf", "S2F1RES.pdf"}
//
//	if err := preloadScrapeTable(db, events, expectedFiles); err != nil {
//		logger.Error("Failed to preload scrape table", slog.String("error", err.Error()))
//		return
//	}
func (c *SXChecker) preloadScrapeTable(db *sql.DB, events []Event, expectedFiles []string) error {
	for _, event := range events {
		for _, fileName := range expectedFiles {
			scrapeFile := ScrapeFile{
				EventID:     event.EventID,
				FileName:    fileName,
				FilePath:    "", // Пока неизвестен
				Status:      "pending",
				LastChecked: time.Now(),
			}
			if err := c.insertScrapeFile(db, scrapeFile); err != nil {
				return fmt.Errorf("failed to preload file %s for event %s: %w", fileName, event.EventID, err)
			}
		}
	}
	return nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	checker := SXChecker{
		log:      logger,
		pdfRegex: regexp.MustCompile(PDFNameMask),
	}

	baseID := "S25"

	// Подключение к базе данных
	db, err := sql.Open("postgres", "your_connection_string")
	if err != nil {
		logger.Error("Failed to connect to database", slog.String("error", err.Error()))
		return
	}
	defer db.Close()

	// Load events from database
	events, err := checker.LoadEventsFromDB(db, baseID)
	if err != nil {
		logger.Error("Failed to load events from database", slog.String("error", err.Error()))
		return
	}

	// Check events and store file details
	if err := checker.Check(events, db); err != nil {
		logger.Error("Failed to check events", slog.String("error", err.Error()))
	}
}
