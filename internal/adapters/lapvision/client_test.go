package lapvision

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mx_news_bot/internal/domain"
)

// Compile-time proof that the adapter is a drop-in for the port the handlers depend on.
var _ domain.ResultsProvider = (*Client)(nil)

type capture struct {
	path  string
	query string
}

func serve(t *testing.T, body string, got *capture) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.path = r.URL.Path
		got.query = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return server
}

// The envelope keys are lap_vision's, not the adapter's guess. A wrong key decodes to an empty
// list rather than an error, so it has to be pinned per endpoint.
func TestClientUnwrapsResponseEnvelopes(t *testing.T) {
	var got capture

	t.Run("seasons", func(t *testing.T) {
		server := serve(t, `{"seasons":[2025,2026]}`, &got)
		seasons, err := New(server.URL, "", time.Second).Seasons(context.Background())
		if err != nil {
			t.Fatalf("Seasons: %v", err)
		}
		if len(seasons) != 2 || seasons[1] != 2026 {
			t.Errorf("seasons = %v", seasons)
		}
		if got.path != "/moto/seasons" {
			t.Errorf("path = %q", got.path)
		}
	})

	t.Run("championships", func(t *testing.T) {
		server := serve(t, `{"championships":[{"id":7,"championship_name":"AMA Supercross","season_year":2026,"class_names":["450SX"]}]}`, &got)
		champs, err := New(server.URL, "", time.Second).Championships(context.Background(), 2026, true)
		if err != nil {
			t.Fatalf("Championships: %v", err)
		}
		if len(champs) != 1 || champs[0].ID != 7 || champs[0].Name != "AMA Supercross" {
			t.Fatalf("championships = %+v", champs)
		}
		if len(champs[0].ClassNames) != 1 || champs[0].ClassNames[0] != "450SX" {
			t.Errorf("class_names = %v", champs[0].ClassNames)
		}
		if got.query != "season=2026&with_races=true" {
			t.Errorf("query = %q", got.query)
		}
	})

	// The handler wraps this one as "points", not "points_distribution".
	t.Run("points distribution", func(t *testing.T) {
		server := serve(t, `{"points":[{"position":1,"points":25}]}`, &got)
		points, err := New(server.URL, "", time.Second).PointsDistribution(context.Background(), 7)
		if err != nil {
			t.Fatalf("PointsDistribution: %v", err)
		}
		if len(points) != 1 || points[0].Points != 25 {
			t.Errorf("points = %+v", points)
		}
		if got.path != "/moto/championships/7/points-distribution" {
			t.Errorf("path = %q", got.path)
		}
	})

	t.Run("standings", func(t *testing.T) {
		server := serve(t, `{"standings":[{"rider_id":3,"rider_name":"Jett Lawrence","points":100}]}`, &got)
		standings, err := New(server.URL, "", time.Second).Standings(context.Background(), 7, "450SX", "West")
		if err != nil {
			t.Fatalf("Standings: %v", err)
		}
		if len(standings) != 1 || standings[0].RiderName != "Jett Lawrence" {
			t.Errorf("standings = %+v", standings)
		}
		if got.query != "class=450SX&region=West" {
			t.Errorf("query = %q", got.query)
		}
	})

	// Results and the single event come back bare, without an envelope.
	t.Run("event results", func(t *testing.T) {
		server := serve(t, `{"champ_name":"AMA Supercross","class":"450SX","results":[{"pos":1,"rider":"Jett Lawrence","rider_number":"18"}]}`, &got)
		result, err := New(server.URL, "", time.Second).EventResult(context.Background(), 12, "450SX", "Main Event", "")
		if err != nil {
			t.Fatalf("EventResult: %v", err)
		}
		if len(result.Results) != 1 || result.Results[0].Position != 1 {
			t.Fatalf("results = %+v", result.Results)
		}
		if got.query != "class=450SX&race_type=Main+Event" {
			t.Errorf("query = %q", got.query)
		}
	})

	t.Run("triple crown standings", func(t *testing.T) {
		server := serve(t, `{"event":{"id":12,"name":"Houston"},"class":"450SX","standings":[{"total_position":1,"name":"Jett Lawrence","total_points":6}]}`, &got)
		rows, event, err := New(server.URL, "", time.Second).TripleCrownStandings(context.Background(), 12, "450SX")
		if err != nil {
			t.Fatalf("TripleCrownStandings: %v", err)
		}
		if len(rows) != 1 || rows[0].TotalPosition != 1 {
			t.Errorf("rows = %+v", rows)
		}
		if event.Name != "Houston" {
			t.Errorf("event = %+v", event)
		}
	})
}

func TestClientSendsInternalTokenWhenConfigured(t *testing.T) {
	var header string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header = r.Header.Get("X-Internal-Token")
		_, _ = w.Write([]byte(`{"seasons":[]}`))
	}))
	defer server.Close()

	if _, err := New(server.URL, "tok", time.Second).Seasons(context.Background()); err != nil {
		t.Fatalf("Seasons: %v", err)
	}
	if header != "tok" {
		t.Errorf("X-Internal-Token = %q, want tok", header)
	}

	if _, err := New(server.URL, "", time.Second).Seasons(context.Background()); err != nil {
		t.Fatalf("Seasons without token: %v", err)
	}
	if header != "" {
		t.Errorf("an empty token must not be sent, got %q", header)
	}
}

// A 404 has to be distinguishable, or the bot answers "internal error" when a user simply asked
// for an event that does not exist.
func TestClientMaps404ToNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "no such event", http.StatusNotFound)
	}))
	defer server.Close()

	_, err := New(server.URL, "", time.Second).Event(context.Background(), 999)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("err = %v, want it to wrap domain.ErrNotFound", err)
	}
}

func TestClientReportsServerErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := New(server.URL, "", time.Second).Seasons(context.Background())
	if err == nil {
		t.Fatal("a 500 must be an error")
	}
	if errors.Is(err, domain.ErrNotFound) {
		t.Error("a 500 must not look like a missing resource")
	}

	var statusErr *StatusError
	if !errors.As(err, &statusErr) || statusErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("err = %v, want a StatusError carrying 500", err)
	}
}
