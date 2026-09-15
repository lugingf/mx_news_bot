package dispatcher

import (
	"context"
	"sync"
	"testing"
	"time"
)

// fakeClock replaces both the clock and the waiting, so the tests state what the pacer decided
// rather than how long a machine happened to sleep.
type fakeClock struct {
	mu    sync.Mutex
	now   time.Time
	slept []time.Duration
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.now
}

func (c *fakeClock) Sleep(_ context.Context, wait time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.slept = append(c.slept, wait)
	c.now = c.now.Add(wait)

	return nil
}

func pacerWithClock(pause time.Duration) (*pacer, *fakeClock) {
	clock := &fakeClock{now: time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)}
	p := newPacer(pause)
	p.now = clock.Now
	p.sleep = clock.Sleep

	return p, clock
}

func TestFirstMessageToAChannelDoesNotWait(t *testing.T) {
	p, clock := pacerWithClock(3 * time.Second)

	release, err := p.reserve(context.Background(), paceKey("telegram", "-100"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	release()

	if len(clock.slept) != 0 {
		t.Fatalf("waited %v before the first message", clock.slept)
	}
}

// The results of one round finish within milliseconds of each other. Without this they would land
// in the same chat together and Telegram would answer 429.
func TestSecondMessageToTheSameChannelWaitsOutTheGap(t *testing.T) {
	p, clock := pacerWithClock(3 * time.Second)
	key := paceKey("telegram", "-100")

	release, _ := p.reserve(context.Background(), key)
	release()

	release, err := p.reserve(context.Background(), key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	release()

	if len(clock.slept) != 1 || clock.slept[0] != 3*time.Second {
		t.Fatalf("waited %v, want one wait of 3s", clock.slept)
	}
}

// A slow Telegram channel must not hold up Twitter, and two Telegram channels are two chats with
// two separate limits.
func TestDifferentDestinationsDoNotWaitOnEachOther(t *testing.T) {
	p, clock := pacerWithClock(3 * time.Second)

	release, _ := p.reserve(context.Background(), paceKey("telegram", "-100"))
	release()
	release, _ = p.reserve(context.Background(), paceKey("telegram", "-200"))
	release()
	release, _ = p.reserve(context.Background(), paceKey("twitter", "-100"))
	release()

	if len(clock.slept) != 0 {
		t.Fatalf("waited %v between different destinations", clock.slept)
	}
}

// Time already spent counts: a publish that took two seconds has used most of the gap, and the
// next message waits only for what is left.
func TestTimeAlreadyPassedCountsTowardsTheGap(t *testing.T) {
	p, clock := pacerWithClock(3 * time.Second)
	key := paceKey("telegram", "-100")

	release, _ := p.reserve(context.Background(), key)
	release()
	clock.mu.Lock()
	clock.now = clock.now.Add(2 * time.Second)
	clock.mu.Unlock()

	release, _ = p.reserve(context.Background(), key)
	release()

	if len(clock.slept) != 1 || clock.slept[0] != time.Second {
		t.Fatalf("waited %v, want a single second", clock.slept)
	}
}

// A retry is the case that makes a burst out of an outage, so it goes through the same gate.
func TestARetryWaitsLikeAnyOtherMessage(t *testing.T) {
	p, clock := pacerWithClock(3 * time.Second)
	key := paceKey("telegram", "-100")

	for attempt := 0; attempt < 3; attempt++ {
		release, err := p.reserve(context.Background(), key)
		if err != nil {
			t.Fatalf("attempt %d: %v", attempt, err)
		}
		release()
	}

	if len(clock.slept) != 2 {
		t.Fatalf("three attempts waited %v, want two gaps", clock.slept)
	}
}

// A destination held by another goroutine is waited for, not walked past: two callers that both
// found the channel free would publish together and defeat the point.
func TestConcurrentSendsToOneChannelAreSerialised(t *testing.T) {
	p, clock := pacerWithClock(time.Second)
	key := paceKey("telegram", "-100")

	var wg sync.WaitGroup
	for sender := 0; sender < 4; sender++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			release, err := p.reserve(context.Background(), key)
			if err != nil {
				return
			}
			release()
		}()
	}
	wg.Wait()

	clock.mu.Lock()
	defer clock.mu.Unlock()
	if len(clock.slept) != 3 {
		t.Fatalf("four senders produced %v, want three gaps", clock.slept)
	}
}

// A shutdown while waiting is not a delivery. The caller is told, and the destination is released
// rather than left locked for whatever is still running.
func TestAWaitThatIsCancelledReleasesTheChannel(t *testing.T) {
	p, _ := pacerWithClock(3 * time.Second)
	key := paceKey("telegram", "-100")

	release, _ := p.reserve(context.Background(), key)
	release()

	p.sleep = func(context.Context, time.Duration) error { return context.Canceled }
	if _, err := p.reserve(context.Background(), key); err == nil {
		t.Fatal("a cancelled wait reported success")
	}

	// The slot has to be free, or every later attempt on this channel would block for good.
	p.sleep = func(context.Context, time.Duration) error { return nil }
	done := make(chan struct{})
	go func() {
		release, err := p.reserve(context.Background(), key)
		if err == nil {
			release()
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("the channel stayed locked after a cancelled wait")
	}
}
