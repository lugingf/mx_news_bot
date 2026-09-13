package publishing

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"mx_news_bot/config"
	"mx_news_bot/internal/models"
)

// Declarer writes one channel as the configuration describes it.
type Declarer interface {
	DeclareDeliveryChannel(ctx context.Context, channel models.DeliveryChannelRecord) (models.DeliveryChannelRecord, error)
}

// knownChannelTypes are the ones a publisher exists for. A destination nothing can deliver to
// would sit in the table looking configured while every post to it failed.
var knownChannelTypes = map[string]bool{"telegram": true, "twitter": true, "instagram": true}

// DeclareChannels brings the delivery table in line with the configuration.
//
// The config is where a deployment's channels are written down, and this is what makes the table
// agree with it — so nobody has to run an INSERT by hand or remember which one was run where.
// Channels in the table that the config does not mention are left alone: the administration screen
// can still add one for an experiment without a restart wiping it.
//
// A channel that cannot be written stops the startup rather than being skipped: a bot running with
// half its channels would publish to some of them and quietly not to the rest.
func DeclareChannels(ctx context.Context, store Declarer, channels []config.DeliveryChannel, log *slog.Logger) error {
	for _, declared := range channels {
		record, err := channelRecord(declared)
		if err != nil {
			return err
		}

		saved, err := store.DeclareDeliveryChannel(ctx, record)
		if err != nil {
			return fmt.Errorf("declare channel %s %s: %w", record.Channel, record.Target, err)
		}

		log.Info("delivery channel declared",
			"id", saved.ID,
			"channel", saved.Channel,
			"target", saved.Target,
			"title", saved.Title,
			"enabled", saved.Enabled,
			"rehearsal", saved.Rehearsal,
			"disciplines", []string(saved.Disciplines),
			"championships", []string(saved.Championships),
			"post_types", []string(saved.PostTypes),
		)
	}

	return nil
}

func channelRecord(declared config.DeliveryChannel) (models.DeliveryChannelRecord, error) {
	channel := strings.ToLower(strings.TrimSpace(declared.Channel))
	target := strings.TrimSpace(declared.Target)

	if target == "" {
		return models.DeliveryChannelRecord{}, fmt.Errorf("publishing.channels: a channel needs a target")
	}
	if !knownChannelTypes[channel] {
		return models.DeliveryChannelRecord{}, fmt.Errorf("publishing.channels: unknown channel type %q for %s", declared.Channel, target)
	}

	return models.DeliveryChannelRecord{
		Channel:   channel,
		Target:    target,
		Title:     strings.TrimSpace(declared.Title),
		Enabled:   declared.Enabled,
		Rehearsal: declared.Rehearsal,
		// A discipline and a post type are matched against values this service produces itself, so
		// they are folded to lower case. A championship is matched against what the sender writes
		// in the payload, and "AMA Supercross" is how that arrives.
		Disciplines:   cleanFilter(declared.Disciplines, true),
		Championships: cleanFilter(declared.Championships, false),
		PostTypes:     cleanFilter(declared.PostTypes, true),
	}, nil
}

func cleanFilter(values []string, lower bool) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if lower {
			trimmed = strings.ToLower(trimmed)
		}
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}

	return out
}
