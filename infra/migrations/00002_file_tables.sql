-- +goose Up
CREATE TABLE events_scrape
(
    id               SERIAL PRIMARY KEY,
    championship_id  INT         NOT NULL,        -- ID чемпионата, к которому относится событие
    event_id         VARCHAR(10) NOT NULL UNIQUE, -- Например, "S2510"
    round_number     INT         NOT NULL,        -- Номер раунда
    is_processed     BOOLEAN   DEFAULT FALSE,     -- Событие обработано или нет
    has_results_page BOOLEAN   DEFAULT FALSE,     -- Страница с результатами найдена или нет
    last_checked     TIMESTAMP DEFAULT NOW(),     -- Дата последней проверки
    created_at       TIMESTAMP DEFAULT NOW()
);

CREATE TABLE event_files
(
    id              SERIAL PRIMARY KEY,
    event_scrape_id INT REFERENCES events_scrape (id) ON DELETE CASCADE,
    file_name       VARCHAR(50) NOT NULL,           -- Например, "S1F1RES.pdf"
    file_path       VARCHAR(255) DEFAULT NULL,      -- Локальный путь к файлу
    status          VARCHAR(20)  DEFAULT 'pending', -- pending, downloaded, processed
    last_checked    TIMESTAMP    DEFAULT NOW(),     -- Дата последней проверки
    created_at      TIMESTAMP    DEFAULT NOW()
);

-- +goose Down
drop table events_scrape;
drop table event_files;