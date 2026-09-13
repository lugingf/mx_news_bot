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
	App        *App        `json:"app"`
	Metrics    *Metrics    `json:"metrics"`
	DB         *DB         `json:"database"`
	LapVision  *LapVision  `json:"lap_vision"`
	Publishing *Publishing `json:"publishing"`
}

type App struct {
	Bot Bot `json:"bot"`
}

type Bot struct {
	Port       int    `json:"port"`
	BotToken   string `json:"token"`
	HookUrl    string `json:"hook"`
	BotVerbose bool   `json:"verbose"`
}

// LapVision is the results backend. The bot reads everything it shows through this API and
// receives publication requests from it, so there is no local racing data any more.
type LapVision struct {
	BaseURL        string        `json:"base_url"`
	InternalToken  string        `json:"internal_token"`
	RequestTimeout time.Duration `json:"request_timeout"`
	// WebhookSecret verifies the HMAC on incoming publication requests. Empty means the
	// receiver rejects everything, which is the safe default for a missing configuration.
	WebhookSecret string `json:"webhook_secret"`
}

type Publishing struct {
	Telegram  *TelegramChannel  `json:"telegram"`
	Twitter   *TwitterChannel   `json:"twitter"`
	Instagram *InstagramChannel `json:"instagram"`

	// Channels are the delivery destinations, declared here rather than inserted by hand. Every
	// one listed is written into delivery_channels on startup, so the config is what a deployment
	// is described by and the table is only where it ends up. A channel that is in the table but
	// not here is left alone: the administration screen can still add one for an experiment.
	Channels []DeliveryChannel `json:"channels"`
}

// DeliveryChannel is one destination and the posts it accepts.
//
// The three filters are lists, an empty one meaning "everything", and they are combined with AND:
// disciplines ["moto"] with championships ["AMA Supercross"] is a Supercross channel. Rehearsal
// channels take rehearsal posts and nothing else, and a live channel never takes one.
type DeliveryChannel struct {
	// Channel: "telegram", "twitter" or "instagram".
	Channel string `json:"channel"`
	// Target is the destination as the channel names it: for Telegram either a public @name or a
	// numeric chat id such as -1004362440814, which is the only address a private channel has.
	Target  string `json:"target"`
	Title   string `json:"title"`
	Enabled bool   `json:"enabled"`

	Rehearsal     bool     `json:"rehearsal"`
	Disciplines   []string `json:"disciplines"`
	Championships []string `json:"championships"`
	PostTypes     []string `json:"post_types"`
}

// TelegramChannel posts through the bot token already configured under app.bot; only the
// destination is channel-specific.
type TelegramChannel struct {
	Enabled bool   `json:"enabled"`
	ChatID  string `json:"chat_id"`
}

type TwitterChannel struct {
	Enabled           bool   `json:"enabled"`
	APIKey            string `json:"api_key"`
	APISecret         string `json:"api_secret"`
	AccessToken       string `json:"access_token"`
	AccessTokenSecret string `json:"access_token_secret"`
}

type InstagramChannel struct {
	Enabled     bool   `json:"enabled"`
	AccessToken string `json:"access_token"`
	AccountID   string `json:"account_id"`
}

type Metrics struct {
	Port        string        `json:"port"`
	ReadTimeout time.Duration `json:"read_timeout"`
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
