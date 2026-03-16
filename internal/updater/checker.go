package updater

import (
	"context"
	"log/slog"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/pkg/errors"

	dwn "mx_news_bot/internal/downloader"
	"mx_news_bot/internal/parser"
	"mx_news_bot/internal/storage"
)

type SXChecker struct {
	repo *storage.Repository
	dwnl *dwn.AMASupercross
	log  *slog.Logger
	prsr *parser.AMASupercross
}

const (
	sxChampionshipName        = "Monster Energy AMA Supercross"
	completedLookbackDays     = 30
	pdfArchiveRetentionPeriod = 30 * 24 * time.Hour
)

func New(repo *storage.Repository, dwnl *dwn.AMASupercross, prsr *parser.AMASupercross, log *slog.Logger) *SXChecker {
	return &SXChecker{
		log:  log,
		repo: repo,
		dwnl: dwnl,
		prsr: prsr,
	}
}

// Check fetches and parses event pages for PDF links.
func (c *SXChecker) Check() error {
	archivedCount, err := c.dwnl.ArchiveOldPDFs(pdfArchiveRetentionPeriod)
	if err != nil {
		c.log.Error("can't archive old PDFs", "error", err)
	} else if archivedCount > 0 {
		c.log.Info("Archived old PDF files", "count", archivedCount)
	}

	events, err := c.repo.GetEventsToCheck(sxChampionshipName, completedLookbackDays)
	if err != nil {
		return errors.Wrap(err, "can't collect events for download")
	}

	if len(events) == 0 {
		c.log.Info("No events to check")
		return nil
	}

	for _, event := range events {
		// Create a Chromedp allocator with default options
		allocatorCtx, cancelAllocator := chromedp.NewExecAllocator(context.Background(), chromedp.DefaultExecAllocatorOptions[:]...)

		// Create a Chromedp context
		ctx, cancel := chromedp.NewContext(allocatorCtx)

		c.log.Info("Checking event", slog.String("event_name", event.Name))

		changedFiles, err := c.dwnl.DownloadEventFiles(ctx, event.Name)
		cancel()
		cancelAllocator()
		if err != nil {
			return errors.Wrapf(err, "can't check event %s", event.Name)
		}

		if len(changedFiles) == 0 {
			c.log.Info("No changed files found for event", "event_name", event.Name)
			continue
		}

		c.log.Info("Changed files downloaded", "event_name", event.Name, "count", len(changedFiles))
		for _, file := range changedFiles {
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
	}

	return nil
}
