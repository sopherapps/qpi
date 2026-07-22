package api

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/spf13/cobra"
	"go.nanomsg.org/mangos/v3"
	"go.nanomsg.org/mangos/v3/protocol/pull"
	"go.nanomsg.org/mangos/v3/protocol/push"
	_ "go.nanomsg.org/mangos/v3/transport/tlstcp"

	"qpi/internal/config"
)

func TestEventRegistryDispatch(t *testing.T) {
	registry := NewEventRegistry()

	var handled *Event
	registry.Register(EventJobResult, func(ctx context.Context, app core.App, qpuID string, event *Event) error {
		handled = event
		return nil
	})

	if !registry.Handles(EventJobResult) {
		t.Errorf("expected registry to handle %q", EventJobResult)
	}
	if registry.Handles(EventJobDispatch) {
		t.Errorf("expected registry not to handle unregistered %q", EventJobDispatch)
	}

	event := &Event{ID: "evt_1", Type: EventJobResult}
	if err := registry.Dispatch(context.Background(), nil, "qpu_1", event); err != nil {
		t.Fatalf("dispatch returned error: %v", err)
	}
	if handled == nil || handled.ID != "evt_1" {
		t.Errorf("expected handler to receive event evt_1, got %v", handled)
	}
}

func TestEventRegistryDropsUnknownType(t *testing.T) {
	registry := NewEventRegistry()

	event := &Event{ID: "evt_1", Type: EventType("Unknown")}
	if err := registry.Dispatch(context.Background(), nil, "qpu_1", event); err != nil {
		t.Errorf("expected unknown event type to be dropped without error, got %v", err)
	}
}

func TestEventRegistryReturnsHandlerError(t *testing.T) {
	registry := NewEventRegistry()

	rejected := errors.New("invalid payload")
	registry.Register(EventJobResult, func(ctx context.Context, app core.App, qpuID string, event *Event) error {
		return rejected
	})

	event := &Event{ID: "evt_1", Type: EventJobResult}
	if err := registry.Dispatch(context.Background(), nil, "qpu_1", event); !errors.Is(err, rejected) {
		t.Errorf("expected handler error to be returned, got %v", err)
	}
}

func TestNewEventAssignsDefaults(t *testing.T) {
	event, err := NewEvent("drv_test", EventJobDispatch, map[string]any{"job_id": "job_1"})
	if err != nil {
		t.Fatalf("NewEvent returned error: %v", err)
	}

	if !strings.HasPrefix(event.ID, "evt_") {
		t.Errorf("expected id with evt_ prefix, got %q", event.ID)
	}
	if event.Ts == "" {
		t.Errorf("expected a timestamp to be assigned")
	}
	if event.Driver != "drv_test" || event.Type != EventJobDispatch {
		t.Errorf("unexpected envelope header: %+v", event)
	}

	raw, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var back Event
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if back.ID != event.ID || back.Type != event.Type || back.Driver != event.Driver || back.Ts != event.Ts {
		t.Errorf("envelope did not round-trip through JSON: %+v vs %+v", back, event)
	}

	var payload map[string]any
	if err := json.Unmarshal(back.Payload, &payload); err != nil {
		t.Fatalf("payload unmarshal error: %v", err)
	}
	if payload["job_id"] != "job_1" {
		t.Errorf("expected payload job_id=job_1, got %v", payload["job_id"])
	}
}

