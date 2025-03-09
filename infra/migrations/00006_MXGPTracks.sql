-- +goose Up

INSERT INTO tracks (name, city, state, surface, length_km)
VALUES
    ('Infinito Race Track', 'Córdoba', 'Argentina', NULL, NULL),
    ('Cózar Circuit', 'Cózar', 'Spain', NULL, NULL),
    ('Saint Jean d''Angély', 'Saint-Jean-d''Angély', 'France', 'Hard Pack', 1.62),
    ('Riola Sardo', 'Riola Sardo', 'Italy', NULL, NULL),
    ('Pietramurata', 'Pietramurata', 'Italy', NULL, NULL),
    ('Frauenfeld', 'Frauenfeld', 'Switzerland', NULL, NULL),
    ('Águeda', 'Águeda', 'Portugal', NULL, NULL),
    ('Lugo', 'Lugo', 'Spain', NULL, NULL),
    ('Ernée', 'Ernée', 'France', 'Clay', 1.517),
    ('Teutschenthal', 'Teutschenthal', 'Germany', NULL, NULL),
    ('Ķegums', 'Ķegums', 'Latvia', NULL, NULL),
    ('Matterley Basin', 'Winchester', 'United Kingdom', NULL, NULL),
    ('Indonesia Circuit', 'TBA', 'Indonesia', NULL, NULL),
    ('Loket', 'Loket', 'Czech Republic', NULL, NULL),
    ('Lommel', 'Lommel', 'Belgium', NULL, NULL),
    ('Uddevalla', 'Uddevalla', 'Sweden', NULL, NULL),
    ('Motorportpark Gelderland Midden', 'Arnhem', 'Netherlands', NULL, NULL),
    ('Afyonkarahisar Motor Sports Center', 'Afyonkarahisar', 'Turkey', 'Hard Pack', 1.725),
    ('Shanghai International Off-road Circuit', 'Shanghai', 'China', NULL, NULL),
    ('Hidden Valley Motorsports Complex', 'Darwin', 'Australia', NULL, NULL),
    ('Ironman Raceway', 'Crawfordsville', 'United States', NULL, NULL),
    ('Romagné Circuit', 'Romagné', 'France', NULL, NULL);


INSERT INTO events (championship_id, name, classes, venue_name, round_number, track_id, event_date, event_format, event_code, place, event_status)
VALUES
    (3, 'YPF INFINIA MXGP of Argentina', 'MXGP, MX2', 'Cordoba', 1, 29, '2025-03-02', 'Standard', 'MXGP4001', 'Argentina', 'upcoming'),
    (3, 'MXGP of Castilla la Mancha', 'MXGP, MX2', 'Cozar', 2, 30, '2025-03-16', 'Standard', 'MXGP4002', 'Spain', 'upcoming'),
    (3, 'MXGP of Europe', 'MXGP, MX2', 'St Jean d''Angely', 3, 31, '2025-03-23', 'Standard', 'MXGP4003', 'France', 'upcoming'),
    (3, 'MXGP of Sardegna', 'MXGP, MX2', 'Riola Sardo', 4, 32, '2025-04-06', 'Standard', 'MXGP4004', 'Italy', 'upcoming'),
    (3, 'Monster Energy MXGP of Trentino', 'MXGP, MX2', 'Pietramurata', 5, 33, '2025-04-13', 'Standard', 'MXGP4005', 'Italy', 'upcoming'),
    (3, 'MXGP of Switzerland', 'MXGP, MX2', 'Frauenfeld', 6, 34, '2025-04-21', 'Standard', 'MXGP4006', 'Switzerland', 'upcoming'),
    (3, 'MXGP of Portugal', 'MXGP, MX2', 'Agueda', 7, 35, '2025-05-04', 'Standard', 'MXGP4007', 'Portugal', 'upcoming'),
    (3, 'MXGP of Spain', 'MXGP, MX2', 'Lugo', 8, 36, '2025-05-11', 'Standard', 'MXGP4008', 'Spain', 'upcoming'),
    (3, 'MXGP of France', 'MXGP, MX2', 'Ernee', 9, 37, '2025-05-25', 'Standard', 'MXGP4009', 'France', 'upcoming'),
    (3, 'Liqui Moly MXGP of Germany', 'MXGP, MX2', 'Teutschenthal', 10, 38, '2025-06-01', 'Standard', 'MXGP4010', 'Germany', 'upcoming'),
    (3, 'MXGP of Latvia', 'MXGP, MX2', 'Kegums', 11, 39, '2025-06-08', 'Standard', 'MXGP4011', 'Latvia', 'upcoming'),
    (3, 'MXGP of Great Britain', 'MXGP, MX2', 'Matterley Basin', 12, 40, '2025-06-22', 'Standard', 'MXGP4012', 'United Kingdom', 'upcoming'),
    (3, 'MXGP of Indonesia', 'MXGP, MX2', 'TBA', 13, 41, '2025-07-06', 'Standard', 'MXGP4013', 'Indonesia', 'upcoming'),
    (3, 'MXGP of Czech Republic', 'MXGP, MX2', 'Loket', 14, 42, '2025-07-27', 'Standard', 'MXGP4014', 'Czech Republic', 'upcoming'),
    (3, 'MXGP of Flanders', 'MXGP, MX2', 'Lommel', 15, 43, '2025-08-03', 'Standard', 'MXGP4015', 'Belgium', 'upcoming'),
    (3, 'MXGP of Sweden', 'MXGP, MX2', 'Uddevalla', 16, 44, '2025-08-17', 'Standard', 'MXGP4016', 'Sweden', 'upcoming'),
    (3, 'MXGP of The Netherlands', 'MXGP, MX2', 'Arnhem', 17, 45, '2025-08-24', 'Standard', 'MXGP4017', 'Netherlands', 'upcoming'),
    (3, 'MXGP of Turkiye', 'MXGP, MX2', 'Afyonkarahisar', 18, 46, '2025-09-07', 'Standard', 'MXGP4018', 'Turkey', 'upcoming'),
    (3, 'MXGP of China', 'MXGP, MX2', 'Shanghai', 19, 47, '2025-09-14', 'Standard', 'MXGP4019', 'China', 'upcoming'),
    (3, 'MXGP of Australia', 'MXGP, MX2', 'Darwin', 20, 48, '2025-09-21', 'Standard', 'MXGP4020', 'Australia', 'upcoming'),
    (3, 'Monster Energy FIM MXoN', 'MXGP, MX2', 'Crawfordsville, IN', 21, 49, '2025-10-05', 'Standard', 'MXGP4021', 'United States', 'upcoming');

