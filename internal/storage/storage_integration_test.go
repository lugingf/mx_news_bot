package storage

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"mx_news_bot/internal/domain"
	"mx_news_bot/internal/models"
)

const defaultIntegrationDSN = "host=localhost port=6444 user=mx password=mxpassword dbname=mx sslmode=disable"

// Every query here runs against a real Postgres. Unit tests use fakes and cannot catch a column
// name that does not exist, which is exactly the kind of mistake these statements invite.
func testRepo(t *testing.T) *Repository {
	t.Helper()

	dsn := os.Getenv("MX_TEST_DSN")
	explicit := dsn != ""
	if dsn == "" {
		dsn = defaultIntegrationDSN
	}

	db, err := sqlx.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		if explicit {
			t.Fatalf("connect: %v", err)
		}
		t.Skipf("integration postgres unavailable: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	return New(db, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestIntegrationUserPreferencesRoundTrip(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()
	const tgUserID = int64(-987654321)

	t.Cleanup(func() {
		_, _ = repo.db.Exec(`DELETE FROM users WHERE tg_user_id = $1`, tgUserID)
	})

	if err := repo.EnsureUser(ctx, models.User{TGUserID: tgUserID, Name: "Test"}); err != nil {
		t.Fatalf("EnsureUser: %v", err)
	}
	// Upsert, so a returning user does not fail on the unique constraint.
	if err := repo.EnsureUser(ctx, models.User{TGUserID: tgUserID, Name: "Test Renamed"}); err != nil {
		t.Fatalf("EnsureUser twice: %v", err)
	}

	// A user who never set anything reads back as the default rather than an error.
	pref, err := repo.UserPreference(ctx, tgUserID)
	if err != nil {
		t.Fatalf("UserPreference: %v", err)
	}
	if !pref.NotificationsEnabled {
		t.Error("notifications should default to enabled")
	}

	champID := 42
	if err := repo.UpdateUserPreference(ctx, domain.UserPreferenceUpdate{
		TGUserID: tgUserID, DefaultChampionshipID: &champID,
	}); err != nil {
		t.Fatalf("set championship: %v", err)
	}

	// Changing only notifications must not clear the championship.
	disabled := false
	if err := repo.UpdateUserPreference(ctx, domain.UserPreferenceUpdate{
		TGUserID: tgUserID, NotificationsEnabled: &disabled,
	}); err != nil {
		t.Fatalf("set notifications: %v", err)
	}

	pref, err = repo.UserPreference(ctx, tgUserID)
	if err != nil {
		t.Fatalf("UserPreference after updates: %v", err)
	}
	if pref.DefaultChampionshipID == nil || *pref.DefaultChampionshipID != champID {
		t.Errorf("championship = %v, want %d preserved", pref.DefaultChampionshipID, champID)
	}
	if pref.NotificationsEnabled {
		t.Error("notifications should be off")
	}
}

// Both fields nil used to build invalid SQL. It must now be a no-op.
func TestIntegrationUpdateUserPreferenceWithNothingToChange(t *testing.T) {
	repo := testRepo(t)

	if err := repo.UpdateUserPreference(context.Background(), domain.UserPreferenceUpdate{TGUserID: 1}); err != nil {
		t.Errorf("an update with no fields must be a no-op, got %v", err)
	}
}

func TestIntegrationDeliveryChannelsAndPublications(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()
	const eventID = "it-event-1"

	var channelID int64
	err := repo.db.Get(&channelID, `
INSERT INTO delivery_channels (channel_type, target, title, enabled, event_types)
VALUES ('telegram', $1, 'integration', TRUE, ARRAY['standings'])
RETURNING id`, "@it_"+t.Name())
	if err != nil {
		t.Fatalf("seed channel: %v", err)
	}
	t.Cleanup(func() {
		_, _ = repo.db.Exec(`DELETE FROM publications WHERE event_id = $1`, eventID)
		_, _ = repo.db.Exec(`DELETE FROM delivery_channels WHERE id = $1`, channelID)
	})

	// The type filter is what keeps a channel that subscribed to standings from receiving
	// schedules.
	channels, err := repo.DeliveryChannelsFor(ctx, "standings")
	if err != nil {
		t.Fatalf("DeliveryChannelsFor: %v", err)
	}
	found := false
	for _, c := range channels {
		if c.ID == channelID {
			found = true
			if c.Channel != "telegram" {
				t.Errorf("channel = %q, want telegram: the channel_type column must map onto the field", c.Channel)
			}
			if c.Target == "" {
				t.Error("target came back empty")
			}
		}
	}
	if !found {
		t.Fatal("the seeded channel was not returned for its own event type")
	}

	other, err := repo.DeliveryChannelsFor(ctx, "race_result")
	if err != nil {
		t.Fatalf("DeliveryChannelsFor other type: %v", err)
	}
	for _, c := range other {
		if c.ID == channelID {
			t.Error("a channel subscribed to standings must not receive race_result")
		}
	}

	payload := json.RawMessage(`{"class":"450SX"}`)
	claimed, err := repo.ClaimPublication(ctx, eventID, "standings", payload)
	if err != nil {
		t.Fatalf("ClaimPublication: %v", err)
	}
	if !claimed {
		t.Fatal("the first claim of an event id must succeed")
	}

	again, err := repo.ClaimPublication(ctx, eventID, "standings", payload)
	if err != nil {
		t.Fatalf("ClaimPublication twice: %v", err)
	}
	if again {
		t.Error("claiming the same event id twice must report it as already known")
	}

	if err := repo.MarkDelivery(ctx, eventID, channelID, "delivered", "", "msg-1"); err != nil {
		t.Fatalf("MarkDelivery: %v", err)
	}
	delivered, err := repo.DeliveredChannelIDs(ctx, eventID)
	if err != nil {
		t.Fatalf("DeliveredChannelIDs: %v", err)
	}
	if _, ok := delivered[channelID]; !ok {
		t.Fatal("a delivered channel must be reported, or a retry would post to it again")
	}

	// A retry increments attempts instead of inserting a second row.
	if err := repo.MarkDelivery(ctx, eventID, channelID, "failed", "boom", ""); err != nil {
		t.Fatalf("MarkDelivery retry: %v", err)
	}
	var attempts int
	if err := repo.db.Get(&attempts,
		`SELECT attempts FROM publication_deliveries WHERE event_id = $1 AND channel_id = $2`,
		eventID, channelID); err != nil {
		t.Fatalf("read attempts: %v", err)
	}
	if attempts != 2 {
		t.Errorf("attempts = %d, want 2", attempts)
	}
}
