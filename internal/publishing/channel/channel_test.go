package channel

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	tele "gopkg.in/telebot.v3"

	"mx_news_bot/internal/models"
	"mx_news_bot/internal/publishing/contentmodel"
)

// fakeTelegramSender records every call it is given, so a test can assert on what shape of
// message actually left the publisher without a real bot token.
type fakeTelegramSender struct {
	sent   []any
	albums []tele.Album
	err    error
}

func (f *fakeTelegramSender) Send(_ tele.Recipient, what any, _ ...any) (*tele.Message, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.sent = append(f.sent, what)

	return &tele.Message{ID: len(f.sent)}, nil
}

func (f *fakeTelegramSender) SendAlbum(_ tele.Recipient, a tele.Album, _ ...any) ([]tele.Message, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.albums = append(f.albums, a)

	return []tele.Message{{ID: len(f.albums)}}, nil
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

// A gallery of several pictures — a circuit from more than one angle, say — goes out as one
// Telegram album rather than picking just the first of them to stand for the rest.
func TestTelegramPublish_SendsGalleryAsAlbum(t *testing.T) {
	sender := &fakeTelegramSender{}
	telegram := NewTelegram(sender)

	message := contentmodel.Message{
		Text: "Zandvoort, three angles",
		Media: []contentmodel.Media{
			{URL: "https://lapvision.org/media/zandvoort-1.jpg", Kind: contentmodel.MediaImage},
			{URL: "https://lapvision.org/media/zandvoort-2.jpg", Kind: contentmodel.MediaImage},
			{URL: "https://lapvision.org/media/zandvoort-3.jpg", Kind: contentmodel.MediaImage},
		},
	}

	_, err := telegram.Publish(context.Background(), "-100", message)
	if err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}

	if len(sender.sent) != 0 {
		t.Fatalf("expected no single send for a gallery, got %d", len(sender.sent))
	}
	if len(sender.albums) != 1 {
		t.Fatalf("expected exactly one album, got %d", len(sender.albums))
	}
	album := sender.albums[0]
	if len(album) != len(message.Media) {
		t.Fatalf("expected %d items in the album, got %d", len(message.Media), len(album))
	}
	for i, item := range album {
		photo, ok := item.(*tele.Photo)
		if !ok {
			t.Fatalf("item %d: expected *tele.Photo, got %#v", i, item)
		}
		if photo.FileURL != message.Media[i].URL {
			t.Errorf("item %d: file URL = %q, want %q", i, photo.FileURL, message.Media[i].URL)
		}
	}
	if first, ok := album[0].(*tele.Photo); !ok || first.Caption != message.Text {
		t.Errorf("expected the caption on the first item of the album, got %#v", album[0])
	}
}

// A single-picture post still sends as one photo, not a one-item album: an album is what a
// gallery needs, not what every post with a URL gets upgraded to.
func TestTelegramPublish_SingleImageIsNotAnAlbum(t *testing.T) {
	sender := &fakeTelegramSender{}
	telegram := NewTelegram(sender)

	message := contentmodel.Message{
		Text: "one picture",
		Media: []contentmodel.Media{
			{URL: "https://lapvision.org/media/one.jpg", Kind: contentmodel.MediaImage},
		},
	}

	if _, err := telegram.Publish(context.Background(), "-100", message); err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}

	if len(sender.albums) != 0 {
		t.Fatalf("expected no album for a single picture, got %d", len(sender.albums))
	}
	if len(sender.sent) != 1 {
		t.Fatalf("expected exactly one send, got %d", len(sender.sent))
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

type fakeInstagramHTTP struct {
	requests []*http.Request
	status   int
	body     string
}

func (f *fakeInstagramHTTP) Do(req *http.Request) (*http.Response, error) {
	f.requests = append(f.requests, req)
	status := f.status
	if status == 0 {
		status = http.StatusOK
	}

	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(f.body)),
		Header:     make(http.Header),
	}, nil
}

