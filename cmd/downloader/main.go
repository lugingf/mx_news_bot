package main

import (
	"context"
	"fmt"
	"github.com/chromedp/chromedp"
	"log/slog"
	"os"

	dwl "mx_news_bot/internal/downloader"
)

func main() {
	// Create a Chromedp allocator with default options
	allocatorCtx, cancelAllocator := chromedp.NewExecAllocator(context.Background(), chromedp.DefaultExecAllocatorOptions[:]...)
	defer cancelAllocator()

	// Create a Chromedp context
	ctx, cancel := chromedp.NewContext(allocatorCtx)
	defer cancel()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	downloader := dwl.NewDownloader("https://results.supercrosslive.com/events/", "data/2025", logger)
	eventList := []string{"Philadelphia"}

	for _, eventName := range eventList {
		changed, err := downloader.DownloadEventFiles(ctx, eventName)
		if err != nil {
			fmt.Printf("Error in eventName %s: %v\n", eventName, err)
		} else {
			fmt.Printf("Changed files for %s: %d\n", eventName, len(changed))
		}
	}
}
