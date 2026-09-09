package lapvision

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"mx_news_bot/internal/domain"
	"mx_news_bot/internal/models"
)

const defaultTimeout = 15 * time.Second

// Client reads the lap_vision /moto/* API. It is the only place in the bot that knows racing data
// arrives over HTTP.
type Client struct {
	baseURL       string
	internalToken string
	http          *http.Client
}

func New(baseURL, internalToken string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	return &Client{
		baseURL:       strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		internalToken: strings.TrimSpace(internalToken),
		http:          &http.Client{Timeout: timeout},
	}
}

// StatusError reports a response the API refused. It is kept distinct from a transport failure so
// a caller can tell "you asked for the wrong thing" from "the backend is unreachable".
type StatusError struct {
	StatusCode int
	Path       string
	Body       string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("lap_vision %s: unexpected status %d: %s", e.Path, e.StatusCode, e.Body)
}

func (e *StatusError) Unwrap() error {
	if e.StatusCode == http.StatusNotFound {
		return domain.ErrNotFound
	}

	return nil
}

func (c *Client) get(ctx context.Context, path string, query url.Values, out any) error {
	endpoint := c.baseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("lap_vision %s: build request: %w", path, err)
	}
	req.Header.Set("Accept", "application/json")
	if c.internalToken != "" {
		req.Header.Set("X-Internal-Token", c.internalToken)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("lap_vision %s: %w", path, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return &StatusError{StatusCode: resp.StatusCode, Path: path, Body: strings.TrimSpace(string(body))}
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("lap_vision %s: decode response: %w", path, err)
	}

	return nil
}

func (c *Client) Seasons(ctx context.Context) ([]int, error) {
	var out struct {
		Seasons []int `json:"seasons"`
	}
	if err := c.get(ctx, "/moto/seasons", nil, &out); err != nil {
		return nil, err
	}

	return out.Seasons, nil
}

func (c *Client) Championships(ctx context.Context, season int, withRaces bool) ([]models.Championship, error) {
	query := url.Values{}
	if season > 0 {
		query.Set("season", strconv.Itoa(season))
	}
	if withRaces {
		query.Set("with_races", "true")
	}

	var out struct {
		Championships []models.Championship `json:"championships"`
	}
	if err := c.get(ctx, "/moto/championships", query, &out); err != nil {
		return nil, err
	}

	return out.Championships, nil
}

func (c *Client) ChampionshipClasses(ctx context.Context, champID int) ([]models.RaceClass, error) {
	var out struct {
		Classes []models.RaceClass `json:"classes"`
	}
	if err := c.get(ctx, fmt.Sprintf("/moto/championships/%d/classes", champID), nil, &out); err != nil {
		return nil, err
	}

	return out.Classes, nil
}

func (c *Client) ChampionshipEvents(ctx context.Context, champID int) ([]models.Event, error) {
	var out struct {
		Events []models.Event `json:"events"`
	}
	if err := c.get(ctx, fmt.Sprintf("/moto/championships/%d/events", champID), nil, &out); err != nil {
		return nil, err
	}

	return out.Events, nil
}

func (c *Client) Standings(ctx context.Context, champID int, class, region string) ([]models.Standing, error) {
	query := url.Values{}
	query.Set("class", class)
	if strings.TrimSpace(region) != "" {
		query.Set("region", region)
	}

	var out struct {
		Standings []models.Standing `json:"standings"`
	}
	if err := c.get(ctx, fmt.Sprintf("/moto/championships/%d/standings", champID), query, &out); err != nil {
		return nil, err
	}

	return out.Standings, nil
}

func (c *Client) PointsDistribution(ctx context.Context, champID int) ([]models.PointsRow, error) {
	var out struct {
		Points []models.PointsRow `json:"points"`
	}
	if err := c.get(ctx, fmt.Sprintf("/moto/championships/%d/points-distribution", champID), nil, &out); err != nil {
		return nil, err
	}

	return out.Points, nil
}

func (c *Client) UpcomingEvents(ctx context.Context) ([]models.Event, error) {
	var out struct {
		Events []models.Event `json:"events"`
	}
	if err := c.get(ctx, "/moto/events/upcoming", nil, &out); err != nil {
		return nil, err
	}

	return out.Events, nil
}

func (c *Client) CompletedEvents(ctx context.Context, season int) ([]models.Event, error) {
	query := url.Values{}
	if season > 0 {
		query.Set("season", strconv.Itoa(season))
	}

	var out struct {
		Events []models.Event `json:"events"`
	}
	if err := c.get(ctx, "/moto/events/completed", query, &out); err != nil {
		return nil, err
	}

	return out.Events, nil
}

func (c *Client) Event(ctx context.Context, eventID int) (models.Event, error) {
	var out models.Event
	if err := c.get(ctx, fmt.Sprintf("/moto/events/%d", eventID), nil, &out); err != nil {
		return models.Event{}, err
	}

	return out, nil
}

func (c *Client) EventRaces(ctx context.Context, eventID int) ([]models.EventRace, error) {
	var out struct {
		Races []models.EventRace `json:"races"`
	}
	if err := c.get(ctx, fmt.Sprintf("/moto/events/%d/races", eventID), nil, &out); err != nil {
		return nil, err
	}

	return out.Races, nil
}

func (c *Client) EventResult(ctx context.Context, eventID int, class, raceType, region string) (models.RaceResult, error) {
	query := url.Values{}
	query.Set("class", class)
	query.Set("race_type", raceType)
	if strings.TrimSpace(region) != "" {
		query.Set("region", region)
	}

	var out models.RaceResult
	if err := c.get(ctx, fmt.Sprintf("/moto/events/%d/results", eventID), query, &out); err != nil {
		return models.RaceResult{}, err
	}

	return out, nil
}

func (c *Client) TripleCrownStandings(ctx context.Context, eventID int, class string) ([]models.StandingsRow, models.Event, error) {
	query := url.Values{}
	query.Set("class", class)

	var out struct {
		Event     models.Event          `json:"event"`
		Class     string                `json:"class"`
		Standings []models.StandingsRow `json:"standings"`
	}
	if err := c.get(ctx, fmt.Sprintf("/moto/events/%d/triple-crown-standings", eventID), query, &out); err != nil {
		return nil, models.Event{}, err
	}

	return out.Standings, out.Event, nil
}
