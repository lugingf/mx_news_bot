package service

import (
	"log/slog"

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

func (b *BotBackend) GetEventResultByID(eventID int) ([]models.RaceResult, error) {
	races, err := b.repo.GetEventResultByID(eventID)
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
