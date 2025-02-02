package bot

import (
	"fmt"
	"strconv"
	"strings"

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

// showEventRaceResult shows the details of a specific event
func (b *Bot) showEventRaceResult(c tele.Context, uqData string) error {
	noPref := strings.TrimPrefix(uqData, uqRacePrefix)
	parts := strings.Split(noPref, "_")
	if len(parts) != 3 {
		b.log.Error("Bad unique data parts", "unique", uqData)
		return c.Respond(&tele.CallbackResponse{Text: "Sorry. Race data corrupted. We'll fix it soon"})
	}

	id, err := strconv.Atoi(parts[0])
	if err != nil {
		b.log.Error("Bad unique ID data part", "unique_id", parts[0])
		return c.Respond(&tele.CallbackResponse{Text: "Sorry. Race data corrupted. We'll fix it soon"})
	}

	results, err := b.app.GetEventRaceResultByDetails(id, parts[1], parts[2])
	if err != nil {
		b.log.Error("Failed to fetch results details",
			"eventID", id,
			"class", parts[1],
			"race", parts[2],
			"error", err,
		)

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

// showEventRaces shows the details of a specific event
func (b *Bot) showEventRaces(c tele.Context, uqData string) error {
	eventID, err := strconv.Atoi(strings.TrimPrefix(uqData, uqEventPrefix))
	if err != nil {
		b.log.Error("Failed to parse event ID", "data", uqData, "error", err)
		return c.Respond(&tele.CallbackResponse{Text: "Invalid event ID."})
	}

	races, err := b.app.GetEventRaces(eventID)
	if err != nil {
		b.log.Error("Failed to fetch results details", "eventID", eventID, "error", err)
		return c.Respond(&tele.CallbackResponse{Text: "Failed to fetch results details."})
	}

	if races == nil {
		b.log.Error("No races found for event", "eventID", eventID, "races", len(races))
		return c.Respond(&tele.CallbackResponse{Text: "Sorry, no data for this event races"})
	}

	var inlineButtons [][]tele.InlineButton
	for _, race := range races {
		eventButton := tele.InlineButton{
			Unique: fmt.Sprintf("%s%d_%s_%s", uqEventPrefix, race.EventID, race.Class, race.RaceType),
			Text:   fmt.Sprintf("%s - %s", race.Class, race.RaceType),
		}
		inlineButtons = append(inlineButtons, []tele.InlineButton{eventButton})
	}

	inlineMarkup := &tele.ReplyMarkup{InlineKeyboard: inlineButtons}
	return c.Send("Select a race:", inlineMarkup)
}
