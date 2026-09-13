package publishing

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"mx_news_bot/config"
	"mx_news_bot/internal/models"
)

type declarerStub struct {
	declared []models.DeliveryChannelRecord
	err      error
}

func (d *declarerStub) DeclareDeliveryChannel(_ context.Context, channel models.DeliveryChannelRecord) (models.DeliveryChannelRecord, error) {
	if d.err != nil {
		return models.DeliveryChannelRecord{}, d.err
	}
	channel.ID = int64(len(d.declared) + 1)
	d.declared = append(d.declared, channel)

	return channel, nil
}

func quietLog() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// The config is where a deployment's channels are written down, so what it says has to reach the
// table on every start — nobody should have to remember which INSERT was run where.
func TestDeclaredChannelsReachTheTable(t *testing.T) {
	store := &declarerStub{}
	channels := []config.DeliveryChannel{
		{Channel: "telegram", Target: "-1004362440814", Title: "TestChannel", Enabled: true, Rehearsal: true},
		{Channel: "telegram", Target: "-1004331483764", Title: "Racing Hub: Moto", Enabled: true, Disciplines: []string{"moto"}},
	}

	if err := DeclareChannels(context.Background(), store, channels, quietLog()); err != nil {
		t.Fatalf("DeclareChannels: %v", err)
	}

	if len(store.declared) != 2 {
		t.Fatalf("declared %d channels, want 2", len(store.declared))
	}
	// A numeric chat id is the only address a private channel has, so it must survive verbatim.
	if store.declared[0].Target != "-1004362440814" {
		t.Errorf("target = %q, want the chat id unchanged", store.declared[0].Target)
	}
	if !store.declared[0].Rehearsal {
		t.Error("the rehearsal channel was declared as a live one")
	}
	if store.declared[1].Rehearsal {
		t.Error("a live channel was declared as a rehearsal one")
	}
}

// A discipline is matched against what this service produces, so its spelling is ours to fold. A
// championship is matched against what the sender writes in the payload, and "AMA Supercross" is
// how that arrives.
func TestDeclaredFiltersKeepTheSpellingThatIsMatched(t *testing.T) {
	store := &declarerStub{}
	channels := []config.DeliveryChannel{{
		Channel:       "  Telegram ",
		Target:        " @chan ",
		Enabled:       true,
		Disciplines:   []string{" MOTO ", ""},
		Championships: []string{" AMA Supercross "},
		PostTypes:     []string{"Event_Result"},
	}}

	if err := DeclareChannels(context.Background(), store, channels, quietLog()); err != nil {
		t.Fatalf("DeclareChannels: %v", err)
	}

	declared := store.declared[0]
	if declared.Channel != "telegram" || declared.Target != "@chan" {
		t.Errorf("channel = %q target = %q", declared.Channel, declared.Target)
	}
	if len(declared.Disciplines) != 1 || declared.Disciplines[0] != "moto" {
		t.Errorf("disciplines = %v, want [moto] with the empty entry dropped", declared.Disciplines)
	}
	if len(declared.Championships) != 1 || declared.Championships[0] != "AMA Supercross" {
		t.Errorf("championships = %v, want the published spelling kept", declared.Championships)
	}
	if len(declared.PostTypes) != 1 || declared.PostTypes[0] != "event_result" {
		t.Errorf("post types = %v, want [event_result]", declared.PostTypes)
	}
}

// A destination nothing can deliver to would sit in the table looking configured while every post
// to it failed, so it stops the start instead.
func TestDeclaringRefusesAChannelNothingCanDeliverTo(t *testing.T) {
	cases := map[string]config.DeliveryChannel{
		"unknown type": {Channel: "carrier_pigeon", Target: "@chan"},
		"no target":    {Channel: "telegram", Target: "   "},
	}

	for name, channel := range cases {
		store := &declarerStub{}
		err := DeclareChannels(context.Background(), store, []config.DeliveryChannel{channel}, quietLog())
		if err == nil {
			t.Errorf("%s was accepted", name)
		}
		if len(store.declared) != 0 {
			t.Errorf("%s reached the table", name)
		}
	}
}

// Half the channels written and the rest missing is worse than not starting: the bot would publish
// to some destinations and quietly not to the others.
func TestDeclaringStopsOnTheFirstFailure(t *testing.T) {
	store := &declarerStub{err: errors.New("database is down")}
	channels := []config.DeliveryChannel{
		{Channel: "telegram", Target: "@one", Enabled: true},
		{Channel: "telegram", Target: "@two", Enabled: true},
	}

	err := DeclareChannels(context.Background(), store, channels, quietLog())
	if err == nil {
		t.Fatal("a failing database was reported as a successful start")
	}
	if !strings.Contains(err.Error(), "@one") {
		t.Errorf("error = %v, want it to name the channel that failed", err)
	}
}

func TestDeclaringNothingIsFine(t *testing.T) {
	store := &declarerStub{}
	if err := DeclareChannels(context.Background(), store, nil, quietLog()); err != nil {
		t.Errorf("a config with no channels must not fail: %v", err)
	}
}
