package storage

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/pkg/errors"

	"mx_news_bot/internal/domain"
	"mx_news_bot/internal/models"
	"mx_news_bot/internal/publishing/contract"
)

// Repository holds the bot's own state only: its users, their preferences, the channels it
// delivers to and what it has already published. Racing data lives in lap_vision.
type Repository struct {
	db  *sqlx.DB
	log *slog.Logger
}

func New(db *sqlx.DB, log *slog.Logger) *Repository {
	return &Repository{db: db, log: log}
}

func (r *Repository) EnsureUser(ctx context.Context, user models.User) error {
	if _, err := r.db.ExecContext(ctx, sqlUpsertUser, user.TGUserID, user.Name, user.IsPremium); err != nil {
		return errors.Wrap(err, "storage: upsert user")
	}

	return nil
}

func (r *Repository) UserPreference(ctx context.Context, tgUserID int64) (models.UserPreference, error) {
	var pref models.UserPreference
	err := r.db.GetContext(ctx, &pref, sqlSelectUserPreference, tgUserID)
	if errors.Is(err, sql.ErrNoRows) {
		return models.UserPreference{TGUserID: tgUserID, NotificationsEnabled: true}, nil
	}
	if err != nil {
		return models.UserPreference{}, errors.Wrap(err, "storage: select user preference")
	}

	return pref, nil
}

func (r *Repository) UpdateUserPreference(ctx context.Context, update domain.UserPreferenceUpdate) error {
	if update.DefaultChampionshipID == nil && update.NotificationsEnabled == nil {
		return nil
	}

	_, err := r.db.ExecContext(ctx, sqlUpsertUserPreference,
		update.TGUserID, update.DefaultChampionshipID, update.NotificationsEnabled)
	if err != nil {
		return errors.Wrap(err, "storage: upsert user preference")
	}

	return nil
}

// DeliveryChannelsFor lists the enabled channels a publication matches. A dimension the payload
// says nothing about reaches only the channels that do not filter on it.
func (r *Repository) DeliveryChannelsFor(ctx context.Context, match contract.Match) ([]models.DeliveryChannel, error) {
	var channels []models.DeliveryChannel
	err := r.db.SelectContext(ctx, &channels, sqlListDeliveryChannels,
		match.EventType, match.Discipline, match.Championship, match.PostType, match.Rehearsal)
	if err != nil {
		return nil, errors.Wrap(err, "storage: list delivery channels")
	}

	return channels, nil
}

// ListDeliveryChannels returns every registered channel, enabled or not.
func (r *Repository) ListDeliveryChannels(ctx context.Context) ([]models.DeliveryChannelRecord, error) {
	var channels []models.DeliveryChannelRecord
	if err := r.db.SelectContext(ctx, &channels, sqlListAllDeliveryChannels); err != nil {
		return nil, errors.Wrap(err, "storage: list all delivery channels")
	}

	return channels, nil
}

func (r *Repository) CreateDeliveryChannel(ctx context.Context, channel models.DeliveryChannelRecord) (models.DeliveryChannelRecord, error) {
	var created models.DeliveryChannelRecord
	err := r.db.GetContext(ctx, &created, sqlInsertDeliveryChannel,
		channel.Channel, channel.Target, channel.Title, channel.Enabled, channel.Rehearsal,
		pq.Array(channel.Disciplines), pq.Array(channel.Championships), pq.Array(channel.PostTypes))
	if err != nil {
		return models.DeliveryChannelRecord{}, errors.Wrap(err, "storage: create delivery channel")
	}

	return created, nil
}

// UpdateDeliveryChannel rewrites everything but the channel type: moving a row from Telegram to
// Twitter would keep the delivery history of a post that never went there.
func (r *Repository) UpdateDeliveryChannel(ctx context.Context, channel models.DeliveryChannelRecord) (models.DeliveryChannelRecord, error) {
	var updated models.DeliveryChannelRecord
	err := r.db.GetContext(ctx, &updated, sqlUpdateDeliveryChannel,
		channel.ID, channel.Target, channel.Title, channel.Enabled, channel.Rehearsal,
		pq.Array(channel.Disciplines), pq.Array(channel.Championships), pq.Array(channel.PostTypes))
	if errors.Is(err, sql.ErrNoRows) {
		return models.DeliveryChannelRecord{}, domain.ErrChannelNotFound
	}
	if err != nil {
		return models.DeliveryChannelRecord{}, errors.Wrap(err, "storage: update delivery channel")
	}

	return updated, nil
}

func (r *Repository) DeleteDeliveryChannel(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, sqlDeleteDeliveryChannel, id)
	if err != nil {
		return errors.Wrap(err, "storage: delete delivery channel")
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "storage: delete delivery channel rows")
	}
	if affected == 0 {
		return domain.ErrChannelNotFound
	}

	return nil
}

// ClaimPublication records an accepted publication. It reports false when the event id was
// already stored, which is how a repeated webhook is recognised.
func (r *Repository) ClaimPublication(ctx context.Context, eventID, eventType string, payload []byte) (bool, error) {
	result, err := r.db.ExecContext(ctx, sqlClaimPublication, eventID, eventType, payload)
	if err != nil {
		return false, errors.Wrap(err, "storage: claim publication")
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return false, errors.Wrap(err, "storage: claim publication rows")
	}

	return affected > 0, nil
}

func (r *Repository) DeliveredChannelIDs(ctx context.Context, eventID string) (map[int64]struct{}, error) {
	var ids []int64
	if err := r.db.SelectContext(ctx, &ids, sqlDeliveredChannels, eventID); err != nil {
		return nil, errors.Wrap(err, "storage: list delivered channels")
	}

	delivered := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		delivered[id] = struct{}{}
	}

	return delivered, nil
}

func (r *Repository) MarkDelivery(ctx context.Context, eventID string, channelID int64, status, lastError, externalRef string) error {
	var deliveredAt *time.Time
	if status == "delivered" {
		now := time.Now().UTC()
		deliveredAt = &now
	}

	_, err := r.db.ExecContext(ctx, sqlMarkDelivery,
		eventID, channelID, status, lastError, externalRef, deliveredAt)
	if err != nil {
		return errors.Wrap(err, "storage: mark delivery")
	}

	return nil
}
