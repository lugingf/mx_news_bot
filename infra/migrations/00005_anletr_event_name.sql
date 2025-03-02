-- +goose Up

ALTER TABLE ama_supercross_results
    ALTER COLUMN event_name TYPE VARCHAR(100);
