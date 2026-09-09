package domain

import (
	"context"
	"errors"

	"mx_news_bot/internal/models"
)

// ErrNotFound lets a caller tell "no such event" from "the backend is broken", which the bot
// answers very differently.
var ErrNotFound = errors.New("not found")

// ResultsProvider is everything the bot needs to know about racing. It is exactly the surface the
// Telegram handlers call, so an implementation is either the lap_vision API or a test double —
// the bot itself computes nothing.
type ResultsProvider interface {
	Seasons(ctx context.Context) ([]int, error)
	Championships(ctx context.Context, season int, withRaces bool) ([]models.Championship, error)
	ChampionshipClasses(ctx context.Context, champID int) ([]models.RaceClass, error)
	ChampionshipEvents(ctx context.Context, champID int) ([]models.Event, error)
	Standings(ctx context.Context, champID int, class, region string) ([]models.Standing, error)
	PointsDistribution(ctx context.Context, champID int) ([]models.PointsRow, error)
	UpcomingEvents(ctx context.Context) ([]models.Event, error)
	CompletedEvents(ctx context.Context, season int) ([]models.Event, error)
	Event(ctx context.Context, eventID int) (models.Event, error)
	EventRaces(ctx context.Context, eventID int) ([]models.EventRace, error)
	EventResult(ctx context.Context, eventID int, class, raceType, region string) (models.RaceResult, error)
	TripleCrownStandings(ctx context.Context, eventID int, class string) ([]models.StandingsRow, models.Event, error)
}

// UserPreferenceUpdate carries only the fields a caller means to change; a nil field is left
// alone. Both nil is a no-op rather than an error, which is what the old repository got wrong.
type UserPreferenceUpdate struct {
	TGUserID              int64
	DefaultChampionshipID *int
	NotificationsEnabled  *bool
}

// PreferenceStore is the bot's own state: who its users are and what they asked for.
type PreferenceStore interface {
	EnsureUser(ctx context.Context, user models.User) error
	UserPreference(ctx context.Context, tgUserID int64) (models.UserPreference, error)
	UpdateUserPreference(ctx context.Context, update UserPreferenceUpdate) error
}
