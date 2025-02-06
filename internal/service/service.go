package service

import (
	"fmt"
	"log/slog"
	"sort"
	"strconv"

	"github.com/pkg/errors"

	"mx_news_bot/internal/models"
	"mx_news_bot/internal/storage"
)

type BotBackend struct {
	repo *storage.Repository
	log  *slog.Logger
}

// NewApp initializes a new instance of the service layer
func NewApp(repo *storage.Repository, log *slog.Logger) *BotBackend {
	return &BotBackend{repo: repo, log: log}
}

// GetAllChampionships fetches all championships available
func (b *BotBackend) GetAllChampionships() ([]models.Championship, error) {
	championships, err := b.repo.GetAllChampionships()
	if err != nil {
		return nil, errors.Wrap(err, "bot: could not get all championships")
	}
	return championships, nil
}

// GetChampionshipClasses fetches championships available
func (b *BotBackend) GetChampionshipClasses(champID int) ([]string, error) {
	classes, err := b.repo.GetChampionshipClasses(champID)
	if err != nil {
		return nil, errors.Wrap(err, "bot: could not get championship classes")
	}

	return classes, nil
}

// GetChampionshipsWithRaces fetches championships available
func (b *BotBackend) GetChampionshipsWithRaces() ([]models.Championship, error) {
	championships, err := b.repo.GetChampionshipsWithRaces()
	if err != nil {
		return nil, errors.Wrap(err, "bot: could not get championships with races")
	}
	return championships, nil
}

func (b *BotBackend) GetCurrentStandings(champID int, class string) ([]models.Standing, error) {
	currentChampionship, err := b.repo.GetCurrentChampionship(champID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current championship: %w", err)
	}

	events, err := b.repo.GetCompletedEventsByChampionship(currentChampionship.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get completed events: %w", err)
	}

	b.log.Info("Got events for current championship", "champ_id", champID, "event_count", len(events))

	// Use rider name as the unique identifier.
	riderPoints := make(map[string]int)
	// This map stores the rider names (the key is the rider's name itself).
	riderNames := make(map[string]string)

	// Process each event.
	for _, event := range events {
		switch event.Format {
		case "Standard":
			// For standard events, use the finishing positions from the main race.
			resultsMap, err := b.repo.GetRaceResultByDetails(event.ID, class, storage.RaceTypeMainEvent)
			if err != nil {
				return nil, fmt.Errorf("failed to get race result for event %d: %w", event.ID, err)
			}

			for _, raceResult := range resultsMap {
				for _, rider := range raceResult.Results {
					pos, err := strconv.Atoi(rider.Position)
					if err != nil {
						return nil, fmt.Errorf("failed to convert position %q to int: %w", rider.Position, err)
					}

					points, err := b.repo.GetPointsForPosition(currentChampionship.ID, pos)
					if err != nil {
						return nil, fmt.Errorf("failed to get points for position %d: %w", pos, err)
					}

					riderPoints[rider.Name] += points
					riderNames[rider.Name] = rider.Name
				}
			}

		case "Triple Crown":
			// For Tripple Crown events, aggregate finishing positions from three races.
			sumPositions := make(map[string]int)
			for _, raceType := range []string{storage.RaceTypeRace1, storage.RaceTypeRace2, storage.RaceTypeRace3} {
				resultsMap, err := b.repo.GetRaceResultByDetails(event.ID, class, raceType)
				if err != nil {
					return nil, fmt.Errorf("failed to get race result for event %d race type %s: %w", event.ID, raceType, err)
				}
				for _, raceResult := range resultsMap {
					for _, rider := range raceResult.Results {
						pos, err := strconv.Atoi(rider.Position)
						if err != nil {
							return nil, fmt.Errorf("failed to convert position %q to int: %w", rider.Position, err)
						}

						sumPositions[rider.Name] += pos
						riderNames[rider.Name] = rider.Name
					}
				}
			}

			// Create a slice to rank riders based on the sum of finishing positions (lower is better).
			type riderScore struct {
				Name string
				Sum  int
			}
			var scores []riderScore
			for name, sum := range sumPositions {
				scores = append(scores, riderScore{Name: name, Sum: sum})
			}
			sort.Slice(scores, func(i, j int) bool {
				return scores[i].Sum < scores[j].Sum
			})

			// Assign championship points based on the ranking.
			for rank, rs := range scores {
				// Ranking is one-indexed.
				rankPosition := rank + 1
				points, err := b.repo.GetPointsForPosition(currentChampionship.ID, rankPosition)
				if err != nil {
					return nil, fmt.Errorf("failed to get points for rank %d: %w", rankPosition, err)
				}

				riderPoints[rs.Name] += points
			}

		default:
			// Skip events with unknown format.
			b.log.Info(fmt.Sprintf("Skipping event %d with unknown format: %s", event.ID, event.Format))
			continue
		}

		b.log.Info("Race calculated", "race", event.Name, "format", event.Format)
	}

	b.log.Info("All race calculated", "rider_count", len(riderPoints))
	// Build and sort the overall standings by total championship points (descending).
	var standings []models.Standing
	for name, pts := range riderPoints {
		standings = append(standings, models.Standing{
			RiderName: name,
			Points:    pts,
		})
	}

	sort.Slice(standings, func(i, j int) bool {
		return standings[i].Points > standings[j].Points
	})

	b.log.Info("Current Championship Standings:")
	for pos, s := range standings {
		b.log.Info(fmt.Sprintf("%d. %s - %d points", pos+1, s.RiderName, s.Points))
	}

	return standings, nil
}

