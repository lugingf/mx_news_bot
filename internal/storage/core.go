package storage

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"

	"mx_news_bot/internal/domain"
	"mx_news_bot/internal/models"
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

// DeliveryChannelsFor lists the enabled channels that accept the given event type.
func (r *Repository) DeliveryChannelsFor(ctx context.Context, eventType string) ([]models.DeliveryChannel, error) {
	var channels []models.DeliveryChannel
	if err := r.db.SelectContext(ctx, &channels, sqlListDeliveryChannels, eventType); err != nil {
		return nil, errors.Wrap(err, "storage: list delivery channels")
	}

	return channels, nil
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
