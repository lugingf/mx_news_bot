package formatter

import (
	"fmt"
	"mx_news_bot/internal/models"
	"strings"
)

const (
	EmojiError    = "❗"
	EmojiCalendar = "📅"
	EmojiBowl     = "🏆"
	EmojiPin      = "📍"
	EmojiBook     = "📖"
	EmojiFlag     = "🏁"
)

type TgFormatter struct{}

func NewTelegram() *TgFormatter {
	return &TgFormatter{}
}

func (f *TgFormatter) FormatErrorMessage(message string) string {
	return EmojiError + " " + message
}

func (f *TgFormatter) FormatUpcomingEvents(events []models.Event) string {
	if len(events) == 0 {
		return "No upcoming events at the moment."
	}

	var builder strings.Builder
	builder.WriteString("🏁 *Upcoming Events* 🏁\n\n")
	for _, event := range events {
		builder.WriteString(fmt.Sprintf(
			`*%s*
%s *%s*
%s Championship: %s
%s Location: %s
%s Status: %s

`,
			event.Name,
			EmojiCalendar, event.Date.Format("02.01.2006"),
			EmojiBowl, event.ChampionshipName,
			EmojiPin, event.Stadium,
			EmojiBook, event.Status,
		))
	}

	return builder.String()
}

func (f *TgFormatter) FormatEventResultTable(event models.RaceResult) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf(
		"🏆 *%s - %s*\n\n",
		event.ChampName, event.EventName,
	))
	builder.WriteString(fmt.Sprintf(
		`%s *Date:* %s
%s *Location:* %s, %s
%s *Track:* %s
%s *Race:* %s
%s *Round:* %s of %s
%s *Class:* %s

`,
		EmojiCalendar, event.Date.Format("02.01.2006"),
		EmojiPin, event.City, event.State,
		EmojiBowl, event.Track,
		EmojiFlag, event.RaceType,
		EmojiBook, event.Round, event.TotalRounds,
		EmojiBook, event.Class,
	))

	if len(event.Results) == 0 {
		builder.WriteString("No race results available.")
		return builder.String()
	}

	builder.WriteString("*Race Results:*\n")
	builder.WriteString("```\n")
	builder.WriteString(fmt.Sprintf("%-3s | %-3s | %-20s | %-7s\n", "Pos", "#", "Rider", "Bike"))
	builder.WriteString(strings.Repeat("-", 45) + "\n")
	for _, rider := range event.Results {
		builder.WriteString(fmt.Sprintf(
			"%-3s | %-3s | %-20s | %-7s\n",
			rider.Position, rider.RiderNumber, rider.Rider, rider.Bike,
		))
	}
	builder.WriteString("```\n")
	text := builder.String()

	return text
}
