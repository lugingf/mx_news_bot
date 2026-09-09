package render

import (
	"strings"

	"mx_news_bot/internal/publishing/contentmodel"
)

// Renderer turns a channel-independent Post into one channel's message.
type Renderer interface {
	Channel() string
	Capabilities() contentmodel.Capabilities
	Render(post contentmodel.Post) (contentmodel.Message, error)
}

// Telegram messages cap at 4096 characters.
const telegramLimit = 4096

type Telegram struct{}

func NewTelegram() Telegram { return Telegram{} }

func (Telegram) Channel() string { return "telegram" }

func (Telegram) Capabilities() contentmodel.Capabilities {
	return contentmodel.Capabilities{
		MaxRunes:       telegramLimit,
		SupportsTables: true,
		SupportsMedia:  true,
		SupportsLinks:  true,
	}
}

// Render keeps the layout the bot's own replies already use: a bold header, a labelled meta
// block, then the results in a fenced monospace table so the columns line up.
//
// Over the length limit it drops table rows and re-renders, rather than cutting the string:
// a cut would leave the code fence unterminated and Telegram would reject the whole message.
func (t Telegram) Render(post contentmodel.Post) (contentmodel.Message, error) {
	text := t.compose(post)
	if len([]rune(text)) > telegramLimit && post.Table != nil {
		text = t.composeTruncated(post)
	}
	if len([]rune(text)) > telegramLimit {
		runes := []rune(text)
		text = string(runes[:telegramLimit])
	}

	return contentmodel.Message{Text: text, ParseMode: "Markdown", Media: post.Media}, nil
}

func (t Telegram) composeTruncated(post contentmodel.Post) string {
	rows := post.Table.Rows
	for len(rows) > 1 {
		rows = rows[:len(rows)-1]

		table := *post.Table
		table.Rows = rows
		trimmed := post
		trimmed.Table = &table
		trimmed.Sections = append(append([]contentmodel.Section{}, post.Sections...),
			contentmodel.Section{Heading: "Note", Body: "List truncated to fit one message."})

		if text := t.compose(trimmed); len([]rune(text)) <= telegramLimit {
			return text
		}
	}

	return t.compose(post)
}

func (t Telegram) compose(post contentmodel.Post) string {
	var b strings.Builder

	if post.Title != "" {
		b.WriteString("🏆 *")
		b.WriteString(escapeMarkdown(post.Title))
		b.WriteString("*\n")
	}
	if post.Subtitle != "" {
		b.WriteString("_")
		b.WriteString(escapeMarkdown(post.Subtitle))
		b.WriteString("_\n")
	}
	if post.Title != "" || post.Subtitle != "" {
		b.WriteString("\n")
	}

	for _, field := range post.Meta {
		if strings.TrimSpace(field.Value) == "" {
			continue
		}
		if field.Emoji != "" {
			b.WriteString(field.Emoji)
			b.WriteString(" ")
		}
		b.WriteString("*")
		b.WriteString(escapeMarkdown(field.Label))
		b.WriteString(":* ")
		b.WriteString(escapeMarkdown(field.Value))
		b.WriteString("\n")
	}

	for _, section := range post.Sections {
		b.WriteString("\n*")
		b.WriteString(escapeMarkdown(section.Heading))
		b.WriteString("*\n")
		b.WriteString(escapeMarkdown(section.Body))
		b.WriteString("\n")
	}

	if post.Table != nil && len(post.Table.Rows) > 0 {
		b.WriteString("\n```\n")
		b.WriteString(RenderTable(*post.Table, 0))
		b.WriteString("```\n")
	}

	if tags := renderTags(post.Tags); tags != "" {
		b.WriteString("\n")
		b.WriteString(tags)
		b.WriteString("\n")
	}

	if post.Link != "" {
		b.WriteString("\n")
		b.WriteString(post.Link)
	}

	return b.String()
}

func renderTags(tags []string) string {
	parts := make([]string, 0, len(tags))
	for _, tag := range tags {
		if strings.TrimSpace(tag) != "" {
			parts = append(parts, "#"+tag)
		}
	}

	return strings.Join(parts, " ")
}

// escapeMarkdown protects the characters Telegram's legacy Markdown treats as formatting. A
// rider name with an underscore would otherwise swallow the rest of the message.
func escapeMarkdown(value string) string {
	replacer := strings.NewReplacer("_", "\\_", "*", "\\*", "`", "\\`", "[", "\\[")

	return replacer.Replace(value)
}
