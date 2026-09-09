package webhook

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"mx_news_bot/internal/models"
	"mx_news_bot/internal/publishing/builder"
	"mx_news_bot/internal/publishing/channel"
	"mx_news_bot/internal/publishing/contentmodel"
	"mx_news_bot/internal/publishing/contract"
	"mx_news_bot/internal/publishing/dispatcher"
	"mx_news_bot/internal/publishing/render"
)

const secret = "shared-secret"

// stubStore mirrors the real repository: MarkDelivery records what succeeded and
// DeliveredChannelIDs reads it back, which is what makes a redelivery skip an already-posted
// channel instead of posting twice.
type stubStore struct {
	mu        sync.Mutex
	claimed   map[string]bool
	delivered map[string]map[int64]struct{}
}

func (s *stubStore) ClaimPublication(_ context.Context, id, _ string, _ []byte) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.claimed == nil {
		s.claimed = map[string]bool{}
	}
	if s.claimed[id] {
		return false, nil
	}
	s.claimed[id] = true

	return true, nil
}

func (s *stubStore) DeliveryChannelsFor(context.Context, string) ([]models.DeliveryChannel, error) {
	return []models.DeliveryChannel{{ID: 1, Channel: "telegram", Target: "@chan", Enabled: true}}, nil
}

func (s *stubStore) DeliveredChannelIDs(_ context.Context, eventID string) (map[int64]struct{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make(map[int64]struct{}, len(s.delivered[eventID]))
	for id := range s.delivered[eventID] {
		out[id] = struct{}{}
	}

	return out, nil
}

func (s *stubStore) MarkDelivery(_ context.Context, eventID string, channelID int64, status, _, _ string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if status != "delivered" {
		return nil
	}
	if s.delivered == nil {
		s.delivered = map[string]map[int64]struct{}{}
	}
	if s.delivered[eventID] == nil {
		s.delivered[eventID] = map[int64]struct{}{}
	}
	s.delivered[eventID][channelID] = struct{}{}

	return nil
}

type countingPublisher struct {
	mu    sync.Mutex
	posts int
}

func (*countingPublisher) Name() string { return "telegram" }

func (p *countingPublisher) Publish(context.Context, string, contentmodel.Message) (channel.Receipt, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.posts++

	return channel.Receipt{Channel: "telegram", Ref: "1"}, nil
}

func (p *countingPublisher) count() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.posts
}

func testHandler(t *testing.T, secretValue string) (*Handler, *countingPublisher) {
	t.Helper()

	publisher := &countingPublisher{}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	d := dispatcher.New(&stubStore{}, builder.DefaultRegistry(), log)
	d.Register(render.NewTelegram(), publisher)

	return New(secretValue, d, log), publisher
}

func validBody(t *testing.T) []byte {
	t.Helper()

	payload, err := json.Marshal(contract.EventUpcomingPayload{
		Championship: models.Championship{Name: "AMA Supercross"},
		Event:        models.Event{Name: "Anaheim 1", RoundNumber: 1},
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	body, err := json.Marshal(contract.Envelope{
		ID:      "evt-42",
		Type:    contract.TypeEventUpcoming,
		Version: contract.Version,
		Payload: payload,
	})
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}

	return body
}

func post(h *Handler, body []byte, signature, eventID string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/internal/publications", strings.NewReader(string(body)))
	if signature != "" {
		req.Header.Set(contract.HeaderSignature, signature)
	}
	if eventID != "" {
		req.Header.Set(contract.HeaderEventID, eventID)
	}
	rec := httptest.NewRecorder()
	h.Publications(rec, req)

	return rec
}

func TestWebhookAcceptsSignedRequest(t *testing.T) {
	h, publisher := testHandler(t, secret)
	body := validBody(t)

	rec := post(h, body, contract.Sign(secret, body), "evt-42")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if publisher.count() != 1 {
		t.Errorf("published %d times, want 1", publisher.count())
	}
}

// An unsigned or wrongly signed body must never reach a builder, let alone a channel.
func TestWebhookRejectsBadSignature(t *testing.T) {
	body := validBody(t)

	cases := map[string]string{
		"missing":    "",
		"wrong":      contract.Sign("not-the-secret", body),
		"other body": contract.Sign(secret, []byte(`{"id":"x"}`)),
		"not a mac":  "sha256=deadbeef",
	}

	for name, signature := range cases {
		t.Run(name, func(t *testing.T) {
			h, publisher := testHandler(t, secret)
			rec := post(h, body, signature, "evt-42")

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want 401", rec.Code)
			}
			if publisher.count() != 0 {
				t.Errorf("published %d times, want 0", publisher.count())
			}
		})
	}
}

// With no secret the endpoint is open to anyone who can reach the port, so it must refuse to
// serve at all rather than accept unverified requests.
func TestWebhookRefusesWhenSecretIsMissing(t *testing.T) {
	h, publisher := testHandler(t, "")
	body := validBody(t)

	rec := post(h, body, contract.Sign("", body), "evt-42")
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rec.Code)
	}
	if publisher.count() != 0 {
		t.Errorf("published %d times, want 0", publisher.count())
	}
}

// The header is what the sender retries on. If it disagrees with the signed body, the request is
// ambiguous and must be refused rather than guessed at.
func TestWebhookRejectsMismatchedEventID(t *testing.T) {
	h, publisher := testHandler(t, secret)
	body := validBody(t)

	rec := post(h, body, contract.Sign(secret, body), "evt-999")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
	if publisher.count() != 0 {
		t.Errorf("published %d times, want 0", publisher.count())
	}
}

func TestWebhookReportsDuplicateWithoutReposting(t *testing.T) {
	h, publisher := testHandler(t, secret)
	body := validBody(t)
	signature := contract.Sign(secret, body)

	if rec := post(h, body, signature, "evt-42"); rec.Code != http.StatusOK {
		t.Fatalf("first status = %d", rec.Code)
	}

	rec := post(h, body, signature, "evt-42")
	if rec.Code != http.StatusOK {
		t.Fatalf("second status = %d, want 200 so the sender stops retrying", rec.Code)
	}

	var response struct {
		Duplicate bool `json:"duplicate"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Duplicate {
		t.Error("the second delivery should be reported as a duplicate")
	}
	if publisher.count() != 1 {
		t.Errorf("published %d times, want exactly 1", publisher.count())
	}
}
