package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/chromedp/chromedp"
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

func (d *Downloader) DownloadEventFiles(ctx context.Context, eventName string) error {
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
		return fmt.Errorf("failed to extract event links: %w", err)
	}

	// Parse the JSON result
	var events []Event
	if err := json.Unmarshal([]byte(eventsJSON), &events); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
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
		return fmt.Errorf("no event link found for '%s'", eventName)
	}

	fmt.Printf("Visiting event URL: %s\n", eventURL)

	// Step 2: Visit the event page and separate links for "250 Main Event" and "450 Main Event"
	links250, links450, err := d.getMainEvents(ctx, eventURL, eventName)
	if err != nil {
		return errors.Wrap(err, "failed to get main events")
	}

	// Step 3: Download files
	for _, link := range links250 {
		err := d.download(link, "250", eventName)
		if err != nil {
			return err
		}
	}

	for _, link := range links450 {
		err := d.download(link, "450", eventName)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *Downloader) getMainEvents(ctx context.Context, eventURL, eventName string) ([]string, []string, error) {
	var links250 []string
	var links450 []string
	err := chromedp.Run(ctx,
		chromedp.Navigate(eventURL),
		chromedp.WaitVisible(`a`, chromedp.ByQuery), // Ensure the links are visible
		chromedp.Evaluate(`Array.from(document.querySelectorAll('a')).filter(a => a.textContent.includes('250 Main Event')).map(a => a.href)`, &links250),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('a')).filter(a => a.textContent.includes('450 Main Event')).map(a => a.href)`, &links450),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to extract download links: %w", err)
	}

	if len(links250) == 0 && len(links450) == 0 {
		return nil, nil, fmt.Errorf("no download links found for '%s'", eventName)
	}

	return links250, links450, nil
}

func (d *Downloader) download(link, class, eventName string) error {
	// main event link contains "p=view_race_result"
	if !strings.Contains(link, "p=view_race_result") {
		fmt.Printf("Skipping link: %s (missing 'p=view_race_result')\n", link)
		return nil
	}

	pdfURL := fmt.Sprintf("%s&export=pdf", link)
	fileName := fmt.Sprintf("%s_%s.pdf", eventName, getTitle(class))
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

func getTitle(event string) string {
	switch event {
	case "250":
		return "250_MainEvent"
	case "450":
		return "450_MainEvent"
	}

	return "UnknownEvent"
}

// Example usage
func main() {
	// Create a Chromedp allocator with default options
	allocatorCtx, cancelAllocator := chromedp.NewExecAllocator(context.Background(), chromedp.DefaultExecAllocatorOptions[:]...)
	defer cancelAllocator()

	// Create a Chromedp context
	ctx, cancel := chromedp.NewContext(allocatorCtx)
	defer cancel()

	downloader := NewDownloader("https://results.supercrosslive.com/events/", "data/2025")
	eventList := []string{"Anaheim 1", "San Diego"}

	for _, eventName := range eventList {
		if err := downloader.DownloadEventFiles(ctx, eventName); err != nil {
			fmt.Printf("Error in eventName %s: %v\n", eventName, err)
		} else {
			fmt.Printf("All files downloaded successfully for event name  %s.\n", eventName)
		}
	}
}
