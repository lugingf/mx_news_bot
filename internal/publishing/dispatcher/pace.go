package dispatcher

import (
	"context"
	"sync"
	"time"
)

// How close together two messages may land in the same chat.
//
// Telegram counts per chat, not per bot, and a channel that is sent to faster than about one
// message a second starts answering 429 with a retry-after. Two results of the same round finish
// within milliseconds of each other in the fan-out below, and a retry of a failed delivery can
// arrive at any moment on top of that — so the gate is on the destination and every attempt goes
// through it, first try or fifth.
const defaultChannelPause = 3 * time.Second

// pacer keeps the last moment anything was sent to each destination.
//
// Deliberately in memory rather than in the table: what it protects against is a burst, and a
// burst does not survive a restart. Persisting it would buy nothing and would put a write in front
// of every message.
type pacer struct {
	pause time.Duration

	mu    sync.Mutex
	slots map[string]*sync.Mutex
	last  map[string]time.Time
	now   func() time.Time
	sleep func(context.Context, time.Duration) error
}

func newPacer(pause time.Duration) *pacer {
	if pause <= 0 {
		pause = defaultChannelPause
	}

	return &pacer{
		pause: pause,
		slots: map[string]*sync.Mutex{},
		last:  map[string]time.Time{},
		now:   time.Now,
		sleep: sleepUntil,
	}
}

func sleepUntil(ctx context.Context, wait time.Duration) error {
	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// reserve blocks until this destination may be written to again, and returns a function to call
// once the attempt is over.
//
// The destination is held for the whole attempt, not only for the wait: two goroutines that both
// found the channel free would otherwise publish together and defeat the point. Different
// destinations never wait on each other — a slow Telegram channel must not hold up Twitter.
func (p *pacer) reserve(ctx context.Context, key string) (func(), error) {
	p.mu.Lock()
	slot, ok := p.slots[key]
	if !ok {
		slot = &sync.Mutex{}
		p.slots[key] = slot
	}
	p.mu.Unlock()

	slot.Lock()

	p.mu.Lock()
	last := p.last[key]
	p.mu.Unlock()

	if !last.IsZero() {
		if wait := p.pause - p.now().Sub(last); wait > 0 {
			if err := p.sleep(ctx, wait); err != nil {
				slot.Unlock()

				return nil, err
			}
		}
	}

	return func() {
		// Timed from the end of the attempt rather than the start: a publish that took two
		// seconds has already spent part of the gap, and the next message must still not follow
		// the one before it too closely.
		p.mu.Lock()
		p.last[key] = p.now()
		p.mu.Unlock()
		slot.Unlock()
	}, nil
}

func paceKey(channel string, target string) string {
	return channel + "\x00" + target
}
