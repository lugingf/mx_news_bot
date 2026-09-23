-- +goose Up
-- Long-lived Instagram tokens are bootstrapped from config, then refreshed by the bot and kept
-- here. The config file is mounted read-only in production, so it is not a place for rotation
-- state.
CREATE TABLE IF NOT EXISTS instagram_tokens (
    account_id   TEXT        PRIMARY KEY,
    access_token TEXT        NOT NULL,
    expires_at   TIMESTAMPTZ,
    refreshed_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS instagram_tokens;
