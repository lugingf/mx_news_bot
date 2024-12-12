package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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

	cfg, err := config.New(ctx)
	if err != nil {
		slog.Error("Config initialization failed", "error", err)
		return
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

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

	botClient := bot.New(cfg, application, logger)

	config.InitMetrics()
	go runMetricServer(cfg.Metrics, logger)

	go func() {
		logger.Info("Listening on port", "port", cfg.App.Port)
		logger.Info("Start bot")
		botClient.Start()
	}()

	select {
	case <-ctx.Done():
		logger.Info("Context done signal received")
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
