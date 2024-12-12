package bot

import (
	tele "gopkg.in/telebot.v3"

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
	b.Client.Handle(&tele.ReplyButton{Text: ButtonBackToMainMenu}, func(c tele.Context) error {
		return b.startCmd(c)
	}, md.WithLogMiddleware)
	b.Client.Handle(&tele.ReplyButton{Text: ButtonBackToSettings}, b.showSettingsMenu, md.WithLogMiddleware)
}
