package qpidriver

import (
	"encoding/json"
	"testing"
)

func TestNewEventFillsIDAndTimestamp(t *testing.T) {
	event := NewEvent(JobResult, "qpu_1", map[string]any{"job_id": "j1"})
	if event.ID == "" {
		t.Error("expected NewEvent to generate an ID")
	}
	if event.Ts == "" {
		t.Error("expected NewEvent to generate a timestamp")
	}
	if event.Driver != "qpu_1" || event.Type != JobResult {
		t.Errorf("unexpected event fields: %+v", event)
	}
}

func TestNewEventNilPayloadBecomesEmptyObject(t *testing.T) {
	raw, err := json.Marshal(NewEvent(CryostatReading, "cryo", nil))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var wire map[string]any
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	payload, ok := wire["payload"].(map[string]any)
	if !ok {
		t.Fatalf("expected payload object, got %T", wire["payload"])
	}
	if len(payload) != 0 {
		t.Errorf("expected empty payload, got %v", payload)
	}
}

func TestEventEnvelopeRoundTrip(t *testing.T) {
	original := Event{
		ID:      "evt_1",
		Driver:  "cryo",
		Type:    CryostatReading,
		Ts:      "2026-01-01T00:00:00.000Z",
		Payload: map[string]any{"readings": map[string]any{"mapper.bf.tmc": 0.02}},
	}
	raw, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var wire map[string]any
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, key := range []string{"id", "driver", "type", "ts", "payload"} {
		if _, ok := wire[key]; !ok {
			t.Errorf("envelope missing key %q", key)
		}
	}
	if wire["type"] != "CryostatReading" {
		t.Errorf("expected type CryostatReading, got %v", wire["type"])
	}

	decoded, err := decodeEvent(raw)
	if err != nil {
		t.Fatalf("decodeEvent: %v", err)
	}
	if decoded.ID != original.ID || decoded.Type != original.Type || decoded.Driver != original.Driver {
		t.Errorf("round trip mismatch: %+v", decoded)
	}
}

func TestDecodeEventMalformed(t *testing.T) {
	if _, err := decodeEvent([]byte("not json")); err == nil {
		t.Error("expected an error decoding malformed input")
	}
}
