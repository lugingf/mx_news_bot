package bot

import (
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

func (b *Bot) Start() {
	b.Client.Start()
}

func (b *Bot) Stop() {
	b.Client.Stop()
}

func New(cfg *config.Bot, app *service.BotBackend, log *slog.Logger) *Bot {
	var tls *tele.WebhookTLS
	if !cfg.Local {
		tls = &tele.WebhookTLS{
			Cert: "/etc/letsencrypt/live/lugingfwebhookambot.com/fullchain.pem",
			Key:  "/etc/letsencrypt/live/lugingfwebhookambot.com/privkey.pem",
		}
	}
	botClient, err := tele.NewBot(tele.Settings{
		Token: cfg.BotToken,
		Poller: &tele.Webhook{
			Listen: fmt.Sprintf("0.0.0.0:%d", cfg.Port),
			Endpoint: &tele.WebhookEndpoint{
				PublicURL: cfg.HookUrl,
			},
			TLS: tls,
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
