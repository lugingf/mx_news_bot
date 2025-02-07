package storage

const (
	querySelectUser = `
	SELECT id, tg_user_id, name, birthdate, birthplace, living_place, is_premium 
		FROM users 
		WHERE tg_user_id = $1;
`
	sqlGetAllChampionships = `
	SELECT id, championship_name, class_names, season_year 
		FROM championships 
		WHERE season_year = $1
	;	
`

	sqlGetChampionshipWithRaces = `
		SELECT DISTINCT c.*
	FROM championships c
	JOIN events e ON c.id = e.championship_id
	WHERE e.event_status = 'completed' AND season_year = $1
;
`
	sqlGetChampionshipClasses = `
	WITH ama_supercross_with_region AS (
		SELECT DISTINCT 
			sr.class, 
			CASE 
				WHEN sr.class = '450SX' THEN ''
				WHEN e.classes ILIKE '%East%' THEN 'East'
				WHEN e.classes ILIKE '%West%' THEN 'West'
				ELSE ''
			END AS region
		FROM ama_supercross_results sr
		JOIN events e ON sr.event_code = e.event_code
		WHERE sr.championship_id = $1
	)
	SELECT class, region FROM ama_supercross_with_region
	
	UNION
	SELECT DISTINCT class, '' AS region FROM ama_promotocross_results WHERE championship_id = $1
	UNION
	SELECT DISTINCT class, '' AS region FROM mxgp_results WHERE championship_id = $1
	UNION
	SELECT DISTINCT class, '' AS region FROM wsx_results WHERE championship_id = $1;
	;
`

	sqlGetSXEventResultByDetails = `
SELECT 
    championships.championship_name AS champ_name,
    events.name AS event_name,
    events.event_code,
    a.race_type,
    tracks.city,
    tracks.state,
    tracks.name AS track,
    events.event_date,
    events.round_number AS round,
    (SELECT COUNT(*) FROM events WHERE championship_id = 1) AS total_rounds,
    a.class,
    a.position,
    a.rider_number,
    riders.full_name AS rider,
    a.bike,
    rider_teams.team_name AS team
FROM ama_supercross_results a
JOIN events ON events.event_code = a.event_code
JOIN tracks ON events.track_id = tracks.id
JOIN championships ON events.championship_id = championships.id
JOIN riders ON riders.id = a.rider_id
JOIN rider_teams ON a.rider_team_id = rider_teams.id
WHERE a.championship_id = 1
  AND events.id = $1
  AND a.class = $2
  AND a.race_type = $3
  AND (
        $4 = '' OR -- Если $4 пустое, выбираем все
        ($4 = 'East' AND events.classes ILIKE '%East%') OR
        ($4 = 'West' AND events.classes ILIKE '%West%')
      )
ORDER BY a.class DESC, a.event_name, events.round_number, a.position
;
`

	sqlGetSXTripleCrownStandings = `
SELECT
    championships.championship_name AS champ_name,
    events.name AS event_name,
    events.event_code,
    a.race_type,
    tracks.city,
    tracks.state,
    tracks.name AS track,
    events.event_date,
    events.round_number AS round,
    (SELECT COUNT(*) FROM events WHERE championship_id = 1) AS total_rounds,
    a.class,
    a.position,
    a.rider_number,
    riders.full_name AS rider,
    a.bike,
    rider_teams.team_name AS team
FROM ama_supercross_results a
         JOIN events ON events.event_code = a.event_code
         JOIN tracks ON events.track_id = tracks.id
         JOIN championships ON events.championship_id = championships.id
         JOIN riders ON riders.id = a.rider_id
         JOIN rider_teams ON a.rider_team_id = rider_teams.id
WHERE a.championship_id = $1
  AND events.id = $2
  AND a.class = $3
ORDER BY a.class DESC, a.event_name, events.round_number, a.position, race_type
;
`

	sqlGetSXEventRaces = `
SELECT distinct events.id, a.class, a.race_type
FROM ama_supercross_results a
         JOIN events ON events.event_code = a.event_code
         JOIN tracks ON events.track_id = tracks.id
         JOIN championships ON events.championship_id = championships.id
         JOIN riders ON riders.id = a.rider_id
         JOIN rider_teams ON a.rider_team_id = rider_teams.id
WHERE a.championship_id = 1
  AND events.id = $1
;
`
	sqlGetNextEventToCheck = `
SELECT 
    championship_id, 
    name, 
    events.classes, 
    round_number, 
    event_format,
    event_date 
FROM events 
WHERE event_status = 'upcoming' 
AND event_date <= NOW()
ORDER BY event_date LIMIT 1;
	`

	sqlGetEventFormat = `
SELECT e.event_format FROM events e WHERE e.id = $1
;
	`

	sqlGetUpcomingEvents = `
SELECT
    e.id,
    c.championship_name,
    e.name,
    e.classes,
    e.venue_name,
    e.round_number,
    e.track_id,
    e.event_date,
    event_format,
    event_status
FROM
    events e
        JOIN
    championships c ON e.championship_id = c.id
        LEFT JOIN
    tracks t ON e.track_id = t.id
WHERE
    e.event_date BETWEEN now() AND now() + interval '8 days'
ORDER BY
    e.event_date ASC;
	`

	sqlGetEventsByChampIDFromNow = `
SELECT
    e.id,
    c.championship_name,
    e.name,
    e.classes,
    e.venue_name,
    e.round_number,
    e.track_id,
    e.event_date,
    event_format,
    event_status
FROM
    events e
        JOIN
    championships c ON e.championship_id = c.id
        LEFT JOIN
    tracks t ON e.track_id = t.id
WHERE
    c.id = $1 AND
    e.event_date >= now()
ORDER BY
    e.event_date ASC;
	`

	sqlGetCompletedEvents = `
SELECT
    e.id,
    c.championship_name,
    e.name,
    e.classes,
    e.venue_name,
    e.round_number,
    e.track_id,
    e.event_date,
    event_format,
    event_status
FROM
    events e
        JOIN
    championships c ON e.championship_id = c.id
        LEFT JOIN
    tracks t ON e.track_id = t.id
WHERE
    e.event_status = 'completed'
ORDER BY
    e.event_date DESC;
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