type fakeInstagramTokenStore struct {
	tokens  []models.InstagramToken
	updated []models.InstagramToken
}

func (s *fakeInstagramTokenStore) ListInstagramTokens(context.Context) ([]models.InstagramToken, error) {
	return append([]models.InstagramToken{}, s.tokens...), nil
}

func (s *fakeInstagramTokenStore) UpdateInstagramToken(_ context.Context, token models.InstagramToken) error {
	s.updated = append(s.updated, token)
	return nil
}

func TestInstagramTokenRefresherRefreshesDueToken(t *testing.T) {
	store := &fakeInstagramTokenStore{
		tokens: []models.InstagramToken{{AccountID: "1789001", AccessToken: "old-token"}},
	}
	cache := NewInstagramTokenCache(nil)
	client := &fakeInstagramHTTP{body: `{"access_token":"new-token","token_type":"bearer","expires_in":5184000}`}
	refresher := NewInstagramTokenRefresher(store, cache, client, nil)
	now := time.Date(2026, time.September, 22, 12, 0, 0, 0, time.UTC)
	refresher.now = func() time.Time { return now }

	refresher.RefreshDue(context.Background())

	if len(client.requests) != 1 {
		t.Fatalf("refresh requests = %d, want 1", len(client.requests))
	}
	if got := client.requests[0].URL.Path; got != "/refresh_access_token" {
		t.Fatalf("path = %q, want /refresh_access_token", got)
	}
	query := client.requests[0].URL.Query()
	if query.Get("grant_type") != "ig_refresh_token" || query.Get("access_token") != "old-token" {
		t.Fatalf("unexpected query: %s", client.requests[0].URL.RawQuery)
	}

	if len(store.updated) != 1 {
		t.Fatalf("updated tokens = %d, want 1", len(store.updated))
	}
	updated := store.updated[0]
	if updated.AccountID != "1789001" || updated.AccessToken != "new-token" {
		t.Fatalf("unexpected update: %+v", updated)
	}
	if updated.ExpiresAt == nil || !updated.ExpiresAt.Equal(now.Add(5184000*time.Second)) {
		t.Fatalf("expires_at = %v, want %v", updated.ExpiresAt, now.Add(5184000*time.Second))
	}
	if updated.RefreshedAt == nil || !updated.RefreshedAt.Equal(now) {
		t.Fatalf("refreshed_at = %v, want %v", updated.RefreshedAt, now)
	}
	if token, ok := cache.Token(context.Background(), "1789001"); !ok || token != "new-token" {
		t.Fatalf("cache token = %q/%v, want new-token/true", token, ok)
	}
}

func TestInstagramTokenRefresherSkipsTokenFarFromExpiry(t *testing.T) {
	expiresAt := time.Date(2026, time.October, 30, 12, 0, 0, 0, time.UTC)
	store := &fakeInstagramTokenStore{
		tokens: []models.InstagramToken{{AccountID: "1789001", AccessToken: "current-token", ExpiresAt: &expiresAt}},
	}
	cache := NewInstagramTokenCache(nil)
	client := &fakeInstagramHTTP{body: `{"access_token":"new-token","expires_in":5184000}`}
	refresher := NewInstagramTokenRefresher(store, cache, client, nil)
	refresher.now = func() time.Time { return time.Date(2026, time.September, 22, 12, 0, 0, 0, time.UTC) }

	refresher.RefreshDue(context.Background())

	if len(client.requests) != 0 {
		t.Fatalf("refresh requests = %d, want 0", len(client.requests))
	}
	if len(store.updated) != 0 {
		t.Fatalf("updated tokens = %d, want 0", len(store.updated))
	}
	if token, ok := cache.Token(context.Background(), "1789001"); !ok || token != "current-token" {
		t.Fatalf("cache token = %q/%v, want current-token/true", token, ok)
	}
}
