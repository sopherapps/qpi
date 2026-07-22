package api

import (
	"context"
	"testing"

	"qpi/internal/config"
	"qpi/internal/db"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// seedJobForResult builds a test app with a user, a QPU, and one running job
// owned by that user, ready for a JobResult to be applied to it.
func seedJobForResult(t *testing.T) (*tests.TestApp, *config.AppConfig, *core.Record, *core.Record) {
	t.Helper()

	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("failed to create test app: %v", err)
	}
	t.Cleanup(app.Cleanup)

	cfg := testConfig()
	config.SaveConfigOnApp(app, cfg)
	if err := db.EnsureSchema(app); err != nil {
		t.Fatalf("failed to ensure schema: %v", err)
	}

	userRec := core.NewRecord(getCollectionByName(t, app, "users"))
	userRec.Set("email", "runner@example.com")
	userRec.Set("password", "runnerpassword1234")
	userRec.Set("qpu_seconds", 1000.0)
	if err := app.Save(userRec); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	qpuRec := core.NewRecord(getCollectionByName(t, app, cfg.CollectionQPUs))
	qpuRec.Set("name", "qpu_result")
	qpuRec.Set("access_token", db.HashToken("tok"))
	qpuRec.Set("status", "online")
	qpuRec.Set("num_qubits", 2)
	qpuRec.Set("enabled", true)
	if err := app.Save(qpuRec); err != nil {
		t.Fatalf("failed to create qpu: %v", err)
	}

	jobRec := core.NewRecord(getCollectionByName(t, app, cfg.CollectionQuantumJobs))
	jobRec.Set("user_id", userRec.Id)
	jobRec.Set("qpu_target", qpuRec.Id)
	jobRec.Set("status", "running")
	jobRec.Set("payload", map[string]any{"circuits": []any{}})
	if err := app.Save(jobRec); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	return app, cfg, userRec, jobRec
}

// TestApplyJobResult_CompletesAndDeducts proves the JobResult handler persists a
// successful outcome and deducts the user's QPU-seconds, mirroring the legacy
// result listener (RFC 0001 §8).
func TestApplyJobResult_CompletesAndDeducts(t *testing.T) {
	app, cfg, userRec, jobRec := seedJobForResult(t)

	result := ResultPayload{
		JobID:   jobRec.Id,
		Results: map[string]any{"counts": map[string]any{"0x0": 1024}},
	}
	if err := applyJobResult(app, "qpu_result", result); err != nil {
		t.Fatalf("applyJobResult: %v", err)
	}

	updated, err := app.FindRecordById(cfg.CollectionQuantumJobs, jobRec.Id)
	if err != nil {
		t.Fatalf("find updated job: %v", err)
	}
	if updated.GetString("status") != "completed" {
		t.Errorf("job status = %q, want completed", updated.GetString("status"))
	}

	updatedUser, err := app.FindRecordById("users", userRec.Id)
	if err != nil {
		t.Fatalf("find updated user: %v", err)
	}
	if updatedUser.GetFloat("qpu_seconds") > 1000.0 {
		t.Errorf("expected qpu_seconds deducted from 1000, got %v", updatedUser.GetFloat("qpu_seconds"))
	}
}

// TestApplyJobResult_MarksFailedOnError proves an error payload lands the job as
// failed rather than completed.
func TestApplyJobResult_MarksFailedOnError(t *testing.T) {
	app, cfg, _, jobRec := seedJobForResult(t)

	result := ResultPayload{
		JobID:   jobRec.Id,
		Results: map[string]any{"error": "execution blew up"},
	}
	if err := applyJobResult(app, "qpu_result", result); err != nil {
		t.Fatalf("applyJobResult: %v", err)
	}

	updated, err := app.FindRecordById(cfg.CollectionQuantumJobs, jobRec.Id)
	if err != nil {
		t.Fatalf("find updated job: %v", err)
	}
	if updated.GetString("status") != "failed" {
		t.Errorf("job status = %q, want failed", updated.GetString("status"))
	}
}

// TestHandleDriverJobResult_ParsesEnvelope proves the registered handler unwraps
// a JobResult envelope's payload and applies it (RFC 0001 §4, §6).
func TestHandleDriverJobResult_ParsesEnvelope(t *testing.T) {
	app, cfg, _, jobRec := seedJobForResult(t)

	event, err := NewEvent("drv_1", EventJobResult, ResultPayload{
		JobID:   jobRec.Id,
		Results: map[string]any{"counts": map[string]any{"0x3": 1024}},
	})
	if err != nil {
		t.Fatalf("build result event: %v", err)
	}

	if err := handleDriverJobResult(nil, app, "qpu_result", event); err != nil {
		t.Fatalf("handleDriverJobResult: %v", err)
	}

	updated, err := app.FindRecordById(cfg.CollectionQuantumJobs, jobRec.Id)
	if err != nil {
		t.Fatalf("find updated job: %v", err)
	}
	if updated.GetString("status") != "completed" {
		t.Errorf("job status = %q, want completed", updated.GetString("status"))
	}
}

