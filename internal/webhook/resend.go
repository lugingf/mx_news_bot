package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"mx_news_bot/internal/domain"
	"mx_news_bot/internal/models"
	"mx_news_bot/internal/publishing/contract"
)

const resendTimeout = 60 * time.Second

// ResendStore is what Resend needs to look a past publication and a channel back up by id, apart
// from the dispatcher's own matching and dedupe logic, which this deliberately bypasses.
type ResendStore interface {
	GetPublication(ctx context.Context, eventID string) (models.PublicationRecord, error)
	GetDeliveryChannel(ctx context.Context, id int64) (models.DeliveryChannel, error)
}

// Resend sends a publication the bot already has on record to one channel again, chosen by an
// administrator rather than by the channel's own filters. This is what fixes a post that never
// arrived, replaces one deleted from a channel by mistake, or simply shows a channel's real
// rendering without waiting for the next real event.
func (h *Handler) Resend(w http.ResponseWriter, r *http.Request) {
	body, ok := h.authorize(w, r)
	if !ok {
		return
	}
	if h.resend == nil {
		http.Error(w, "resend is not configured", http.StatusServiceUnavailable)
		return
	}

	eventID := strings.TrimSpace(chi.URLParam(r, "event_id"))
	if eventID == "" {
		http.Error(w, "event id is required", http.StatusBadRequest)
		return
	}

	var req struct {
		ChannelID int64 `json:"channel_id"`
	}
	if err := json.Unmarshal(body, &req); err != nil || req.ChannelID <= 0 {
		http.Error(w, "channel_id is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), resendTimeout)
	defer cancel()

	record, err := h.resend.GetPublication(ctx, eventID)
	if errors.Is(err, domain.ErrNotFound) {
		// A structured body, not just the 404 status, so a caller resending on lap_vision's behalf
		// can tell this apart from "no such channel" below instead of guessing from one status code.
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "no such publication"})
		return
	}
	if err != nil {
		h.log.Error("resend: load publication failed", "event_id", eventID, "err", err)
		http.Error(w, "cannot load publication", http.StatusInternalServerError)
		return
	}

	target, err := h.resend.GetDeliveryChannel(ctx, req.ChannelID)
	if errors.Is(err, domain.ErrChannelNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "no such channel"})
		return
	}
	if err != nil {
		h.log.Error("resend: load channel failed", "channel_id", req.ChannelID, "err", err)
		http.Error(w, "cannot load channel", http.StatusInternalServerError)
		return
	}

	envelope := contract.Envelope{ID: record.EventID, Type: record.EventType, Payload: record.Payload}

	status, ref, deliverErr := h.dispatcher.Redeliver(ctx, envelope, target)
	if status == "" {
		h.log.Error("resend failed", "event_id", eventID, "channel_id", req.ChannelID, "err", deliverErr)
		http.Error(w, "resend failed", http.StatusInternalServerError)
		return
	}

	h.log.Info("publication resent", "event_id", eventID, "channel_id", req.ChannelID, "status", status)

	writeJSON(w, http.StatusOK, map[string]any{
		"event_id":   eventID,
		"channel_id": req.ChannelID,
		"status":     status,
		"ref":        ref,
	})
}
