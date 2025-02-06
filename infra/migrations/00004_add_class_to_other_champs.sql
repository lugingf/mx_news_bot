-- +goose Up

ALTER TABLE ama_promotocross_results
    ADD COLUMN class VARCHAR(50) NOT NULL DEFAULT '450';

ALTER TABLE mxgp_results
    ADD COLUMN class VARCHAR(50) NOT NULL DEFAULT '450';

ALTER TABLE wsx_results
    ADD COLUMN class VARCHAR(50) NOT NULL DEFAULT '450';

