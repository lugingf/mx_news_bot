package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/pkg/errors"

	"mx_news_bot/internal/domain"
	"mx_news_bot/internal/models"
)

// BotBackend answers the Telegram handlers. Since lap_vision became the source of truth it holds
// no racing data and computes no points: every racing answer is a pass-through to the provider.
// What is left here is presentation policy, such as which extra buttons a format needs.
type BotBackend struct {
	results domain.ResultsProvider
	prefs   domain.PreferenceStore
	log     *slog.Logger
}

const (
	eventTypeTripleCrown = "Triple Crown"
	// EventTypeTripleCrownStandings is not a race lap_vision knows about. It is a synthetic entry
	// the bot adds so a user can ask for the combined result of a Triple Crown round.
	EventTypeTripleCrownStandings = "Triple Crown Standings"
)

func NewApp(results domain.ResultsProvider, prefs domain.PreferenceStore, log *slog.Logger) *BotBackend {
	return &BotBackend{results: results, prefs: prefs, log: log}
}

func (b *BotBackend) GetAllChampionships(ctx context.Context) ([]models.Championship, error) {
	return b.GetAllChampionshipsBySeason(ctx, time.Now().Year())
}

func (b *BotBackend) GetAllChampionshipsBySeason(ctx context.Context, season int) ([]models.Championship, error) {
	champs, err := b.results.Championships(ctx, season, false)
	if err != nil {
		return nil, errors.Wrap(err, "bot: could not get all championships")
	}

	return champs, nil
}

func (b *BotBackend) GetChampionshipsWithRacesBySeason(ctx context.Context, season int) ([]models.Championship, error) {
	champs, err := b.results.Championships(ctx, season, true)
	if err != nil {
		return nil, errors.Wrap(err, "bot: could not get championships with races")
	}

	return champs, nil
}

func (b *BotBackend) GetChampionshipClasses(ctx context.Context, champID int) ([]models.RaceClass, error) {
	classes, err := b.results.ChampionshipClasses(ctx, champID)
	if err != nil {
		return nil, errors.Wrap(err, "bot: could not get championship classes")
	}

	return classes, nil
}

func (b *BotBackend) GetAvailableSeasons(ctx context.Context) ([]int, error) {
	seasons, err := b.results.Seasons(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "bot: could not get available seasons")
	}

	return seasons, nil
}

func (b *BotBackend) GetCurrentStandings(ctx context.Context, champID int, class, region string) ([]models.Standing, error) {
	standings, err := b.results.Standings(ctx, champID, class, region)
	if err != nil {
		return nil, errors.Wrap(err, "bot: could not get standings")
	}

	return standings, nil
}

func (b *BotBackend) GetPointsDistribution(ctx context.Context, champID int) ([]models.PointsRow, error) {
	points, err := b.results.PointsDistribution(ctx, champID)
	if err != nil {
		return nil, errors.Wrap(err, "bot: could not get points distribution")
	}

	return points, nil
}

func (b *BotBackend) GetUpcomingEvents(ctx context.Context) ([]models.Event, error) {
	events, err := b.results.UpcomingEvents(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "bot: could not fetch upcoming events")
	}

	return events, nil
}

func (b *BotBackend) GetChampEvents(ctx context.Context, champID int) ([]models.Event, error) {
	events, err := b.results.ChampionshipEvents(ctx, champID)
	if err != nil {
		return nil, errors.Wrap(err, "bot: could not fetch champ events")
	}

	return events, nil
}

func (b *BotBackend) GetCompletedEvents(ctx context.Context) ([]models.Event, error) {
	return b.GetCompletedEventsBySeason(ctx, time.Now().Year())
}

func (b *BotBackend) GetCompletedEventsBySeason(ctx context.Context, season int) ([]models.Event, error) {
	events, err := b.results.CompletedEvents(ctx, season)
	if err != nil {
		return nil, errors.Wrap(err, "bot: could not fetch completed events")
	}

	return events, nil
}

// GetEventRaces lists the races a user can ask about. A Triple Crown round gets one extra entry
// per class for the combined standings, which is a button rather than a race.
func (b *BotBackend) GetEventRaces(ctx context.Context, eventID int) ([]models.EventRace, error) {
	races, err := b.results.EventRaces(ctx, eventID)
	if err != nil {
		return nil, errors.Wrap(err, "bot: could not fetch event races")
	}
	if len(races) == 0 {
		return nil, nil
	}

	if races[0].EventFormat != eventTypeTripleCrown {
		return races, nil
	}

	seen := make(map[string]struct{}, len(races))
	classes := make([]string, 0, len(races))
	for _, race := range races {
		if _, ok := seen[race.Class]; ok {
			continue
		}
		seen[race.Class] = struct{}{}
		classes = append(classes, race.Class)
	}

	// Appended in the order the classes appear, so the buttons do not shuffle between calls the
	// way ranging over a map would make them.
	for _, class := range classes {
		races = append(races, models.EventRace{
			RaceType: EventTypeTripleCrownStandings,
			EventID:  eventID,
			Class:    class,
		})
	}

	return races, nil
}

func (b *BotBackend) GetEventRaceResultByDetails(ctx context.Context, eventID int, class, raceType string) (models.RaceResult, error) {
	result, err := b.results.EventResult(ctx, eventID, class, raceType, "")
	if err != nil {
		b.log.Error("Failed to get event result", "error", err)
		return models.RaceResult{}, errors.Wrap(err, "bot: could not fetch event result")
	}
	if len(result.Results) == 0 {
		return result, errors.New("no data found")
	}

	return result, nil
}

func (b *BotBackend) GetTripleCrownStandings(ctx context.Context, eventID int, class string) ([]models.StandingsRow, models.Event, error) {
	rows, event, err := b.results.TripleCrownStandings(ctx, eventID, class)
	if err != nil {
		b.log.Error("Failed to get triple crown standings", "error", err)
		return nil, models.Event{}, errors.Wrap(err, "bot: could not fetch triple crown standings")
	}
	if len(rows) == 0 {
		return nil, event, errors.New("no data found")
	}

	return rows, event, nil
}

func (b *BotBackend) EnsureUser(ctx context.Context, user models.User) error {
	return b.prefs.EnsureUser(ctx, user)
}

func (b *BotBackend) UpdateUserPreference(ctx context.Context, update domain.UserPreferenceUpdate) error {
	return b.prefs.UpdateUserPreference(ctx, update)
}

func (b *BotBackend) UserPreference(ctx context.Context, tgUserID int64) (models.UserPreference, error) {
	return b.prefs.UserPreference(ctx, tgUserID)
}
