package config

import (
	"sync"
	"time"

	kjp "github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	_ "github.com/lib/pq"
)

var once sync.Once

type Config struct {
	App     *App     `json:"app"`
	Metrics *Metrics `json:"metrics"`
	Http    *Http    `json:"http"`
	DB      *DB      `json:"database"`
	Yandex  *Yandex  `json:"yandex"`
}

type Yandex struct {
	BaseURL  string `json:"base_url,required"`
	FolderID string `json:"folder_id,required"`
	ApiKey   string `json:"api_key,required"`
}

type App struct {
	Bot          Bot                 `json:"bot"`
	ChampConfigs ChampionshipConfigs `json:"champ_configs"`
}

type Bot struct {
	Port       string `json:"port,required"`
	BotToken   string `json:"token,required"`
	HookUrl    string `json:"hook,required"`
	BotVerbose bool   `json:"verbose"`
}

type ChampionshipConfigs struct {
	SXConfig    ChampionshipConfig `json:"sx"`
	ProMXConfig ChampionshipConfig `json:"promx"`
}

type ChampionshipConfig struct {
	BaseURL string `json:"base_url"`
}

type Metrics struct {
	Port        string        `json:"port"`
	ReadTimeout time.Duration `json:"read_timeout"`
}

type Http struct {
	ReadTimeout time.Duration `json:"read_timeout"`
	IsLocal     bool          `json:"local"`
}

func New(configPath string) *Config {
	var c Config

	once.Do(func() {
		var ko = koanf.New(".")
		var cc Config

		err := ko.Load(file.Provider(configPath), kjp.Parser())
		if err != nil {
			panic(err)
		}

		err = ko.UnmarshalWithConf("", &cc, koanf.UnmarshalConf{Tag: "json", FlatPaths: false})
		if err != nil {
			panic(err)
		}

		c = cc
	})

	return &c
}

//func New(ctx context.Context) (*Config, error) {
//	var cfg Config
//
//	if err := envconfig.Process(ctx, &cfg); err != nil {
//		return nil, fmt.Errorf("process: %w", err)
//	}
//	return &cfg, nil
//}
