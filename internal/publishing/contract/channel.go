package contract

import "time"

// ChannelsPath is where the bot serves the delivery-channel administration. lap_vision owns the
// screen; the rows live here, next to the tokens that can actually reach them.
const ChannelsPath = "/internal/channels"

// ChannelSpec is one delivery channel and the posts it accepts.
//
// Target is whatever the channel calls its destination: for Telegram either a public @name or a
// numeric chat id such as -1004362440814, which is the only form a private channel has.
//
// The three lists are filters, and an empty list means "everything". They are combined with AND:
// a channel with Championships ["AMA Supercross"] and PostTypes ["event_result"] takes the results
// of Supercross rounds and nothing else. That is what separates a moto channel from an F1 one, a
// Supercross channel from a MotoGP one, and a rehearsal channel — which filters nothing and so
// sees every post — from both.
type ChannelSpec struct {
	ID      int64  `json:"id,omitempty"`
	Channel string `json:"channel"`
	Target  string `json:"target"`
	Title   string `json:"title"`
	Enabled bool   `json:"enabled"`

	// Rehearsal channels take rehearsal posts and nothing else, and a live channel never takes one.
	// That is what lets a new kind of post be tried out while the live channels keep working.
	Rehearsal bool `json:"rehearsal"`

	Disciplines   []string `json:"disciplines"`
	Championships []string `json:"championships"`
	PostTypes     []string `json:"post_types"`
}

type ChannelList struct {
	Items []ChannelSpec `json:"items"`
}

// Match is what one publication looks like to the channel table: the three things a channel may
// filter on, taken from the payload. Anything a payload does not carry stays empty and then only
// matches a channel that filters on nothing.
type Match struct {
	EventType    string
	Discipline   string
	Championship string
	PostType     string
	Rehearsal    bool
}

// DeliveriesPath is where the bot reports what became of publications: for each event, one line
// per channel it was sent to. The ids to ask about come in the query, comma separated.
const DeliveriesPath = "/internal/publications/deliveries"

// DeliveryReport is what became of one publication in one channel. Channel is the channel's title;
// the chat id is not reported as a field, though the text of an error may quote it the way the
// Telegram client wrote it.
type DeliveryReport struct {
	EventID     string     `json:"event_id"`
	ChannelID   int64      `json:"channel_id"`
	Channel     string     `json:"channel"`
	Rehearsal   bool       `json:"rehearsal"`
	Status      string     `json:"status"`
	Attempts    int        `json:"attempts"`
	Error       string     `json:"error,omitempty"`
	DeliveredAt *time.Time `json:"delivered_at,omitempty"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type DeliveryReportList struct {
	Items []DeliveryReport `json:"items"`
}
