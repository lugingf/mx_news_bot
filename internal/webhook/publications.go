package webhook

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"mx_news_bot/internal/publishing/contract"
	"mx_news_bot/internal/publishing/dispatcher"
)

// maxBody caps what the receiver will read. The signature covers the whole body, so it has to be
// buffered before anything is trusted.
const (
	maxBody        = 4 << 20
	publishTimeout = 60 * time.Second
)

type Handler struct {
	secret     string
	dispatcher *dispatcher.Dispatcher
	log        *slog.Logger
}

func New(secret string, d *dispatcher.Dispatcher, log *slog.Logger) *Handler {
	return &Handler{secret: strings.TrimSpace(secret), dispatcher: d, log: log}
}

// Publications accepts a publication request from lap_vision.
//
// The body is verified before it is parsed: an unsigned request is never decoded, let alone
// posted anywhere. A repeat of an event id already seen answers 200 without publishing again, so
// the sender's at-least-once retries are safe.
func (h *Handler) Publications(w http.ResponseWriter, r *http.Request) {
	if h.secret == "" {
		h.log.Error("publication rejected: webhook secret is not configured")
		http.Error(w, "webhook is not configured", http.StatusServiceUnavailable)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxBody))
	if err != nil {
		http.Error(w, "cannot read body", http.StatusBadRequest)
		return
	}

	if !contract.Verify(h.secret, body, r.Header.Get(contract.HeaderSignature)) {
		h.log.Warn("publication rejected: bad signature", "event_id", r.Header.Get(contract.HeaderEventID))
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	var envelope contract.Envelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	// The header is the idempotency key the sender retries on; the body must agree with it.
	if headerID := strings.TrimSpace(r.Header.Get(contract.HeaderEventID)); headerID != "" {
		if envelope.ID != "" && envelope.ID != headerID {
			http.Error(w, "event id does not match signature header", http.StatusBadRequest)
			return
		}
		envelope.ID = headerID
	}

	if envelope.ID == "" || envelope.Type == "" {
		http.Error(w, "id and type are required", http.StatusBadRequest)
		return
	}

	// Dispatch outlives the request: the sender should not wait on Telegram, and a client
	// timeout must not abort a publication that is already under way.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), publishTimeout)
	defer cancel()

	result, err := h.dispatcher.Dispatch(ctx, envelope)
	if err != nil {
		h.log.Error("dispatch failed", "event_id", envelope.ID, "type", envelope.Type, "err", err)
		http.Error(w, "dispatch failed", http.StatusInternalServerError)
		return
	}

	h.log.Info("publication handled",
		"event_id", envelope.ID, "type", envelope.Type, "duplicate", result.Duplicate,
		"delivered", result.Delivered, "skipped", result.Skipped, "failed", result.Failed)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"event_id":  envelope.ID,
		"duplicate": result.Duplicate,
		"delivered": result.Delivered,
		"skipped":   result.Skipped,
		"failed":    result.Failed,
	})
}
