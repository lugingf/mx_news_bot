package channel

import (
	"context"
	"errors"
	"strings"
	"testing"

	tele "gopkg.in/telebot.v3"

	"mx_news_bot/internal/publishing/contentmodel"
)

// fakeTelegramSender records every call it is given, so a test can assert on what shape of
// message actually left the publisher without a real bot token.
type fakeTelegramSender struct {
	sent []any
	err  error
}

func (f *fakeTelegramSender) Send(_ tele.Recipient, what any, _ ...any) (*tele.Message, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.sent = append(f.sent, what)

	return &tele.Message{ID: len(f.sent)}, nil
}

func TestTelegramPublish_TextOnlyWhenNoMedia(t *testing.T) {
	sender := &fakeTelegramSender{}
	telegram := NewTelegram(sender)

	_, err := telegram.Publish(context.Background(), "-100", contentmodel.Message{Text: "hello"})
	if err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}

	if len(sender.sent) != 1 {
		t.Fatalf("expected exactly one send, got %d", len(sender.sent))
	}
	text, ok := sender.sent[0].(string)
	if !ok || text != "hello" {
		t.Fatalf("expected a plain text send of %q, got %#v", "hello", sender.sent[0])
	}
}

func TestTelegramPublish_AttachesImageAsPhoto(t *testing.T) {
	sender := &fakeTelegramSender{}
	telegram := NewTelegram(sender)

	message := contentmodel.Message{
		Text: "short caption",
		Media: []contentmodel.Media{
			{URL: "https://lapvision.org/api/publications/media/card.png", Kind: contentmodel.MediaImage},
		},
	}

	_, err := telegram.Publish(context.Background(), "-100", message)
	if err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}

	if len(sender.sent) != 1 {
		t.Fatalf("expected exactly one send for a short caption, got %d", len(sender.sent))
	}
	photo, ok := sender.sent[0].(*tele.Photo)
	if !ok {
		t.Fatalf("expected a *tele.Photo, got %#v", sender.sent[0])
	}
	if photo.FileURL != message.Media[0].URL {
		t.Errorf("photo file URL = %q, want %q", photo.FileURL, message.Media[0].URL)
	}
	if photo.Caption != message.Text {
		t.Errorf("photo caption = %q, want %q", photo.Caption, message.Text)
	}
}

func TestTelegramPublish_AttachesVideo(t *testing.T) {
	sender := &fakeTelegramSender{}
	telegram := NewTelegram(sender)

	message := contentmodel.Message{
		Text: "clip",
		Media: []contentmodel.Media{
			{URL: "https://lapvision.org/api/publications/media/clip.mp4", Kind: contentmodel.MediaVideo},
		},
	}

	if _, err := telegram.Publish(context.Background(), "-100", message); err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}

	video, ok := sender.sent[0].(*tele.Video)
	if !ok {
		t.Fatalf("expected a *tele.Video, got %#v", sender.sent[0])
	}
	if video.FileURL != message.Media[0].URL {
		t.Errorf("video file URL = %q, want %q", video.FileURL, message.Media[0].URL)
	}
}

// A rendered post with a full standings table easily runs past Telegram's 1024-character caption
// cap. Sending it as the caption anyway gets the whole call rejected — image included — so the
// text has to travel as its own message instead.
func TestTelegramPublish_LongTextFollowsPhotoSeparately(t *testing.T) {
	sender := &fakeTelegramSender{}
	telegram := NewTelegram(sender)

	longText := strings.Repeat("a", telegramCaptionLimit+1)
	message := contentmodel.Message{
		Text: longText,
		Media: []contentmodel.Media{
			{URL: "https://lapvision.org/api/publications/media/card.png", Kind: contentmodel.MediaImage},
		},
	}

	_, err := telegram.Publish(context.Background(), "-100", message)
	if err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}

	if len(sender.sent) != 2 {
		t.Fatalf("expected a photo followed by a text message, got %d sends", len(sender.sent))
	}
	photo, ok := sender.sent[0].(*tele.Photo)
	if !ok {
		t.Fatalf("expected the first send to be a *tele.Photo, got %#v", sender.sent[0])
	}
	if photo.Caption != "" {
		t.Errorf("photo caption = %q, want empty when the text follows separately", photo.Caption)
	}
	text, ok := sender.sent[1].(string)
	if !ok || text != longText {
		t.Fatalf("expected the second send to be the full text, got %#v", sender.sent[1])
	}
}

func TestTelegramPublish_IgnoresMediaWithoutURL(t *testing.T) {
	sender := &fakeTelegramSender{}
	telegram := NewTelegram(sender)

	message := contentmodel.Message{
		Text:  "no real attachment",
		Media: []contentmodel.Media{{URL: "  ", Kind: contentmodel.MediaImage}},
	}

	if _, err := telegram.Publish(context.Background(), "-100", message); err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}

	if _, ok := sender.sent[0].(string); !ok {
		t.Fatalf("expected a plain text send when no media has a real URL, got %#v", sender.sent[0])
	}
}

func TestTelegramPublish_NotConfigured(t *testing.T) {
	telegram := NewTelegram(nil)

	_, err := telegram.Publish(context.Background(), "-100", contentmodel.Message{Text: "hi"})
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured, got %v", err)
	}
}
