package middleware

import (
	"encoding/json"
	"log/slog"
	"os"
	"time"

	tb "gopkg.in/telebot.v3"

	"mx_news_bot/config"
)

func WithLogMiddleware(next tb.HandlerFunc) tb.HandlerFunc {
	return func(c tb.Context) error {
		start := time.Now()
		_, err := json.Marshal(c.Update())
		logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
		if err != nil {
			logger.Error("unmarshal update failed", "error", err)
		} else {
			logger.Info("update received", "user_id", c.Sender().ID, "username", c.Sender().Username)
		}

		status := "success"
		err = next(c)
		if err != nil {
			status = "failed"
		}

		config.SaveHTTPDuration(start, status)
		return err
	}
}
