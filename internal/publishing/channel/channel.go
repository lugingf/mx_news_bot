package channel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	tele "gopkg.in/telebot.v3"

	"mx_news_bot/internal/models"
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
	SendAlbum(to tele.Recipient, a tele.Album, opts ...any) ([]tele.Message, error)
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

	// A gallery — several pictures filed to be sent together, a circuit from more than one angle
	// say — goes out as one album instead of picking a single one of them to stand for the rest.
	if gallery := filedMedia(message.Media); len(gallery) > 1 {
		return t.publishAlbum(target, gallery, message.Text, opts)
	}

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

// primaryMedia is the one attachment a single-photo send gets — the first item in the list that
// actually carries a URL.
func primaryMedia(media []contentmodel.Media) *contentmodel.Media {
	for i := range media {
		if strings.TrimSpace(media[i].URL) != "" {
			return &media[i]
		}
	}

	return nil
}

// filedMedia is every item that actually carries a URL, in order. A caller decides from the count
// whether that is one attachment or an album.
func filedMedia(media []contentmodel.Media) []contentmodel.Media {
	out := make([]contentmodel.Media, 0, len(media))
	for _, item := range media {
		if strings.TrimSpace(item.URL) != "" {
			out = append(out, item)
		}
	}

	return out
}

// publishAlbum sends several pictures as one Telegram message. The caption goes on the first
// item — telebot's own Album.SetCaption does this — following the same overflow rule a single
// photo does: text too long for a caption is sent as its own message afterward instead of being
// silently cut down to fit.
func (t *Telegram) publishAlbum(target string, media []contentmodel.Media, text string, opts *tele.SendOptions) (Receipt, error) {
	caption := text
	textFollowsSeparately := len([]rune(caption)) > telegramCaptionLimit
	if textFollowsSeparately {
		caption = ""
	}

	album := make(tele.Album, 0, len(media))
	for _, item := range media {
		if item.Kind == contentmodel.MediaVideo {
			album = append(album, &tele.Video{File: tele.FromURL(item.URL)})
			continue
		}
		album = append(album, &tele.Photo{File: tele.FromURL(item.URL)})
	}
	album.SetCaption(caption)

	sent, err := t.sender.SendAlbum(recipient(target), album, opts)
	if err != nil {
		return Receipt{}, fmt.Errorf("telegram: send album to %s: %w", target, err)
	}

	if textFollowsSeparately {
		if _, err := t.sender.Send(recipient(target), text, opts); err != nil {
			return Receipt{}, fmt.Errorf("telegram: send follow-up text to %s: %w", target, err)
		}
	}

	ref := ""
	if len(sent) > 0 {
		ref = fmt.Sprintf("%d", sent[0].ID)
	}

	return Receipt{Channel: t.Name(), Ref: ref}, nil
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

type InstagramTokenSource interface {
	Token(ctx context.Context, accountID string) (string, bool)
}

// InstagramTokenCache is the in-process view of the DB-backed tokens. Publishing reads from it on
// every post, and the refresher writes new tokens into it after they are persisted.
type InstagramTokenCache struct {
	mu     sync.RWMutex
	tokens map[string]string
}

func NewInstagramTokenCache(accounts []InstagramAccount) *InstagramTokenCache {
	cache := &InstagramTokenCache{tokens: make(map[string]string, len(accounts))}
	for _, account := range accounts {
		cache.Set(account.AccountID, account.AccessToken)
	}

	return cache
}

func (c *InstagramTokenCache) Set(accountID string, accessToken string) {
	accountID = strings.TrimSpace(accountID)
	accessToken = strings.TrimSpace(accessToken)
	if c == nil || accountID == "" || accessToken == "" {
		return
	}

	c.mu.Lock()
	c.tokens[accountID] = accessToken
	c.mu.Unlock()
}

func (c *InstagramTokenCache) Token(_ context.Context, accountID string) (string, bool) {
	if c == nil {
		return "", false
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	token, ok := c.tokens[strings.TrimSpace(accountID)]
	return token, ok
}

func (c *InstagramTokenCache) Empty() bool {
	if c == nil {
		return true
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.tokens) == 0
}

// maxCarouselItems is Instagram's own limit on how many images one carousel post may carry.
const maxCarouselItems = 10

// Instagram posts through the Graph API: a media container is created from a public image URL,
// then published. Several images become a carousel — child containers first, then a parent that
// lists them.
type Instagram struct {
	tokens  InstagramTokenSource
	client  InstagramHTTPClient
	baseURL string
}

func NewInstagram(enabled bool, accounts []InstagramAccount, client InstagramHTTPClient) *Instagram {
	return NewInstagramWithTokenSource(enabled, NewInstagramTokenCache(accounts), client)
}

func NewInstagramWithTokenSource(enabled bool, tokens InstagramTokenSource, client InstagramHTTPClient) *Instagram {
	if !enabled {
		return &Instagram{}
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
	if i.tokens == nil {
		return Receipt{}, ErrNotConfigured
	}
	token, ok := i.tokens.Token(ctx, strings.TrimSpace(target))
	if !ok {
		if cache, cacheOK := i.tokens.(*InstagramTokenCache); cacheOK && cache.Empty() {
			return Receipt{}, ErrNotConfigured
		}
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

type InstagramTokenStore interface {
	ListInstagramTokens(ctx context.Context) ([]models.InstagramToken, error)
	UpdateInstagramToken(ctx context.Context, token models.InstagramToken) error
}

type InstagramTokenRefresher struct {
	store   InstagramTokenStore
	cache   *InstagramTokenCache
	client  InstagramHTTPClient
	baseURL string
	now     func() time.Time
	log     *slog.Logger

	interval time.Duration
	lead     time.Duration
}

const (
	defaultInstagramRefreshInterval = 24 * time.Hour
	defaultInstagramRefreshLead     = 14 * 24 * time.Hour
)

func NewInstagramTokenRefresher(store InstagramTokenStore, cache *InstagramTokenCache, client InstagramHTTPClient, log *slog.Logger) *InstagramTokenRefresher {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	return &InstagramTokenRefresher{
		store:    store,
		cache:    cache,
		client:   client,
		baseURL:  "https://graph.instagram.com",
		now:      time.Now,
		log:      log,
		interval: defaultInstagramRefreshInterval,
		lead:     defaultInstagramRefreshLead,
	}
}

func (r *InstagramTokenRefresher) Run(ctx context.Context) {
	if r == nil || r.store == nil || r.cache == nil {
		return
	}

	r.RefreshDue(ctx)

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.RefreshDue(ctx)
		}
	}
}

func (r *InstagramTokenRefresher) RefreshDue(ctx context.Context) {
	tokens, err := r.store.ListInstagramTokens(ctx)
	if err != nil {
		r.logError("instagram token list failed", err, "")
		return
	}

	for _, token := range tokens {
		if !r.due(token) {
			r.cache.Set(token.AccountID, token.AccessToken)
			continue
		}

		refreshed, err := r.refresh(ctx, token.AccessToken)
		if err != nil {
			r.logError("instagram token refresh failed", err, token.AccountID)
			continue
		}

		now := r.now().UTC()
		expiresAt := now.Add(time.Duration(refreshed.ExpiresIn) * time.Second)
		next := models.InstagramToken{
			AccountID:   token.AccountID,
			AccessToken: refreshed.AccessToken,
			ExpiresAt:   &expiresAt,
			RefreshedAt: &now,
		}
		if err := r.store.UpdateInstagramToken(ctx, next); err != nil {
			r.logError("instagram token save failed", err, token.AccountID)
			continue
		}

		r.cache.Set(next.AccountID, next.AccessToken)
		if r.log != nil {
			r.log.Info("instagram token refreshed", "account_id", next.AccountID, "expires_at", expiresAt)
		}
	}
}

func (r *InstagramTokenRefresher) due(token models.InstagramToken) bool {
	if token.AccountID == "" || token.AccessToken == "" {
		return false
	}
	if token.ExpiresAt == nil {
		return true
	}

	return token.ExpiresAt.Sub(r.now().UTC()) <= r.lead
}

type instagramRefreshResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

func (r *InstagramTokenRefresher) refresh(ctx context.Context, accessToken string) (instagramRefreshResponse, error) {
	endpoint := strings.TrimRight(r.baseURL, "/") + "/refresh_access_token"
	reqURL, err := url.Parse(endpoint)
	if err != nil {
		return instagramRefreshResponse{}, err
	}
	query := reqURL.Query()
	query.Set("grant_type", "ig_refresh_token")
	query.Set("access_token", accessToken)
	reqURL.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return instagramRefreshResponse{}, err
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return instagramRefreshResponse{}, fmt.Errorf("instagram token refresh request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return instagramRefreshResponse{}, err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return instagramRefreshResponse{}, fmt.Errorf("instagram token refresh answered %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var refreshed instagramRefreshResponse
	if err := json.Unmarshal(body, &refreshed); err != nil {
		return instagramRefreshResponse{}, err
	}
	if strings.TrimSpace(refreshed.AccessToken) == "" || refreshed.ExpiresIn <= 0 {
		return instagramRefreshResponse{}, errors.New("instagram token refresh returned no usable token")
	}

	return refreshed, nil
}

func (r *InstagramTokenRefresher) logError(message string, err error, accountID string) {
	if r.log == nil {
		return
	}
	if accountID == "" {
		r.log.Error(message, "err", err)
		return
	}
	r.log.Error(message, "account_id", accountID, "err", err)
}
