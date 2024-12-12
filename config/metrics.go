package config

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const AppName = "mx_news_bot"

const (
	labelApp    = "app"
	labelStatus = "status"
)

var (
	httpDurationSummary *prometheus.SummaryVec
)

func InitMetrics() {
	if httpDurationSummary == nil {
		httpDurationSummary = promauto.NewSummaryVec(prometheus.SummaryOpts{
			Name:       "http_request_duration_summary",
			Help:       "Duration of HTTP requests Summary.",
			Objectives: map[float64]float64{0.5: 0.5, 0.9: 0.9, 1: 1},
			AgeBuckets: 3,
			MaxAge:     120 * time.Second,
		}, []string{labelApp, labelStatus})
	}
}

func SaveHTTPDuration(timeSince time.Time, status string) {
	httpDurationSummary.With(map[string]string{
		labelApp:    AppName,
		labelStatus: status,
	}).Observe(float64(time.Since(timeSince).Milliseconds()))
}
