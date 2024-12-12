package storage

import (
	"database/sql"
	"errors"
	"log/slog"
	"strings"

	"github.com/jmoiron/sqlx"

	"mx_news_bot/internal/models"
)

type Repository struct {
	db  *sqlx.DB
	log *slog.Logger
}

// New initializes a new Repository instance
func New(db *sqlx.DB, log *slog.Logger) *Repository {
	return &Repository{db: db, log: log}
}

// GetAllChampionships fetches all available championships
func (r *Repository) GetAllChampionships() ([]models.Championship, error) {
	var championships []models.Championship
	err := r.db.Select(&championships, sqlGetAllChampionships)
	if err != nil {
		r.log.Error("Failed to fetch championships", "error", err)
		return nil, errors.New("unable to fetch championships from database")
	}
	return championships, nil
}

// GetUpcomingEvents fetches upcoming events for a given championship
func (r *Repository) GetUpcomingEvents() ([]models.Event, error) {
	var events []models.Event

	err := r.db.Select(&events, sqlGetUpcomingEvents)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		r.log.Error("Failed to fetch upcoming events", "error", err)
		return nil, errors.New("unable to fetch upcoming events from database")
	}
	return events, nil
}

// UpdateUserPreference updates the user preferences
type UserPreferenceUpdate struct {
	TGUserID              int64
	DefaultChampionshipID *int
	NotificationsEnabled  *bool
}

func (r *Repository) UpdateUserPreference(update UserPreferenceUpdate) error {
	query := sqlUpdateUserPreference
	if update.DefaultChampionshipID != nil {
		query += " default_championship_id = :default_championship_id,"
	}
	if update.NotificationsEnabled != nil {
		query += " notifications_enabled = :notifications_enabled,"
	}
	query = strings.TrimSuffix(query, ",")
	query += " WHERE tg_user_id = :tg_user_id"

	_, err := r.db.NamedExec(query, update)
	if err != nil {
		r.log.Error("Failed to update user preferences", "error", err)
		return errors.New("could not update user preferences in the database")
	}
	return nil
}