// TestEventRoundTripOverNNG is the RFC 0001 Phase 0 spike: it proves one
// UI→driver event (JobDispatch) and one driver→UI event (JobResult) round-trip
// over the existing TLS-secured NNG PUSH/PULL channel, decoded from the shared
// envelope and routed through the registry on each side.
func TestEventRoundTripOverNNG(t *testing.T) {
	cfg, caCertPath := newTestConfig(t)
	serverTLS := cfg.GetTlsConfig()
	clientTLS := newClientTLSConfig(t, caCertPath)

	dispatchPort := freePort(t)
	resultPort := freePort(t)

	// QPI-UI side: PUSH for dispatch, PULL for results (mirrors nng.go).
	uiPush, err := push.NewSocket()
	if err != nil {
		t.Fatalf("ui push socket: %v", err)
	}
	defer uiPush.Close()
	dispatchListener, err := getListener(uiPush, dispatchPort, serverTLS)
	if err != nil {
		t.Fatalf("dispatch listener: %v", err)
	}
	if err := dispatchListener.Listen(); err != nil {
		t.Fatalf("dispatch listen: %v", err)
	}
	if err := uiPush.SetOption(mangos.OptionSendDeadline, 5*time.Second); err != nil {
		t.Fatalf("ui push deadline: %v", err)
	}

	uiPull, err := pull.NewSocket()
	if err != nil {
		t.Fatalf("ui pull socket: %v", err)
	}
	defer uiPull.Close()
	resultListener, err := getListener(uiPull, resultPort, serverTLS)
	if err != nil {
		t.Fatalf("result listener: %v", err)
	}
	if err := resultListener.Listen(); err != nil {
		t.Fatalf("result listen: %v", err)
	}
	if err := uiPull.SetOption(mangos.OptionRecvDeadline, 5*time.Second); err != nil {
		t.Fatalf("ui pull deadline: %v", err)
	}

	// Driver side: PULL for dispatch, PUSH for results (mirrors driver.py).
	drvPull, err := pull.NewSocket()
	if err != nil {
		t.Fatalf("driver pull socket: %v", err)
	}
	defer drvPull.Close()
	if err := drvPull.SetOption(mangos.OptionRecvDeadline, 5*time.Second); err != nil {
		t.Fatalf("driver pull deadline: %v", err)
	}
	if err := drvPull.DialOptions(dialAddr(dispatchPort), tlsDialOptions(clientTLS)); err != nil {
		t.Fatalf("driver pull dial: %v", err)
	}

	drvPush, err := push.NewSocket()
	if err != nil {
		t.Fatalf("driver push socket: %v", err)
	}
	defer drvPush.Close()
	if err := drvPush.SetOption(mangos.OptionSendDeadline, 5*time.Second); err != nil {
		t.Fatalf("driver push deadline: %v", err)
	}
	if err := drvPush.DialOptions(dialAddr(resultPort), tlsDialOptions(clientTLS)); err != nil {
		t.Fatalf("driver push dial: %v", err)
	}

	// Give the TLS pipes time to attach before pushing.
	time.Sleep(300 * time.Millisecond)

	// 1. UI → driver: dispatch a job.
	dispatch, err := NewEvent("drv_test", EventJobDispatch, map[string]any{"job_id": "job_1"})
	if err != nil {
		t.Fatalf("build dispatch: %v", err)
	}
	dispatchRaw, err := json.Marshal(dispatch)
	if err != nil {
		t.Fatalf("marshal dispatch: %v", err)
	}
	if err := uiPush.Send(dispatchRaw); err != nil {
		t.Fatalf("ui send dispatch: %v", err)
	}

	// 2. Driver receives the dispatch, routes it through its registry, then emits a result.
	inboundRaw, err := drvPull.Recv()
	if err != nil {
		t.Fatalf("driver recv dispatch: %v", err)
	}
	var inbound Event
	if err := json.Unmarshal(inboundRaw, &inbound); err != nil {
		t.Fatalf("driver unmarshal dispatch: %v", err)
	}

	dispatched := make(chan string, 1)
	driverRegistry := NewEventRegistry()
	driverRegistry.Register(EventJobDispatch, func(ctx context.Context, app core.App, qpuID string, event *Event) error {
		var payload struct {
			JobID string `json:"job_id"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return err
		}
		dispatched <- payload.JobID
		return nil
	})
	if err := driverRegistry.Dispatch(context.Background(), nil, "qpu_test", &inbound); err != nil {
		t.Fatalf("driver dispatch handler: %v", err)
	}

	result, err := NewEvent("drv_test", EventJobResult, map[string]any{"job_id": "job_1", "status": "completed"})
	if err != nil {
		t.Fatalf("build result: %v", err)
	}
	resultRaw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	if err := drvPush.Send(resultRaw); err != nil {
		t.Fatalf("driver send result: %v", err)
	}

	// 3. UI receives the result and routes it through its registry.
	outboundRaw, err := uiPull.Recv()
	if err != nil {
		t.Fatalf("ui recv result: %v", err)
	}
	var outbound Event
	if err := json.Unmarshal(outboundRaw, &outbound); err != nil {
		t.Fatalf("ui unmarshal result: %v", err)
	}

	received := make(chan *Event, 1)
	uiRegistry := NewEventRegistry()
	uiRegistry.Register(EventJobResult, func(ctx context.Context, app core.App, qpuID string, event *Event) error {
		received <- event
		return nil
	})
	if err := uiRegistry.Dispatch(context.Background(), nil, "qpu_test", &outbound); err != nil {
		t.Fatalf("ui result handler: %v", err)
	}

	select {
	case jobID := <-dispatched:
		if jobID != "job_1" {
			t.Errorf("driver handled job_id %q, want job_1", jobID)
		}
	default:
		t.Fatal("driver did not handle the dispatch event")
	}

	select {
	case event := <-received:
		if event.Type != EventJobResult {
			t.Errorf("ui received type %q, want %q", event.Type, EventJobResult)
		}
		if event.ID != result.ID {
			t.Errorf("ui received id %q, want %q", event.ID, result.ID)
		}
	default:
		t.Fatal("ui did not receive the result event")
	}
}

// newTestConfig builds an AppConfig backed by freshly generated TLS material in a
// temp dir, returning the config and the path to its CA certificate.
func newTestConfig(t *testing.T) (*config.AppConfig, string) {
	t.Helper()

	tmpDir := t.TempDir()
	emptyConfig := filepath.Join(tmpDir, "empty.yaml")
	if err := os.WriteFile(emptyConfig, []byte{}, 0644); err != nil {
		t.Fatalf("write empty config: %v", err)
	}

	caCertPath := filepath.Join(tmpDir, "t.ca.pem")
	t.Setenv("QPI_TLS_CERT_FILE", filepath.Join(tmpDir, "t.cert.pem"))
	t.Setenv("QPI_TLS_KEY_FILE", filepath.Join(tmpDir, "t.key"))
	t.Setenv("QPI_TLS_CA_CERT_FILE", caCertPath)
	t.Setenv("QPI_TLS_CA_KEY_FILE", filepath.Join(tmpDir, "t.ca.key"))
	t.Setenv("QPI_IP_ADDR", "127.0.0.1")
	t.Setenv("QPI_CONFIG_FILE", emptyConfig)

	cmd := &cobra.Command{}
	config.BindFlags(cmd)
	cfg, err := config.NewFromFlags(cmd)
	if err != nil {
		t.Fatalf("build config: %v", err)
	}
	return cfg, caCertPath
}

// newClientTLSConfig builds a driver-side TLS config that pins the server's CA,
// mirroring how driver.py trusts the downloaded root CA.
func newClientTLSConfig(t *testing.T, caCertPath string) *tls.Config {
	t.Helper()

	caPEM, err := os.ReadFile(caCertPath)
	if err != nil {
		t.Fatalf("read CA cert: %v", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		t.Fatalf("failed to add CA cert to pool")
	}
	return &tls.Config{
		RootCAs:    pool,
		ServerName: "127.0.0.1",
		MinVersion: tls.VersionTLS12,
	}
}

// freePort asks the OS for an available TCP port on the loopback interface.
func freePort(t *testing.T) int {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("allocate free port: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func dialAddr(port int) string {
	return "tls+tcp://127.0.0.1:" + strconv.Itoa(port)
}

func tlsDialOptions(tlsConfig *tls.Config) map[string]any {
	return map[string]any{mangos.OptionTLSConfig: tlsConfig}
}
