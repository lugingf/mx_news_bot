-- +goose Up

CREATE TABLE points_distribution
(
    championship_id INTEGER NOT NULL,
    position        INTEGER NOT NULL,
    points          INTEGER NOT NULL
);


INSERT INTO points_distribution (championship_id, position, points)
VALUES (1, 1, 25),
       (1, 2, 22),
       (1, 3, 20),
       (1, 4, 18),
       (1, 5, 17),
       (1, 6, 16),
       (1, 7, 15),
       (1, 8, 14),
       (1, 9, 13),
       (1, 10, 12),
       (1, 11, 11),
       (1, 12, 10),
       (1, 13, 9),
       (1, 14, 8),
       (1, 15, 7),
       (1, 16, 6),
       (1, 17, 5),
       (1, 18, 4),
       (1, 19, 3),
       (1, 20, 2),
       (1, 21, 1),
       (1, 22, 0),
       (1, 23, 0),
       (1, 24, 0),
       (1, 25, 0),
       (1, 26, 0),
       (1, 27, 0),
       (1, 28, 0),
       (1, 29, 0),
       (1, 30, 0);

