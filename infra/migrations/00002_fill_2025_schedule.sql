-- +goose Up

-- Step 1: Fill the Championships Table
INSERT INTO championships (championship_name, class_names, season_year)
VALUES ('Monster Energy AMA Supercross',
        ARRAY ['450SX', '250SX West', 'KTM Junior', '250SX East', 'Supercross Futures', '250SX East/West Showdown'],
        2025),
       ('Pro Motocross Championship', ARRAY ['450 Class', '250 Class'], 2025);

-- Step 2: Fill the Tracks Table
INSERT INTO tracks (name, city, state, track_map_filename, surface, length_km)
VALUES ('Angel Stadium', 'Anaheim', 'CA', 'anaheim-1-track-map-2025.jpg', 'hardpack', 1.2),
       ('Snapdragon Stadium', 'San Diego', 'CA', 'san-diego-track-map-2025.jpg', 'hardpack', 1.1),
       ('State Farm Stadium', 'Glendale', 'AZ', 'glendale-track-map-2025.jpg', 'hardpack', 1.3),
       ('Raymond James Stadium', 'Tampa', 'FL', 'tampa-track-map-2025.jpg', 'hardpack', 1.0),
       ('Ford Field', 'Detroit', 'MI', 'detroit-track-map-2025.jpg', 'hardpack', 1.1),
       ('AT&T Stadium', 'Arlington', 'TX', 'arlington-track-map-2025.jpg', 'hardpack', 1.2),
       ('Daytona International Speedway', 'Daytona Beach', 'FL', 'daytona-track-map-2025.jpg', 'sand', 1.5),
       ('Lucas Oil Stadium', 'Indianapolis', 'IN', 'indianapolis-track-map-2025.jpg', 'hardpack', 1.2),
       ('Protective Stadium', 'Birmingham', 'AL', 'birmingham-track-map-2025.jpg', 'hardpack', 1.1),
       ('Lumen Field', 'Seattle', 'WA', 'seattle-track-map-2025.jpg', 'hardpack', 1.3),
       ('Gillette Stadium', 'Foxborough', 'MA', 'foxborough-track-map-2025.jpg', 'hardpack', 1.0),
       ('Lincoln Financial Field', 'Philadelphia', 'PA', 'philadelphia-track-map-2025.jpg', 'hardpack', 1.2),
       ('MetLife Stadium', 'East Rutherford', 'NJ', 'east-rutherford-track-map-2025.jpg', 'hardpack', 1.1),
       ('Acrisure Stadium', 'Pittsburgh', 'PA', 'pittsburgh-track-map-2025.jpg', 'hardpack', 1.2),
       ('Empower Field at Mile High', 'Denver', 'CO', 'denver-track-map-2025.jpg', 'hardpack', 1.3),
       ('Rice-Eccles Stadium', 'Salt Lake City', 'UT', 'salt-lake-city-track-map-2025.jpg', 'hardpack', 1.2),
       ('Fox Raceway at Pala', 'Pala', 'CA', 'fox-raceway-track-map-2025.jpg', 'sand', 1.6),
       ('Prairie City SVRA', 'Rancho Cordova', 'CA', 'prairie-city-track-map-2025.jpg', 'sand', 1.5),
       ('Thunder Valley Motocross Park', 'Lakewood', 'CO', 'thunder-valley-track-map-2025.jpg', 'hardpack', 1.7),
       ('High Point Raceway', 'Mt. Morris', 'PA', 'high-point-track-map-2025.jpg', 'hardpack', 1.8),
       ('The Wick 338', 'Southwick', 'MA', 'the-wick-338-track-map-2025.jpg', 'sand', 1.9),
       ('RedBud MX', 'Buchanan', 'MI', 'redbud-track-map-2025.jpg', 'hardpack', 1.6),
       ('Spring Creek MX Park', 'Millville', 'MN', 'spring-creek-track-map-2025.jpg', 'sand', 1.8),
       ('Washougal MX Park', 'Washougal', 'WA', 'washougal-track-map-2025.jpg', 'hardpack', 1.7),
       ('Ironman Raceway', 'Crawfordsville', 'IN', 'ironman-track-map-2025.jpg', 'hardpack', 1.6),
       ('Unadilla MX', 'New Berlin', 'NY', 'unadilla-track-map-2025.jpg', 'hardpack', 1.9),
       ('Budds Creek Motocross Park', 'Mechanicsville', 'MD', 'budds-creek-track-map-2025.jpg', 'sand', 1.7);

