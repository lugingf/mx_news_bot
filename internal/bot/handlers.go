package bot

import (
	"fmt"
	"github.com/pkg/errors"
	tele "gopkg.in/telebot.v3"
	"mx_news_bot/internal/models"
	"os"
	"path/filepath"
)

// startCmd - command to start bot and show main menu
func (b *Bot) startCmd(c tele.Context) error {
	return c.Send("Welcome to MX News Bot! Use the menu to navigate.", b.mainMenu())
}

func (b *Bot) mainMenu() *tele.ReplyMarkup {
	return &tele.ReplyMarkup{
		ReplyKeyboard: [][]tele.ReplyButton{
			{tele.ReplyButton{Text: ButtonUpcomingEvents}, tele.ReplyButton{Text: ButtonCurrentStandings}},
			{tele.ReplyButton{Text: ButtonChampionshipSchedules}, tele.ReplyButton{Text: ButtonEventResults}},
			{tele.ReplyButton{Text: ButtonPointsDistribution}, tele.ReplyButton{Text: ButtonSettings}},
		}, ResizeKeyboard: true,
	}
}

func (b *Bot) showUpcomingEvents(c tele.Context) error {
	events, err := b.app.GetUpcomingEvents()
	if err != nil {
		b.log.Error("Failed to get upcoming events: ", "error", err)
		return c.Send("Sorry, can't get upcoming events")
	}

	if len(events) == 0 {
		return c.Send("Sorry, no upcoming events in next 10 days")
	}

	for _, event := range events {
		resultText := b.formatter.FormatUpcomingEvents(event)
		err := c.Send(resultText)
		if err != nil {
			b.log.Error("Failed to send event", "error", err)
			continue
		}

		err = b.sendEventMaps(c, event)
		if err != nil {
			b.log.Error("Failed to send map pics", "error", err)
			continue
		}
	}

	return nil
}

func (b *Bot) handleSelectChampionshipForEvents(c tele.Context) error {
	championships, err := b.app.GetAllChampionships()
	if err != nil {
		b.log.Error("Failed to fetch championships", "error", err)
		return c.Send("Unable to fetch championships at the moment.")
	}

	buttons := make([]tele.InlineButton, len(championships))
	for i, champ := range championships {
		buttons[i] = tele.InlineButton{
			Text:   champ.Name,
			Unique: fmt.Sprintf("select_champ_%d", champ.ID),
		}
	}

	replyMarkup := &tele.ReplyMarkup{InlineKeyboard: buttonsToGrid(buttons, 2)}
	return c.Send("Please select a championship:", replyMarkup)
}

func (b *Bot) showCurrentStandingsMenu(c tele.Context) error {
	replyMarkup := &tele.ReplyMarkup{
		ReplyKeyboard: [][]tele.ReplyButton{
			{tele.ReplyButton{Text: ButtonChampionshipSchedules}},
			{tele.ReplyButton{Text: ButtonBackToMainMenu}},
		}, ResizeKeyboard: true,
	}
	return c.Send(ButtonCurrentStandings, replyMarkup)
}

func (b *Bot) showChampionshipSchedulesMenu(c tele.Context) error {
	replyMarkup := &tele.ReplyMarkup{
		ReplyKeyboard: [][]tele.ReplyButton{
			{tele.ReplyButton{Text: ButtonEventResults}, tele.ReplyButton{Text: ButtonPointsDistribution}},
			{tele.ReplyButton{Text: ButtonBackToMainMenu}},
		}, ResizeKeyboard: true,
	}
	return c.Send(ButtonChampionshipSchedules, replyMarkup)
}

func (b *Bot) showEventResults(c tele.Context) error {
	events, err := b.app.GetCompletedEvents()
	if err != nil {
		b.log.Error("Failed to fetch events", "error", err)
		return c.Send("An error occurred while fetching events. Please try again later.")
	}

	const maxVisibleEvents = 5
	var inlineButtons [][]tele.InlineButton

	for i, event := range events {
		if i >= maxVisibleEvents {
			break
		}
		eventButton := tele.InlineButton{
			Unique: fmt.Sprintf("%s%d", uqEventPrefix, event.ID),
			Text:   fmt.Sprintf("%s (%s)", event.Name, event.Date.Format("02.01.2006")),
		}
		inlineButtons = append(inlineButtons, []tele.InlineButton{eventButton})
	}

	if len(events) > maxVisibleEvents {
		showAllButton := tele.InlineButton{
			Unique: uqShowAllEvents,
			Text:   "Show All Events",
		}
		inlineButtons = append(inlineButtons, []tele.InlineButton{showAllButton})
	}

	b.log.Info("Showing event results", "events", events, "buttons", inlineButtons)

	inlineMarkup := &tele.ReplyMarkup{InlineKeyboard: inlineButtons}
	return c.Send("Select an event to see the results:", inlineMarkup)
}

