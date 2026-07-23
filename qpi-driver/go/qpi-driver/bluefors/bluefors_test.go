package bluefors

import (
	"net/http"
	"net/http/httptest"
	"testing"

	qpidriver "github.com/sopherapps/qpi/qpi-driver/go"
)

func TestReadChannelParsesLatestValidValue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/values/mapper/bf/tmc" {
			t.Errorf("unexpected path %q (dots should become slashes)", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":{"content":{"latest_valid_value":{"value":0.0123,"status":"OK"}}}}`))
	}))
	defer server.Close()

	d := New(Options{BaseURL: server.URL, Channels: map[string]string{"mapper.bf.tmc": "K"}})
	r := d.readChannel("mapper.bf.tmc", "K")

	if r.Status != "OK" || r.Unit != "K" || r.Value == nil || *r.Value != 0.0123 {
		t.Fatalf("unexpected reading: %+v", r)
	}
}

func TestReadChannelFallsBackToLatestValue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"content":{"latest_value":{"value":"1.5","status":"STALE"}}}}`))
	}))
	defer server.Close()

	d := New(Options{BaseURL: server.URL, Channels: map[string]string{"mapper.bf.pmc": "mbar"}})
	r := d.readChannel("mapper.bf.pmc", "mbar")

	if r.Status != "STALE" || r.Value == nil || *r.Value != 1.5 {
		t.Fatalf("expected fallback to latest_value with string coercion, got %+v", r)
	}
}

func TestReadChannelErrorStatusOnFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	d := New(Options{BaseURL: server.URL, Channels: map[string]string{"x": ""}})
	r := d.readChannel("x", "")

	if r.Status != "ERROR" || r.Value != nil {
		t.Fatalf("expected an ERROR reading with nil value, got %+v", r)
	}
}

func TestParseValueVariants(t *testing.T) {
	if v := parseValue([]byte("null")); v != nil {
		t.Errorf("null should parse to nil, got %v", *v)
	}
	if v := parseValue(nil); v != nil {
		t.Errorf("empty should parse to nil, got %v", *v)
	}
	if v := parseValue([]byte(`"not-a-number"`)); v != nil {
		t.Errorf("non-numeric string should parse to nil, got %v", *v)
	}
	if v := parseValue([]byte("42")); v == nil || *v != 42 {
		t.Errorf("expected 42, got %v", v)
	}
}

func TestParseChannels(t *testing.T) {
	got := ParseChannels("mapper.bf.tmc:K, mapper.bf.pmc:mbar ,bare")
	want := map[string]string{"mapper.bf.tmc": "K", "mapper.bf.pmc": "mbar", "bare": ""}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("channel %q = %q, want %q", k, got[k], v)
		}
	}
}

func TestHandleEventIgnoresInbound(t *testing.T) {
	d := New(Options{Channels: map[string]string{"x": "K"}})
	d.HandleEvent(qpidriver.NewEvent(qpidriver.JobDispatch, "qpu_1", nil)) // must not panic
}
