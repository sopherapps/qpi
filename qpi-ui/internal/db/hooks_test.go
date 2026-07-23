package db

import (
	"testing"

	"qpi/internal/config"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/hook"
)

// TestOnDriverCreate_HashesTokenOnSave proves a driver's raw token gets
// hashed the same way OnQpuCreate hashes a QPU's access_token — via the
// real app.Save() hook chain, mirroring how main.go wires it (RFC 0001 §7, §9).
func TestOnDriverCreate_HashesTokenOnSave(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("failed to create test app: %v", err)
	}
	defer app.Cleanup()

	cfg := testConfig()
	config.SaveConfigOnApp(app, cfg)
	if err := EnsureSchema(app); err != nil {
		t.Fatalf("failed to ensure schema: %v", err)
	}

	app.OnRecordCreate().Bind(&hook.Handler[*core.RecordEvent]{
		Func: RegisterCollectionHooks(app, CollectionHookMap{
			config.DefaultDriversCollection: OnDriverCreate,
		}),
	})

	qpuCol, err := app.FindCollectionByNameOrId(config.DefaultQpusCollection)
	if err != nil {
		t.Fatalf("qpus collection not found: %v", err)
	}
	qpuRec := core.NewRecord(qpuCol)
	qpuRec.Set("name", "qpu_for_driver_test")
	qpuRec.Set("access_token", "qpu_raw_token")
	qpuRec.Set("status", "offline")
	qpuRec.Set("enabled", true)
	if err := app.Save(qpuRec); err != nil {
		t.Fatalf("failed to create qpu: %v", err)
	}

	driversCol, err := app.FindCollectionByNameOrId(config.DefaultDriversCollection)
	if err != nil {
		t.Fatalf("drivers collection not found: %v", err)
	}
	driverRec := core.NewRecord(driversCol)
	driverRec.Set("name", "driver_1")
	driverRec.Set("qpu", qpuRec.Id)
	driverRec.Set("kind", "mock")
	driverRec.Set("language", "python")
	driverRec.Set("events", []string{"JobDispatch", "JobResult"})
	driverRec.Set("status", "offline")
	driverRec.Set("enabled", true)
	driverRec.Set("token", "raw_driver_token")

	if err := app.Save(driverRec); err != nil {
		t.Fatalf("failed to create driver: %v", err)
	}

	got := driverRec.GetString("token")
	want := HashToken("raw_driver_token")
	if got != want {
		t.Errorf("stored token = %q, want hashed %q", got, want)
	}
	if got == "raw_driver_token" {
		t.Errorf("expected the raw token not to be stored as-is")
	}
}
