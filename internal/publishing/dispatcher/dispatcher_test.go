package dispatcher

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"mx_news_bot/internal/models"
	"mx_news_bot/internal/publishing/builder"
	"mx_news_bot/internal/publishing/channel"
	"mx_news_bot/internal/publishing/contentmodel"
	"mx_news_bot/internal/publishing/contract"
	"mx_news_bot/internal/publishing/render"
)

type fakeStore struct {
	mu         sync.Mutex
	claimed    map[string]bool
	channels   []models.DeliveryChannel
	delivered  map[int64]struct{}
	marks      []mark
	claimCalls int
}

type mark struct {
	channelID int64
	status    string
	lastError string
	ref       string
}

func newStore(channels ...models.DeliveryChannel) *fakeStore {
	return &fakeStore{
		claimed:   map[string]bool{},
		channels:  channels,
		delivered: map[int64]struct{}{},
	}
}

func (s *fakeStore) ClaimPublication(_ context.Context, eventID, _ string, _ []byte) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.claimCalls++
	if s.claimed[eventID] {
		return false, nil
	}
	s.claimed[eventID] = true

	return true, nil
}

func (s *fakeStore) DeliveryChannelsFor(context.Context, string) ([]models.DeliveryChannel, error) {
	return s.channels, nil
}

func (s *fakeStore) DeliveredChannelIDs(context.Context, string) (map[int64]struct{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make(map[int64]struct{}, len(s.delivered))
	for id := range s.delivered {
		out[id] = struct{}{}
	}

	return out, nil
}

func (s *fakeStore) MarkDelivery(_ context.Context, _ string, channelID int64, status, lastError, ref string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.marks = append(s.marks, mark{channelID: channelID, status: status, lastError: lastError, ref: ref})
	if status == "delivered" {
		s.delivered[channelID] = struct{}{}
	}

	return nil
}

func (s *fakeStore) statusFor(channelID int64) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, m := range s.marks {
		if m.channelID == channelID {
			return m.status
		}
	}

	return ""
}

type recordingPublisher struct {
	name string
	mu   sync.Mutex
	sent []string
	err  error
}

func (p *recordingPublisher) Name() string { return p.name }

func (p *recordingPublisher) Publish(_ context.Context, target string, message contentmodel.Message) (channel.Receipt, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.err != nil {
		return channel.Receipt{}, p.err
	}
	p.sent = append(p.sent, target+":"+message.Text)

	return channel.Receipt{Channel: p.name, Ref: "ref-1"}, nil
}

func (p *recordingPublisher) count() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	return len(p.sent)
}

