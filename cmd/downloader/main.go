package main

import (
	"context"
	"fmt"

	"github.com/chromedp/chromedp"

	dwl "mx_news_bot/internal/downloader"
)

func main() {
	// Create a Chromedp allocator with default options
	allocatorCtx, cancelAllocator := chromedp.NewExecAllocator(context.Background(), chromedp.DefaultExecAllocatorOptions[:]...)
	defer cancelAllocator()

	// Create a Chromedp context
	ctx, cancel := chromedp.NewContext(allocatorCtx)
	defer cancel()

	downloader := dwl.NewDownloader("https://results.supercrosslive.com/events/", "data/2025")
	eventList := []string{"Anaheim 2"}

	for _, eventName := range eventList {
		if err := downloader.DownloadEventFiles(ctx, eventName); err != nil {
			fmt.Printf("Error in eventName %s: %v\n", eventName, err)
		} else {
			fmt.Printf("All files downloaded successfully for event name  %s.\n", eventName)
		}
	}
}
