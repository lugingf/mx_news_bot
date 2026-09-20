package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"mx_news_bot/internal/domain"
	"mx_news_bot/internal/models"
	"mx_news_bot/internal/publishing/builder"
	"mx_news_bot/internal/publishing/channel"
	"mx_news_bot/internal/publishing/contentmodel"
	"mx_news_bot/internal/publishing/contract"
	"mx_news_bot/internal/publishing/dispatcher"
	"mx_news_bot/internal/publishing/render"
)

type stubResendStore struct {
	publications map[string]models.PublicationRecord
	channels     map[int64]models.DeliveryChannel
}

func (s *stubResendStore) GetPublication(_ context.Context, eventID string) (models.PublicationRecord, error) {
	record, ok := s.publications[eventID]
	if !ok {
		return models.PublicationRecord{}, domain.ErrNotFound
	}

	return record, nil
}

func (s *stubResendStore) GetDeliveryChannel(_ context.Context, id int64) (models.DeliveryChannel, error) {
	target, ok := s.channels[id]
	if !ok {
		return models.DeliveryChannel{}, domain.ErrChannelNotFound
	}

	return target, nil
}

func resendHandler(t *testing.T, store *stubResendStore, publisher channel.Publisher) *Handler {
	t.Helper()

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	d := dispatcher.New(&stubStore{}, builder.DefaultRegistry(), log)
	d.Register(render.NewTelegram(), publisher)

	return New(secret, d, nil, store, log)
}

func resendPublication(eventID string, t *testing.T) models.PublicationRecord {
	t.Helper()

	payload, err := json.Marshal(contract.RenderedPostPayload{
		Discipline: "moto",
		PostType:   "standings_after",
		Title:      "450SMX",
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	return models.PublicationRecord{EventID: eventID, EventType: contract.TypeRenderedPost, Payload: payload}
}

func requestWithEventID(body []byte, eventID, signature string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/internal/publications/"+eventID+"/resend", strings.NewReader(string(body)))
	req.Header.Set(contract.HeaderSignature, signature)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("event_id", eventID)

	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func TestResendSendsToTheChosenChannel(t *testing.T) {
	store := &stubResendStore{
		publications: map[string]models.PublicationRecord{"evt-1": resendPublication("evt-1", t)},
		channels:     map[int64]models.DeliveryChannel{9: {ID: 9, Channel: "telegram", Target: "@chosen", Enabled: true}},
	}
	publisher := &countingPublisher{}
	h := resendHandler(t, store, publisher)

	body := []byte(`{"channel_id":9}`)
	rr := httptest.NewRecorder()
	h.Resend(rr, requestWithEventID(body, "evt-1", contract.Sign(secret, body)))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if publisher.count() != 1 {
		t.Errorf("published %d times, want 1", publisher.count())
	}

	var response struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Status != "delivered" {
		t.Errorf("status = %q, want delivered", response.Status)
	}
}

func TestResendRejectsBadSignature(t *testing.T) {
	store := &stubResendStore{
		publications: map[string]models.PublicationRecord{"evt-1": resendPublication("evt-1", t)},
		channels:     map[int64]models.DeliveryChannel{9: {ID: 9, Channel: "telegram", Target: "@chosen", Enabled: true}},
	}
	h := resendHandler(t, store, &countingPublisher{})

	body := []byte(`{"channel_id":9}`)
	rr := httptest.NewRecorder()
	h.Resend(rr, requestWithEventID(body, "evt-1", "sha256=deadbeef"))

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}

func TestResendReportsMissingPublication(t *testing.T) {
	store := &stubResendStore{
		publications: map[string]models.PublicationRecord{},
		channels:     map[int64]models.DeliveryChannel{9: {ID: 9, Channel: "telegram", Target: "@chosen", Enabled: true}},
	}
	h := resendHandler(t, store, &countingPublisher{})

	body := []byte(`{"channel_id":9}`)
	rr := httptest.NewRecorder()
	h.Resend(rr, requestWithEventID(body, "evt-missing", contract.Sign(secret, body)))

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rr.Code)
	}
}

func TestResendReportsMissingChannel(t *testing.T) {
	store := &stubResendStore{
		publications: map[string]models.PublicationRecord{"evt-1": resendPublication("evt-1", t)},
		channels:     map[int64]models.DeliveryChannel{},
	}
	h := resendHandler(t, store, &countingPublisher{})

	body := []byte(`{"channel_id":404}`)
	rr := httptest.NewRecorder()
	h.Resend(rr, requestWithEventID(body, "evt-1", contract.Sign(secret, body)))

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rr.Code)
	}
}

func TestResendRequiresChannelID(t *testing.T) {
	store := &stubResendStore{
		publications: map[string]models.PublicationRecord{"evt-1": resendPublication("evt-1", t)},
		channels:     map[int64]models.DeliveryChannel{},
	}
	h := resendHandler(t, store, &countingPublisher{})

	body := []byte(`{}`)
	rr := httptest.NewRecorder()
	h.Resend(rr, requestWithEventID(body, "evt-1", contract.Sign(secret, body)))

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

func TestResendPropagatesPublisherFailure(t *testing.T) {
	store := &stubResendStore{
		publications: map[string]models.PublicationRecord{"evt-1": resendPublication("evt-1", t)},
		channels:     map[int64]models.DeliveryChannel{9: {ID: 9, Channel: "telegram", Target: "@chosen", Enabled: true}},
	}
	h := resendHandler(t, store, failingPublisher{err: errors.New("chat not found")})

	body := []byte(`{"channel_id":9}`)
	rr := httptest.NewRecorder()
	h.Resend(rr, requestWithEventID(body, "evt-1", contract.Sign(secret, body)))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s, want 200 carrying a failed status", rr.Code, rr.Body.String())
	}

	var response struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Status != "failed" {
		t.Errorf("status = %q, want failed", response.Status)
	}
}

type failingPublisher struct {
	err error
}

func (failingPublisher) Name() string { return "telegram" }

func (p failingPublisher) Publish(context.Context, string, contentmodel.Message) (channel.Receipt, error) {
	return channel.Receipt{}, p.err
}
