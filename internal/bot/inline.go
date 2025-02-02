package bot

import (
	"fmt"

	tele "gopkg.in/telebot.v3"
)

// showAllEvents shows all completed events with inline buttons
func (b *Bot) showAllEvents(c tele.Context) error {
	events, err := b.app.GetCompletedEvents()
	if err != nil {
		b.log.Error("Failed to fetch events", "error", err)
		return c.Send("An error occurred while fetching events. Please try again later.")
	}

	var inlineButtons [][]tele.InlineButton
	for _, event := range events {
		eventButton := tele.InlineButton{
			Unique: fmt.Sprintf("%s%d", uqEventPrefix, event.ID),
			Text:   fmt.Sprintf("%s (%s)", event.Name, event.Date.Format("02.01.2006")),
		}
		inlineButtons = append(inlineButtons, []tele.InlineButton{eventButton})
	}

	b.log.Info("Showing all events", "events", events, "buttons", inlineButtons)

	inlineMarkup := &tele.ReplyMarkup{InlineKeyboard: inlineButtons}
	return c.Edit("All events:", inlineMarkup)
}

// showEventDetails shows the details of a specific event
func (b *Bot) showEventDetails(c tele.Context, eventID int) error {
	results, err := b.app.GetEventRacesResultByID(eventID)
	if err != nil {
		b.log.Error("Failed to fetch results details", "eventID", eventID, "error", err)
		return c.Respond(&tele.CallbackResponse{Text: "Failed to fetch results details."})
	}

	for _, result := range results {
		message := b.formatter.FormatEventResultTable(result)

		err = c.Send(message, &tele.SendOptions{ParseMode: tele.ModeMarkdown})
		if err != nil {
			b.log.Error("Failed to send event results", "error", err)
			return c.Respond(&tele.CallbackResponse{Text: "Failed to send event results."})
		}
	}

	return nil
}
