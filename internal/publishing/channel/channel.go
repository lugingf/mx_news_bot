package channel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

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

// telegramCaptionLimit is Telegram's own cap on a photo or video caption. A message longer than
// this cannot ride along as the caption — sending it as one would have the whole call rejected,
// image included — so it follows as its own plain message instead of being cut off silently.
const telegramCaptionLimit = 1024

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
	opts := &tele.SendOptions{ParseMode: parseMode}

	media := primaryMedia(message.Media)
	if media == nil {
		sent, err := t.sender.Send(recipient(target), message.Text, opts)
		if err != nil {
			return Receipt{}, fmt.Errorf("telegram: send to %s: %w", target, err)
		}

		return Receipt{Channel: t.Name(), Ref: fmt.Sprintf("%d", sent.ID)}, nil
	}

	caption := message.Text
	textFollowsSeparately := len([]rune(caption)) > telegramCaptionLimit
	if textFollowsSeparately {
		caption = ""
	}

	file := tele.FromURL(media.URL)
	var attachment any
	switch media.Kind {
	case contentmodel.MediaVideo:
		attachment = &tele.Video{File: file, Caption: caption}
	default:
		attachment = &tele.Photo{File: file, Caption: caption}
	}

	sent, err := t.sender.Send(recipient(target), attachment, opts)
	if err != nil {
		return Receipt{}, fmt.Errorf("telegram: send to %s: %w", target, err)
	}

	if textFollowsSeparately {
		if _, err := t.sender.Send(recipient(target), message.Text, opts); err != nil {
			return Receipt{}, fmt.Errorf("telegram: send follow-up text to %s: %w", target, err)
		}
	}

	return Receipt{Channel: t.Name(), Ref: fmt.Sprintf("%d", sent.ID)}, nil
}

// primaryMedia is the one attachment Telegram gets — a single photo or video per post, not an
// album — the first item in the list that actually carries a URL.
func primaryMedia(media []contentmodel.Media) *contentmodel.Media {
	for i := range media {
		if strings.TrimSpace(media[i].URL) != "" {
			return &media[i]
		}
	}

	return nil
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

// InstagramHTTPClient is the slice of http.Client the publisher needs, so tests do not need a
// real Graph API.
type InstagramHTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// InstagramAccount is one authorised account: a Graph API access token is issued to a single
// Instagram business account and cannot post for another.
type InstagramAccount struct {
	AccountID   string
	AccessToken string
}

// maxCarouselItems is Instagram's own limit on how many images one carousel post may carry.
const maxCarouselItems = 10

// Instagram posts through the Graph API: a media container is created from a public image URL,
// then published. Several images become a carousel — child containers first, then a parent that
// lists them.
type Instagram struct {
	tokens  map[string]string
	client  InstagramHTTPClient
	baseURL string
}

func NewInstagram(enabled bool, accounts []InstagramAccount, client InstagramHTTPClient) *Instagram {
	if !enabled {
		return &Instagram{}
	}

	tokens := make(map[string]string, len(accounts))
	for _, account := range accounts {
		if account.AccountID == "" || account.AccessToken == "" {
			continue
		}
		tokens[account.AccountID] = account.AccessToken
	}
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	return &Instagram{tokens: tokens, client: client, baseURL: "https://graph.instagram.com"}
}

func (*Instagram) Name() string { return "instagram" }

// Publish expects target to be the Instagram business account's own id — the same value the
// delivery channel names it by — so a token bound to the wrong account is never used.
func (i *Instagram) Publish(ctx context.Context, target string, message contentmodel.Message) (Receipt, error) {
	if len(i.tokens) == 0 {
		return Receipt{}, ErrNotConfigured
	}
	token, ok := i.tokens[target]
	if !ok {
		return Receipt{}, fmt.Errorf("instagram: no access token configured for account %s", target)
	}
	if len(message.Media) == 0 {
		return Receipt{}, errors.New("instagram: post has no image to publish")
	}

	media := message.Media
	if len(media) > maxCarouselItems {
		media = media[:maxCarouselItems]
	}

	creationID, err := i.createContainer(ctx, target, token, media, message.Text)
	if err != nil {
		return Receipt{}, err
	}

	mediaID, err := i.publishContainer(ctx, target, token, creationID)
	if err != nil {
		return Receipt{}, err
	}

	return Receipt{Channel: i.Name(), Ref: mediaID}, nil
}

func (i *Instagram) createContainer(ctx context.Context, accountID, token string, media []contentmodel.Media, caption string) (string, error) {
	if len(media) == 1 {
		return i.createMediaContainer(ctx, accountID, token, url.Values{
			"image_url": {media[0].URL},
			"caption":   {caption},
		})
	}

	children := make([]string, 0, len(media))
	for _, item := range media {
		childID, err := i.createMediaContainer(ctx, accountID, token, url.Values{
			"image_url":        {item.URL},
			"is_carousel_item": {"true"},
		})
		if err != nil {
			return "", err
		}
		children = append(children, childID)
	}

	return i.createMediaContainer(ctx, accountID, token, url.Values{
		"media_type": {"CAROUSEL"},
		"children":   {strings.Join(children, ",")},
		"caption":    {caption},
	})
}

func (i *Instagram) createMediaContainer(ctx context.Context, accountID, token string, form url.Values) (string, error) {
	var created struct {
		ID string `json:"id"`
	}
	if err := i.call(ctx, accountID+"/media", token, form, &created); err != nil {
		return "", err
	}

	return created.ID, nil
}

func (i *Instagram) publishContainer(ctx context.Context, accountID, token, creationID string) (string, error) {
	var published struct {
		ID string `json:"id"`
	}
	form := url.Values{"creation_id": {creationID}}
	if err := i.call(ctx, accountID+"/media_publish", token, form, &published); err != nil {
		return "", err
	}

	return published.ID, nil
}

func (i *Instagram) call(ctx context.Context, path, token string, form url.Values, out any) error {
	form.Set("access_token", token)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, i.baseURL+"/"+path, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := i.client.Do(req)
	if err != nil {
		return fmt.Errorf("instagram: request to %s failed: %w", path, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("instagram: graph api answered %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return json.Unmarshal(body, out)
}
