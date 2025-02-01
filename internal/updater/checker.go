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
	if event.ChampionshipID == "0" {
		c.log.Info("No next event to check")
		return nil
	}

	// Create a Chromedp allocator with default options
	allocatorCtx, cancelAllocator := chromedp.NewExecAllocator(context.Background(), chromedp.DefaultExecAllocatorOptions[:]...)
	defer cancelAllocator()

	// Create a Chromedp context
	ctx, cancel := chromedp.NewContext(allocatorCtx)
	defer cancel()

	c.log.Info("Checking event", slog.String("event_name", event.Name))

	n, err := c.dwnl.DownloadEventFiles(ctx, event.Name)
	if err != nil {
		return errors.Wrapf(err, "can't check event %s", event.Name)
	}

	if n == 0 {
		c.log.Info("No files downloaded", "event_name", event.Name)
		return nil
	}

	c.log.Info("Files downloaded", "event_name", event.Name)

	files, err := c.prsr.CollectFiles([]string{event.Name})
	if err != nil {
		return errors.Wrapf(err, "can't collect files for event %s", event.Name)
	}

	c.log.Info("Files collected for parsing", "event_name", event.Name, "count", len(files))
	for _, file := range files {
		c.log.Info("Parsing file", "event_name", event.Name, "file_name", file)
		raceResult, err := c.prsr.ParseFile(file, event)
		if err != nil {
			c.log.Error("can't parse file", "error", err.Error(), "file_name", file)
			continue
		}

		c.log.Info("Uploading result", "event_name", event.Name, "file_name", file)
		err = c.prsr.UploadRaceResult(raceResult)
		if err != nil {
			c.log.Error("can't upload race result", "error", err.Error(), "file_name", file)
			continue
		}
	}

	return nil
}
