package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/robfig/cron/v3"
	"log/slog"
	"mx_news_bot/internal/downloader"
	"mx_news_bot/internal/parser"
	"mx_news_bot/internal/updater"
	"net/http"
	"os"
	"os/signal"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"mx_news_bot/config"
	"mx_news_bot/internal/bot"
	"mx_news_bot/internal/service"
	"mx_news_bot/internal/storage"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	cfg := config.New("config.json")
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	// Metrics
	config.InitMetrics()
	go runMetricServer(cfg.Metrics, logger)

	dbm, err := config.OpenSQLXConn(cfg.DB)
	if err != nil {
		slog.Error("DB Connection Failed", "error", err)
		return
	}

	repository := storage.New(dbm, logger)
	application := service.NewApp(
		repository,
		logger,
	)
	botClient := bot.New(&cfg.App.Bot, application, logger)

	go func() {
		logger.Info("Listening on port", "port", cfg.App.Bot.Port)
		logger.Info("Start bot")
		botClient.Start()
	}()

	// Results Checker
	sxCfg := cfg.App.ChampConfigs.SXConfig
	dwnlr := downloader.NewDownloader(sxCfg.BaseURL, sxCfg.DataDir, logger)
	prCfg := cfg.App.ParserConfig
	prsr := parser.New(repository, &parser.Config{
		DryRun:     prCfg.DryRun,
		DataDir:    prCfg.DataDir,
		OutputFile: prCfg.OutputFile,
	}, logger)

	updService := updater.New(repository, dwnlr, prsr, logger)

	c := cron.New()
	_, err = c.AddFunc(prCfg.CronRule, func() {
		logger.Info("Checking events")
		err := updService.Check()
		if err != nil {
			slog.Error("checker check failed", "error", err)
		}
	})
	logger.Info("Start events checker")
	c.Start()

	if err != nil {
		slog.Error("checker func add cron failed", "error", err)
		return
	}

	select {
	case <-ctx.Done():
		logger.Info("Context done signal received")
		c.Stop()
		//botClient.Stop()
	}

	logger.Info("Collector gracefully shutdown")
}

func runMetricServer(cfg *config.Metrics, log *slog.Logger) {
	mh := chi.NewRouter()
	mh.HandleFunc("/metrics", promhttp.Handler().ServeHTTP)

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
