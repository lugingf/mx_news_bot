package storage

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/pkg/errors"

	"mx_news_bot/internal/models"
)

type Repository struct {
	db  *sqlx.DB
	log *slog.Logger
}

const (
	eventStatusCompleted  = "completed"
	eventStatusProcessing = "processing"
	eventStatusUpcoming   = "upcoming"
)

// New initializes a new Repository instance
func New(db *sqlx.DB, log *slog.Logger) *Repository {
	return &Repository{db: db, log: log}
}

// GetAllChampionshipsBySeason fetches all championships for a given season.
func (r *Repository) GetAllChampionshipsBySeason(season int) ([]models.Championship, error) {
	var championships []models.Championship
	err := r.db.Select(&championships, sqlGetAllChampionships, season)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch championships from database")
	}
	return championships, nil
}

func (r *Repository) GetAvailableSeasons() ([]int, error) {
	var seasons []int

	err := r.db.Select(&seasons, sqlGetAvailableSeasons)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch available seasons from database")
	}

	return seasons, nil
}

// GetChampionshipClasses fetches all available championships
func (r *Repository) GetChampionshipClasses(champID int) ([]models.RaceClass, error) {
	var classes []models.RaceClass
	err := r.db.Select(&classes, sqlGetChampionshipClasses, champID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch championships from database")
	}
	return classes, nil
}

// GetChampionshipsWithRacesBySeason fetches championships with completed races for a given season.
func (r *Repository) GetChampionshipsWithRacesBySeason(season int) ([]models.Championship, error) {
	var championships []models.Championship
	err := r.db.Select(&championships, sqlGetChampionshipWithRaces, season)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch championships from database")
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
		return nil, errors.Wrap(err, "unable to fetch upcoming events from database")
	}
	return events, nil
}

func (r *Repository) GetChampEventsFromNow(champID int) ([]models.Event, error) {
	var events []models.Event

	err := r.db.Select(&events, sqlGetEventsByChampIDFromNow, champID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch upcoming events from database")
	}
	return events, nil
}

func (r *Repository) GetChampEvents(champID int) ([]models.Event, error) {
	var events []models.Event

	err := r.db.Select(&events, sqlGetEventsByChampID, champID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch events from database")
	}

	return events, nil
}

func (r *Repository) GetEventsToCheck(championshipName string, completedLookbackDays int) ([]models.EventToCheck, error) {
	var events []models.EventToCheck

	err := r.db.Select(&events, sqlGetEventsToCheck, championshipName, completedLookbackDays)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch events to check from database")
	}

	return events, nil
}

func (r *Repository) GetEventByID(ID int) (models.Event, error) {
	var event models.Event

	row := r.db.QueryRow(sqlGetEventByID, ID)
	err := row.Scan(
		&event.ID, &event.ChampionshipName, &event.Name, &event.Classes,
		&event.Stadium, &event.RoundNumber, &event.TrackID, &event.Date, &event.Format, &event.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return event, nil
	}

	if err != nil {
		return event, errors.Wrap(err, "unable to get event event from database")
	}

	return event, nil
}

func (r *Repository) GetTripleCrownRaceResults(eventID int, class string) (map[string]models.RaceResult, error) {
	result := make(map[string]models.RaceResult)

	rows, err := r.db.Query(sqlGetSXTripleCrownStandings, eventID, class)
	if err != nil {
		r.log.Error("Failed to execute query", "error", err)
		return nil, errors.New("unable to fetch race results from the database")
	}
	defer rows.Close()

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	for rows.Next() {
		var rider models.Rider
		var raceResult models.RaceResult

		err := rows.Scan(
			&raceResult.ChampName,
			&raceResult.EventName,
			&raceResult.EventCode,
			&raceResult.RaceType,
			&raceResult.City,
			&raceResult.State,
			&raceResult.Track,
			&raceResult.Date,
			&raceResult.Round,
			&raceResult.TotalRounds,
			&raceResult.Class,
			&rider.Position,
			&rider.RiderNumber,
			&rider.Name,
			&rider.Bike,
			&rider.Team,
		)
		if err != nil {
			r.log.Error("Failed to scan row", "error", err)
			return nil, errors.New("triple crown: error scanning race results")
		}

		if existingResult, ok := result[raceResult.RaceType]; ok {
			existingResult.Results = append(existingResult.Results, rider)
			result[raceResult.RaceType] = existingResult
		} else {
			raceResult.Results = []models.Rider{rider}
			result[raceResult.RaceType] = raceResult
		}
	}

	if err = rows.Err(); err != nil {
		r.log.Error("Row iteration error", "error", err)
		return nil, errors.New("error iterating over race results")
	}

	return result, nil
}

