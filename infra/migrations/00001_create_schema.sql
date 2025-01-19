-- +goose Up
-- Table 1: Riders
CREATE TABLE riders
(
    id            SERIAL PRIMARY KEY,
    full_name     VARCHAR(100) NOT NULL,
    nationality   VARCHAR(50),
    date_of_birth DATE,
    UNIQUE (full_name)
);

-- Table 2: Tracks
CREATE TABLE tracks
(
    id                 SERIAL PRIMARY KEY,
    name               VARCHAR(100) NOT NULL,
    city               VARCHAR(50),
    state              VARCHAR(50),
    track_map_filename VARCHAR(100),
    surface            VARCHAR(50),   -- E.g., sand, hardpack
    length_km          DECIMAL(5, 2), -- Track length in kilometers
    UNIQUE (name, city, state)
);

-- Table 3: Championships
CREATE TABLE championships
(
    id                SERIAL PRIMARY KEY,
    championship_name VARCHAR(100) NOT NULL, -- E.g., "AMA Pro Motocross"
    class_names       TEXT[]       NOT NULL, -- Array of classes, e.g., {"450cc", "250cc", "MXGP"}
    season_year       INT          NOT NULL, -- Year of the championship
    UNIQUE (championship_name, season_year)  -- Ensure unique championships per season
);

-- Table 4: Rider Teams
CREATE TABLE rider_teams
(
    id              SERIAL PRIMARY KEY,
    rider_id        INT REFERENCES riders (id) ON DELETE CASCADE,
    team_name       VARCHAR(100) NOT NULL,
    bike_brand      VARCHAR(50),       -- Optional field
    championship_id INT REFERENCES championships (id) ON DELETE CASCADE,
    UNIQUE (rider_id, championship_id) -- One rider per team per championship
);

-- Table 5: Events
CREATE TABLE events
(
    id               SERIAL PRIMARY KEY,
    championship_id  INT REFERENCES championships (id),
    name             VARCHAR(100),
    classes          VARCHAR(255),
    venue_name       VARCHAR(100),
    round_number     INT         NOT NULL, -- Round number in the championship
    track_id         INT,
    event_date       DATE        NOT NULL,
    event_format     VARCHAR(50),          -- E.g., "Triple Crown", "Standard",
    venue_info_url   VARCHAR(255),
    surface_override VARCHAR(50),          -- Optional: specific surface type for this event
    event_code       VARCHAR(10) NOT NULL DEFAULT '',
    UNIQUE (championship_id, round_number, event_code, event_date)
);

-- Alter
ALTER TABLE events
    ADD COLUMN place              VARCHAR(100),
    ADD COLUMN weather_conditions VARCHAR(50);
-- Optional override for track surface

-- Updated Table: Events
ALTER TABLE events
    ADD COLUMN event_status VARCHAR(50) NOT NULL DEFAULT 'upcoming';
-- Event status (e.g., "upcoming", "result_pending", "downloaded", "completed")

-- {
--   "event": "monster energy ama supercross",
--   "race_type": "MainEvent",
--   "city": "salt lake city",
--   "state": "UT",
--   "stadium": "rice-eccles stadium",
--   "date": "may 11, 2024",
--   "round": "17",
--   "total_rounds": "17",
--   "class": "450SX",
--   "results": [
--     {
--       "pos": "1",
--       "rider_number": "1",
--       "rider": "Chase Sexton",
--       "hometown": "LaMoille, IL",
--       "bike": "KTM 450 SX-F FE",
--       "team": "Red Bull KTM Factory Racing"
--     },]}
-- Table 7: AMA Supercross Results
CREATE TABLE ama_supercross_results
(
    id              SERIAL PRIMARY KEY,
    championship_id INT REFERENCES championships (id),
    event_code      VARCHAR(10) NOT NULL DEFAULT 0000,
    event_name      VARCHAR(10),
    race_type       VARCHAR(50) NOT NULL,            -- "Heat", "LCQ", "Main Event", or "Triple Crown"
    class           VARCHAR(50) NOT NULL DEFAULT '',
    round           VARCHAR(50) NOT NULL DEFAULT '',
    rider_id        INT REFERENCES riders (id),
    rider_team_id   INT REFERENCES rider_teams (id), -- Optional, for event-specific team tracking
    rider_number    VARCHAR(50) NOT NULL DEFAULT '',
    bike            VARCHAR(50) NOT NULL DEFAULT '',
    position        INT         NOT NULL,
    points_awarded  INT,
    UNIQUE (championship_id, event_code, race_type, event_name, class, round, rider_id)
);

-- Table 6: AMA Pro Motocross Results
CREATE TABLE ama_promotocross_results
(
    id              SERIAL PRIMARY KEY,
    championship_id INT REFERENCES championships (id),
    moto_number     INT NOT NULL,                    -- 1 or 2
    rider_id        INT REFERENCES riders (id),
    rider_team_id   INT REFERENCES rider_teams (id), -- Optional, for event-specific team tracking
    position        INT NOT NULL,
    points_awarded  INT NOT NULL
);



-- Table 8: MXGP Results
CREATE TABLE mxgp_results
(
    id              SERIAL PRIMARY KEY,
    championship_id INT REFERENCES championships (id),
    moto_number     INT NOT NULL,                    -- 1 or 2
    rider_id        INT REFERENCES riders (id),
    rider_team_id   INT REFERENCES rider_teams (id), -- Optional, for event-specific team tracking
    position        INT NOT NULL,
    points_awarded  INT NOT NULL
);

-- Table 9: World Supercross Results
CREATE TABLE wsx_results
(
    id              SERIAL PRIMARY KEY,
    championship_id INT REFERENCES championships (id),
    race_type       VARCHAR(50) NOT NULL,            -- "Heat", "LCQ", "Main Event", or "Triple Crown"
    race_number     INT,                             -- For Triple Crown events (1, 2, or 3)
    rider_id        INT REFERENCES riders (id),
    rider_team_id   INT REFERENCES rider_teams (id), -- Optional, for event-specific team tracking
    position        INT         NOT NULL,
    points_awarded  INT
);

-- Table 10: Points Distribution
CREATE TABLE points_distribution
(
    championship_id INT REFERENCES championships (id),
    position        INT NOT NULL, -- Rider's position in the race
    points_awarded  INT NOT NULL,
    UNIQUE (championship_id, position)
);

-- Table 11: Championship Standings
CREATE TABLE standings
(
    standing_id     SERIAL PRIMARY KEY,
    championship_id INT REFERENCES championships (id),
    rider_id        INT REFERENCES riders (id),
    total_points    INT NOT NULL,
    position        INT,               -- Calculated dynamically based on total_points
    UNIQUE (championship_id, rider_id) -- One standing per rider per championship
);


-- Updated Table: Championships
ALTER TABLE championships
    ADD COLUMN default_class VARCHAR(100), -- Default class to be used for standings and results
    ADD COLUMN description   TEXT;
-- Additional details about the championship


-- New Table: UserPreferences
CREATE TABLE user_preferences
(
    preference_id           SERIAL PRIMARY KEY,
    tg_user_id              BIGINT NOT NULL,
    default_championship_id INT REFERENCES championships (id),
    notifications_enabled   BOOLEAN DEFAULT TRUE,
    UNIQUE (tg_user_id)
);


-- +goose Down

drop table events;