func (b *Bot) showPointsDistributionMenu(c tele.Context) error {
	replyMarkup := &tele.ReplyMarkup{
		ReplyKeyboard: [][]tele.ReplyButton{
			{tele.ReplyButton{Text: ButtonChampionshipSchedules}},
			{tele.ReplyButton{Text: ButtonBackToMainMenu}},
		}, ResizeKeyboard: true,
	}
	return c.Send(ButtonPointsDistribution, replyMarkup)
}

func (b *Bot) showSettingsMenu(c tele.Context) error {
	replyMarkup := &tele.ReplyMarkup{
		ReplyKeyboard: [][]tele.ReplyButton{
			{tele.ReplyButton{Text: ButtonNotifications}, tele.ReplyButton{Text: ButtonDefaultChampionship}},
			{tele.ReplyButton{Text: ButtonBackToMainMenu}},
		}, ResizeKeyboard: true,
	}
	return c.Send(ButtonSettings, replyMarkup)
}

func (b *Bot) manageNotifications(c tele.Context) error {
	replyMarkup := &tele.ReplyMarkup{
		ReplyKeyboard: [][]tele.ReplyButton{
			{tele.ReplyButton{Text: ButtonEnableNotifications}, tele.ReplyButton{Text: ButtonDisableNotifications}},
			{tele.ReplyButton{Text: ButtonBackToSettings}},
		}, ResizeKeyboard: true,
	}
	return c.Send(ButtonNotifications, replyMarkup)
}

// Set Default Championship Handler
func (b *Bot) setDefaultChampionship(c tele.Context) error {
	// Fetch championships from the repository
	championships, err := b.app.GetAllChampionships()
	if err != nil {
		b.log.Error("Failed to fetch championships", "error", err)
		return c.Send("Unable to fetch championships at the moment.")
	}

	// Display championships as InlineKeyboard options
	buttons := make([]tele.InlineButton, len(championships))
	for i, champ := range championships {
		buttons[i] = tele.InlineButton{
			Text:   champ.Name,
			Unique: fmt.Sprintf("default_champ_%d", champ.ID),
		}
	}

	replyMarkup := &tele.ReplyMarkup{InlineKeyboard: buttonsToGrid(buttons, 2)}
	return c.Send("Please select a championship to set as default:", replyMarkup)
}

func (b *Bot) sendEventMaps(c tele.Context, event models.Event) error {
	// Build file name pattern, e.g. "Rd05*.png"
	pattern := fmt.Sprintf("Rd%02d*.png", event.RoundNumber)
	matches, err := filepath.Glob(fmt.Sprintf("./maps/SX/%d/%s", event.Date.Year(), pattern))
	if err != nil {
		b.log.Error("Error searching files", "pattern", pattern, "error", err)
		return errors.Wrap(err, "searching files")
	}

	if len(matches) == 0 {
		b.log.Info("No image files found for event", "pattern", pattern)
		return nil
	}

	var album tele.Album
	for _, fileName := range matches {
		if _, err := os.Stat(fileName); os.IsNotExist(err) {
			b.log.Error("File not found on disk", "file", fileName)
			continue
		}
		photo := &tele.Photo{
			File: tele.FromDisk(fileName),
		}
		album = append(album, photo)
	}
	if len(album) > 0 {
		if err := c.SendAlbum(album); err != nil {
			b.log.Error("Failed to send album", "error", err)
		}
	}

	return nil
}

// Function to create a grid of buttons (helper for inline button layout)
func buttonsToGrid(buttons []tele.InlineButton, cols int) [][]tele.InlineButton {
	var grid [][]tele.InlineButton
	for i := 0; i < len(buttons); i += cols {
		end := i + cols
		if end > len(buttons) {
			end = len(buttons)
		}
		grid = append(grid, buttons[i:end])
	}
	return grid
}
