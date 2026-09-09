-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS users (
    id          SERIAL PRIMARY KEY,
    tg_user_id  BIGINT      NOT NULL UNIQUE,
    name        TEXT        NOT NULL DEFAULT '',
    is_premium  BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- default_championship_id is a lap_vision championship id: this service no longer owns
-- championships, so the column is deliberately not a foreign key.
CREATE TABLE IF NOT EXISTS user_preferences (
    preference_id           SERIAL PRIMARY KEY,
    tg_user_id              BIGINT      NOT NULL UNIQUE REFERENCES users(tg_user_id) ON DELETE CASCADE,
    default_championship_id INT,
    notifications_enabled   BOOLEAN     NOT NULL DEFAULT TRUE,
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS delivery_channels (
    id           SERIAL PRIMARY KEY,
    channel_type TEXT        NOT NULL CHECK (channel_type IN ('telegram', 'twitter', 'instagram')),
    target       TEXT        NOT NULL,
    title        TEXT        NOT NULL DEFAULT '',
    enabled      BOOLEAN     NOT NULL DEFAULT TRUE,
    event_types  TEXT[]      NOT NULL DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (channel_type, target)
);

CREATE INDEX IF NOT EXISTS idx_delivery_channels_enabled ON delivery_channels(enabled);

-- One row per publication accepted from lap_vision. The primary key is the sender's event id,
-- which is what makes redelivery of the same webhook a no-op.
CREATE TABLE IF NOT EXISTS publications (
    event_id    TEXT        PRIMARY KEY,
    event_type  TEXT        NOT NULL,
    payload     JSONB       NOT NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Delivery is tracked per channel so one failing channel neither blocks the others nor causes
-- an already-published channel to be posted twice on retry.
CREATE TABLE IF NOT EXISTS publication_deliveries (
    event_id     TEXT        NOT NULL REFERENCES publications(event_id) ON DELETE CASCADE,
    channel_id   INT         NOT NULL REFERENCES delivery_channels(id) ON DELETE CASCADE,
    status       TEXT        NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'delivered', 'failed', 'skipped')),
    attempts     INT         NOT NULL DEFAULT 0,
    last_error   TEXT        NOT NULL DEFAULT '',
    external_ref TEXT        NOT NULL DEFAULT '',
    delivered_at TIMESTAMPTZ,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id, channel_id)
);

CREATE INDEX IF NOT EXISTS idx_publication_deliveries_status ON publication_deliveries(status);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS publication_deliveries;
DROP TABLE IF EXISTS publications;
DROP TABLE IF EXISTS delivery_channels;
DROP TABLE IF EXISTS user_preferences;
DROP TABLE IF EXISTS users;

-- +goose StatementEnd