func (r *Repository) GetRaceResultByDetails(eventID int, class, raceType, region string) (models.RaceResult, error) {
	raceResult := models.RaceResult{}

	rows, err := r.db.Query(sqlGetSXEventResultByDetails, eventID, class, raceType, region)
	if err != nil {
		r.log.Error("Failed to execute query", "error", err)
		return raceResult, errors.New("unable to fetch race results from the database")
	}
	defer rows.Close()

	if errors.Is(err, sql.ErrNoRows) {
		return raceResult, nil
	}

	for rows.Next() {
		var rider models.Rider

		// Сканирование данных строки
		err := rows.Scan(
			&raceResult.ChampName,
			&raceResult.EventName,
			&raceResult.EventCode,
			&raceResult.RaceType,
			&raceResult.City,
			&raceResult.State,
			&raceResult.Track,
			&raceResult.Date,
			&raceResult.Round,
			&raceResult.TotalRounds,
			&raceResult.Class,
			&rider.Position,
			&rider.RiderNumber,
			&rider.Name,
			&rider.Bike,
			&rider.Team,
		)

		if err != nil {
			r.log.Error("Failed to scan row", "error", err)
			return raceResult, errors.New("error scanning race results")
		}

		raceResult.Results = append(raceResult.Results, rider)
	}

	if err = rows.Err(); err != nil {
		r.log.Error("Row iteration error", "error", err)
		return raceResult, errors.New("error iterating over race results")
	}

	return raceResult, nil
}

func (r *Repository) GetRaceKey(class, race string) string {
	return fmt.Sprintf("%s %s", class, race)
}

func (r *Repository) GetChampRoundsCount(champID int) (int, error) {
	var count int

	err := r.db.QueryRow(sqlChampRoundsCount, champID).Scan(&count)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}

	if err != nil {
		r.log.Error("Failed to fetch completed events", "error", err)
		return 0, errors.New("unable to fetch completed events from database")
	}

	return count, nil
}

func (r *Repository) GetCompletedEvents() ([]models.Event, error) {
	var events []models.Event

	err := r.db.Select(&events, sqlGetCompletedEvents)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		r.log.Error("Failed to fetch completed events", "error", err)
		return nil, errors.New("unable to fetch completed events from database")
	}

	return events, nil
}

func (r *Repository) GetCompletedEventsBySeason(season int) ([]models.Event, error) {
	var events []models.Event

	err := r.db.Select(&events, sqlGetCompletedEventsBySeason, season)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		r.log.Error("Failed to fetch completed events by season", "season", season, "error", err)
		return nil, errors.New("unable to fetch completed events by season from database")
	}

	return events, nil
}

func (r *Repository) GetEventRaces(eventID int) ([]models.EventRace, error) {
	var events []models.EventRace

	err := r.db.Select(&events, sqlGetSXEventRaces, eventID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, errors.Wrap(err, "unable to get events race list")
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
	err = tx.Get(&championshipID, insertChampionshipQuery, result.ChampName, result.Date.Year(), pq.Array([]string{result.Class}))
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
	err = tx.Get(&eventCode, insertEventQuery, championshipID, result.Round, trackID, result.EventCode, result.Date, result.Track, eventStatusUpcoming)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return errors.Wrap(err, "failed to insert event")
	}

	_, err = tx.Exec(deleteRaceResultsByRace, championshipID, result.EventCode, result.RaceType, result.Class)
	if err != nil {
		return errors.Wrap(err, "failed to delete previous race results")
	}

	for _, rider := range result.Results {
		// Insert rider if not exists
		riderID := 0
		err = tx.Get(&riderID, insertRiderQuery, rider.Name)
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

	racesUploaded := 0
	err = tx.Get(&racesUploaded, getSXEventRacesResultCount, result.EventCode, championshipID)
	if err != nil {
		return errors.Wrap(err, "failed to get race count")
	}

	if racesUploaded >= 6 {
		_, err = tx.Exec(completeEvent,
			eventStatusCompleted, championshipID, result.Round, result.EventCode)
		if err != nil {
			return errors.Wrap(err, "failed to complete event")
		}

		r.log.Warn("Event marked as COMPLETED", "event_name", result.EventName, "champ", result.ChampName)
	} else {
		r.log.Warn("Event updated but NOT completed", "event_name", result.EventName, "champ", result.ChampName)
	}

	return nil
}

// GetPointsForPosition returns the championship points for a given finishing position
// by querying the points_distribution table.
func (r *Repository) GetPointsForPosition(championshipID int, position int) (int, error) {
	var points int
	query := `SELECT points FROM points_distribution WHERE championship_id = $1 AND position = $2`
	err := r.db.Get(&points, query, championshipID, position)
	if err != nil {
		return 0, err
	}
	return points, nil
}

// The following two methods are assumed to exist. If they are not available,
// you need to implement them according to your database schema.

// GetCurrentChampionship retrieves the current championship.
func (r *Repository) GetCurrentChampionship(champID int) (*models.Championship, error) {
	// For example, one might query by current season.
	// Adjust the query as needed.
	var champ models.Championship
	query := `
SELECT id, championship_name, class_names, season_year 
	FROM championships 
	WHERE season_year = EXTRACT(YEAR FROM CURRENT_DATE)
	AND id = $1
	LIMIT 1
;
`
	err := r.db.Get(&champ, query, champID)
	if err != nil {
		return nil, err
	}
	return &champ, nil
}

// GetCompletedEventsByChampionship retrieves all completed events for a given championship.
func (r *Repository) GetCompletedEventsByChampionship(champID int) ([]models.Event, error) {
	// Adjust the query according to your schema.
	var events []models.Event
	query := `
		SELECT e.id, c.championship_name, e.name, e.classes, e.venue_name, e.round_number, e.track_id, e.event_date, e.event_format, e.event_status
		FROM events e
		INNER JOIN championships c ON c.id = e.championship_id
		WHERE c.id = $1 AND e.event_status = 'completed'
	`
	err := r.db.Select(&events, query, champID)
	if err != nil {
		return nil, err
	}
	return events, nil
}
