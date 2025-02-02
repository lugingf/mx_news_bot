package downloader

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/chromedp/chromedp"
	"github.com/pkg/errors"
)

type Downloader struct {
	BaseURL string
	DataDir string
}

type Event struct {
	Name string `json:"name"`
	Link string `json:"link"`
}

func NewDownloader(baseURL, dataDir string) *Downloader {
	return &Downloader{BaseURL: baseURL, DataDir: dataDir}
}

func (d *Downloader) DownloadEventFiles(ctx context.Context, eventName string) (int, error) {
	// Variable to store results
	var eventsJSON string

	// Step 1: Find the event by name and extract its link
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
		return 0, fmt.Errorf("failed to extract event links: %w", err)
	}

	// Parse the JSON result
	var events []Event
	if err := json.Unmarshal([]byte(eventsJSON), &events); err != nil {
		return 0, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Find the event with the given name
	var eventURL string
	for _, event := range events {
		if event.Name == eventName {
			eventURL = event.Link
			break
		}
	}

	if eventURL == "" {
		return 0, fmt.Errorf("no event link found for '%s'", eventName)
	}

	fmt.Printf("Visiting event URL: %s\n", eventURL)

	// Step 2: Visit the event page and separate links for "250 Main Event" and "450 Main Event"
	races, err := d.getMainEvents(ctx, eventURL)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get main events")
	}

	// Step 3: Download files
	count := 0
	for name, links := range races.links {
		for _, link := range links {
			err := d.download(link, name, eventName)
			if err != nil {
				return 0, errors.Wrap(err, "can't download file")
			}

			count++
		}
	}

	return count, nil
}

type raceSet struct {
	links map[string][]string
}

func (d *Downloader) getMainEvents(ctx context.Context, eventURL string) (raceSet, error) {
	var links250, links250H1, links250H2 []string
	var links250R1, links250R2, links250R3 []string
	var links450, links450H1, links450H2 []string
	var links450R1, links450R2, links450R3 []string

	err := chromedp.Run(ctx,
		chromedp.Navigate(eventURL),
		chromedp.WaitVisible(`a`, chromedp.ByQuery), // Ensure the links are visible
		chromedp.Evaluate(`Array.from(document.querySelectorAll('a')).filter(a => a.textContent.includes('250 Main Event')).map(a => a.href)`, &links250),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('a')).filter(a => /250 Heat (\#?1)/.test(a.textContent)).map(a => a.href)`, &links250H1),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('a')).filter(a => /250 Heat (\#?2)/.test(a.textContent)).map(a => a.href)`, &links250H2),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('a')).filter(a => /250 Race (\#?1)/.test(a.textContent)).map(a => a.href)`, &links250R1),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('a')).filter(a => /250 Race (\#?2)/.test(a.textContent)).map(a => a.href)`, &links250R2),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('a')).filter(a => /250 Race (\#?3)/.test(a.textContent)).map(a => a.href)`, &links250R3),

		chromedp.Evaluate(`Array.from(document.querySelectorAll('a')).filter(a => a.textContent.includes('450 Main Event')).map(a => a.href)`, &links450),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('a')).filter(a => /450 Heat (\#?1)/.test(a.textContent)).map(a => a.href)`, &links450H1),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('a')).filter(a => /450 Heat (\#?2)/.test(a.textContent)).map(a => a.href)`, &links450H2),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('a')).filter(a => /450 Race (\#?1)/.test(a.textContent)).map(a => a.href)`, &links450R1),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('a')).filter(a => /450 Race (\#?2)/.test(a.textContent)).map(a => a.href)`, &links450R2),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('a')).filter(a => /450 Race (\#?3)/.test(a.textContent)).map(a => a.href)`, &links450R3),
	)
	if err != nil {
		return raceSet{}, fmt.Errorf("failed to extract download links: %w", err)
	}

	result := raceSet{
		links: map[string][]string{
			"250 Main_Event": links250,
			"250 Heat_1":     links250H1,
			"250 Heat_2":     links250H2,
			"250 Race#1":     links250R1,
			"250 Race#2":     links250R2,
			"250 Race#3":     links250R3,

			"450 Main_Event": links450,
			"450 Heat_1":     links450H1,
			"450 Heat_2":     links450H2,
			"450 Race#1":     links450R1,
			"450 Race#2":     links450R2,
			"450 Race#3":     links450R3,
		},
	}

	return result, nil
}

func (d *Downloader) download(link, race, eventName string) error {
	// main event link contains "p=view_race_result"
	if !strings.Contains(link, "p=view_race_result") {
		fmt.Printf("Skipping link: %s (missing 'p=view_race_result')\n", link)
		return nil
	}

	title, err := d.getTitle(race)
	if err != nil {
		return fmt.Errorf("failed to get title: %w", err)
	}

	fileName := fmt.Sprintf("%s_%s.pdf", eventName, title)

	pdfURL := fmt.Sprintf("%s&export=pdf", link)
	fmt.Printf("Downloading PDF from: %s\n", pdfURL)

	if err := d.downloadFile(pdfURL, eventName, fileName); err != nil {
		return fmt.Errorf("failed to download file from %s: %w", pdfURL, err)
	}

	fmt.Printf("Successfully downloaded: %s\n", fileName)

	return nil
}

func (d *Downloader) downloadFile(url, eventDir, fileName string) error {
	// Ensure the directory exists
	err := os.MkdirAll(fmt.Sprintf("%s/%s", d.DataDir, eventDir), os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Perform HTTP GET request
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to make GET request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("non-OK HTTP status: %s", resp.Status)
	}

	// Create the file
	out, err := os.Create(fmt.Sprintf("%s/%s/%s", d.DataDir, eventDir, fileName))
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	// Write the response body to the file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	return nil
}

func (d *Downloader) getTitle(event string) (string, error) {
	parts := strings.Split(event, " ")
	if len(parts) < 2 {
		return "", errors.New("wrong race name")
	}

	return fmt.Sprintf("%s_%s", parts[0], parts[1]), nil
}
