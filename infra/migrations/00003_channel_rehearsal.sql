-- +goose Up
-- A rehearsal channel exists to look at a post before it is published: a new kind of post, or a
-- change to how one reads. It takes rehearsals and nothing else, and a live channel never takes
-- one — otherwise trying something out would interrupt the channels already in service.
ALTER TABLE delivery_channels ADD COLUMN IF NOT EXISTS rehearsal BOOLEAN NOT NULL DEFAULT FALSE;
