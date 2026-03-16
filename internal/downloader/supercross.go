package downloader

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/pkg/errors"
)

type AMASupercross struct {
	BaseURL string
	DataDir string
	log     *slog.Logger
}

type Event struct {
	Name string `json:"name"`
	Link string `json:"link"`
}

func NewDownloader(baseURL, dataDir string, log *slog.Logger) *AMASupercross {
	return &AMASupercross{BaseURL: baseURL, DataDir: dataDir, log: log}
}

func (d *AMASupercross) DownloadEventFiles(ctx context.Context, eventName string) ([]string, error) {
	ctxt, cancel := context.WithTimeout(ctx, 300*time.Second)
	defer cancel()

	events, err := d.getEventLinks(ctxt)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get event links")
	}

	eventURL, resolvedEventName, err := d.findEventURL(events, eventName)
	if err != nil {
		return nil, err
	}

	// Step 2: Visit the event page and group links by class/race.
	races, err := d.getRaces(ctxt, eventURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get race links")
	}

	changedFiles := make(map[string]struct{})
	for name, links := range races.links {
		for _, link := range links {
			filePath, changed, err := d.download(link, name, resolvedEventName)
			if err != nil {
				return nil, errors.Wrap(err, "can't download file")
			}

			if changed {
				changedFiles[filePath] = struct{}{}
			}
		}
	}

	result := make([]string, 0, len(changedFiles))
	for filePath := range changedFiles {
		result = append(result, filePath)
	}
	sort.Strings(result)

	return result, nil
}

func (d *AMASupercross) getEventLinks(ctx context.Context) ([]Event, error) {
	// Variable to store results
	var eventsJSON string

	// Step 1: Find the event by name and extract its link
	d.log.Info("Visiting URL", "url", d.BaseURL)
	err := chromedp.Run(ctx,
		chromedp.Navigate(d.BaseURL),
		chromedp.WaitVisible(`table`, chromedp.ByQuery), // Ensure the table is visible
		chromedp.Evaluate(`JSON.stringify(Array.from(document.querySelectorAll('tbody tr')).map(row => {
			const nameCell = row.querySelector('td a');
			return {
				name: nameCell ? nameCell.textContent.trim() : '',
				link: nameCell ? nameCell.href : ''
			};
		}))`, &eventsJSON),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to extract event links: %w", err)
	}

	// Parse the JSON result
	var events []Event
	if err := json.Unmarshal([]byte(eventsJSON), &events); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return events, nil
}

type raceSet struct {
	links map[string][]string
}

type anchorLink struct {
	Text string `json:"text"`
	Href string `json:"href"`
}

var (
	rgxNonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)
	rgxSpaces      = regexp.MustCompile(`\s+`)
	rgxHeat1       = regexp.MustCompile(`(?i)heat[\s_-]*#?\s*1\b`)
	rgxHeat2       = regexp.MustCompile(`(?i)heat[\s_-]*#?\s*2\b`)
	rgxRace1       = regexp.MustCompile(`(?i)race[\s_-]*#?\s*1\b`)
	rgxRace2       = regexp.MustCompile(`(?i)race[\s_-]*#?\s*2\b`)
	rgxRace3       = regexp.MustCompile(`(?i)race[\s_-]*#?\s*3\b`)
)

func (d *AMASupercross) findEventURL(events []Event, eventName string) (string, string, error) {
	target := normalizeEventName(eventName)

	for _, event := range events {
		if normalizeEventName(event.Name) == target {
			return event.Link, event.Name, nil
		}
	}

	for _, event := range events {
		normalized := normalizeEventName(event.Name)
		if strings.Contains(normalized, target) || strings.Contains(target, normalized) {
			return event.Link, event.Name, nil
		}
	}

	return "", "", fmt.Errorf("no event link found for %q", eventName)
}

func (d *AMASupercross) getRaces(ctx context.Context, eventURL string) (raceSet, error) {
	var anchorsJSON string

	err := chromedp.Run(ctx,
		chromedp.Navigate(eventURL),
		chromedp.WaitVisible(`a`, chromedp.ByQuery), // Ensure the links are visible
		chromedp.Evaluate(`JSON.stringify(Array.from(document.querySelectorAll('a')).map(a => ({
			text: (a.textContent || '').trim(),
			href: a.href
		})))`, &anchorsJSON),
	)
	if err != nil {
		return raceSet{}, fmt.Errorf("failed to extract download links: %w", err)
	}

	var anchors []anchorLink
	if err := json.Unmarshal([]byte(anchorsJSON), &anchors); err != nil {
		return raceSet{}, fmt.Errorf("failed to parse race links JSON: %w", err)
	}

	dedup := make(map[string]map[string]struct{})
	for _, anchor := range anchors {
		if anchor.Href == "" || !strings.Contains(anchor.Href, "p=view_race_result") {
			continue
		}

		raceName, ok := classifyRaceLabel(anchor.Text)
		if !ok {
			continue
		}

		if _, exists := dedup[raceName]; !exists {
			dedup[raceName] = make(map[string]struct{})
		}
		dedup[raceName][anchor.Href] = struct{}{}
	}

	result := raceSet{
		links: make(map[string][]string),
	}

	for raceName, linksSet := range dedup {
		links := make([]string, 0, len(linksSet))
		for link := range linksSet {
			links = append(links, link)
		}
		sort.Strings(links)
		result.links[raceName] = links
	}

	return result, nil
}

