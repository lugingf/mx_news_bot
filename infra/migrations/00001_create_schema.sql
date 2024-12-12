-- +goose Up
-- Table 1: Riders
CREATE TABLE riders
(
    rider_id      SERIAL PRIMARY KEY,
    full_name     VARCHAR(100) NOT NULL,
    nationality   VARCHAR(50),
    date_of_birth DATE,
    UNIQUE (full_name)
);

-- Table 2: Tracks
CREATE TABLE tracks
(
    track_id           SERIAL PRIMARY KEY,
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
    championship_id   SERIAL PRIMARY KEY,
    championship_name VARCHAR(100) NOT NULL, -- E.g., "AMA Pro Motocross"
    class_names       TEXT[]       NOT NULL, -- Array of classes, e.g., {"450cc", "250cc", "MXGP"}
    season_year       INT          NOT NULL, -- Year of the championship
    UNIQUE (championship_name, season_year)  -- Ensure unique championships per season
);

-- Table 4: Rider Teams
CREATE TABLE rider_teams
(
    rider_team_id   SERIAL PRIMARY KEY,
    rider_id        INT REFERENCES riders (rider_id) ON DELETE CASCADE,
    team_name       VARCHAR(100) NOT NULL,
    bike_brand      VARCHAR(50),       -- Optional field
    championship_id INT REFERENCES championships (championship_id) ON DELETE CASCADE,
    UNIQUE (rider_id, championship_id) -- One rider per team per championship
);

-- Table 5: Events
CREATE TABLE events
(
    event_id         SERIAL PRIMARY KEY,
    championship_id  INT REFERENCES championships (championship_id),
    classes          VARCHAR(255),
    venue_name       VARCHAR(100),
    round_number     INT  NOT NULL, -- Round number in the championship
    track_id         INT REFERENCES tracks (track_id),
    event_date       DATE NOT NULL,
    event_format     VARCHAR(50),   -- E.g., "Triple Crown", "Standard",
    venue_info_url   VARCHAR(255),
    surface_override VARCHAR(50),   -- Optional: specific surface type for this event
    UNIQUE (championship_id, round_number)
);


-- Table 6: AMA Pro Motocross Results
CREATE TABLE ama_promotocross_results
(
    result_id       SERIAL PRIMARY KEY,
    championship_id INT REFERENCES championships (championship_id),
    event_id        INT REFERENCES events (event_id),
    moto_number     INT NOT NULL,                               -- 1 or 2
    rider_id        INT REFERENCES riders (rider_id),
    rider_team_id   INT REFERENCES rider_teams (rider_team_id), -- Optional, for event-specific team tracking
    position        INT NOT NULL,
    points_awarded  INT NOT NULL,
    lap_time        VARCHAR(50)                                 -- Optional: lap time (string or seconds)
);

-- Table 7: AMA Supercross Results
CREATE TABLE ama_supercross_results
(
    result_id       SERIAL PRIMARY KEY,
    championship_id INT REFERENCES championships (championship_id),
    event_id        INT REFERENCES events (event_id),
    race_type       VARCHAR(50) NOT NULL,                       -- "Heat", "LCQ", "Main Event", or "Triple Crown"
    race_number     INT,                                        -- For Triple Crown events (1, 2, or 3)
    rider_id        INT REFERENCES riders (rider_id),
    rider_team_id   INT REFERENCES rider_teams (rider_team_id), -- Optional, for event-specific team tracking
    position        INT         NOT NULL,
    points_awarded  INT,
    lap_time        VARCHAR(50)                                 -- Optional: lap time (string or seconds)
);

-- Table 8: MXGP Results
CREATE TABLE mxgp_results
(
    result_id       SERIAL PRIMARY KEY,
    championship_id INT REFERENCES championships (championship_id),
    event_id        INT REFERENCES events (event_id),
    moto_number     INT NOT NULL,                               -- 1 or 2
    rider_id        INT REFERENCES riders (rider_id),
    rider_team_id   INT REFERENCES rider_teams (rider_team_id), -- Optional, for event-specific team tracking
    position        INT NOT NULL,
    points_awarded  INT NOT NULL,
    lap_time        VARCHAR(50)                                 -- Optional: lap time (string or seconds)
);

-- Table 9: World Supercross Results
CREATE TABLE wsx_results
(
    result_id       SERIAL PRIMARY KEY,
    championship_id INT REFERENCES championships (championship_id),
    event_id        INT REFERENCES events (event_id),
    race_type       VARCHAR(50) NOT NULL,                       -- "Heat", "LCQ", "Main Event", or "Triple Crown"
    race_number     INT,                                        -- For Triple Crown events (1, 2, or 3)
    rider_id        INT REFERENCES riders (rider_id),
    rider_team_id   INT REFERENCES rider_teams (rider_team_id), -- Optional, for event-specific team tracking
    position        INT         NOT NULL,
    points_awarded  INT,
    lap_time        VARCHAR(50)                                 -- Optional: lap time (string or seconds)
);

-- Table 10: Points Distribution
CREATE TABLE points_distribution
(
    championship_id INT REFERENCES championships (championship_id),
    position        INT NOT NULL, -- Rider's position in the race
    points_awarded  INT NOT NULL,
    UNIQUE (championship_id, position)
);

-- Table 11: Championship Standings
CREATE TABLE standings
(
    standing_id     SERIAL PRIMARY KEY,
    championship_id INT REFERENCES championships (championship_id),
    rider_id        INT REFERENCES riders (rider_id),
    total_points    INT NOT NULL,
    position        INT,               -- Calculated dynamically based on total_points
    UNIQUE (championship_id, rider_id) -- One standing per rider per championship
);

-- Alter
ALTER TABLE events
    ADD COLUMN place              VARCHAR(100),
    ADD COLUMN weather_conditions VARCHAR(50);
-- Optional override for track surface

-- Updated Table: Championships
ALTER TABLE championships
    ADD COLUMN default_class VARCHAR(100), -- Default class to be used for standings and results
    ADD COLUMN description   TEXT;
-- Additional details about the championship

-- Updated Table: Events
ALTER TABLE events
    ADD COLUMN event_status VARCHAR(50) NOT NULL DEFAULT 'upcoming';
-- Event status (e.g., "upcoming", "completed")

-- New Table: UserPreferences
CREATE TABLE user_preferences
(
    preference_id           SERIAL PRIMARY KEY,
    tg_user_id              BIGINT NOT NULL,
    default_championship_id INT REFERENCES championships (championship_id),
    notifications_enabled   BOOLEAN DEFAULT TRUE,
    UNIQUE (tg_user_id)
);

-- New Table: EventResults
CREATE TABLE event_results
(
    result_id      SERIAL PRIMARY KEY,
    event_id       INT REFERENCES events (event_id),
    class_name     VARCHAR(100) NOT NULL,
    rider_id       INT REFERENCES riders (rider_id),
    position       INT          NOT NULL,
    points_awarded INT          NOT NULL,
    UNIQUE (event_id, class_name, rider_id)
);


-- +goose Down

drop table events;