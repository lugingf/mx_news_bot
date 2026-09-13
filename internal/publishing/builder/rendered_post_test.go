package builder

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"mx_news_bot/internal/publishing/contract"
)

func envelopeFor(t *testing.T, payload contract.RenderedPostPayload) contract.Envelope {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	return contract.Envelope{
		ID:         "post-1",
		Type:       contract.TypeRenderedPost,
		Version:    contract.Version,
		OccurredAt: time.Date(2026, time.September, 13, 12, 0, 0, 0, time.UTC),
		Payload:    contract.RawPayload(raw),
	}
}

func samplePayload() contract.RenderedPostPayload {
	return contract.RenderedPostPayload{
		Discipline: "moto",
		PostType:   "event_result",
		Title:      "MXGP of China · MXGP",
		Subtitle:   "FIM Motocross World Championship, этап 18",
		Lines:      []string{"Победа: Jeffrey Herlings", "Отрыв от второго: 6 очк."},
		Table: &contract.RenderedTable{
			Header: []string{"#", "Гонщик", "Очки"},
			Rows:   [][]string{{"1", "Jeffrey Herlings", "50"}, {"2", "Tim Gajser", "44"}},
		},
		Tags: []string{"FIM Motocross World Championship"},
	}
}

func TestRenderedPostIsPassedThroughAsComposed(t *testing.T) {
	post, err := RenderedPostBuilder{}.Build(envelopeFor(t, samplePayload()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if post.Title != "MXGP of China · MXGP" || post.Subtitle == "" {
		t.Fatalf("unexpected heading: %+v", post)
	}
	// The lines are statements, not labelled fields, so they belong in the body rather than in a
	// meta block written for dates and tracks.
	if len(post.Sections) != 1 || !strings.Contains(post.Sections[0].Body, "Jeffrey Herlings") {
		t.Fatalf("unexpected body: %+v", post.Sections)
	}
	if post.Table == nil || len(post.Table.Rows) != 2 {
		t.Fatalf("unexpected table: %+v", post.Table)
	}
	if !post.OccurredAt.Equal(time.Date(2026, time.September, 13, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("expected the event time from the envelope, got %v", post.OccurredAt)
	}
}

// The discipline is what routes a post to the moto channel or the F1 one, so it has to survive as
// a tag even when the sender lists no tags of its own.
func TestDisciplineBecomesTheFirstTag(t *testing.T) {
	payload := samplePayload()
	payload.Tags = nil
	post, err := RenderedPostBuilder{}.Build(envelopeFor(t, payload))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(post.Tags) != 1 || post.Tags[0] != "moto" {
		t.Fatalf("unexpected tags: %+v", post.Tags)
	}
}

// A description of a picture nobody has composed yet is not something a channel can send.
func TestOnlyAnExistingPictureIsAttached(t *testing.T) {
	payload := samplePayload()
	payload.Image = &contract.RenderedImage{
		Layers: []contract.RenderedLayer{{Kind: "rider", Key: "Jeffrey Herlings"}},
		Title:  "MXGP of China",
	}
	post, err := RenderedPostBuilder{}.Build(envelopeFor(t, payload))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(post.Media) != 0 {
		t.Fatalf("expected no attachment for a picture that does not exist yet: %+v", post.Media)
	}

	payload.Image.URL = "https://media.lapvision.org/post-1.png"
	post, err = RenderedPostBuilder{}.Build(envelopeFor(t, payload))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(post.Media) != 1 || post.Media[0].URL != "https://media.lapvision.org/post-1.png" {
		t.Fatalf("expected the picture to be attached: %+v", post.Media)
	}
}

func TestAPostWithoutATitleIsRefused(t *testing.T) {
	payload := samplePayload()
	payload.Title = "  "
	_, err := RenderedPostBuilder{}.Build(envelopeFor(t, payload))
	if err == nil {
		t.Fatal("expected a titleless post to be refused")
	}
}

func TestTheRegistryKnowsTheType(t *testing.T) {
	post, err := DefaultRegistry().Build(envelopeFor(t, samplePayload()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if post.Title == "" {
		t.Fatal("expected the registry to route rendered posts")
	}
}
