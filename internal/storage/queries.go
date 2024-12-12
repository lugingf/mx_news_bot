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
		e.event_date BETWEEN now() AND now() + interval '10 days'
	ORDER BY 
		e.event_date ASC;
	`
	sqlUpdateUserPreference = `UPDATE user_preferences SET`
)