func testEnvelope(t *testing.T) contract.Envelope {
	t.Helper()

	payload, err := json.Marshal(contract.StandingsPayload{
		Championship: models.Championship{Name: "AMA Supercross"},
		Class:        "450SX",
		Standings: []models.Standing{
			{RiderName: "Jett Lawrence", Points: 100},
			{RiderName: "Chase Sexton", Points: 92},
		},
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	return contract.Envelope{
		ID:      "evt-1",
		Type:    contract.TypeStandings,
		Version: contract.Version,
		Payload: payload,
	}
}

func testDispatcher(store Store, publishers ...channel.Publisher) *Dispatcher {
	d := New(store, builder.DefaultRegistry(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	renderers := map[string]render.Renderer{
		"telegram":  render.NewTelegram(),
		"twitter":   render.NewTwitter(),
		"instagram": render.NewInstagram(),
	}
	for _, p := range publishers {
		d.Register(renderers[p.Name()], p)
	}

	return d
}

// One channel failing must not stop the others: a broken Twitter token cannot cost the Telegram
// audience its post.
func TestDispatchIsolatesAFailingChannel(t *testing.T) {
	store := newStore(
		models.DeliveryChannel{ID: 1, Channel: "telegram", Target: "@chan", Enabled: true},
		models.DeliveryChannel{ID: 2, Channel: "twitter", Target: "acct", Enabled: true},
	)
	telegram := &recordingPublisher{name: "telegram"}
	twitter := &recordingPublisher{name: "twitter", err: errors.New("token expired")}

	result, err := testDispatcher(store, telegram, twitter).Dispatch(context.Background(), testEnvelope(t))
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}

	if result.Delivered != 1 || result.Failed != 1 {
		t.Errorf("result = %+v, want 1 delivered and 1 failed", result)
	}
	if telegram.count() != 1 {
		t.Errorf("telegram received %d posts, want 1", telegram.count())
	}
	if got := store.statusFor(2); got != "failed" {
		t.Errorf("twitter recorded as %q, want failed", got)
	}
}

// A stub channel is not an outage. Recording it as skipped keeps the failure count meaningful.
func TestDispatchRecordsUnconfiguredChannelAsSkipped(t *testing.T) {
	store := newStore(models.DeliveryChannel{ID: 3, Channel: "twitter", Target: "acct", Enabled: true})
	stub := &recordingPublisher{name: "twitter", err: channel.ErrNotConfigured}

	result, err := testDispatcher(store, stub).Dispatch(context.Background(), testEnvelope(t))
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}

	if result.Skipped != 1 || result.Failed != 0 {
		t.Errorf("result = %+v, want 1 skipped and 0 failed", result)
	}
	if got := store.statusFor(3); got != "skipped" {
		t.Errorf("recorded as %q, want skipped", got)
	}
}

// lap_vision retries at least once, so the same event id must not produce a second post.
func TestDispatchDoesNotRepostOnRedelivery(t *testing.T) {
	store := newStore(models.DeliveryChannel{ID: 1, Channel: "telegram", Target: "@chan", Enabled: true})
	telegram := &recordingPublisher{name: "telegram"}
	d := testDispatcher(store, telegram)
	envelope := testEnvelope(t)

	first, err := d.Dispatch(context.Background(), envelope)
	if err != nil {
		t.Fatalf("first dispatch: %v", err)
	}
	if first.Duplicate || first.Delivered != 1 {
		t.Fatalf("first = %+v, want a fresh delivery", first)
	}

	second, err := d.Dispatch(context.Background(), envelope)
	if err != nil {
		t.Fatalf("second dispatch: %v", err)
	}
	if !second.Duplicate {
		t.Error("second delivery of the same event id must be reported as a duplicate")
	}
	if telegram.count() != 1 {
		t.Errorf("telegram received %d posts, want exactly 1", telegram.count())
	}
	if second.Skipped != 1 {
		t.Errorf("second = %+v, want the already-delivered channel skipped", second)
	}
}

// The post is built once and rendered per channel, so the same standings reach Telegram as a
// monospace table and Twitter as flat lines.
func TestDispatchRendersPerChannel(t *testing.T) {
	store := newStore(
		models.DeliveryChannel{ID: 1, Channel: "telegram", Target: "@chan", Enabled: true},
		models.DeliveryChannel{ID: 2, Channel: "twitter", Target: "acct", Enabled: true},
	)
	telegram := &recordingPublisher{name: "telegram"}
	twitter := &recordingPublisher{name: "twitter"}

	if _, err := testDispatcher(store, telegram, twitter).Dispatch(context.Background(), testEnvelope(t)); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}

	if !strings.Contains(telegram.sent[0], "```") {
		t.Error("the telegram message should carry a fenced table")
	}
	if strings.Contains(twitter.sent[0], "```") {
		t.Error("the twitter message must not carry a code fence")
	}
	for _, sent := range []string{telegram.sent[0], twitter.sent[0]} {
		if !strings.Contains(sent, "Jett Lawrence") {
			t.Errorf("rider missing from %q", sent)
		}
	}
}

func TestDispatchFailsOnUnknownEventType(t *testing.T) {
	store := newStore(models.DeliveryChannel{ID: 1, Channel: "telegram", Target: "@chan", Enabled: true})
	envelope := testEnvelope(t)
	envelope.Type = "something_new"

	if _, err := testDispatcher(store, &recordingPublisher{name: "telegram"}).Dispatch(context.Background(), envelope); err == nil {
		t.Error("an unknown event type must be an error, not a silently dropped publication")
	}
}
