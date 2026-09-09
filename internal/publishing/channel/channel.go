package channel

import (
	"context"
	"errors"
	"fmt"

	tele "gopkg.in/telebot.v3"

	"mx_news_bot/internal/publishing/contentmodel"
)

// ErrNotConfigured marks a channel that exists in code but has no working credentials. The
// dispatcher records it as skipped rather than failed, so a stub never looks like an outage.
var ErrNotConfigured = errors.New("channel is not configured")

// Receipt is what a channel returns on success. Ref is the channel's own id for the post, kept
// so a later edit or delete can find it.
type Receipt struct {
	Channel string
	Ref     string
}

// Publisher sends one rendered message to one channel.
type Publisher interface {
	Name() string
	Publish(ctx context.Context, target string, message contentmodel.Message) (Receipt, error)
}

// TelegramSender is the slice of telebot the publisher needs, so tests do not need a real bot.
type TelegramSender interface {
	Send(to tele.Recipient, what any, opts ...any) (*tele.Message, error)
}

type recipient string

func (r recipient) Recipient() string { return string(r) }

type Telegram struct {
	sender TelegramSender
}

func NewTelegram(sender TelegramSender) *Telegram {
	return &Telegram{sender: sender}
}

func (*Telegram) Name() string { return "telegram" }

func (t *Telegram) Publish(ctx context.Context, target string, message contentmodel.Message) (Receipt, error) {
	if t.sender == nil {
		return Receipt{}, ErrNotConfigured
	}
	if err := ctx.Err(); err != nil {
		return Receipt{}, err
	}

	parseMode := message.ParseMode
	if parseMode == "" {
		parseMode = tele.ModeMarkdown
	}

	sent, err := t.sender.Send(recipient(target), message.Text, &tele.SendOptions{ParseMode: parseMode})
	if err != nil {
		return Receipt{}, fmt.Errorf("telegram: send to %s: %w", target, err)
	}

	return Receipt{Channel: t.Name(), Ref: fmt.Sprintf("%d", sent.ID)}, nil
}

// Twitter is a stub with the real signature. It is registered so the wiring, the channel table
// and the dispatcher are exercised end to end before the API client exists.
type Twitter struct {
	Configured bool
}

func NewTwitter(configured bool) *Twitter { return &Twitter{Configured: configured} }

func (*Twitter) Name() string { return "twitter" }

func (t *Twitter) Publish(context.Context, string, contentmodel.Message) (Receipt, error) {
	if !t.Configured {
		return Receipt{}, ErrNotConfigured
	}

	return Receipt{}, fmt.Errorf("twitter: publishing is not implemented yet")
}

type Instagram struct {
	Configured bool
}

func NewInstagram(configured bool) *Instagram { return &Instagram{Configured: configured} }

func (*Instagram) Name() string { return "instagram" }

func (i *Instagram) Publish(context.Context, string, contentmodel.Message) (Receipt, error) {
	if !i.Configured {
		return Receipt{}, ErrNotConfigured
	}

	return Receipt{}, fmt.Errorf("instagram: publishing is not implemented yet")
}
