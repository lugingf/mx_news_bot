package formatter

import (
	"fmt"
	"strings"

	"mx_news_bot/internal/models"
)

const (
	EmojiError = "❗"
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
			"📅 *%s*\n🏆 Championship: %s\n📍 Location: %s\n📖 Status: %s\n\n",
			event.Name,
			event.ChampionshipName,
			event.Location,
			event.Status,
		))
	}

	return builder.String()
}
