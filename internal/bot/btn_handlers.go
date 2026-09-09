package bot

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	tele "gopkg.in/telebot.v3"

	"mx_news_bot/internal/models"
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
			//{tele.ReplyButton{Text: ButtonPointsDistribution}, tele.ReplyButton{Text: ButtonSettings}},
		}, ResizeKeyboard: true,
	}
}

func (b *Bot) showUpcomingEvents(c tele.Context) error {
	ctx, cancel := b.reqCtx()
	defer cancel()

	events, err := b.app.GetUpcomingEvents(ctx)
	if err != nil {
		b.log.Error("Failed to get upcoming events: ", "error", err)
		return c.Send("Sorry, can't get upcoming events")
	}

	if len(events) == 0 {
		return c.Send("Sorry, no upcoming events in next 10 days")
	}

	for _, event := range events {
		resultText := b.formatter.FormatUpcomingEvents(event)
		err := c.Send(resultText, &tele.SendOptions{ParseMode: tele.ModeMarkdown})
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

func (b *Bot) showCurrentStandingsMenu(c tele.Context) error {
	return b.showSeasonSelectionMenu(c, uqSeasonResultPrefix, "Please select a season for standings:")
}

func (b *Bot) showChampionshipSchedulesMenu(c tele.Context) error {
	return b.showSeasonSelectionMenu(c, uqSeasonSchedulePrefix, "Please select a season for schedule:")
}

func (b *Bot) showEventResults(c tele.Context) error {
	return b.showSeasonSelectionMenu(c, uqSeasonEventsPrefix, "Please select a season for event results:")
}

func (b *Bot) showSeasonSelectionMenu(c tele.Context, prefix, title string) error {
	ctx, cancel := b.reqCtx()
	defer cancel()

	seasons, err := b.app.GetAvailableSeasons(ctx)
	if err != nil {
		b.log.Error("Failed to fetch seasons", "error", err)
		return c.Send("An error occurred while fetching seasons. Please try again later.")
	}

	if len(seasons) == 0 {
		return c.Send("No seasons found.")
	}

	buttons := make([]tele.InlineButton, len(seasons))
	for i, season := range seasons {
		buttons[i] = tele.InlineButton{
			Text:   strconv.Itoa(season),
			Unique: fmt.Sprintf("%s%d", prefix, season),
		}
	}

	replyMarkup := &tele.ReplyMarkup{InlineKeyboard: buttonsToGrid(buttons, 2)}
	return c.Send(title, replyMarkup)
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
	ctx, cancel := b.reqCtx()
	defer cancel()

	championships, err := b.app.GetAllChampionships(ctx)
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
	if !strings.Contains(strings.ToLower(event.ChampionshipName), "supercross") {
		return nil
	}

	// Build file name pattern, e.g. "Rd05*.png"
	seasonYear := event.Date.Year()
	pattern := fmt.Sprintf("Rd%02d*.png", event.RoundNumber)
	matches, err := filepath.Glob(filepath.Join(".", "maps", "SX", strconv.Itoa(seasonYear), pattern))
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
