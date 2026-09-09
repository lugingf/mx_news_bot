package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"

	"github.com/pressly/goose/v3"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"mx_news_bot/config"
	"mx_news_bot/internal/adapters/lapvision"
	"mx_news_bot/internal/bot"
	"mx_news_bot/internal/publishing/builder"
	"mx_news_bot/internal/publishing/channel"
	"mx_news_bot/internal/publishing/dispatcher"
	"mx_news_bot/internal/publishing/render"
	"mx_news_bot/internal/service"
	"mx_news_bot/internal/storage"
	"mx_news_bot/internal/webhook"
)

func main() {
	configPath := flag.String("config", "config.json", "path to the configuration file")
	migrateUp := flag.Bool("migrate-up", false, "apply pending database migrations and exit")
	migrationsDir := flag.String("migrations", "infra/migrations", "directory holding the SQL migrations")
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	cfg := config.New(*configPath)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	dbm, err := config.OpenSQLXConn(cfg.DB)
	if err != nil {
		slog.Error("DB Connection Failed", "error", err)
		return
	}

	// Migrations run as their own invocation of this binary rather than at startup: a deploy
	// applies them once, before any new container serves traffic, instead of every replica
	// racing to migrate the same database.
	if *migrateUp {
		if err := runMigrations(dbm.DB, *migrationsDir); err != nil {
			logger.Error("migrations failed", "error", err)
			os.Exit(1)
		}
		logger.Info("migrations applied", "dir", *migrationsDir)
		return
	}

	repository := storage.New(dbm, logger)
	results := lapvision.New(cfg.LapVision.BaseURL, cfg.LapVision.InternalToken, cfg.LapVision.RequestTimeout)
	application := service.NewApp(results, repository, logger)
	botClient := bot.New(&cfg.App.Bot, application, logger)

	// The dispatcher fans one publication out to every registered channel. Telegram posts
	// through the same bot client that serves user requests; the other two are stubs until
	// their APIs are wired, and report themselves as not configured.
	publisher := dispatcher.New(repository, builder.DefaultRegistry(), logger)
	publisher.Register(render.NewTelegram(), channel.NewTelegram(botClient.Client))
	publisher.Register(render.NewTwitter(), channel.NewTwitter(cfg.Publishing.Twitter.Enabled))
	publisher.Register(render.NewInstagram(), channel.NewInstagram(cfg.Publishing.Instagram.Enabled))

	webhookHandler := webhook.New(cfg.LapVision.WebhookSecret, publisher, logger)

	// Metrics
	config.InitMetrics()
	go runMetricServer(cfg.Metrics, webhookHandler, logger)

	go func() {
		logger.Info("Listening on port", "port", cfg.App.Bot.Port)
		logger.Info("Start bot")
		botClient.Start()
	}()

	<-ctx.Done()
	logger.Info("Context done signal received")

	logger.Info("Collector gracefully shutdown")
}

// runMetricServer also carries the publication webhook. lap_vision posts there; Telegram does
// not, so this stays a separate listener from the bot's own webhook port.
func runMetricServer(cfg *config.Metrics, wh *webhook.Handler, log *slog.Logger) {
	mh := chi.NewRouter()
	mh.HandleFunc("/metrics", promhttp.Handler().ServeHTTP)
	mh.Post("/internal/publications", wh.Publications)

	// The bot itself only speaks Telegram webhooks, which a deploy cannot probe without
	// impersonating Telegram. This is what the blue-green health check calls instead.
	mh.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	srv := &http.Server{
		Addr:        fmt.Sprintf(":%s", cfg.Port),
		Handler:     mh,
		ReadTimeout: cfg.ReadTimeout,
	}

	log.Info(fmt.Sprintf("starting Metric exporter server: listening on %s", cfg.Port))

	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		log.Error("failed to listen promhandler server")
	}
}

func runMigrations(db *sql.DB, dir string) error {
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set migration dialect: %w", err)
	}
	if err := goose.Up(db, dir); err != nil {
		return fmt.Errorf("apply migrations from %s: %w", dir, err)
	}
	return nil
}
