package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mx_news_bot/internal/models"
	"mx_news_bot/internal/publishing/contract"
)

type stubReports struct {
	asked   []string
	records []models.DeliveryRecord
	err     error
}

func (s *stubReports) ListDeliveries(_ context.Context, ids []string) ([]models.DeliveryRecord, error) {
	s.asked = ids

	return s.records, s.err
}

func reportRequest(secret string, query string) *http.Request {
	request := httptest.NewRequest(http.MethodGet, contract.DeliveriesPath+query, nil)
	request.Header.Set(contract.HeaderSignature, contract.Sign(secret, nil))

	return request
}

func TestDeliveriesReportsWhatBecameOfEachPublicationPerChannel(t *testing.T) {
	reports := &stubReports{records: []models.DeliveryRecord{{
		EventID: "e1", ChannelID: 3, Title: "Racing Hub: F1", Status: "failed", Attempts: 2,
		LastError: "telegram: failed to get HTTP URL content (400)", UpdatedAt: time.Now(),
	}}}
	handler := New("secret", nil, nil, nil, slog.Default()).WithDeliveryReports(reports)

	recorder := httptest.NewRecorder()
	handler.Deliveries(recorder, reportRequest("secret", "?ids=e1,%20e2,"))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", recorder.Code, recorder.Body.String())
	}
	if len(reports.asked) != 2 || reports.asked[0] != "e1" || reports.asked[1] != "e2" {
		t.Fatalf("asked about %v", reports.asked)
	}
	var list contract.DeliveryReportList
	if err := json.Unmarshal(recorder.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 1 || list.Items[0].Status != "failed" || list.Items[0].Channel != "Racing Hub: F1" || list.Items[0].Attempts != 2 {
		t.Fatalf("unexpected report: %+v", list)
	}
}

func TestDeliveriesRefusesAnUnsignedRequest(t *testing.T) {
	handler := New("secret", nil, nil, nil, slog.Default()).WithDeliveryReports(&stubReports{})

	recorder := httptest.NewRecorder()
	handler.Deliveries(recorder, reportRequest("another", "?ids=e1"))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", recorder.Code)
	}
}

func TestDeliveriesIsNotConfiguredWithoutAStore(t *testing.T) {
	handler := New("secret", nil, nil, nil, slog.Default())

	recorder := httptest.NewRecorder()
	handler.Deliveries(recorder, reportRequest("secret", "?ids=e1"))

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", recorder.Code)
	}
}

func TestDeliveriesAnswersAStoreFailureAs500(t *testing.T) {
	handler := New("secret", nil, nil, nil, slog.Default()).WithDeliveryReports(&stubReports{err: errors.New("db down")})

	recorder := httptest.NewRecorder()
	handler.Deliveries(recorder, reportRequest("secret", "?ids=e1"))

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
}
