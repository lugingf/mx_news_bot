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

	"github.com/go-chi/chi/v5"

	"mx_news_bot/internal/domain"
	"mx_news_bot/internal/models"
	"mx_news_bot/internal/publishing/contract"
)

type fakeChannels struct {
	mu      sync.Mutex
	rows    []models.DeliveryChannelRecord
	nextID  int64
	deleted []int64
}

func (f *fakeChannels) ListDeliveryChannels(context.Context) ([]models.DeliveryChannelRecord, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]models.DeliveryChannelRecord{}, f.rows...), nil
}

func (f *fakeChannels) CreateDeliveryChannel(_ context.Context, channel models.DeliveryChannelRecord) (models.DeliveryChannelRecord, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.nextID++
	channel.ID = f.nextID
	f.rows = append(f.rows, channel)

	return channel, nil
}

func (f *fakeChannels) UpdateDeliveryChannel(_ context.Context, channel models.DeliveryChannelRecord) (models.DeliveryChannelRecord, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for index, row := range f.rows {
		if row.ID == channel.ID {
			channel.Channel = row.Channel
			f.rows[index] = channel

			return channel, nil
		}
	}

	return models.DeliveryChannelRecord{}, domain.ErrChannelNotFound
}

func (f *fakeChannels) DeleteDeliveryChannel(_ context.Context, id int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	for index, row := range f.rows {
		if row.ID == id {
			f.rows = append(f.rows[:index], f.rows[index+1:]...)
			f.deleted = append(f.deleted, id)

			return nil
		}
	}

	return domain.ErrChannelNotFound
}

const channelSecret = "s3cret"

func channelRouter(store ChannelStore) chi.Router {
	handler := New(channelSecret, nil, store, slog.New(slog.NewTextHandler(io.Discard, nil)))
	router := chi.NewRouter()
	router.Get(contract.ChannelsPath, handler.Channels)
	router.Post(contract.ChannelsPath, handler.Channels)
	router.Put(contract.ChannelsPath+"/{id}", handler.Channels)
	router.Delete(contract.ChannelsPath+"/{id}", handler.Channels)

	return router
}

func signedRequest(t *testing.T, method, path, body string) *http.Request {
	t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set(contract.HeaderSignature, contract.Sign(channelSecret, []byte(body)))

	return req
}

// A numeric chat id is the only address a private Telegram channel has, so it has to survive the
// round trip exactly — no trimming of the minus sign, no parsing into a number.
func TestCreateChannelKeepsNumericChatID(t *testing.T) {
	store := &fakeChannels{}
	body := `{"channel":"telegram","target":"-1004362440814","title":"Moto","enabled":true,"disciplines":["moto"],"championships":["AMA Supercross"],"post_types":["event_result"]}`

	rec := httptest.NewRecorder()
	channelRouter(store).ServeHTTP(rec, signedRequest(t, http.MethodPost, contract.ChannelsPath, body))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var created contract.ChannelSpec
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if created.Target != "-1004362440814" {
		t.Errorf("target = %q, want the chat id unchanged", created.Target)
	}
	if created.ID == 0 {
		t.Error("the created channel came back without an id")
	}
	if len(created.Disciplines) != 1 || created.Disciplines[0] != "moto" {
		t.Errorf("disciplines = %v, want [moto]", created.Disciplines)
	}
}

// A row for a channel nothing can publish to would sit in the table looking configured while
// every delivery to it failed.
func TestCreateChannelRejectsUnknownType(t *testing.T) {
	store := &fakeChannels{}
	body := `{"channel":"carrier_pigeon","target":"@chan"}`

	rec := httptest.NewRecorder()
	channelRouter(store).ServeHTTP(rec, signedRequest(t, http.MethodPost, contract.ChannelsPath, body))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if len(store.rows) != 0 {
		t.Error("a channel with an unknown type was stored")
	}
}

