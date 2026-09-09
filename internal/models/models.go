package models

import (
	"time"
)

// The racing types below mirror the lap_vision /moto/* responses. The bot owns no racing data any
// more, so they carry json tags and no db tags: nothing here is ever selected from a table.

type Championship struct {
	ID         int      `json:"id"`
	Name       string   `json:"championship_name"`
	ClassNames []string `json:"class_names"`
	SeasonYear int      `json:"season_year"`
	Discipline string   `json:"discipline"`
}

type Event struct {
	ID               int       `json:"id"`
	ChampionshipName string    `json:"championship_name"`
	Name             string    `json:"name"`
	Classes          string    `json:"classes"`
	Location         string    `json:"location"`
	Stadium          string    `json:"venue_name"`
	RoundNumber      int       `json:"round_number"`
	TrackID          int       `json:"track_id"`
	Date             time.Time `json:"event_date"`
	Format           string    `json:"event_format"`
	Status           string    `json:"event_status"`
}

type RaceClass struct {
	Class  string `json:"class"`
	Region string `json:"region"`
}

type EventRace struct {
	EventID     int    `json:"event_id"`
	Class       string `json:"class"`
	RaceType    string `json:"race_type"`
	EventFormat string `json:"event_format"`
}

type Rider struct {
	RiderID     int    `json:"rider_id"`
	Position    int    `json:"pos"`
	RiderNumber string `json:"rider_number"`
	Name        string `json:"rider"`
	Hometown    string `json:"hometown"`
	Bike        string `json:"bike"`
	Team        string `json:"team"`
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

type Standing struct {
	RiderID   int    `json:"rider_id"`
	RiderName string `json:"rider_name"`
	Points    int    `json:"points"`
}

type StandingsRow struct {
	RiderID       int    `json:"rider_id"`
	TotalPosition int    `json:"total_position"`
	RiderNumber   string `json:"rider_number"`
	Name          string `json:"name"`
	Bike          string `json:"bike"`
	R1            int    `json:"r1"`
	R2            int    `json:"r2"`
	R3            int    `json:"r3"`
	TotalPoints   int    `json:"total_points"`
}

type PointsRow struct {
	Position int `json:"position"`
	Points   int `json:"points"`
}

// The types below are the bot's own state and do come from its database.

type User struct {
	ID        int64  `db:"id"`
	TGUserID  int64  `db:"tg_user_id"`
	Name      string `db:"name"`
	IsPremium bool   `db:"is_premium"`
}

type UserPreference struct {
	PreferenceID          int   `db:"preference_id"`
	TGUserID              int64 `db:"tg_user_id"`
	DefaultChampionshipID *int  `db:"default_championship_id"`
	NotificationsEnabled  bool  `db:"notifications_enabled"`
}

// DeliveryChannel is a registered destination for published content.
type DeliveryChannel struct {
	ID      int64  `db:"id"`
	Channel string `db:"channel"`
	Target  string `db:"target"`
	Enabled bool   `db:"enabled"`
}
