package webhook

import (
	"context"
	"net/http"
	"strings"
	"time"

	"mx_news_bot/internal/models"
	"mx_news_bot/internal/publishing/contract"
)

// maxReportedEvents bounds one report: the screen asks about the publications it is showing, and a
// request for more than that is a mistake rather than a use.
const maxReportedEvents = 200

// DeliveryReportStore is the read of what became of publications.
type DeliveryReportStore interface {
	ListDeliveries(ctx context.Context, eventIDs []string) ([]models.DeliveryRecord, error)
}

// WithDeliveryReports lets the handler answer what became of a publication. Without it the report
// endpoint answers that it is not configured.
func (h *Handler) WithDeliveryReports(store DeliveryReportStore) *Handler {
	h.reports = store

	return h
}

// Deliveries answers, for the events named in the query, what became of each in every channel it
// was sent to. The screen in lap_vision shows it so that a post that did not arrive is seen
// instead of being found out by the readers.
func (h *Handler) Deliveries(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authorize(w, r); !ok {
		return
	}
	if h.reports == nil {
		http.Error(w, "delivery reports are not configured", http.StatusServiceUnavailable)
		return
	}

	ids := make([]string, 0, maxReportedEvents)
	for _, id := range strings.Split(r.URL.Query().Get("ids"), ",") {
		if trimmed := strings.TrimSpace(id); trimmed != "" {
			ids = append(ids, trimmed)
		}
	}
	if len(ids) > maxReportedEvents {
		http.Error(w, "too many ids", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	records, err := h.reports.ListDeliveries(ctx, ids)
	if err != nil {
		h.log.Error("list deliveries failed", "err", err)
		http.Error(w, "cannot list deliveries", http.StatusInternalServerError)
		return
	}

	items := make([]contract.DeliveryReport, 0, len(records))
	for _, record := range records {
		items = append(items, contract.DeliveryReport{
			EventID:     record.EventID,
			ChannelID:   record.ChannelID,
			Channel:     record.Title,
			Rehearsal:   record.Rehearsal,
			Status:      record.Status,
			Attempts:    record.Attempts,
			Error:       record.LastError,
			DeliveredAt: record.DeliveredAt,
			UpdatedAt:   record.UpdatedAt,
		})
	}

	writeJSON(w, http.StatusOK, contract.DeliveryReportList{Items: items})
}