-- Step 3: Fill the Events Table
INSERT INTO events (championship_id, round_number, event_date, venue_name, classes, venue_info_url,
                    track_id, event_format, surface_override)
VALUES (1, 1, '2025-01-11', 'Angel Stadium', '450SX, 250SX West, KTM Junior',
        'https://www.supercrosslive.com/tickets/anaheim-ca/jan-11-2025/', 1, 'Standard', NULL),
       (1, 2, '2025-01-18', 'Snapdragon Stadium', '450SX, 250SX West, KTM Junior',
        'https://www.supercrosslive.com/tickets/san-diego-ca/jan-18-2025/', 2, 'Standard', NULL),
       (1, 3, '2025-01-25', 'Angel Stadium', '450SX, 250SX West, KTM Junior',
        'https://www.supercrosslive.com/tickets/anaheim-ca/jan-25-2025/', 1, 'Standard', NULL),
       (1, 4, '2025-02-01', 'State Farm Stadium', '450SX, 250SX West, KTM Junior, Supercross Futures',
        'https://www.supercrosslive.com/tickets/glendale-az/feb-1-2025/', 3, 'Triple Crown', NULL),
       (1, 5, '2025-02-08', 'Raymond James Stadium', '450SX, 250SX East',
        'https://www.supercrosslive.com/tickets/tampa-fl/feb-8-2025/', 4, 'Standard', NULL),
       (1, 6, '2025-02-15', 'Ford Field', '450SX, 250SX East, KTM Junior',
        'https://www.supercrosslive.com/tickets/detroit-mi/feb-15-2025/', 5, 'Standard', NULL),
       (1, 7, '2025-02-22', 'AT&T Stadium', '450SX, 250SX West, KTM Junior',
        'https://www.supercrosslive.com/tickets/arlington-tx/feb-22-2025/', 6, 'Triple Crown', NULL),
       (1, 8, '2025-03-01', 'Daytona International Speedway', '450SX, 250SX East, Supercross Futures',
        'https://www.supercrosslive.com/tickets/daytona-beach-fl/mar-1-2025/', 7, 'Standard', 'sand'),
       (1, 9, '2025-03-08', 'Lucas Oil Stadium', '450SX, 250SX East/West Showdown, KTM Junior',
        'https://www.supercrosslive.com/tickets/indianapolis-in/mar-8-2025/', 8, 'Standard', NULL),
       (1, 10, '2025-03-22', 'Protective Stadium', '450SX, 250SX East, KTM Junior, Supercross Futures',
        'https://www.supercrosslive.com/tickets/birmingham-al/mar-22-2025/', 9, 'Triple Crown', NULL),
       (1, 11, '2025-03-29', 'Lumen Field', '450SX, 250SX West',
        'https://www.supercrosslive.com/tickets/seattle-wa/mar-29-2025/', 10, 'Standard', NULL),
       (1, 12, '2025-04-05', 'Gillette Stadium', '450SX, 250SX East, Supercross Futures',
        'https://www.supercrosslive.com/tickets/foxborough-ma/apr-5-2025/', 11, 'Standard', NULL),
       (1, 13, '2025-04-12', 'Lincoln Financial Field', '450SX, 250SX East/West Showdown',
        'https://www.supercrosslive.com/tickets/philadelphia-pa/apr-12-2025/', 12, 'Standard', NULL),
       (1, 14, '2025-04-19', 'MetLife Stadium', '450SX, 250SX East, KTM Junior',
        'https://www.supercrosslive.com/tickets/east-rutherford-nj/apr-19-2025/', 13, 'Standard', NULL),
       (1, 15, '2025-04-26', 'Acrisure Stadium', '450SX, 250SX East, Supercross Futures',
        'https://www.supercrosslive.com/tickets/pittsburgh-pa/apr-26-2025/', 14, 'Standard', NULL),
       (1, 16, '2025-05-03', 'Empower Field at Mile High', '450SX, 250SX West, KTM Junior',
        'https://www.supercrosslive.com/tickets/denver-co/may-3-2025/', 15, 'Standard', NULL),
       (1, 17, '2025-05-10', 'Rice-Eccles Stadium', '450SX, 250SX East/West Showdown',
        'https://www.supercrosslive.com/tickets/salt-lake-city-ut/may-10-2025/', 16, 'Standard', NULL),
       (2, 1, '2025-05-24', 'Fox Raceway at Pala', '450 Class, 250 Class',
        'https://promotocross.com/race/fox-raceway-national', 17, 'Standard', 'sand'),
       (2, 2, '2025-05-31', 'Prairie City SVRA', '450 Class, 250 Class',
        'https://promotocross.com/race/prairie-city-national', 18, 'Standard', 'sand'),
       (2, 3, '2025-06-07', 'Thunder Valley Motocross Park', '450 Class, 250 Class',
        'https://promotocross.com/race/thunder-valley-national', 19, 'Standard', NULL),
       (2, 4, '2025-06-14', 'High Point Raceway', '450 Class, 250 Class',
        'https://promotocross.com/race/high-point-national', 20, 'Standard', NULL),
       (2, 5, '2025-06-28', 'The Wick 338', '450 Class, 250 Class',
        'https://promotocross.com/race/southwick-national', 21, 'Standard', 'sand'),
       (2, 6, '2025-07-05', 'RedBud MX', '450 Class, 250 Class', 'https://promotocross.com/race/redbud-national',
        22, 'Standard', NULL),
       (2, 7, '2025-07-12', 'Spring Creek MX Park', '450 Class, 250 Class',
        'https://promotocross.com/race/spring-creek-national', 23, 'Standard', 'sand'),
       (2, 8, '2025-07-19', 'Washougal MX Park', '450 Class, 250 Class',
        'https://promotocross.com/race/washougal-national', 24, 'Standard', NULL),
       (2, 9, '2025-08-09', 'Ironman Raceway', '450 Class, 250 Class',
        'https://promotocross.com/race/ironman-national', 25, 'Standard', NULL),
       (2, 10, '2025-08-16', 'Unadilla MX', '450 Class, 250 Class',
        'https://promotocross.com/race/unadilla-national', 26, 'Standard', NULL),
       (2, 11, '2025-08-23', 'Budds Creek Motocross Park', '450 Class, 250 Class',
        'https://promotocross.com/race/budds-creek-national', 27, 'Standard', 'sand');

UPDATE events
SET event_code = CASE
                     WHEN championship_id = 1 THEN
                         CASE
                             WHEN round_number = 18 THEN CONCAT('S', EXTRACT(YEAR FROM event_date) - 2000, '99') -- Exception for SX Round 18
                             ELSE CONCAT('S', EXTRACT(YEAR FROM event_date) - 2000, LPAD((round_number * 5)::TEXT, 2, '0')) -- General SX logic
                             END
                     WHEN championship_id = 2 THEN
                         CONCAT('M', EXTRACT(YEAR FROM event_date) - 2000, LPAD((round_number * 5)::TEXT, 2, '0')) -- MX logic
                     ELSE NULL -- Default case if necessary
END;



-- +goose Down
