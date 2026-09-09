package contract

import (
	"encoding/json"
	"testing"
	"time"
)

// The exact bytes lap_vision must produce. This is the only place the two services agree, and
// they agree by duplication, so a change here is a change to the wire format.
const goldenEnvelope = `{"id":"01J000000000000000000000","type":"race_result","version":1,"occurred_at":"2026-09-06T14:30:00Z","payload":{"class":"MXGP"}}`

func TestEnvelopeGoldenRoundTrip(t *testing.T) {
	var envelope Envelope
	if err := json.Unmarshal([]byte(goldenEnvelope), &envelope); err != nil {
		t.Fatalf("decode golden envelope: %v", err)
	}

	if envelope.ID != "01J000000000000000000000" {
		t.Errorf("ID = %q", envelope.ID)
	}
	if envelope.Type != TypeRaceResult {
		t.Errorf("Type = %q, want %q", envelope.Type, TypeRaceResult)
	}
	if envelope.Version != Version {
		t.Errorf("Version = %d, want %d", envelope.Version, Version)
	}
	if want := time.Date(2026, time.September, 6, 14, 30, 0, 0, time.UTC); !envelope.OccurredAt.Equal(want) {
		t.Errorf("OccurredAt = %v, want %v", envelope.OccurredAt, want)
	}

	// The payload must survive untouched: the dispatcher stores these raw bytes and a builder
	// decodes them later.
	var payload struct {
		Class string `json:"class"`
	}
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.Class != "MXGP" {
		t.Errorf("payload class = %q", payload.Class)
	}

	encoded, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("re-encode: %v", err)
	}
	if string(encoded) != goldenEnvelope {
		t.Errorf("re-encoded envelope drifted:\n got %s\nwant %s", encoded, goldenEnvelope)
	}
}

func TestSignatureRoundTrip(t *testing.T) {
	body := []byte(goldenEnvelope)
	signature := Sign("s3cret", body)

	if !Verify("s3cret", body, signature) {
		t.Error("a signature must verify against the body that produced it")
	}
	if Verify("other", body, signature) {
		t.Error("a different secret must not verify")
	}
	if Verify("s3cret", append(body, ' '), signature) {
		t.Error("a modified body must not verify")
	}
}

// An unconfigured receiver must reject everything. Treating an empty secret as "no verification
// needed" would leave the publication endpoint open to anyone who can reach it.
func TestVerifyRejectsEmptySecret(t *testing.T) {
	body := []byte(`{}`)
	if Verify("", body, Sign("", body)) {
		t.Error("an empty secret must never verify, even against its own signature")
	}
}

func TestSignatureIsPrefixed(t *testing.T) {
	if got := Sign("k", []byte("x")); len(got) != len("sha256=")+64 || got[:7] != "sha256=" {
		t.Errorf("Sign returned %q, want a sha256= prefix and 64 hex characters", got)
	}
}
