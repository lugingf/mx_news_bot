package bot

import (
	tele "gopkg.in/telebot.v3"
	"strings"

	md "mx_news_bot/internal/bot/middleware"
)

const (
	StartCommand                = "/start"
	ButtonUpcomingEvents        = "🏁 Upcoming Events"
	ButtonCurrentStandings      = "📊 Current Standings"
	ButtonChampionshipSchedules = "📅 Championship Schedules"
	ButtonEventResults          = "📜 Event Results"
	ButtonPointsDistribution    = "🏆 Points Distribution"
	ButtonSettings              = "⚙️ Settings"
	ButtonNotifications         = "🔔 Notifications"
	ButtonDefaultChampionship   = "📅 Default Championship"
	ButtonBackToMainMenu        = "⬅️ Back to Main Menu"
	ButtonBackToSettings        = "⬅️ Back to Settings"
	ButtonEnableNotifications   = "✅ Enable Notifications"
	ButtonDisableNotifications  = "❌ Disable Notifications"
)

func (b *Bot) setupHandlers() {
	b.Client.Handle(StartCommand, b.startCmd, md.WithLogMiddleware)
	b.Client.Handle(&tele.ReplyButton{Text: ButtonUpcomingEvents}, b.showUpcomingEvents, md.WithLogMiddleware)
	b.Client.Handle(&tele.ReplyButton{Text: ButtonCurrentStandings}, b.showCurrentStandingsMenu, md.WithLogMiddleware)
	b.Client.Handle(&tele.ReplyButton{Text: ButtonChampionshipSchedules}, b.showChampionshipSchedulesMenu, md.WithLogMiddleware)
	b.Client.Handle(&tele.ReplyButton{Text: ButtonEventResults}, b.showEventResults, md.WithLogMiddleware)
	b.Client.Handle(&tele.ReplyButton{Text: ButtonPointsDistribution}, b.showPointsDistributionMenu, md.WithLogMiddleware)
	b.Client.Handle(&tele.ReplyButton{Text: ButtonSettings}, b.showSettingsMenu, md.WithLogMiddleware)
	b.Client.Handle(&tele.ReplyButton{Text: ButtonNotifications}, b.manageNotifications, md.WithLogMiddleware)
	b.Client.Handle(&tele.ReplyButton{Text: ButtonDefaultChampionship}, b.setDefaultChampionship, md.WithLogMiddleware)
	b.Client.Handle(&tele.ReplyButton{Text: ButtonBackToMainMenu}, func(c tele.Context) error { return b.startCmd(c) }, md.WithLogMiddleware)
	b.Client.Handle(&tele.ReplyButton{Text: ButtonBackToSettings}, b.showSettingsMenu, md.WithLogMiddleware)
}

const (
	uqShowAllEvents          = "show_all_events"
	uqEventPrefix            = "event_"
	uqRacePrefix             = "race_"
	uqChampSchedulePrefix    = "champ_schedule_"
	uqChampResultPrefix      = "champ_result_"
	uqChampClassResultPrefix = "champ_class_result_"
)

// Middleware to handle inline button callbacks
func (b *Bot) setupInlineHandlers() {
	b.Client.Handle(tele.OnCallback, func(c tele.Context) error {
		data := strings.TrimPrefix(c.Callback().Data, "\u000c")

		switch {
		case data == uqShowAllEvents:
			return b.showAllEvents(c)

		case strings.HasPrefix(data, uqEventPrefix):
			return b.showEventRaces(c, data)

		case strings.HasPrefix(data, uqRacePrefix):
			return b.showEventRaceResult(c, data)

		case strings.HasPrefix(data, uqChampSchedulePrefix):
			return b.showChampionshipScheduleFromNow(c, data)

		case strings.HasPrefix(data, uqChampResultPrefix):
			return b.showChampClassesMenuStandings(c, data)

		case strings.HasPrefix(data, uqChampClassResultPrefix):
			return b.showCurrentStandings(c, data)
		}

		b.log.Error("Failed to determine callback", "data", data)
		return nil
	}, md.WithLogMiddleware)
}
