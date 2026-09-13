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

	// Each filter is "empty means everything", and they are combined with AND. A channel is
	// therefore described by what it narrows down — a discipline, a championship, a kind of post —
	// and a channel that narrows nothing, the rehearsal one, receives every publication.
	sqlListDeliveryChannels = `
		SELECT id, channel_type AS channel, target, enabled
		FROM delivery_channels
		WHERE enabled = TRUE
		  AND (cardinality(event_types) = 0   OR $1 = ANY (event_types))
		  AND (cardinality(disciplines) = 0   OR $2 = ANY (disciplines))
		  AND (cardinality(championships) = 0 OR $3 = ANY (championships))
		  AND (cardinality(post_types) = 0    OR $4 = ANY (post_types))
		  AND rehearsal = $5
		ORDER BY id
	`

	// The administration screen sees every channel, including the disabled ones: a channel that
	// has been switched off still has to be findable to be switched back on.
	sqlListAllDeliveryChannels = `
		SELECT id, channel_type, target, title, enabled, rehearsal, disciplines, championships, post_types
		FROM delivery_channels
		ORDER BY channel_type, id
	`

	sqlInsertDeliveryChannel = `
		INSERT INTO delivery_channels (channel_type, target, title, enabled, rehearsal, disciplines, championships, post_types)
		VALUES ($1, $2, $3, $4, $5, COALESCE($6::text[], '{}'), COALESCE($7::text[], '{}'), COALESCE($8::text[], '{}'))
		RETURNING id, channel_type, target, title, enabled, rehearsal, disciplines, championships, post_types
	`

	// The declared form of a channel. A deployment describes its channels in the config, and this
	// writes that description into the table on every start — so the two never drift, and nobody
	// has to remember which INSERT was run where.
	// COALESCE on the filters: a list nobody set arrives as NULL, and these columns are NOT NULL.
	// An unset filter means "everything", which is the empty array, so it is written as one.
	sqlUpsertDeliveryChannelByTarget = `
		INSERT INTO delivery_channels (channel_type, target, title, enabled, rehearsal, disciplines, championships, post_types)
		VALUES ($1, $2, $3, $4, $5, COALESCE($6::text[], '{}'), COALESCE($7::text[], '{}'), COALESCE($8::text[], '{}'))
		ON CONFLICT (channel_type, target) DO UPDATE
		SET title         = EXCLUDED.title,
		    enabled       = EXCLUDED.enabled,
		    rehearsal     = EXCLUDED.rehearsal,
		    disciplines   = EXCLUDED.disciplines,
		    championships = EXCLUDED.championships,
		    post_types    = EXCLUDED.post_types,
		    updated_at    = now()
		RETURNING id, channel_type, target, title, enabled, rehearsal, disciplines, championships, post_types
	`

	sqlUpdateDeliveryChannel = `
		UPDATE delivery_channels
		SET target = $2, title = $3, enabled = $4, rehearsal = $5,
		    disciplines   = COALESCE($6::text[], '{}'),
		    championships = COALESCE($7::text[], '{}'),
		    post_types    = COALESCE($8::text[], '{}'),
		    updated_at    = now()
		WHERE id = $1
		RETURNING id, channel_type, target, title, enabled, rehearsal, disciplines, championships, post_types
	`

	sqlDeleteDeliveryChannel = `DELETE FROM delivery_channels WHERE id = $1`

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
