package config

import (
	"context"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/sethvargo/go-envconfig"
)

type Config struct {
	App     *App     `env:",prefix=APP_"`
	Metrics *Metrics `env:",prefix=METRICS_"`
	Http    *Http    `env:",prefix=HTTP_"`
	DB      *DB      `env:",prefix=DB_MAIN_"`
}

type App struct {
	Port       string `env:"PORT"`
	BotToken   string `env:"BOT_TOKEN, required"`
	HookUrl    string `env:"BOT_HOOK, required"`
	BotVerbose bool   `env:"BOT_VERBOSE"`
}

type Metrics struct {
	Port        string        `env:"PORT"`
	ReadTimeout time.Duration `env:"READ_TIMEOUT"`
}

type Http struct {
	ReadTimeout time.Duration `env:"READ_TIMEOUT"`
	IsLocal     bool          `env:"LOCAL"`
}

func New(ctx context.Context) (*Config, error) {
	var cfg Config

	if err := envconfig.Process(ctx, &cfg); err != nil {
		return nil, fmt.Errorf("process: %w", err)
	}
	return &cfg, nil
}