func (d *AMASupercross) download(link, race, eventName string) (string, bool, error) {
	title, err := d.getTitle(race)
	if err != nil {
		return "", false, fmt.Errorf("failed to get title: %w", err)
	}

	fileName := fmt.Sprintf("%s_%s.pdf", eventName, title)

	pdfURL := fmt.Sprintf("%s&export=pdf", link)
	filePath, changed, err := d.downloadFile(pdfURL, eventName, fileName)
	if err != nil {
		return "", false, fmt.Errorf("failed to download file from %s: %w", pdfURL, err)
	}

	return filePath, changed, nil
}

func (d *AMASupercross) downloadFile(url, eventDir, fileName string) (string, bool, error) {
	filePath := filepath.Join(d.DataDir, eventDir, fileName)

	// Ensure the directory exists
	err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
	if err != nil {
		return "", false, fmt.Errorf("failed to create directory: %w", err)
	}

	// Perform HTTP GET request
	resp, err := http.Get(url)
	if err != nil {
		return "", false, fmt.Errorf("failed to make GET request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf("non-OK HTTP status: %s", resp.Status)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", false, fmt.Errorf("failed to read response body: %w", err)
	}

	current, err := os.ReadFile(filePath)
	if err == nil && bytes.Equal(current, content) {
		return filePath, false, nil
	}
	if err != nil && !os.IsNotExist(err) {
		return "", false, fmt.Errorf("failed to read existing file: %w", err)
	}

	err = os.WriteFile(filePath, content, 0o644)
	if err != nil {
		return "", false, fmt.Errorf("failed to write file: %w", err)
	}

	return filePath, true, nil
}

func (d *AMASupercross) getTitle(event string) (string, error) {
	event = strings.TrimSpace(event)
	if event == "" {
		return "", errors.New("wrong race name")
	}

	return strings.ReplaceAll(event, " ", "_"), nil
}

func (d *AMASupercross) ArchiveOldPDFs(maxAge time.Duration) (int, error) {
	if maxAge <= 0 {
		return 0, nil
	}

	cutoff := time.Now().Add(-maxAge)
	archiveRoot := filepath.Join(d.DataDir, "archive", time.Now().Format("2006-01"))
	archived := 0

	err := filepath.Walk(d.DataDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if info.IsDir() {
			if path == filepath.Join(d.DataDir, "archive") {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.EqualFold(filepath.Ext(info.Name()), ".pdf") {
			return nil
		}
		if !info.ModTime().Before(cutoff) {
			return nil
		}

		rel, err := filepath.Rel(d.DataDir, path)
		if err != nil {
			return err
		}
		dst := filepath.Join(archiveRoot, rel)
		if err := moveFile(path, dst); err != nil {
			return err
		}

		archived++
		return nil
	})
	if err != nil {
		return 0, err
	}

	return archived, nil
}

func classifyRaceLabel(label string) (string, bool) {
	normalized := normalizeEventName(label)
	if normalized == "" {
		return "", false
	}

	class := ""
	switch {
	case strings.Contains(normalized, "450"):
		class = "450"
	case strings.Contains(normalized, "250"):
		class = "250"
	default:
		return "", false
	}

	raceType := ""
	switch {
	case strings.Contains(normalized, "main") || strings.Contains(normalized, "final"):
		raceType = "Main_Event"
	case strings.Contains(normalized, "west") && strings.Contains(normalized, "heat"):
		raceType = "West_Heat"
	case strings.Contains(normalized, "east") && strings.Contains(normalized, "heat"):
		raceType = "East_Heat"
	case rgxHeat1.MatchString(normalized):
		raceType = "Heat_1"
	case rgxHeat2.MatchString(normalized):
		raceType = "Heat_2"
	case rgxRace1.MatchString(normalized):
		raceType = "Race#1"
	case rgxRace2.MatchString(normalized):
		raceType = "Race#2"
	case rgxRace3.MatchString(normalized):
		raceType = "Race#3"
	default:
		return "", false
	}

	return fmt.Sprintf("%s %s", class, raceType), true
}

func normalizeEventName(value string) string {
	value = strings.ToLower(value)
	value = rgxNonAlphaNum.ReplaceAllString(value, " ")
	value = rgxSpaces.ReplaceAllString(value, " ")
	return strings.TrimSpace(value)
}

func moveFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}

	if err := out.Close(); err != nil {
		return err
	}

	return os.Remove(src)
}
