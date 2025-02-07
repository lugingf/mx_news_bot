package bot

import (
	"fmt"
	"github.com/pkg/errors"
	"mx_news_bot/internal/service"
	"sort"
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

func (b *Bot) showChampionshipScheduleFromNow(c tele.Context, uqData string) error {
	noPref := strings.TrimPrefix(uqData, uqChampSchedulePrefix)

	id, err := strconv.Atoi(noPref)
	if err != nil {
		b.log.Error("Bad unique Name data part", "unique_id", noPref, "error", err)
		return c.Respond(&tele.CallbackResponse{Text: "Sorry. Race data corrupted. We'll fix it soon"})
	}

	events, err := b.app.GetChampEvents(id)
	if err != nil {
		b.log.Error("Can't get champ events", "unique_id", noPref, "error", err)
		return c.Respond(&tele.CallbackResponse{Text: "Sorry. Data corrupted. We'll fix it soon"})
	}

	resultText := b.formatter.FormatEventsSchedule(events)
	err = c.Send(resultText, &tele.SendOptions{ParseMode: tele.ModeMarkdown})
	if err != nil {
		b.log.Error("Failed to send event schedule", "error", err)
		return errors.Wrap(err, "failed to send event schedule")
	}

	return nil
}

func (b *Bot) showChampionshipScheduleFull(c tele.Context, uqData string) error {
	return nil
}

// showEventRaceResult shows the details of a specific event
func (b *Bot) showEventRaceResult(c tele.Context, uqData string) error {
	noPref := strings.TrimPrefix(uqData, uqRacePrefix)
	parts := strings.Split(noPref, "_")
	if len(parts) != 3 {
		b.log.Error("Bad unique data parts", "unique", uqData)
		return c.Respond(&tele.CallbackResponse{Text: "Sorry. Race data corrupted. We'll fix it soon"})
	}

	eventID, err := strconv.Atoi(parts[0])
	if err != nil {
		b.log.Error("Bad unique Name data part", "unique_id", parts[0])
		return c.Respond(&tele.CallbackResponse{Text: "Sorry. Race data corrupted. We'll fix it soon"})
	}

	class := parts[1]
	raceType := parts[2]
	switch {
	case raceType == service.EventTypeTripleCrownStandings:
		results, event, err := b.app.GetTripleCrownStandings(eventID, class)
		if err != nil {
			b.log.Error("Event Race Result Triple: Failed to fetch result details",
				"eventID", eventID,
				"class", parts[1],
				"race", parts[2],
				"error", err,
			)

			return c.Respond(&tele.CallbackResponse{Text: "Failed to fetch result details."})
		}

		message := b.formatter.FormatTripleCrownResultTable(event, class, results)
		err = c.Send(message, &tele.SendOptions{ParseMode: tele.ModeMarkdown})
		if err != nil {
			b.log.Error("Failed to send event result", "error", err)
			return c.Respond(&tele.CallbackResponse{Text: "Failed to send event result."})
		}

	default:
		result, err := b.app.GetEventRaceResultByDetails(eventID, class, raceType)
		if err != nil {
			b.log.Error("Event Race Result Standard: to fetch result details",
				"eventID", eventID,
				"class", parts[1],
				"race", parts[2],
				"error", err,
			)

			return c.Respond(&tele.CallbackResponse{Text: "Failed to fetch result details."})
		}

		message := b.formatter.FormatEventResultTable(result)

		err = c.Send(message, &tele.SendOptions{ParseMode: tele.ModeMarkdown})
		if err != nil {
			b.log.Error("Failed to send event result", "error", err)
			return c.Respond(&tele.CallbackResponse{Text: "Failed to send event result."})
		}
	}

	return b.showEventRaces(c, fmt.Sprintf("%s%d", uqEventPrefix, eventID))
}

// showEventRaces shows the details of a specific event
func (b *Bot) showEventRaces(c tele.Context, uqData string) error {
	eventID, err := strconv.Atoi(strings.TrimPrefix(uqData, uqEventPrefix))
	if err != nil {
		b.log.Error("Failed to parse event Name", "data", uqData, "error", err)
		return c.Respond(&tele.CallbackResponse{Text: "Invalid event Name."})
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
	// Define the pattern: first row with 2 buttons, then 1, then 2, then 1, and repeat
	pattern := []int{2, 1, 2, 1}
	if len(races) == 8 {
		// Looks like we have Triple Crown
		pattern = []int{2, 1, 1, 2, 1, 1}
	}
	// Need to ensure we have correct order
	sort.Slice(races, func(i, j int) bool {
		if races[i].Class != races[j].Class {
			return races[i].Class < races[j].Class
		}
		return races[i].RaceType < races[j].RaceType
	})

	patternIndex := 0
	i := 0

	for i < len(races) {
		// Number of buttons in the current row according to the pattern
		rowCount := pattern[patternIndex]
		var row []tele.InlineButton

		// Add up to rowCount buttons to the current row
		for j := 0; j < rowCount && i < len(races); j++ {
			race := races[i]
			eventButton := tele.InlineButton{
				Unique: fmt.Sprintf("%s%d_%s_%s", uqRacePrefix, race.EventID, race.Class, race.RaceType),
				Text:   fmt.Sprintf(" %s - %s ", race.Class, race.RaceType),
			}
			row = append(row, eventButton)
			i++
		}

		// Append the current row to the inline buttons slice
		inlineButtons = append(inlineButtons, row)
		// Move to the next pattern element, wrapping around if necessary
		patternIndex = (patternIndex + 1) % len(pattern)
	}

	inlineMarkup := &tele.ReplyMarkup{InlineKeyboard: inlineButtons}
	return c.Send("Select a race:", inlineMarkup)
}

func (b *Bot) showChampClassesMenuStandings(c tele.Context, uqData string) error {
	noPref := strings.TrimPrefix(uqData, uqChampResultPrefix)

	id, err := strconv.Atoi(noPref)
	if err != nil {
		b.log.Error("Bad unique Name data part", "unique_id", noPref, "error", err)
		return c.Respond(&tele.CallbackResponse{Text: "Sorry. Race data corrupted. We'll fix it soon"})
	}

	classes, err := b.app.GetChampionshipClasses(id)
	if err != nil {
		b.log.Error("Failed to get championships with races", "error", err)
		return c.Send("An error occurred while listing champs. Please try again later.")
	}

	buttons := make([]tele.InlineButton, len(classes))
	for i, class := range classes {
		buttons[i] = tele.InlineButton{
			Text:   fmt.Sprintf("%s%s", class.Class, class.Region),
			Unique: fmt.Sprintf("%s%d_%s_%s", uqChampClassResultPrefix, id, class.Class, class.Region),
		}
	}

	replyMarkup := &tele.ReplyMarkup{InlineKeyboard: buttonsToGrid(buttons, 2)}
	return c.Send("Please select a class:", replyMarkup)
}

func (b *Bot) showCurrentStandings(c tele.Context, uqData string) error {
	noPref := strings.TrimPrefix(uqData, uqChampClassResultPrefix)
	parts := strings.Split(noPref, "_")
	if len(parts) < 3 {
		b.log.Error("Bad unique data parts", "unique", uqData)
		return c.Respond(&tele.CallbackResponse{Text: "Sorry. Race data corrupted. We'll fix it soon"})
	}

	id, err := strconv.Atoi(parts[0])
	if err != nil {
		b.log.Error("Bad unique Name data part", "unique_id", parts[0])
		return c.Respond(&tele.CallbackResponse{Text: "Sorry. Race data corrupted. We'll fix it soon"})
	}

	standings, err := b.app.GetCurrentStandings(id, parts[1], parts[2])
	if err != nil {
		b.log.Error("Failed to prepare standings", "error", err)
		return c.Send("Unable to prepare standings at the moment.")
	}

	resultText := b.formatter.FormatStandings(standings)
	err = c.Send(resultText, &tele.SendOptions{ParseMode: tele.ModeMarkdown})
	if err != nil {
		b.log.Error("Failed to send standings", "error", err)
		return errors.Wrap(err, "failed to send standings")
	}

	return b.showChampClassesMenuStandings(c, fmt.Sprintf("%s%d", uqChampResultPrefix, id))
}
