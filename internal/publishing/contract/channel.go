package contract

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
