package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"mx_news_bot/internal/domain"
	"mx_news_bot/internal/models"
	"mx_news_bot/internal/publishing/contract"
)

const channelTimeout = 15 * time.Second

// ChannelStore is the delivery-channel bookkeeping. It is separate from the dispatcher's Store so
// the administration endpoints cannot publish anything and the dispatcher cannot edit a channel.
type ChannelStore interface {
	ListDeliveryChannels(ctx context.Context) ([]models.DeliveryChannelRecord, error)
	CreateDeliveryChannel(ctx context.Context, channel models.DeliveryChannelRecord) (models.DeliveryChannelRecord, error)
	UpdateDeliveryChannel(ctx context.Context, channel models.DeliveryChannelRecord) (models.DeliveryChannelRecord, error)
	DeleteDeliveryChannel(ctx context.Context, id int64) error
}

// knownChannels are the channel types the dispatcher has a publisher for. A row for anything else
// would be a destination nothing can ever deliver to, so it is refused on the way in.
var knownChannels = map[string]bool{"telegram": true, "twitter": true, "instagram": true}

// Channels serves the delivery-channel CRUD that the lap_vision administration screen drives.
// The rows live here because this is the service that owns the tokens; the screen lives there
// because that is where an administrator already is.
func (h *Handler) Channels(w http.ResponseWriter, r *http.Request) {
	body, ok := h.authorize(w, r)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), channelTimeout)
	defer cancel()

	switch r.Method {
	case http.MethodGet:
		h.listChannels(ctx, w)
	case http.MethodPost:
		h.createChannel(ctx, w, body)
	case http.MethodPut:
		h.updateChannel(ctx, w, r, body)
	case http.MethodDelete:
		h.deleteChannel(ctx, w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// authorize reads and verifies the body once. A request whose signature does not cover the exact
// bytes that follow is never acted on, whatever its method.
func (h *Handler) authorize(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	if h.secret == "" {
		h.log.Error("channel request rejected: webhook secret is not configured")
		http.Error(w, "webhook is not configured", http.StatusServiceUnavailable)
		return nil, false
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxBody))
	if err != nil {
		http.Error(w, "cannot read body", http.StatusBadRequest)
		return nil, false
	}

	if !contract.Verify(h.secret, body, r.Header.Get(contract.HeaderSignature)) {
		h.log.Warn("channel request rejected: bad signature", "method", r.Method)
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return nil, false
	}

	return body, true
}

func (h *Handler) listChannels(ctx context.Context, w http.ResponseWriter) {
	if h.channels == nil {
		http.Error(w, "channel administration is not configured", http.StatusServiceUnavailable)
		return
	}

	records, err := h.channels.ListDeliveryChannels(ctx)
	if err != nil {
		h.log.Error("list delivery channels failed", "err", err)
		http.Error(w, "cannot list channels", http.StatusInternalServerError)
		return
	}

	items := make([]contract.ChannelSpec, 0, len(records))
	for _, record := range records {
		items = append(items, specOf(record))
	}

	writeJSON(w, http.StatusOK, contract.ChannelList{Items: items})
}

func (h *Handler) createChannel(ctx context.Context, w http.ResponseWriter, body []byte) {
	if h.channels == nil {
		http.Error(w, "channel administration is not configured", http.StatusServiceUnavailable)
		return
	}

	spec, err := decodeSpec(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !knownChannels[spec.Channel] {
		http.Error(w, "unknown channel type "+spec.Channel, http.StatusBadRequest)
		return
	}

	created, err := h.channels.CreateDeliveryChannel(ctx, recordOf(spec))
	if err != nil {
		h.log.Error("create delivery channel failed", "err", err)
		http.Error(w, "cannot create channel", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, specOf(created))
}

func (h *Handler) updateChannel(ctx context.Context, w http.ResponseWriter, r *http.Request, body []byte) {
	if h.channels == nil {
		http.Error(w, "channel administration is not configured", http.StatusServiceUnavailable)
		return
	}

	id, err := channelID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	spec, err := decodeSpec(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	spec.ID = id

	updated, err := h.channels.UpdateDeliveryChannel(ctx, recordOf(spec))
	if errors.Is(err, domain.ErrChannelNotFound) {
		http.Error(w, "no such channel", http.StatusNotFound)
		return
	}
	if err != nil {
		h.log.Error("update delivery channel failed", "id", id, "err", err)
		http.Error(w, "cannot update channel", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, specOf(updated))
}

func (h *Handler) deleteChannel(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if h.channels == nil {
		http.Error(w, "channel administration is not configured", http.StatusServiceUnavailable)
		return
	}

	id, err := channelID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = h.channels.DeleteDeliveryChannel(ctx, id)
	if errors.Is(err, domain.ErrChannelNotFound) {
		http.Error(w, "no such channel", http.StatusNotFound)
		return
	}
	if err != nil {
		h.log.Error("delete delivery channel failed", "id", id, "err", err)
		http.Error(w, "cannot delete channel", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"deleted": id})
}

func channelID(r *http.Request) (int64, error) {
	raw := strings.TrimSpace(chi.URLParam(r, "id"))
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("channel id must be a positive number")
	}

	return id, nil
}

// decodeSpec normalises what the screen sends: a target with stray spaces, a discipline in
// capitals and an empty entry in a list are all the same row as far as delivery is concerned.
func decodeSpec(body []byte) (contract.ChannelSpec, error) {
	var spec contract.ChannelSpec
	if err := json.Unmarshal(body, &spec); err != nil {
		return contract.ChannelSpec{}, errors.New("invalid request body")
	}

	spec.Channel = strings.ToLower(strings.TrimSpace(spec.Channel))
	spec.Target = strings.TrimSpace(spec.Target)
	spec.Title = strings.TrimSpace(spec.Title)
	spec.Disciplines = cleanList(spec.Disciplines, true)
	spec.PostTypes = cleanList(spec.PostTypes, true)
	// A championship keeps its own spelling: it is matched against what the sender puts in the
	// payload, and "AMA Supercross" is how that arrives.
	spec.Championships = cleanList(spec.Championships, false)

	if spec.Target == "" {
		return contract.ChannelSpec{}, errors.New("target is required")
	}

	return spec, nil
}

func cleanList(values []string, lower bool) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if lower {
			trimmed = strings.ToLower(trimmed)
		}
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}

	return out
}

func specOf(record models.DeliveryChannelRecord) contract.ChannelSpec {
	return contract.ChannelSpec{
		ID:            record.ID,
		Channel:       record.Channel,
		Target:        record.Target,
		Title:         record.Title,
		Enabled:       record.Enabled,
		Rehearsal:     record.Rehearsal,
		Disciplines:   append([]string{}, record.Disciplines...),
		Championships: append([]string{}, record.Championships...),
		PostTypes:     append([]string{}, record.PostTypes...),
	}
}

func recordOf(spec contract.ChannelSpec) models.DeliveryChannelRecord {
	return models.DeliveryChannelRecord{
		ID:            spec.ID,
		Channel:       spec.Channel,
		Target:        spec.Target,
		Title:         spec.Title,
		Enabled:       spec.Enabled,
		Rehearsal:     spec.Rehearsal,
		Disciplines:   spec.Disciplines,
		Championships: spec.Championships,
		PostTypes:     spec.PostTypes,
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
