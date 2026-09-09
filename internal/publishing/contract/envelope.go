// Package contract is the wire format lap_vision uses to ask for a publication.
//
// It is duplicated rather than imported from lap_vision on purpose: the two services build from
// separate repositories into separate images, so a `replace` directive pointing at a sibling
// checkout would not resolve inside either Dockerfile. The duplication is kept honest by the
// golden test in envelope_test.go, which pins the exact JSON both sides must agree on.
package contract

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"mx_news_bot/internal/models"
)

const (
	HeaderEventID   = "X-LV-Event-Id"
	HeaderSignature = "X-LV-Signature"

	Version = 1
)

// Event types. A channel may subscribe to a subset of these.
const (
	TypeRaceResult    = "race_result"
	TypeStandings     = "standings"
	TypeEventUpcoming = "event_upcoming"
	TypeEventSchedule = "event_schedule"
)

// Envelope wraps every publication request. The payload is complete: the receiver renders from
// it and never calls back for more.
type Envelope struct {
	ID         string     `json:"id"`
	Type       string     `json:"type"`
	Version    int        `json:"version"`
	OccurredAt time.Time  `json:"occurred_at"`
	Payload    RawPayload `json:"payload"`
}

// RawPayload is decoded twice: once to route on Type, once into the concrete payload.
type RawPayload []byte

func (p RawPayload) MarshalJSON() ([]byte, error) {
	if len(p) == 0 {
		return []byte("null"), nil
	}

	return p, nil
}

func (p *RawPayload) UnmarshalJSON(data []byte) error {
	*p = append((*p)[:0], data...)

	return nil
}

type RaceResultPayload struct {
	Championship models.Championship `json:"championship"`
	Event        models.Event        `json:"event"`
	Result       models.RaceResult   `json:"result"`
}

type StandingsPayload struct {
	Championship models.Championship `json:"championship"`
	Class        string              `json:"class"`
	Region       string              `json:"region"`
	AfterEvent   models.Event        `json:"after_event"`
	Standings    []models.Standing   `json:"standings"`
}

type EventUpcomingPayload struct {
	Championship models.Championship `json:"championship"`
	Event        models.Event        `json:"event"`
}

type EventSchedulePayload struct {
	Championship models.Championship `json:"championship"`
	Events       []models.Event      `json:"events"`
}

// Sign returns the value of the signature header for a body.
func Sign(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)

	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// Verify compares a received signature against the body in constant time. An empty secret always
// fails: an unconfigured receiver must reject, not accept everything.
func Verify(secret string, body []byte, signature string) bool {
	if strings.TrimSpace(secret) == "" {
		return false
	}

	return hmac.Equal([]byte(Sign(secret, body)), []byte(strings.TrimSpace(signature)))
}
