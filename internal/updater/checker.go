package updater

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

type SXChecker struct {
	log *slog.Logger
}

func (c *SXChecker) Check() []string {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	pdfRegex := regexp.MustCompile(`(?i)href\s*=\s*['"]([^'" ]+\.pdf)['"]`)

	files := make([]string, 0)
	eventLinks := generateEventLinks()
	for _, link := range eventLinks {
		fmt.Println("Checking event link:", link)

		var pageContent string

		err := chromedp.Run(ctx,
			chromedp.Navigate(link),
			chromedp.Sleep(5*time.Second), // Wait for loading dynamic content
			chromedp.OuterHTML("html", &pageContent),
		)

		if err != nil {
			c.log.With("link", link).Error("couldn't visit event page", "error", err)
			continue
		}

		matches := pdfRegex.FindAllStringSubmatch(pageContent, -1)

		for _, match := range matches {
			if len(match) > 1 {
				pdfLink := match[1]

				if !strings.HasPrefix(pdfLink, "http") {
					pdfLink = toAbsoluteURL(pdfLink, link)
				}

				files = append(files, pdfLink)
			}
		}
	}

	return files
}

func toAbsoluteURL(href string, baseURL string) string {
	base, err := url.Parse(baseURL)
	if err != nil {
		return href
	}
	if u, err := url.Parse(href); err == nil {
		return base.ResolveReference(u).String()
	}
	return href
}

func generateEventLinks() []string {
	var eventLinks []string
	for year := 24; year <= 24; year++ {
		yearString := fmt.Sprintf("S%d", year)
		eventID := fmt.Sprintf("%s%02d", yearString, 99)
		eventLink := fmt.Sprintf("https://archives.amasupercross.com/%d/index.html?EventID=%s", time.Now().Year(), eventID)
		eventLinks = append(eventLinks, eventLink)
	}
	return eventLinks
}
