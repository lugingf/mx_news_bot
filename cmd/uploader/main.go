package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"mx_news_bot/config"
	"mx_news_bot/internal/models"
)

// Uploader handles uploading CSV data to the database
type Uploader struct {
	DB               *sqlx.DB
	ChampionshipName string
	EventName        string
	EventVenue       string
	EventDate        string
	EventRoundNumber int
}

func main() {
	championshipName := flag.String("championship", "Monster Energy AMA Supercross", "Championship name")
	eventName := flag.String("event_name", "Main Event", "Event name")
	eventVenue := flag.String("event_venue", "Stadium X", "Event venue")
	eventDate := flag.String("event_date", "2024-01-01", "Event date (YYYY-MM-DD)")
	eventRound := flag.Int("event_round", 1, "Event round number")
	csvFile := flag.String("csv_file", "output/csv/results.csv", "CSV file path")

	flag.Parse()

	cfg, err := config.New(context.Background())
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := config.OpenSQLXConn(cfg.DB)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	uploader := Uploader{
		DB:               db,
		ChampionshipName: *championshipName,
		EventName:        *eventName,
		EventVenue:       *eventVenue,
		EventDate:        *eventDate,
		EventRoundNumber: *eventRound,
	}

	if err := uploader.Upload(*csvFile); err != nil {
		log.Fatalf("Upload failed: %v", err)
	}

	fmt.Println("Data successfully loaded into the database.")
}

func (u *Uploader) Upload(csvPath string) error {
	file, err := os.Open(csvPath)
	if err != nil {
		return fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read CSV records: %w", err)
	}

	for _, record := range records[1:] {
		rider := models.SXResultsRider{
			Position: record[0],
			Number:   record[1],
			Name:     record[2],
			Hometown: record[3],
			Bike:     record[4],
			Interval: record[5],
			BestLap:  record[6],
			Team:     record[7],
		}

		riderID := insertRider(u.DB, rider.Name, rider.Hometown)
		championshipID := getOrCreateChampionship(u.DB, u.ChampionshipName, "{450SX,250SX West,KTM Junior,250SX East,Supercross Futures,250SX East/West Showdown}", 2025)
		teamID := insertRiderTeam(u.DB, riderID, rider.Team, rider.Bike, championshipID)
		eventID := insertEvent(u.DB, championshipID, u.EventName, u.EventVenue, u.EventRoundNumber, u.EventDate)
		insertResult(u.DB, championshipID, eventID, "Main Event", 1, riderID, teamID, rider.Position, rider.BestLap)
	}
	return nil
}

func insertRider(db *sqlx.DB, fullName, nationality string) int {
	var riderID int
	err := db.QueryRow(`INSERT INTO riders (full_name, nationality) VALUES ($1, $2) ON CONFLICT (full_name) DO UPDATE SET full_name=EXCLUDED.full_name RETURNING rider_id`, fullName, nationality).Scan(&riderID)
	if err != nil {
		log.Fatalf("Failed to insert rider: %v", err)
	}
	return riderID
}

func getOrCreateChampionship(db *sqlx.DB, name, classes string, year int) int {
	var championshipID int
	err := db.QueryRow(`SELECT championship_id FROM championships WHERE championship_name=$1 AND season_year=$2`, name, year).Scan(&championshipID)
	if errors.Is(err, sql.ErrNoRows) {
		err = db.QueryRow(`INSERT INTO championships (championship_name, class_names, season_year) VALUES ($1, $2, $3) RETURNING championship_id`, name, classes, year).Scan(&championshipID)
		if err != nil {
			log.Fatalf("Failed to insert championship: %v", err)
		}
	} else if err != nil {
		log.Fatalf("Failed to check championship: %v", err)
	}
	return championshipID
}

func insertRiderTeam(db *sqlx.DB, riderID int, teamName, bikeBrand string, championshipID int) int {
	var riderTeamID int
	err := db.QueryRow(`INSERT INTO rider_teams (rider_id, team_name, bike_brand, championship_id) VALUES ($1, $2, $3, $4) ON CONFLICT (rider_id, championship_id) DO UPDATE SET team_name=EXCLUDED.team_name RETURNING rider_team_id`, riderID, teamName, bikeBrand, championshipID).Scan(&riderTeamID)
	if err != nil {
		log.Fatalf("Failed to insert rider team: %v", err)
	}
	return riderTeamID
}

func insertEvent(db *sqlx.DB, championshipID int, name, venue string, roundNumber int, eventDate string) int {
	var eventID int
	err := db.QueryRow(`INSERT INTO events (championship_id, venue_name, round_number, event_date) VALUES ($1, $2, $3, $4) ON CONFLICT (championship_id, round_number) DO UPDATE SET venue_name=EXCLUDED.venue_name RETURNING event_id`, championshipID, venue, roundNumber, eventDate).Scan(&eventID)
	if err != nil {
		log.Fatalf("Failed to insert event: %v", err)
	}
	return eventID
}

func insertResult(db *sqlx.DB, championshipID, eventID int, raceType string, raceNumber, riderID, riderTeamID int, position, lapTime string) {
	_, err := db.Exec(`INSERT INTO ama_supercross_results (championship_id, event_id, race_type, race_number, rider_id, rider_team_id, position, lap_time) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`, championshipID, eventID, raceType, raceNumber, riderID, riderTeamID, position, lapTime)
	if err != nil {
		log.Fatalf("Failed to insert race result: %v", err)
	}
}
