package storage

const (
	querySelectUser = `
	SELECT id, tg_user_id, name, birthdate, birthplace, living_place, is_premium 
		FROM users 
		WHERE tg_user_id = $1;
`
	sqlGetAllChampionships = `SELECT * FROM championships`
	sqlGetUpcomingEvents   = `
	SELECT 
		e.event_id,
		c.championship_name,
		e.venue_name AS name,
		e.event_date,
		COALESCE(t.city || ', ' || t.state, 'Unknown Location') AS location,
		'upcoming' AS event_status
	FROM 
		events e
	JOIN 
		championships c ON e.championship_id = c.championship_id
	LEFT JOIN 
		tracks t ON e.track_id = t.track_id
	WHERE 
		e.event_date BETWEEN now() AND now() + interval '8 days'
	ORDER BY 
		e.event_date ASC;
	`
	sqlUpdateUserPreference = `UPDATE user_preferences SET`
)

const (
	insertChampionshipQuery = `
		INSERT INTO championships (championship_name, season_year, class_names)
		VALUES ($1, $2, $3)
		ON CONFLICT (championship_name, season_year) DO UPDATE SET championship_name = EXCLUDED.championship_name
		RETURNING id;
	`

	insertTrackQuery = `
		INSERT INTO tracks (name, city, state)
		VALUES ($1, $2, $3)
		ON CONFLICT (name, city, state) DO NOTHING
		RETURNING id;
	`

	insertRiderQuery = `
		INSERT INTO riders (full_name)
		VALUES ($1)
		ON CONFLICT (full_name) DO UPDATE SET full_name = EXCLUDED.full_name
		RETURNING id;
	`

	insertRiderTeamQuery = `
		INSERT INTO rider_teams (rider_id, team_name, bike_brand, championship_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (rider_id, championship_id) DO UPDATE SET team_name = EXCLUDED.team_name
		RETURNING id;
	`

	insertEventQuery = `
		INSERT INTO events (championship_id, round_number, track_id, event_code, event_date, venue_name, event_status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (championship_id, round_number, event_code, event_date) DO UPDATE SET event_status = $7
		RETURNING event_code;
	`

	insertRaceResultQuery = `
		INSERT INTO ama_supercross_results (
			championship_id, event_code, event_name, race_type, class, round, 
			rider_id, rider_team_id, rider_number, bike, position
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT DO NOTHING;
	`
)

//CREATE TABLE events
//(
//id               SERIAL PRIMARY KEY,
//championship_id  INT REFERENCES championships (id),
//classes          VARCHAR(255),
//venue_name       VARCHAR(100),
//round_number     INT  NOT NULL, -- Round number in the championship
//track_id         INT,
//event_date       DATE NOT NULL,
//event_format     VARCHAR(50),   -- E.g., "Triple Crown", "Standard",
//venue_info_url   VARCHAR(255),
//surface_override VARCHAR(50),   -- Optional: specific surface type for this event
//UNIQUE (championship_id, round_number)
//);
//
//-- Alter
//ALTER TABLE events
//ADD COLUMN place              VARCHAR(100),
//ADD COLUMN weather_conditions VARCHAR(50);
//-- Optional override for track surface
//
//-- Updated Table: Events
//ALTER TABLE events
//ADD COLUMN event_status VARCHAR(50) NOT NULL DEFAULT 'upcoming',
//-- Event status (e.g., "upcoming", "result_pending", "downloaded", "completed")
//ADD COLUMN event_code   VARCHAR(10) NOT NULL DEFAULT '';
