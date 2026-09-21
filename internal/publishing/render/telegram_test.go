package render

import (
	"strings"
	"testing"

	"mx_news_bot/internal/publishing/contentmodel"
)

// A section with no heading of its own — the body every builder produces from plain statements —
// must not leave a blank markdown heading ("**") sitting between the title and the text, which
// read on the phone as an oversized gap between the headline and the body.
func TestAHeadinglessSectionLeavesNoStrayMarkup(t *testing.T) {
	post := contentmodel.Post{
		Title:    "Azerbaijan Grand Prix",
		Subtitle: "Baku, Azerbaijan",
		Sections: []contentmodel.Section{{Body: "Round 15 · 26 September 2026"}},
	}

	message, err := NewTelegram().Render(post)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(message.Text, "**") {
		t.Errorf("rendered text still carries an empty heading marker: %q", message.Text)
	}
	if strings.Contains(message.Text, "\n\n\n") {
		t.Errorf("rendered text has more than one blank line before the body: %q", message.Text)
	}
}

// A section that does carry a heading keeps it — only the empty case collapses.
func TestAHeadedSectionKeepsItsHeading(t *testing.T) {
	post := contentmodel.Post{
		Title:    "MXGP of China",
		Sections: []contentmodel.Section{{Heading: "Classes", Body: "MXGP, MX2"}},
	}

	message, err := NewTelegram().Render(post)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(message.Text, "*Classes*") {
		t.Errorf("expected the heading to survive: %q", message.Text)
	}
}
