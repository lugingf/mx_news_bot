package storage

import (
	"database/sql"
	"log/slog"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/pkg/errors"

	"mx_news_bot/internal/models"
)

type Repository struct {
	db  *sqlx.DB
	log *slog.Logger
}

const eventStatusCompleted = "completed"

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

func (r *Repository) UploadRaceResultsSMX(result models.RaceResult) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	// Insert championship if not exists
	championshipID := 0
	err = tx.Get(&championshipID, insertChampionshipQuery, result.ChampName, time.Now().Year(), pq.Array([]string{result.Class}))
	if err != nil {
		return errors.Wrap(err, "failed to insert championship")
	}

	// Insert track if not exists
	trackID := 0
	err = tx.Get(&trackID, insertTrackQuery, result.Track, result.City, result.State)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return errors.Wrap(err, "failed to insert track")
	}

	eventCode := ""
	// championship_id, round_number, track_id, eventCode, event_date, event_status
	err = tx.Get(&eventCode, insertEventQuery, championshipID, result.Round, trackID, result.EventCode, result.Date, result.Track, eventStatusCompleted)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return errors.Wrap(err, "failed to insert event")
	}

	for _, rider := range result.Results {
		// Insert rider if not exists
		riderID := 0
		err = tx.Get(&riderID, insertRiderQuery, rider.Rider)
		if err != nil {
			return errors.Wrap(err, "failed to insert rider")
		}

		// Insert rider team if not exists
		riderTeamID := 0
		err = tx.Get(&riderTeamID, insertRiderTeamQuery, riderID, rider.Team, rider.Bike, championshipID)
		if err != nil {
			return errors.Wrap(err, "failed to insert rider team")
		}

		// Insert race result
		_, err = tx.Exec(insertRaceResultQuery,
			championshipID, result.EventCode, result.EventName, result.RaceType, result.Class,
			result.Round, riderID, riderTeamID, rider.RiderNumber, rider.Bike,
			rider.Position)
		if err != nil {
			return errors.Wrap(err, "failed to insert race result")
		}
	}

	return nil
}

func getEventCode() {

}
