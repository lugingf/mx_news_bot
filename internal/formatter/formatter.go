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
	EmojiScroll   = "📜"
)

type TgFormatter struct{}

func NewTelegram() *TgFormatter {
	return &TgFormatter{}
}

func (f *TgFormatter) FormatErrorMessage(message string) string {
	return EmojiError + " " + message
}

func (f *TgFormatter) FormatStandings(standings []models.Standing) string {
	if len(standings) == 0 {
		return EmojiError + " No standings data available."
	}

	var sb strings.Builder

	sb.WriteString("```\n")
	sb.WriteString(fmt.Sprintf("%s Championship Standings %s\n", EmojiBowl, EmojiBowl))
	sb.WriteString(strings.Repeat("=", 35) + "\n\n")

	sb.WriteString(fmt.Sprintf("%-3s | %-20s | %-6s\n", "#", "Rider", "Points"))
	sb.WriteString(strings.Repeat("-", 35) + "\n")

	for i, s := range standings {
		if i == 0 {
			sb.WriteString(fmt.Sprintf("%-3d | %-20s | %-3d %s\n", i+1, s.RiderName, s.Points, EmojiFlag))
		} else {
			sb.WriteString(fmt.Sprintf("%-3d | %-20s | %-3d\n", i+1, s.RiderName, s.Points))
		}
	}

	sb.WriteString("```")

	return sb.String()
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
		sb.WriteString(" *")
		sb.WriteString(event.Name)
		sb.WriteString("*\n")
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
	builder.WriteString(fmt.Sprintf("%-3s | %-3s | %-20s | %-7s\n", "Pos", "#", "Name", "Bike"))
	builder.WriteString(strings.Repeat("-", 45) + "\n")
	for _, rider := range event.Results {
		builder.WriteString(fmt.Sprintf(
			"%-3s | %-3s | %-20s | %-7s\n",
			rider.Position, rider.RiderNumber, rider.Name, rider.Bike,
		))
	}
	builder.WriteString("```\n")
	text := builder.String()

	return text
}

func (f *TgFormatter) FormatTripleCrownResultTable(event models.Event, class string, results []models.StandingsRow) string {
	var builder strings.Builder

	// Header with championship and event name
	builder.WriteString(fmt.Sprintf(
		"🏆 *%s - %s*\n\n",
		event.ChampionshipName, event.Name,
	))
	builder.WriteString(fmt.Sprintf(
		`%s *Date:* %s
%s *Track:* %s
%s *Race:* %s
%s *Round:* %s
%s *Class:* %s

`,
		EmojiCalendar, event.Date.Format("02.01.2006"),
		EmojiPin, event.Stadium,
		EmojiFlag, event.Format,
		EmojiBook, event.RoundNumber,
		EmojiBook, class,
	))
	builder.WriteString("```\n")
	// Table header
	builder.WriteString(fmt.Sprintf("%-2s | %-2s | %-13s | %-4s | %-2s | %-2s | %-2s | %-5s\n",
		" P", "#", "Name", "Bike", "R1", "R2", "R3", "Total"))
	builder.WriteString(strings.Repeat("-", 60) + "\n")

	// Table rows
	for _, row := range results {
		builder.WriteString(fmt.Sprintf("%-2d | %-2s | %-13s | %-4s | %-2d | %-2d | %-2d | %-5d\n",
			row.TotalPosition, row.RiderNumber, shortenName(row.Name), shortenBike(row.Bike), row.R1, row.R2, row.R3, row.TotalPoints))
	}
	builder.WriteString("\n```")

	return builder.String()
}

func shortenName(fullName string) string {
	parts := strings.Fields(fullName)
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return fmt.Sprintf("%c. %s", parts[0][0], parts[len(parts)-1])
}

func shortenBike(bike string) string {
	if len(bike) <= 3 {
		return bike
	}
	return bike[:3]
}