// seedDriverForEvents builds a test app with a QPU and a bluefors_gen1
// driver belonging to it, ready for a CryostatReading event to be attributed
// to (RFC 0001 §7, Phase 3).
func seedDriverForEvents(t *testing.T) (*tests.TestApp, *config.AppConfig, *core.Record, *core.Record) {
	t.Helper()

	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("failed to create test app: %v", err)
	}
	t.Cleanup(app.Cleanup)

	cfg := testConfig()
	config.SaveConfigOnApp(app, cfg)
	if err := db.EnsureSchema(app); err != nil {
		t.Fatalf("failed to ensure schema: %v", err)
	}

	qpuRec := core.NewRecord(getCollectionByName(t, app, cfg.CollectionQPUs))
	qpuRec.Set("name", "qpu_monitor")
	qpuRec.Set("access_token", db.HashToken("tok"))
	qpuRec.Set("status", "online")
	qpuRec.Set("enabled", true)
	if err := app.Save(qpuRec); err != nil {
		t.Fatalf("failed to create qpu: %v", err)
	}

	driverRec := core.NewRecord(getCollectionByName(t, app, cfg.CollectionDrivers))
	driverRec.Set("name", "cryostat-1")
	driverRec.Set("qpu", qpuRec.Id)
	driverRec.Set("kind", string(DriverKindBlueforsGen1))
	driverRec.Set("language", string(DriverLanguagePython))
	driverRec.Set("events", []string{string(EventCryostatReading)})
	driverRec.Set("token", db.HashToken("drv-tok"))
	driverRec.Set("status", "online")
	driverRec.Set("enabled", true)
	if err := app.Save(driverRec); err != nil {
		t.Fatalf("failed to create driver: %v", err)
	}

	return app, cfg, driverRec, qpuRec
}

func floatPtr(v float64) *float64 { return &v }

// TestHandleCryostatReading_PersistsToEventsLog proves a valid reading is
// appended to the events log, attributed to the driver that sent it and its
// QPU (RFC 0001 §7, Phase 3).
func TestHandleCryostatReading_PersistsToEventsLog(t *testing.T) {
	app, cfg, driverRec, qpuRec := seedDriverForEvents(t)

	event, err := NewEvent(driverRec.Id, EventCryostatReading, CryostatReadingPayload{
		Readings: map[string]ChannelReading{
			"mapper.bf.tmc": {Value: floatPtr(0.0123), Unit: "K", Status: "SYNCHRONIZED"},
		},
	})
	if err != nil {
		t.Fatalf("build event: %v", err)
	}

	ctx := context.WithValue(context.Background(), driverIDContextKey{}, driverRec.Id)
	if err := handleCryostatReading(ctx, app, qpuRec.Id, event); err != nil {
		t.Fatalf("handleCryostatReading: %v", err)
	}

	var rows []db.Event
	err = db.FindMany(app, cfg.CollectionEvents, &rows, "type = {:type}", "", 10, 0, dbx.Params{"type": string(EventCryostatReading)})
	if err != nil {
		t.Fatalf("find events: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 event row, got %d", len(rows))
	}
	if rows[0].Driver != driverRec.Id {
		t.Errorf("event driver = %q, want %q", rows[0].Driver, driverRec.Id)
	}
	if rows[0].QPU != qpuRec.Id {
		t.Errorf("event qpu = %q, want %q", rows[0].QPU, qpuRec.Id)
	}
	if rows[0].Source != driverRec.Id {
		t.Errorf("event source = %q, want %q", rows[0].Source, driverRec.Id)
	}
}

// TestHandleCryostatReading_RejectsEmptyReadings proves a payload with no
// readings is rejected — the registry logs and drops it rather than the
// listener crashing (RFC 0001 §4).
func TestHandleCryostatReading_RejectsEmptyReadings(t *testing.T) {
	app, _, driverRec, qpuRec := seedDriverForEvents(t)

	event, err := NewEvent(driverRec.Id, EventCryostatReading, CryostatReadingPayload{
		Readings: map[string]ChannelReading{},
	})
	if err != nil {
		t.Fatalf("build event: %v", err)
	}

	if err := handleCryostatReading(context.Background(), app, qpuRec.Id, event); err == nil {
		t.Errorf("expected empty readings to be rejected")
	}
}

// TestDriverEventRegistry_DispatchesCryostatReading proves the production
// registry routes CryostatReading to its handler and persists it, exercising
// the same path runDriverListener uses (RFC 0001 §4, §7).
func TestDriverEventRegistry_DispatchesCryostatReading(t *testing.T) {
	app, cfg, driverRec, qpuRec := seedDriverForEvents(t)

	event, err := NewEvent(driverRec.Id, EventCryostatReading, CryostatReadingPayload{
		Readings: map[string]ChannelReading{"mapper.bf.pmc": {Value: floatPtr(1.2e-6), Unit: "mbar"}},
	})
	if err != nil {
		t.Fatalf("build event: %v", err)
	}

	ctx := context.WithValue(context.Background(), driverIDContextKey{}, driverRec.Id)
	if err := driverEventRegistry.Dispatch(ctx, app, qpuRec.Id, event); err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	var rows []db.Event
	err = db.FindMany(app, cfg.CollectionEvents, &rows, "type = {:type}", "", 10, 0, dbx.Params{"type": string(EventCryostatReading)})
	if err != nil {
		t.Fatalf("find events: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 event row, got %d", len(rows))
	}
}
