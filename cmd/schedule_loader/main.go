package main

import (
	"encoding/json"
	"flag"
	"log/slog"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/pkg/errors"

	"mx_news_bot/config"
)

type seasonFile struct {
	Championships []championshipInput `json:"championships"`
}

type championshipInput struct {
	Name       string       `json:"name"`
	SeasonYear int          `json:"season_year"`
	Classes    []string     `json:"classes"`
	Events     []eventInput `json:"events"`
}

type eventInput struct {
	Name            string     `json:"name"`
	Classes         string     `json:"classes"`
	VenueName       string     `json:"venue_name"`
	Round           int        `json:"round"`
	Date            string     `json:"date"`
	Format          string     `json:"format"`
	EventCode       string     `json:"event_code"`
	VenueInfoURL    string     `json:"venue_info_url"`
	SurfaceOverride string     `json:"surface_override"`
	Track           trackInput `json:"track"`
}

type trackInput struct {
	Name  string `json:"name"`
	City  string `json:"city"`
	State string `json:"state"`
}

const (
	insertChampionship = `
		INSERT INTO championships (championship_name, season_year, class_names)
		VALUES ($1, $2, $3)
		ON CONFLICT (championship_name, season_year)
		DO UPDATE SET class_names = EXCLUDED.class_names
		RETURNING id;
	`
	insertTrack = `
		INSERT INTO tracks (name, city, state)
		VALUES ($1, $2, $3)
		ON CONFLICT (name, city, state)
		DO UPDATE SET name = EXCLUDED.name
		RETURNING id;
	`
	insertEvent = `
		INSERT INTO events (
		    championship_id, name, classes, venue_name, round_number, track_id, event_date,
		    event_format, event_code, venue_info_url, surface_override, event_status
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, 'upcoming')
		ON CONFLICT (championship_id, round_number, event_code, event_date)
		DO UPDATE SET
		    name = EXCLUDED.name,
		    classes = EXCLUDED.classes,
		    venue_name = EXCLUDED.venue_name,
		    track_id = EXCLUDED.track_id,
		    event_format = EXCLUDED.event_format,
		    venue_info_url = EXCLUDED.venue_info_url,
		    surface_override = EXCLUDED.surface_override;
	`
)

func main() {
	var filePath string
	flag.StringVar(&filePath, "file", "", "path to schedule json")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	if filePath == "" {
		logger.Error("missing required -file")
		os.Exit(1)
	}

	cfg := config.New("config.json")
	db, err := config.OpenSQLXConn(cfg.DB)
	if err != nil {
		logger.Error("db connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	payload, err := loadSchedule(filePath)
	if err != nil {
		logger.Error("load schedule failed", "error", err)
		os.Exit(1)
	}

	if err := importSchedule(db, payload); err != nil {
		logger.Error("import schedule failed", "error", err)
		os.Exit(1)
	}

	logger.Info("season schedule imported", "championship_count", len(payload.Championships))
}

func loadSchedule(path string) (*seasonFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "read input file")
	}

	var payload seasonFile
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, errors.Wrap(err, "unmarshal input json")
	}

	if len(payload.Championships) == 0 {
		return nil, errors.New("no championships in input file")
	}

	return &payload, nil
}

func importSchedule(db *sqlx.DB, payload *seasonFile) error {
	tx, err := db.Beginx()
	if err != nil {
		return errors.Wrap(err, "begin tx")
	}
	defer tx.Rollback()

	for _, championship := range payload.Championships {
		champID, err := upsertChampionship(tx, championship)
		if err != nil {
			return err
		}

		for _, event := range championship.Events {
			trackID, err := upsertTrack(tx, event.Track)
			if err != nil {
				return err
			}

			eventDate, err := time.Parse("2006-01-02", event.Date)
			if err != nil {
				return errors.Wrapf(err, "parse event date %q", event.Date)
			}

			if _, err := tx.Exec(
				insertEvent,
				champID,
				event.Name,
				event.Classes,
				event.VenueName,
				event.Round,
				trackID,
				eventDate,
				event.Format,
				event.EventCode,
				event.VenueInfoURL,
				event.SurfaceOverride,
			); err != nil {
				return errors.Wrapf(err, "upsert event %q", event.Name)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "commit tx")
	}

	return nil
}

func upsertChampionship(tx *sqlx.Tx, championship championshipInput) (int, error) {
	if championship.Name == "" {
		return 0, errors.New("championship name is required")
	}
	if championship.SeasonYear == 0 {
		return 0, errors.Errorf("championship %q has invalid season year", championship.Name)
	}

	champID := 0
	if err := tx.Get(&champID, insertChampionship, championship.Name, championship.SeasonYear, pq.Array(championship.Classes)); err != nil {
		return 0, errors.Wrapf(err, "upsert championship %q", championship.Name)
	}

	return champID, nil
}

func upsertTrack(tx *sqlx.Tx, track trackInput) (int, error) {
	if track.Name == "" {
		return 0, errors.New("track name is required")
	}

	trackID := 0
	if err := tx.Get(&trackID, insertTrack, track.Name, track.City, track.State); err != nil {
		return 0, errors.Wrapf(err, "upsert track %q", track.Name)
	}

	return trackID, nil
}
