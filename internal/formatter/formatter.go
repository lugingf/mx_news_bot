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

func (f *TgFormatter) FormatEventsSchedule(events []models.Event) string {
	if len(events) == 0 {
		return ""
	}

	var sb strings.Builder

	// Header: display the championship name only once, enclosed in flag emojis
	sb.WriteString(EmojiFlag)
	sb.WriteString(" *")
	sb.WriteString(events[0].ChampionshipName)
	sb.WriteString("* ")
	sb.WriteString(EmojiFlag)
	sb.WriteString("\n")

	// Loop through each event and build the formatted schedule
	for _, event := range events {
		sb.WriteString(EmojiCalendar)
		sb.WriteString(" ")
		sb.WriteString(event.Date.Format("02 Jan 2006"))
		sb.WriteString("\n• ")
		sb.WriteString(EmojiBowl)
		sb.WriteString("*")
		sb.WriteString(event.Name)
		sb.WriteString("*")
		sb.WriteString(" Round: ")
		sb.WriteString(event.RoundNumber)
		sb.WriteString(" | ")
		sb.WriteString(EmojiBook)
		sb.WriteString(" ")
		sb.WriteString(event.Format)
		if event.Classes != "" {
			sb.WriteString(" | ")
			sb.WriteString(" (")
			sb.WriteString(event.Classes)
			sb.WriteString(")")
		}
		sb.WriteString("\n\n")
	}

	return sb.String()
}

func (f *TgFormatter) FormatUpcomingEvents(event models.Event) string {
	var builder strings.Builder
	builder.WriteString("🏁 *Upcoming Events* 🏁\n\n")
	builder.WriteString(fmt.Sprintf(
		`*%s - %s*

%s *Date %s*
%s *Round*: %s
%s *Staduim*: %s
%s *Format*: %s

`,
		event.Name, event.ChampionshipName,
		EmojiCalendar, event.Date.Format("02 Jan 2006"),
		EmojiBowl, event.RoundNumber,
		EmojiPin, event.Stadium,
		EmojiBook, event.Format,
	))

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
