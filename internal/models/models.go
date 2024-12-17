package models

type User struct {
	ID        int64  `db:"id"`
	TGUserID  int64  `db:"tg_user_id"`
	Name      string `db:"name"`
	IsPremium bool   `db:"is_premium"`
}

type SXResultsRider struct {
	Position string `db:"position"`
	Number   string `db:"number"`
	Name     string `db:"name"`
	Hometown string `db:"hometown"`
	Bike     string `db:"bike"`
	Interval string `db:"interval"`
	BestLap  string `db:"best_lap"`
	Team     string `db:"team"`
}

// AI
type RaceResult struct {
	Event       string  `json:"event"`
	City        string  `json:"city"`
	Stadium     string  `json:"stadium"`
	Date        string  `json:"date"`
	Round       string  `json:"round"`
	TotalRounds string  `json:"total_rounds"`
	Class       string  `json:"class"`
	Results     []Rider `json:"results"`
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
	ID               int    `db:"event_id"`
	ChampionshipName int    `db:"championship_name"`
	Name             string `db:"name"`
	Date             string `db:"event_date"`
	Location         string `db:"location"`
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
