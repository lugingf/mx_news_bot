package builder

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"mx_news_bot/internal/models"
	"mx_news_bot/internal/publishing/contentmodel"
	"mx_news_bot/internal/publishing/contract"
)

// Builder turns one payload type into a channel-independent Post.
type Builder interface {
	Type() string
	Build(envelope contract.Envelope) (contentmodel.Post, error)
}

// Registry picks the builder for an envelope. A new content type is a new Builder plus one
// Register call; nothing existing changes.
type Registry struct {
	builders map[string]Builder
}

func NewRegistry(builders ...Builder) *Registry {
	registry := &Registry{builders: make(map[string]Builder, len(builders))}
	for _, b := range builders {
		registry.Register(b)
	}

	return registry
}

func (r *Registry) Register(b Builder) {
	r.builders[b.Type()] = b
}

func (r *Registry) Build(envelope contract.Envelope) (contentmodel.Post, error) {
	b, ok := r.builders[envelope.Type]
	if !ok {
		return contentmodel.Post{}, fmt.Errorf("no builder for event type %q", envelope.Type)
	}

	return b.Build(envelope)
}

// DefaultRegistry wires the four types lap_vision publishes today.
func DefaultRegistry() *Registry {
	return NewRegistry(
		RaceResultBuilder{},
		StandingsBuilder{},
		EventUpcomingBuilder{},
		EventScheduleBuilder{},
	)
}

type RaceResultBuilder struct{}

func (RaceResultBuilder) Type() string { return contract.TypeRaceResult }

func (RaceResultBuilder) Build(envelope contract.Envelope) (contentmodel.Post, error) {
	var payload contract.RaceResultPayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		return contentmodel.Post{}, fmt.Errorf("race_result: decode payload: %w", err)
	}

	result := payload.Result
	post := contentmodel.Post{
		Title:      fmt.Sprintf("%s — %s", result.ChampName, result.EventName),
		Subtitle:   strings.TrimSpace(result.Class + " " + result.RaceType),
		OccurredAt: envelope.OccurredAt,
		Meta: []contentmodel.Field{
			{Label: "Date", Value: result.Date.Format("02 Jan 2006"), Emoji: "📅"},
			{Label: "Track", Value: result.Track, Emoji: "🏆"},
		},
		Tags: []string{"moto", tagify(result.ChampName), tagify(result.Class)},
	}

	if location := joinNonEmpty(", ", result.City, result.State); location != "" {
		post.Meta = append(post.Meta, contentmodel.Field{Label: "Location", Value: location, Emoji: "📍"})
	}
	if round := joinNonEmpty(" of ", result.Round, result.TotalRounds); round != "" {
		post.Meta = append(post.Meta, contentmodel.Field{Label: "Round", Value: round, Emoji: "📖"})
	}

	table := &contentmodel.Table{
		Columns: []contentmodel.Column{
			{Header: "Pos", Width: 3, Align: contentmodel.AlignRight},
			{Header: "#", Width: 3, Align: contentmodel.AlignRight},
			{Header: "Rider", Width: 20},
			{Header: "Bike", Width: 7},
		},
	}
	for _, rider := range result.Results {
		table.Rows = append(table.Rows, []string{
			strconv.Itoa(rider.Position), rider.RiderNumber, rider.Name, rider.Bike,
		})
	}
	post.Table = table

	return post, nil
}

type StandingsBuilder struct{}

func (StandingsBuilder) Type() string { return contract.TypeStandings }