func TestCreateChannelRequiresTarget(t *testing.T) {
	store := &fakeChannels{}
	body := `{"channel":"telegram","target":"   "}`

	rec := httptest.NewRecorder()
	channelRouter(store).ServeHTTP(rec, signedRequest(t, http.MethodPost, contract.ChannelsPath, body))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

// Editing is what an administrator does most: switching a channel off, or moving it to another
// discipline. The channel type is not editable, so a row cannot drift away from its history.
func TestUpdateChannelKeepsChannelType(t *testing.T) {
	store := &fakeChannels{rows: []models.DeliveryChannelRecord{{
		ID: 7, Channel: "telegram", Target: "@old", Title: "Moto", Enabled: true,
		Disciplines: []string{"moto"}, Championships: []string{"AMA Supercross"},
	}}, nextID: 7}

	body := `{"channel":"twitter","target":"@new","title":"Moto","enabled":false,"disciplines":["f1"],"championships":[],"post_types":[]}`
	rec := httptest.NewRecorder()
	channelRouter(store).ServeHTTP(rec, signedRequest(t, http.MethodPut, contract.ChannelsPath+"/7", body))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var updated contract.ChannelSpec
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if updated.Channel != "telegram" {
		t.Errorf("channel = %q, want telegram: the type must not be editable", updated.Channel)
	}
	if updated.Target != "@new" || updated.Enabled {
		t.Errorf("update did not apply: %+v", updated)
	}
	if len(updated.Disciplines) != 1 || updated.Disciplines[0] != "f1" {
		t.Errorf("disciplines = %v, want [f1]", updated.Disciplines)
	}
}

func TestUpdateMissingChannelIsNotFound(t *testing.T) {
	store := &fakeChannels{}
	body := `{"channel":"telegram","target":"@chan"}`

	rec := httptest.NewRecorder()
	channelRouter(store).ServeHTTP(rec, signedRequest(t, http.MethodPut, contract.ChannelsPath+"/404", body))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestDeleteChannel(t *testing.T) {
	store := &fakeChannels{rows: []models.DeliveryChannelRecord{{ID: 3, Channel: "telegram", Target: "@chan"}}}

	rec := httptest.NewRecorder()
	channelRouter(store).ServeHTTP(rec, signedRequest(t, http.MethodDelete, contract.ChannelsPath+"/3", ""))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if len(store.rows) != 0 {
		t.Error("the channel was not deleted")
	}
}

// The administration endpoints are as unauthenticated as the publication one without a signature:
// reaching the port must not be enough to rewrite where the channel posts.
func TestChannelsRejectUnsignedRequest(t *testing.T) {
	store := &fakeChannels{rows: []models.DeliveryChannelRecord{{ID: 1, Channel: "telegram", Target: "@chan"}}}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, contract.ChannelsPath+"/1", nil)
	channelRouter(store).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if len(store.rows) != 1 {
		t.Error("an unsigned request deleted a channel")
	}
}

func TestChannelsListReturnsEveryRowIncludingDisabled(t *testing.T) {
	store := &fakeChannels{rows: []models.DeliveryChannelRecord{
		{ID: 1, Channel: "telegram", Target: "@moto", Enabled: true, Disciplines: []string{"moto"}},
		{ID: 2, Channel: "telegram", Target: "-1004362440814", Enabled: false, Disciplines: []string{"f1"}},
	}}

	rec := httptest.NewRecorder()
	channelRouter(store).ServeHTTP(rec, signedRequest(t, http.MethodGet, contract.ChannelsPath, ""))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var list contract.ChannelList
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(list.Items) != 2 {
		t.Fatalf("got %d channels, want both the enabled and the disabled one", len(list.Items))
	}
}

// A rehearsal is a post sent to be looked at. It must reach only the rehearsal channels: trying
// out a new kind of post cannot interrupt the channels already in service, and a live post must
// never land in the rehearsal channel either.
func TestRehearsalFlagRoundTrips(t *testing.T) {
	store := &fakeChannels{}
	body := `{"channel":"telegram","target":"@lv_test","title":"Rehearsal","enabled":true,"rehearsal":true,"disciplines":[],"championships":[],"post_types":[]}`

	rec := httptest.NewRecorder()
	channelRouter(store).ServeHTTP(rec, signedRequest(t, http.MethodPost, contract.ChannelsPath, body))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var created contract.ChannelSpec
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !created.Rehearsal {
		t.Error("the channel came back as a live one")
	}
	if !store.rows[0].Rehearsal {
		t.Error("the stored row is not marked as a rehearsal channel")
	}
}
