package dispatcher

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"mx_news_bot/internal/models"
	"mx_news_bot/internal/publishing/builder"
	"mx_news_bot/internal/publishing/channel"
	"mx_news_bot/internal/publishing/contentmodel"
	"mx_news_bot/internal/publishing/contract"
	"mx_news_bot/internal/publishing/render"
)

// Store is the delivery bookkeeping the dispatcher needs. Keeping it an interface here means the
// fan-out logic is testable without a database.
type Store interface {
	ClaimPublication(ctx context.Context, eventID, eventType string, payload []byte) (bool, error)
	DeliveryChannelsFor(ctx context.Context, match contract.Match) ([]models.DeliveryChannel, error)
	DeliveredChannelIDs(ctx context.Context, eventID string) (map[int64]struct{}, error)
	MarkDelivery(ctx context.Context, eventID string, channelID int64, status, lastError, externalRef string) error
}

// Result reports what happened to one publication, so the webhook can answer honestly instead of
// a bare 200.
type Result struct {
	Duplicate bool
	Delivered int
	Skipped   int
	Failed    int
}

type Dispatcher struct {
	store      Store
	builders   *builder.Registry
	renderers  map[string]render.Renderer
	publishers map[string]channel.Publisher
	pace       *pacer
	log        *slog.Logger
}

func New(store Store, builders *builder.Registry, log *slog.Logger) *Dispatcher {
	return NewWithPause(store, builders, 0, log)
}

// NewWithPause is New with the gap between two messages to the same destination spelled out. Zero
// takes the default, which is what a deployment that has not thought about it should get.
func NewWithPause(store Store, builders *builder.Registry, pause time.Duration, log *slog.Logger) *Dispatcher {
	return &Dispatcher{
		store:      store,
		builders:   builders,
		renderers:  make(map[string]render.Renderer),
		publishers: make(map[string]channel.Publisher),
		pace:       newPacer(pause),
		log:        log,
	}
}

// Register adds a channel. A new channel is one Renderer, one Publisher and this call; nothing
// existing changes.
func (d *Dispatcher) Register(r render.Renderer, p channel.Publisher) {
	d.renderers[r.Channel()] = r
	d.publishers[p.Name()] = p
}

// Dispatch builds the post once and fans it out. A channel that fails is recorded and does not
// stop the others; a channel that already succeeded for this event is not posted to again.
func (d *Dispatcher) Dispatch(ctx context.Context, envelope contract.Envelope) (Result, error) {
	claimed, err := d.store.ClaimPublication(ctx, envelope.ID, envelope.Type, envelope.Payload)
	if err != nil {
		return Result{}, err
	}

	post, err := d.builders.Build(envelope)
	if err != nil {
		return Result{}, err
	}

	channels, err := d.store.DeliveryChannelsFor(ctx, matchOf(envelope))
	if err != nil {
		return Result{}, err
	}

	delivered, err := d.store.DeliveredChannelIDs(ctx, envelope.ID)
	if err != nil {
		return Result{}, err
	}

	result := Result{Duplicate: !claimed}
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, target := range channels {
		if _, done := delivered[target.ID]; done {
			mu.Lock()
			result.Skipped++
			mu.Unlock()
			continue
		}

		wg.Add(1)
		go func(target models.DeliveryChannel) {
			defer wg.Done()

			status, ref, deliveryErr := d.deliver(ctx, target, post)
			if markErr := d.store.MarkDelivery(ctx, envelope.ID, target.ID, status, errText(deliveryErr), ref); markErr != nil {
				d.log.Error("record delivery failed", "event_id", envelope.ID, "channel", target.Channel, "err", markErr)
			}

			mu.Lock()
			defer mu.Unlock()
			switch status {
			case "delivered":
				result.Delivered++
			case "skipped":
				result.Skipped++
			default:
				result.Failed++
			}
		}(target)
	}

	wg.Wait()

	return result, nil
}

func (d *Dispatcher) deliver(ctx context.Context, target models.DeliveryChannel, post contentmodel.Post) (string, string, error) {
	renderer, ok := d.renderers[target.Channel]
	if !ok {
		return "failed", "", errors.New("no renderer registered for channel " + target.Channel)
	}
	publisher, ok := d.publishers[target.Channel]
	if !ok {
		return "failed", "", errors.New("no publisher registered for channel " + target.Channel)
	}

	message, err := renderer.Render(post)
	if err != nil {
		d.log.Error("render failed", "channel", target.Channel, "err", err)
		return "failed", "", err
	}

	// Nothing reaches a destination without waiting its turn — including a retry, which is the
	// case that makes a burst out of an outage.
	release, err := d.pace.reserve(ctx, paceKey(target.Channel, target.Target))
	if err != nil {
		d.log.Error("gave up waiting for the channel", "channel", target.Channel, "err", err)
		return "failed", "", err
	}
	defer release()

	receipt, err := publisher.Publish(ctx, target.Target, message)
	if errors.Is(err, channel.ErrNotConfigured) {
		// A stub is not an outage. Recording it as skipped keeps the failure count meaningful.
		d.log.Info("channel not configured, skipping", "channel", target.Channel)
		return "skipped", "", err
	}
	if err != nil {
		d.log.Error("publish failed", "channel", target.Channel, "err", err)
		return "failed", "", err
	}

	return "delivered", receipt.Ref, nil
}

// matchOf reads what the channel table filters on out of the payload rather than out of fields of
// its own: those values are already there in every payload that has them. A dimension the payload
// says nothing about stays empty, and then only a channel that does not filter on it matches.
func matchOf(envelope contract.Envelope) contract.Match {
	var probe struct {
		Discipline   string `json:"discipline"`
		Championship string `json:"championship"`
		PostType     string `json:"post_type"`
		Rehearsal    bool   `json:"rehearsal"`
	}
	if err := json.Unmarshal(envelope.Payload, &probe); err != nil {
		return contract.Match{EventType: envelope.Type}
	}

	return contract.Match{
		EventType:    envelope.Type,
		Discipline:   strings.TrimSpace(probe.Discipline),
		Championship: strings.TrimSpace(probe.Championship),
		PostType:     strings.TrimSpace(probe.PostType),
		Rehearsal:    probe.Rehearsal,
	}
}

func errText(err error) string {
	if err == nil {
		return ""
	}

	return err.Error()
}
