package storage

const (
	sqlUpsertUser = `
		INSERT INTO users (tg_user_id, name, is_premium)
		VALUES ($1, $2, $3)
		ON CONFLICT (tg_user_id) DO UPDATE
		SET name = EXCLUDED.name, is_premium = EXCLUDED.is_premium
	`

	sqlSelectUserPreference = `
		SELECT preference_id, tg_user_id, default_championship_id, notifications_enabled
		FROM user_preferences
		WHERE tg_user_id = $1
	`

	// COALESCE keeps whatever the caller did not mean to change, so one statement serves every
	// combination of fields. The old code assembled SQL by hand and produced invalid syntax when
	// both fields were nil.
	sqlUpsertUserPreference = `
		INSERT INTO user_preferences (tg_user_id, default_championship_id, notifications_enabled)
		VALUES ($1, $2, COALESCE($3, TRUE))
		ON CONFLICT (tg_user_id) DO UPDATE
		SET default_championship_id = COALESCE($2, user_preferences.default_championship_id),
		    notifications_enabled   = COALESCE($3, user_preferences.notifications_enabled),
		    updated_at              = now()
	`

	// An empty event_types means the channel takes everything; otherwise it must list the type.
	sqlListDeliveryChannels = `
		SELECT id, channel_type AS channel, target, enabled
		FROM delivery_channels
		WHERE enabled = TRUE
		  AND (cardinality(event_types) = 0 OR $1 = ANY (event_types))
		ORDER BY id
	`

	// The insert is the idempotency check: a second delivery of the same lap_vision event id
	// changes nothing and reports that it was already known.
	sqlClaimPublication = `
		INSERT INTO publications (event_id, event_type, payload)
		VALUES ($1, $2, $3)
		ON CONFLICT (event_id) DO NOTHING
	`

	sqlMarkDelivery = `
		INSERT INTO publication_deliveries
			(event_id, channel_id, status, attempts, last_error, external_ref, delivered_at, updated_at)
		VALUES ($1, $2, $3, 1, $4, $5, $6, now())
		ON CONFLICT (event_id, channel_id) DO UPDATE
		SET status       = EXCLUDED.status,
		    attempts     = publication_deliveries.attempts + 1,
		    last_error   = EXCLUDED.last_error,
		    external_ref = EXCLUDED.external_ref,
		    delivered_at = EXCLUDED.delivered_at,
		    updated_at   = now()
	`

	// Used to skip a channel that already succeeded, so a retry of a partially delivered
	// publication does not post to it a second time.
	sqlDeliveredChannels = `
		SELECT channel_id
		FROM publication_deliveries
		WHERE event_id = $1 AND status = 'delivered'
	`
)
