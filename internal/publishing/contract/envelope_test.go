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

// The sender keeps its own copy of this contract: two repositories building into two images
// cannot share a module through a replace directive. This is the same JSON pinned on the
// lap_vision side, so a field renamed there fails here.
func TestRenderedPostGoldenJSON(t *testing.T) {
	const wire = `{"id":"abc123","type":"rendered_post","version":1,"occurred_at":"2026-09-13T12:00:00Z","payload":{"discipline":"moto","post_type":"event_result","championship":"FIM Motocross World Championship","title":"MXGP of China · MXGP","subtitle":"FIM Motocross World Championship, этап 18","lines":["Победа: Jeffrey Herlings"],"table":{"header":["#","Гонщик"],"rows":[["1","Jeffrey Herlings"]]},"tags":["moto"],"image":{"layers":[{"kind":"rider","key":"Jeffrey Herlings"}],"title":"MXGP of China","stats":[{"label":"Jeffrey Herlings","value":"50"}]}}}`

	var envelope Envelope
	if err := json.Unmarshal([]byte(wire), &envelope); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if envelope.Type != TypeRenderedPost || envelope.Version != Version || envelope.ID != "abc123" {
		t.Fatalf("unexpected envelope: %+v", envelope)
	}

	var payload RenderedPostPayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.Title != "MXGP of China · MXGP" || payload.Discipline != "moto" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	// The championship is what a channel narrower than a whole discipline is routed by.
	if payload.Championship != "FIM Motocross World Championship" {
		t.Fatalf("championship = %q", payload.Championship)
	}
	if payload.Table == nil || len(payload.Table.Rows) != 1 || payload.Table.Rows[0][1] != "Jeffrey Herlings" {
		t.Fatalf("unexpected table: %+v", payload.Table)
	}
	if payload.Image == nil || len(payload.Image.Layers) != 1 || payload.Image.Layers[0].Kind != "rider" {
		t.Fatalf("unexpected image: %+v", payload.Image)
	}
	if len(payload.Image.Stats) != 1 || payload.Image.Stats[0].Value != "50" {
		t.Fatalf("unexpected stats: %+v", payload.Image.Stats)
	}
}
