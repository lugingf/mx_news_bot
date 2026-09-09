# Season Schedule

The bot no longer owns the calendar. Championships, rounds and tracks live in `lap_vision`
(`moto_championships`, `moto_events`, `moto_tracks`), which is also what scores them, so a season
is seeded there and the bot reads it back over the API.

## Adding a season

Write a goose migration in `lap_vision/migrations`. The two existing seeds are the pattern to
follow:

- `00009_race_schedule.sql` — AMA Supercross, Pro Motocross and the SMX playoffs for 2026.
- `00020_mxgp_2026_calendar_and_points.sql` — the FIM world championship for 2026, plus the
  points tables for every championship.

Each round needs `round_number`, `event_date`, and an `event_format` that the scorers recognise
(`Standard`, `Triple Crown`, `Two Moto`, `Two Race`) — an unknown format is rejected rather than
silently skipped, so the calendar cannot go half-scored. A championship also needs a row set in
`moto_points_distribution`; without it every rider scores zero.

`season_schedule.template.json` is the input format of the old `cmd/schedule_loader`, which was
removed with the rest of the bot's racing domain. It is kept only as a record of the field names
used for the 2026 season.
