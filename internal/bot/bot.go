package bot

import (
	"context"
	"time"

	"fmt"
	"log/slog"
	"sync"

	tele "gopkg.in/telebot.v3"

	"mx_news_bot/config"
	"mx_news_bot/internal/formatter"
	"mx_news_bot/internal/service"
)

type Bot struct {
	Client          *tele.Bot
	app             *service.BotBackend
	stateController *StateController
	formatter       *formatter.TgFormatter
	log             *slog.Logger
}

// reqCtx bounds one user interaction. telebot has no context of its own, so every call into the
// backend gets its own deadline rather than hanging on an unresponsive lap_vision.
const backendTimeout = 20 * time.Second

func (b *Bot) reqCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), backendTimeout)
}

func (b *Bot) Start() {
	b.Client.Start()
}

func (b *Bot) Stop() {
	b.Client.Stop()
}

// New builds the Telegram client. The webhook is served over plain HTTP because nginx terminates
// TLS in front of it and proxies the webhook path in: the bot has no certificate of its own, and
// the one this used to point at belonged to an unrelated domain, so any host but that one failed
// to start. PublicURL is what Telegram is told to call, and that is the https address.
func New(cfg *config.Bot, app *service.BotBackend, log *slog.Logger) *Bot {
	botClient, err := tele.NewBot(tele.Settings{
		Token: cfg.BotToken,
		Poller: &tele.Webhook{
			Listen: fmt.Sprintf("0.0.0.0:%d", cfg.Port),
			Endpoint: &tele.WebhookEndpoint{
				PublicURL: cfg.HookUrl,
			},
		},
		Verbose: cfg.BotVerbose,
	})

	if err != nil {
		log.Error("tg bot Client initial failed", "error", err)
		return nil
	}

	log.Info("Make bot")

	sc := &StateController{
		userStates: make(map[int64]string),
		app:        app,
		mu:         sync.Mutex{},
		log:        log,
	}

	tgFmt := formatter.NewTelegram()

	b := &Bot{
		Client:          botClient,
		app:             app,
		log:             log,
		stateController: sc,
		formatter:       tgFmt,
	}

	log.Info("Make handlers")
	b.setupHandlers()
	b.setupInlineHandlers()

	return b
}
