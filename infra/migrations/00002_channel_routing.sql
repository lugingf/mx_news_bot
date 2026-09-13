-- +goose Up
-- What a channel accepts, as three independent filters. An empty list means "everything", and
-- the lists are combined with AND, so a channel is described by what it narrows down:
--
--   moto channel      disciplines {moto}
--   Supercross only   championships {AMA Supercross}
--   F1 channel        disciplines {f1}
--   rehearsal channel nothing set — it sees every post
--
-- Discipline alone would not do: Supercross, Pro Motocross and SMX are all moto, and a series
-- added later may deserve a channel of its own without the ones above changing.
ALTER TABLE delivery_channels ADD COLUMN IF NOT EXISTS disciplines   TEXT[] NOT NULL DEFAULT '{}';
ALTER TABLE delivery_channels ADD COLUMN IF NOT EXISTS championships TEXT[] NOT NULL DEFAULT '{}';
ALTER TABLE delivery_channels ADD COLUMN IF NOT EXISTS post_types    TEXT[] NOT NULL DEFAULT '{}';
