package lapvision

import (
	"context"
	"os"
	"testing"
	"time"
)

// Runs the adapter against a real lap_vision. Skipped unless LV_BASE_URL is set, so it never
// breaks a normal `go test ./...`.
func TestLiveAgainstLapVision(t *testing.T) {
	base := os.Getenv("LV_BASE_URL")
	if base == "" {
		t.Skip("LV_BASE_URL not set")
	}

	c := New(base, "", 10*time.Second)
	ctx := context.Background()

	seasons, err := c.Seasons(ctx)
	if err != nil {
		t.Fatalf("Seasons: %v", err)
	}
	t.Logf("seasons=%v", seasons)

	champs, err := c.Championships(ctx, 2026, false)
	if err != nil {
		t.Fatalf("Championships: %v", err)
	}
	if len(champs) == 0 {
		t.Fatal("no championships came back for 2026")
	}
	for _, ch := range champs {
		t.Logf("champ id=%d name=%q discipline=%q classes=%v", ch.ID, ch.Name, ch.Discipline, ch.ClassNames)
		if ch.Name == "" {
			t.Errorf("championship %d decoded with an empty name: envelope or field name is wrong", ch.ID)
		}

		points, err := c.PointsDistribution(ctx, ch.ID)
		if err != nil {
			t.Fatalf("PointsDistribution(%d): %v", ch.ID, err)
		}
		if len(points) == 0 {
			t.Errorf("championship %q has no points rows", ch.Name)
		} else {
			t.Logf("  points rows=%d first=%+v", len(points), points[0])
		}

		events, err := c.ChampionshipEvents(ctx, ch.ID)
		if err != nil {
			t.Fatalf("ChampionshipEvents(%d): %v", ch.ID, err)
		}
		if len(events) == 0 {
			t.Errorf("championship %q has no events", ch.Name)
			continue
		}
		e := events[0]
		t.Logf("  events=%d first: round=%d name=%q date=%s format=%q", len(events), e.RoundNumber, e.Name, e.Date.Format("2006-01-02"), e.Format)
		if e.Name == "" || e.RoundNumber == 0 || e.Date.IsZero() {
			t.Errorf("event decoded with empty fields: %+v", e)
		}

		classes, err := c.ChampionshipClasses(ctx, ch.ID)
		if err != nil {
			t.Fatalf("ChampionshipClasses(%d): %v", ch.ID, err)
		}
		t.Logf("  classes=%+v", classes)

		if len(classes) > 0 {
			st, err := c.Standings(ctx, ch.ID, classes[0].Class, classes[0].Region)
			if err != nil {
				t.Fatalf("Standings(%d,%s): %v", ch.ID, classes[0].Class, err)
			}
			t.Logf("  standings rows=%d", len(st))
		}
	}

	if _, err := c.UpcomingEvents(ctx); err != nil {
		t.Fatalf("UpcomingEvents: %v", err)
	}
	if _, err := c.CompletedEvents(ctx, 2026); err != nil {
		t.Fatalf("CompletedEvents: %v", err)
	}
}
