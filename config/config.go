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
	DB      *DB      `env:",prefix=DB_MAINMX_"`
	Yandex  *Yandex  `env:",prefix=YANDEX_"`
}

type Yandex struct {
	BaseURL  string `env:"BASE_URL, required"`
	FolderID string `env:"FOLDER_ID, required"`
	ApiKey   string `env:"API_KEY, required"`
}

type App struct {
	Port         string              `env:"PORT"`
	BotToken     string              `env:"BOT_TOKEN, required"`
	HookUrl      string              `env:"BOT_HOOK, required"`
	BotVerbose   bool                `env:"BOT_VERBOSE"`
	ChampConfigs ChampionshipConfigs `env:",prefix=CHAMP_CONFIGS_"`
}

type ChampionshipConfigs struct {
	SXConfig    ChampionshipConfig `env:",prefix=SX_CONFIG"`
	ProMXConfig ChampionshipConfig `env:",prefix=PROMX_CONFIG"`
}

type ChampionshipConfig struct {
	BaseURL string `env:"BASE_URL, required"`
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
