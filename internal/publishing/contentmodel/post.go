// Package contentmodel holds the channel-independent shape of a post.
//
// Without it, N channels and M content types would need N*M formatters. A builder turns a
// payload into a Post once per content type, and a renderer turns a Post into a message once per
// channel, which makes it N+M.
package contentmodel

import "time"

type Post struct {
	Title    string
	Subtitle string
	// Meta is the labelled header block: date, round, track. Order is significant, so it is a
	// slice rather than a map.
	Meta     []Field
	Table    *Table
	Sections []Section
	Media    []Media
	Tags     []string
	Link     string
	// OccurredAt is when the underlying event happened, not when the post was built.
	OccurredAt time.Time
}

type Field struct {
	Label string
	Value string
	Emoji string
}

// Table is the results grid. Align says how each column is padded in a fixed-width rendering;
// a renderer that has no monospace mode may ignore it.
type Table struct {
	Columns []Column
	Rows    [][]string
}

type Column struct {
	Header string
	Width  int
	Align  Align
}

type Align int

const (
	AlignLeft Align = iota
	AlignRight
)

type Section struct {
	Heading string
	Body    string
}

type Media struct {
	URL     string
	Caption string
	Kind    MediaKind
}

type MediaKind string

const (
	MediaImage MediaKind = "image"
	MediaVideo MediaKind = "video"
)

// Message is what a renderer produces and a publisher sends.
type Message struct {
	Text      string
	ParseMode string
	Media     []Media
}

// Capabilities lets a renderer trim to what its channel can actually carry, instead of every
// builder having to know about channel limits.
type Capabilities struct {
	MaxRunes       int
	SupportsTables bool
	SupportsMedia  bool
	SupportsLinks  bool
}