func (StandingsBuilder) Build(envelope contract.Envelope) (contentmodel.Post, error) {
	var payload contract.StandingsPayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		return contentmodel.Post{}, fmt.Errorf("standings: decode payload: %w", err)
	}

	post := contentmodel.Post{
		Title:      fmt.Sprintf("%s — Championship Standings", payload.Championship.Name),
		Subtitle:   strings.TrimSpace(payload.Class + " " + payload.Region),
		OccurredAt: envelope.OccurredAt,
		Tags:       []string{"moto", tagify(payload.Championship.Name), tagify(payload.Class)},
	}

	if payload.AfterEvent.Name != "" {
		post.Meta = append(post.Meta, contentmodel.Field{
			Label: "After", Value: payload.AfterEvent.Name, Emoji: "🏁",
		})
	}

	table := &contentmodel.Table{
		Columns: []contentmodel.Column{
			{Header: "#", Width: 3, Align: contentmodel.AlignRight},
			{Header: "Rider", Width: 20},
			{Header: "Points", Width: 6, Align: contentmodel.AlignRight},
		},
	}
	for i, standing := range payload.Standings {
		table.Rows = append(table.Rows, []string{
			strconv.Itoa(i + 1), standing.RiderName, strconv.Itoa(standing.Points),
		})
	}
	post.Table = table

	return post, nil
}

type EventUpcomingBuilder struct{}

func (EventUpcomingBuilder) Type() string { return contract.TypeEventUpcoming }

func (EventUpcomingBuilder) Build(envelope contract.Envelope) (contentmodel.Post, error) {
	var payload contract.EventUpcomingPayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		return contentmodel.Post{}, fmt.Errorf("event_upcoming: decode payload: %w", err)
	}

	event := payload.Event
	post := contentmodel.Post{
		Title:      fmt.Sprintf("Up next: %s", event.Name),
		Subtitle:   firstNonEmpty(event.ChampionshipName, payload.Championship.Name),
		OccurredAt: envelope.OccurredAt,
		Meta: []contentmodel.Field{
			{Label: "Date", Value: event.Date.Format("02 Jan 2006"), Emoji: "📅"},
			{Label: "Round", Value: strconv.Itoa(event.RoundNumber), Emoji: "🏆"},
		},
		Tags: []string{"moto", tagify(firstNonEmpty(event.ChampionshipName, payload.Championship.Name))},
	}

	if event.Stadium != "" {
		post.Meta = append(post.Meta, contentmodel.Field{Label: "Venue", Value: event.Stadium, Emoji: "📍"})
	}
	if event.Format != "" {
		post.Meta = append(post.Meta, contentmodel.Field{Label: "Format", Value: event.Format, Emoji: "📖"})
	}
	if event.Classes != "" {
		post.Sections = append(post.Sections, contentmodel.Section{Heading: "Classes", Body: event.Classes})
	}

	return post, nil
}

type EventScheduleBuilder struct{}

func (EventScheduleBuilder) Type() string { return contract.TypeEventSchedule }

func (EventScheduleBuilder) Build(envelope contract.Envelope) (contentmodel.Post, error) {
	var payload contract.EventSchedulePayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		return contentmodel.Post{}, fmt.Errorf("event_schedule: decode payload: %w", err)
	}

	// Rounds are shown in calendar order regardless of how the sender listed them.
	events := make([]models.Event, len(payload.Events))
	copy(events, payload.Events)
	sort.SliceStable(events, func(a, b int) bool {
		return events[a].Date.Before(events[b].Date)
	})

	post := contentmodel.Post{
		Title:      fmt.Sprintf("%s — Schedule", payload.Championship.Name),
		OccurredAt: envelope.OccurredAt,
		Tags:       []string{"moto", tagify(payload.Championship.Name)},
	}

	table := &contentmodel.Table{
		Columns: []contentmodel.Column{
			{Header: "Rd", Width: 3, Align: contentmodel.AlignRight},
			{Header: "Date", Width: 11},
			{Header: "Event", Width: 22},
		},
	}
	for _, event := range events {
		table.Rows = append(table.Rows, []string{
			strconv.Itoa(event.RoundNumber), event.Date.Format("02 Jan 2006"), event.Name,
		})
	}
	post.Table = table

	return post, nil
}

func tagify(value string) string {
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + 32)
		}
	}

	return b.String()
}

func joinNonEmpty(sep string, values ...string) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			parts = append(parts, strings.TrimSpace(value))
		}
	}

	return strings.Join(parts, sep)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}

	return ""
}
