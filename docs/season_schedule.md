# Season Schedule Import

Use JSON import instead of writing SQL migrations by hand for each season.

## 1) Prepare input

Copy `docs/season_schedule.template.json` and fill all rounds/tracks for the new season.

Date format is `YYYY-MM-DD`.

## 2) Run import

```bash
go run ./cmd/schedule_loader -file ./path/to/season_2026.json
```

The loader upserts:
- championships (`name + season_year`)
- tracks (`name + city + state`)
- events (`championship_id + round_number + event_code + event_date`)

You can run it multiple times; changed rows are updated.