func (b *BotBackend) GetUpcomingEvents() ([]models.Event, error) {
	events, err := b.repo.GetUpcomingEvents()
	if err != nil {
		return nil, errors.Wrap(err, "bot: could not fetch upcoming events")
	}

	if len(events) == 0 {
		return nil, nil
	}

	return events, nil
}

func (b *BotBackend) GetChampEvents(champID int) ([]models.Event, error) {
	events, err := b.repo.GetChampEventsFromNow(champID)
	if err != nil {
		return nil, errors.Wrap(err, "bot: could not fetch upcoming events")
	}

	if len(events) == 0 {
		return nil, nil
	}

	return events, nil
}

func (b *BotBackend) GetCompletedEvents() ([]models.Event, error) {
	events, err := b.repo.GetCompletedEvents()
	if err != nil {
		return nil, errors.Wrap(err, "bot: could not fetch completed events")
	}

	if len(events) == 0 {
		return nil, nil
	}

	return events, nil
}

func (b *BotBackend) GetEventRaces(eventID int) ([]models.EventRace, error) {
	events, err := b.repo.GetEventRaces(eventID)
	if err != nil {
		return nil, errors.Wrap(err, "bot: could not fetch event races")
	}

	if len(events) == 0 {
		return nil, nil
	}

	return events, nil
}

func (b *BotBackend) GetEventRaceResultByDetails(eventID int, class, raceType string) ([]models.RaceResult, error) {
	races, err := b.repo.GetRaceResultByDetails(eventID, class, raceType)
	if err != nil {
		b.log.Error("Failed to get event result", "error", err)
		return nil, errors.New("could not fetch event result")
	}

	if races == nil {
		return nil, errors.New("no data found")
	}

	b.log.Info("Event result fetched", "races", len(races))
	result := make([]models.RaceResult, 0, len(races))

	for _, class := range races {
		result = append(result, class)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Class != result[j].Class {
			return result[i].Class < result[j].Class
		}
		return result[i].RaceType < result[j].RaceType
	})

	return result, nil
}

func (b *BotBackend) UpdateUserPreference(update storage.UserPreferenceUpdate) error {
	err := b.repo.UpdateUserPreference(update)
	if err != nil {
		b.log.Error("Failed to update user preference", "error", err)
		return errors.New("could not update user preferences")
	}
	return nil
}
