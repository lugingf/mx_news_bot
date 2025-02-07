package models

import (
	"github.com/lib/pq"
	"time"
)

type User struct {
	ID        int64  `db:"id"`
	TGUserID  int64  `db:"tg_user_id"`
	Name      string `db:"name"`
	IsPremium bool   `db:"is_premium"`
}

type StandingsRow struct {
	TotalPosition int
	RiderNumber   string
	Name          string
	Bike          string
	R1            int
	R2            int
	R3            int
	TotalPoints   int
}

type RaceResult struct {
	ChampName   string    `json:"champ_name"`
	EventName   string    `json:"event_name"`
	EventCode   string    `json:"event_code"`
	RaceType    string    `json:"race_type"`
	City        string    `json:"city"`
	State       string    `json:"state"`
	Track       string    `json:"track"`
	Date        time.Time `json:"date"`
	Round       string    `json:"round"`
	TotalRounds string    `json:"total_rounds"`
	Class       string    `json:"class"`
	Results     []Rider   `json:"results"`
}

type Rider struct {
	Position    string `json:"pos"`
	RiderNumber string `json:"rider_number"`
	Name        string `json:"rider"`
	Hometown    string `json:"hometown"`
	Bike        string `json:"bike"`
	Team        string `json:"team"`
}

// Championship model representing championship details
type Championship struct {
	ID         int            `db:"id"`
	Name       string         `db:"championship_name"`
	ClassNames pq.StringArray `db:"class_names"`
	SeasonYear int            `db:"season_year"`
}

// Event model representing motocross event details
type Event struct {
	ID               int       `db:"id"`
	ChampionshipName string    `db:"championship_name"`
	Name             string    `db:"name"`
	Classes          string    `db:"classes"`
	Stadium          string    `db:"venue_name"`
	RoundNumber      string    `db:"round_number"`
	TrackID          int       `db:"track_id"`
	Date             time.Time `db:"event_date"`
	Format           string    `db:"event_format"`
	Status           string    `db:"event_status"` // upcoming or completed
}

type RaceClass struct {
	Class  string `db:"class"`
	Region string `db:"region"`
}

type EventToCheck struct {
	ChampionshipID string    `db:"championship_id"`
	Name           string    `db:"name"`
	Classes        string    `db:"classes"`
	RoundNumber    string    `db:"round_number"`
	Format         string    `db:"event_format"`
	Date           time.Time `db:"event_date"`
}

type EventRace struct {
	EventID     int    `db:"id"`
	Class       string `db:"class"`
	RaceType    string `db:"race_type"`
	EventFormat string `db:"event_format"`
}

type Standing struct {
	RiderName string
	Points    int
}

// UserPreference model representing user preferences such as default championship and notifications
type UserPreference struct {
	PreferenceID          int   `db:"preference_id"`
	TGUserID              int64 `db:"tg_user_id"`
	DefaultChampionshipID *int  `db:"default_championship_id"`
	NotificationsEnabled  bool  `db:"notifications_enabled"`
}
