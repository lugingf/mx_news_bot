package builder

import (
	"encoding/json"
	"fmt"
	"strings"

	"mx_news_bot/internal/publishing/contentmodel"
	"mx_news_bot/internal/publishing/contract"
)

// RenderedPostBuilder passes through a post lap_vision has already composed.
//
// The other builders exist because the sender ships raw results and the wording is decided here.
// This one is the opposite arrangement: the sender knows which numbers are worth saying — it is
// the side that computed them — and the bot is left with what it is actually good at, which is
// knowing what each channel can carry.
type RenderedPostBuilder struct{}

func (RenderedPostBuilder) Type() string { return contract.TypeRenderedPost }

func (RenderedPostBuilder) Build(envelope contract.Envelope) (contentmodel.Post, error) {
	var payload contract.RenderedPostPayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		return contentmodel.Post{}, fmt.Errorf("rendered_post: decode payload: %w", err)
	}

	if strings.TrimSpace(payload.Title) == "" {
		return contentmodel.Post{}, fmt.Errorf("rendered_post: title is required")
	}

	post := contentmodel.Post{
		Title:      payload.Title,
		Subtitle:   payload.Subtitle,
		Link:       payload.Link,
		Tags:       tags(payload),
		OccurredAt: envelope.OccurredAt,
	}

	// The lines are the body. They arrive as statements rather than as labelled fields, so they
	// go into one section instead of being forced into a meta block they were not written for.
	body := strings.TrimSpace(strings.Join(payload.Lines, "\n"))
	if hook := strings.TrimSpace(payload.Hook); hook != "" {
		if body != "" {
			body += "\n\n" + hook
		} else {
			body = hook
		}
	}
	if body != "" {
		post.Sections = []contentmodel.Section{{Body: body}}
	}

	if payload.Table != nil && len(payload.Table.Rows) > 0 {
		post.Table = table(*payload.Table)
	}

	// A gallery is a set of pictures already filed, sent together as they are — nothing here draws
	// a card from them. It takes the place of the one drawn image rather than joining it: a post
	// names one or the other, not both.
	if len(payload.Images) > 0 {
		post.Media = make([]contentmodel.Media, 0, len(payload.Images))
		for _, img := range payload.Images {
			if trimmed := strings.TrimSpace(img); trimmed != "" {
				post.Media = append(post.Media, contentmodel.Media{URL: trimmed, Kind: contentmodel.MediaImage})
			}
		}
	} else if payload.Image != nil && strings.TrimSpace(payload.Image.URL) != "" {
		// Only a picture that already exists can be attached. A description of one the media side
		// has not composed yet is not something a channel can send, and dropping it is better than
		// posting a broken attachment.
		post.Media = []contentmodel.Media{{
			URL:     payload.Image.URL,
			Caption: payload.Image.Title,
			Kind:    contentmodel.MediaImage,
		}}
	}

	return post, nil
}

func tags(payload contract.RenderedPostPayload) []string {
	out := make([]string, 0, len(payload.Tags)+1)
	if discipline := strings.TrimSpace(payload.Discipline); discipline != "" {
		out = append(out, discipline)
	}
	for _, tag := range payload.Tags {
		if trimmed := strings.TrimSpace(tag); trimmed != "" {
			out = append(out, tagify(trimmed))
		}
	}

	return out
}

// table keeps the first column narrow and right-aligns the last one: a position and a number of
// points read as columns, a name does not.
func table(source contract.RenderedTable) *contentmodel.Table {
	columns := make([]contentmodel.Column, 0, len(source.Header))
	for index, header := range source.Header {
		align := contentmodel.AlignLeft
		if index > 0 && index == len(source.Header)-1 {
			align = contentmodel.AlignRight
		}
		columns = append(columns, contentmodel.Column{Header: header, Align: align})
	}

	return &contentmodel.Table{Columns: columns, Rows: source.Rows}
}
