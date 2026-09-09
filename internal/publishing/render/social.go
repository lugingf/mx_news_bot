package render

import (
	"strings"
	"unicode/utf8"

	"mx_news_bot/internal/publishing/contentmodel"
)

const twitterLimit = 280

type Twitter struct{}

func NewTwitter() Twitter { return Twitter{} }

func (Twitter) Channel() string { return "twitter" }

func (Twitter) Capabilities() contentmodel.Capabilities {
	return contentmodel.Capabilities{
		MaxRunes:       twitterLimit,
		SupportsTables: false,
		SupportsMedia:  true,
		SupportsLinks:  true,
	}
}

// Render fits the post into 280 characters: headline, then as many table rows as remain. There
// is no monospace, so a row becomes "1. Rider — value" rather than a padded column.
func (t Twitter) Render(post contentmodel.Post) (contentmodel.Message, error) {
	var head strings.Builder
	head.WriteString(post.Title)
	if post.Subtitle != "" {
		head.WriteString(" · ")
		head.WriteString(post.Subtitle)
	}

	tags := renderTags(post.Tags)
	budget := twitterLimit - utf8.RuneCountInString(head.String())
	if tags != "" {
		budget -= utf8.RuneCountInString(tags) + 1
	}
	if post.Link != "" {
		budget -= utf8.RuneCountInString(post.Link) + 1
	}

	var body strings.Builder
	if post.Table != nil {
		for _, row := range post.Table.Rows {
			line := "\n" + flattenRow(row)
			if utf8.RuneCountInString(line) > budget {
				break
			}
			body.WriteString(line)
			budget -= utf8.RuneCountInString(line)
		}
	}

	text := head.String() + body.String()
	if tags != "" {
		text += "\n" + tags
	}
	if post.Link != "" {
		text += "\n" + post.Link
	}
	if runes := []rune(text); len(runes) > twitterLimit {
		text = string(runes[:twitterLimit])
	}

	return contentmodel.Message{Text: text, Media: post.Media}, nil
}

// flattenRow turns a table row into one readable line: the first cell leads, the last one
// trails, and anything in between is dropped since there is no room for it.
func flattenRow(row []string) string {
	switch len(row) {
	case 0:
		return ""
	case 1:
		return row[0]
	default:
		return row[0] + ". " + strings.Join(row[1:len(row)-1], " ") + " — " + row[len(row)-1]
	}
}

type Instagram struct{}

func NewInstagram() Instagram { return Instagram{} }

func (Instagram) Channel() string { return "instagram" }

func (Instagram) Capabilities() contentmodel.Capabilities {
	return contentmodel.Capabilities{
		MaxRunes:       2200,
		SupportsTables: false,
		SupportsMedia:  true,
		SupportsLinks:  false,
	}
}

// Render produces a caption. Instagram does not render links in captions, so the post link is
// deliberately left out rather than printed as dead text.
func (i Instagram) Render(post contentmodel.Post) (contentmodel.Message, error) {
	var b strings.Builder
	b.WriteString(post.Title)
	if post.Subtitle != "" {
		b.WriteString("\n")
		b.WriteString(post.Subtitle)
	}
	b.WriteString("\n")

	for _, field := range post.Meta {
		if strings.TrimSpace(field.Value) == "" {
			continue
		}
		b.WriteString("\n")
		b.WriteString(field.Label)
		b.WriteString(": ")
		b.WriteString(field.Value)
	}

	if post.Table != nil {
		b.WriteString("\n")
		for _, row := range post.Table.Rows {
			b.WriteString("\n")
			b.WriteString(flattenRow(row))
		}
	}

	if tags := renderTags(post.Tags); tags != "" {
		b.WriteString("\n\n")
		b.WriteString(tags)
	}

	text := b.String()
	if runes := []rune(text); len(runes) > i.Capabilities().MaxRunes {
		text = string(runes[:i.Capabilities().MaxRunes])
	}

	return contentmodel.Message{Text: text, Media: post.Media}, nil
}
