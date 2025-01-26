package updater

import (
	"context"
	"log/slog"

	"github.com/chromedp/chromedp"
	"github.com/pkg/errors"

	dwn "mx_news_bot/internal/downloader"
	"mx_news_bot/internal/parser"
	"mx_news_bot/internal/storage"
)

// TODO to constructor
const (
	BaseURL = "https://results.supercrosslive.com/events/"
)

type SXChecker struct {
	repo *storage.Repository
	dwnl *dwn.Downloader
	log  *slog.Logger
	prsr *parser.Parser
}

func New(repo *storage.Repository, dwnl *dwn.Downloader, prsr *parser.Parser, log *slog.Logger) *SXChecker {
	return &SXChecker{
		log:  log,
		repo: repo,
		dwnl: dwnl,
		prsr: prsr,
	}
}

// Check fetches and parses event pages for PDF links.
func (c *SXChecker) Check() error {
	event, err := c.repo.GetNextEvent()
	if err != nil {
		return errors.Wrap(err, "can't collect next event for download")
	}
	// Create a Chromedp allocator with default options
	allocatorCtx, cancelAllocator := chromedp.NewExecAllocator(context.Background(), chromedp.DefaultExecAllocatorOptions[:]...)
	defer cancelAllocator()

	// Create a Chromedp context
	ctx, cancel := chromedp.NewContext(allocatorCtx)
	defer cancel()

	c.log.Info("Checking event", slog.String("event_name", event.Name))

	err = c.dwnl.DownloadEventFiles(ctx, event.Name)
	if err != nil {
		return errors.Wrapf(err, "can't check event %s", event.Name)
	}

	files, err := c.prsr.CollectFiles([]string{event.Name})
	if err != nil {
		return errors.Wrapf(err, "can't collect files for event %s", event.Name)
	}

	for _, file := range files {
		raceResult, err := c.prsr.ParseFile(file, event)
		if err != nil {
			c.log.Error("can't parse file", "error", err.Error(), "file_name", file)
			continue
		}

		err = c.prsr.UploadRaceResult(raceResult)
		if err != nil {
			c.log.Error("can't upload race result", "error", err.Error(), "file_name", file)
			continue
		}
	}

	return nil
}
