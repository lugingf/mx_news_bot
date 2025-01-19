package models

import "time"

type User struct {
	ID        int64  `db:"id"`
	TGUserID  int64  `db:"tg_user_id"`
	Name      string `db:"name"`
	IsPremium bool   `db:"is_premium"`
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
	Rider       string `json:"rider"`
	Hometown    string `json:"hometown"`
	Bike        string `json:"bike"`
	Team        string `json:"team"`
}

// Championship model representing championship details
type Championship struct {
	ID           int    `db:"championship_id"`
	Name         string `db:"name"`
	DefaultClass string `db:"default_class"`
	Description  string `db:"description"`
}

// Event model representing motocross event details
type Event struct {
	ID               int    `db:"id"`
	ChampionshipName string `db:"championship_name"`
	Name             string `db:"name"`
	Classes          string `db:"classes"`
	Stadium          string `db:"venue_name"`
	RoundNumber      string `db:"round_number"`
	TrackID          int    `db:"track_id"`
	Date             string `db:"event_date"`
	Status           string `db:"event_status"` // upcoming or completed
}

// UserPreference model representing user preferences such as default championship and notifications
type UserPreference struct {
	PreferenceID          int   `db:"preference_id"`
	TGUserID              int64 `db:"tg_user_id"`
	DefaultChampionshipID *int  `db:"default_championship_id"`
	NotificationsEnabled  bool  `db:"notifications_enabled"`
}

// EventResult model representing motocross event results for riders
type EventResult struct {
	ResultID      int    `db:"result_id"`
	EventID       int    `db:"event_id"`
	ClassName     string `db:"class_name"`
	RiderID       int    `db:"rider_id"`
	Position      int    `db:"position"`
	PointsAwarded int    `db:"points_awarded"`
}
