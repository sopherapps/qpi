package db

import (
	"fmt"
	"strings"
	"testing"

	"qpi/internal/config"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// testConfig builds an AppConfig with the default collection names, mirroring
// the setup every other db/api test uses.
func testConfig() *config.AppConfig {
	return &config.AppConfig{
		CollectionQPUs:            config.DefaultQpusCollection,
		CollectionTimeSlots:       config.DefaultTimeSlotsCollection,
		CollectionQuantumJobs:     config.DefaultQuantumJobsCollection,
		CollectionAPITokens:       config.DefaultAPITokensCollection,
		CollectionNotifications:   config.DefaultNotificationsCollection,
		CollectionQPUTimeRequests: config.DefaultQPUTimeRequestsCollection,
		CollectionDrivers:         config.DefaultDriversCollection,
		CollectionEvents:          config.DefaultEventsCollection,
	}
}

// TestEnsureSchema_DriversCollectionOffByDefault proves the drivers
// collection is additive: it does not exist while EnableDriverFramework is
// off, so the server behaves exactly as before RFC 0001 (§5, §11).
func TestEnsureSchema_DriversCollectionOffByDefault(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("failed to create test app: %v", err)
	}
	defer app.Cleanup()

	cfg := testConfig()
	cfg.EnableDriverFramework = false
	config.SaveConfigOnApp(app, cfg)

	if err := EnsureSchema(app); err != nil {
		t.Fatalf("failed to ensure schema: %v", err)
	}

	if _, err := app.FindCollectionByNameOrId(config.DefaultDriversCollection); err == nil {
		t.Errorf("expected no drivers collection to exist while EnableDriverFramework is off")
	}
}

// TestEnsureSchema_DriversCollectionOnWithFlag proves that turning the flag
// on creates the drivers collection without touching the qpus collection's
// rules (RFC 0001 §11 done-criteria for Phase 1).
func TestEnsureSchema_DriversCollectionOnWithFlag(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("failed to create test app: %v", err)
	}
	defer app.Cleanup()

	cfg := testConfig()
	cfg.EnableDriverFramework = false
	config.SaveConfigOnApp(app, cfg)
	if err := EnsureSchema(app); err != nil {
		t.Fatalf("failed to ensure schema with flag off: %v", err)
	}

	qpusBefore, err := app.FindCollectionByNameOrId(config.DefaultQpusCollection)
	if err != nil {
		t.Fatalf("qpus collection not found: %v", err)
	}
	rulesBefore := collectionRuleSnapshot(qpusBefore)

	cfg.EnableDriverFramework = true
	config.SaveConfigOnApp(app, cfg)
	if err := EnsureSchema(app); err != nil {
		t.Fatalf("failed to ensure schema with flag on: %v", err)
	}

	if _, err := app.FindCollectionByNameOrId(config.DefaultDriversCollection); err != nil {
		t.Fatalf("expected drivers collection to exist with flag on: %v", err)
	}

	qpusAfter, err := app.FindCollectionByNameOrId(config.DefaultQpusCollection)
	if err != nil {
		t.Fatalf("qpus collection not found after enabling flag: %v", err)
	}
	rulesAfter := collectionRuleSnapshot(qpusAfter)

	if rulesBefore != rulesAfter {
		t.Errorf("expected qpus collection rules unchanged, before=%q after=%q", rulesBefore, rulesAfter)
	}
}

// TestEnsureSchema_EventsCollectionOffByDefault proves the events collection
// is additive just like drivers: it does not exist while
// EnableDriverFramework is off (RFC 0001 §5, §7, §11).
func TestEnsureSchema_EventsCollectionOffByDefault(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("failed to create test app: %v", err)
	}
	defer app.Cleanup()

	cfg := testConfig()
	cfg.EnableDriverFramework = false
	config.SaveConfigOnApp(app, cfg)

	if err := EnsureSchema(app); err != nil {
		t.Fatalf("failed to ensure schema: %v", err)
	}

	if _, err := app.FindCollectionByNameOrId(config.DefaultEventsCollection); err == nil {
		t.Errorf("expected no events collection to exist while EnableDriverFramework is off")
	}
}

// TestEnsureSchema_EventsCollectionOnWithFlag proves turning the flag on
// creates the events collection without touching the drivers collection's
// rules.
func TestEnsureSchema_EventsCollectionOnWithFlag(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("failed to create test app: %v", err)
	}
	defer app.Cleanup()

	cfg := testConfig()
	cfg.EnableDriverFramework = true
	config.SaveConfigOnApp(app, cfg)
	if err := EnsureSchema(app); err != nil {
		t.Fatalf("failed to ensure schema with flag on: %v", err)
	}

	driversBefore, err := app.FindCollectionByNameOrId(config.DefaultDriversCollection)
	if err != nil {
		t.Fatalf("drivers collection not found: %v", err)
	}
	rulesBefore := collectionRuleSnapshot(driversBefore)

	col, err := app.FindCollectionByNameOrId(config.DefaultEventsCollection)
	if err != nil {
		t.Fatalf("expected events collection to exist with flag on: %v", err)
	}

	wantFields := []string{"source", "driver", "qpu", "type", "payload", "ts", "created"}
	for _, name := range wantFields {
		found := false
		for _, f := range col.Fields {
			if f.GetName() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected events collection to have field %q", name)
		}
	}

	wantIndex := fmt.Sprintf("idx_%s_type_ts", config.DefaultEventsCollection)
	if !hasIndex(col, wantIndex) {
		t.Errorf("expected events collection to have index %q, got %v", wantIndex, col.Indexes)
	}

	driversAfter, err := app.FindCollectionByNameOrId(config.DefaultDriversCollection)
	if err != nil {
		t.Fatalf("drivers collection not found after ensuring events: %v", err)
	}
	rulesAfter := collectionRuleSnapshot(driversAfter)
	if rulesBefore != rulesAfter {
		t.Errorf("expected drivers collection rules unchanged, before=%q after=%q", rulesBefore, rulesAfter)
	}
}

// TestEnsureSchema_EventsIndexIdempotent proves running the migration twice
// does not duplicate the (type, ts) index.
func TestEnsureSchema_EventsIndexIdempotent(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("failed to create test app: %v", err)
	}
	defer app.Cleanup()

	cfg := testConfig()
	cfg.EnableDriverFramework = true
	config.SaveConfigOnApp(app, cfg)
	if err := EnsureSchema(app); err != nil {
		t.Fatalf("failed to ensure schema (first pass): %v", err)
	}
	if err := EnsureSchema(app); err != nil {
		t.Fatalf("failed to ensure schema (second pass): %v", err)
	}

	col, err := app.FindCollectionByNameOrId(config.DefaultEventsCollection)
	if err != nil {
		t.Fatalf("events collection not found: %v", err)
	}

	wantIndex := fmt.Sprintf("idx_%s_type_ts", config.DefaultEventsCollection)
	count := 0
	for _, idx := range col.Indexes {
		if strings.Contains(idx, wantIndex) {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly one %q index, got %d: %v", wantIndex, count, col.Indexes)
	}
}

// collectionRuleSnapshot renders a collection's API rules into a comparable
// string, treating a nil rule as the literal "nil".
func collectionRuleSnapshot(col *core.Collection) string {
	rule := func(r *string) string {
		if r == nil {
			return "nil"
		}
		return *r
	}
	return fmt.Sprintf(
		"list=%s view=%s create=%s update=%s delete=%s",
		rule(col.ListRule), rule(col.ViewRule), rule(col.CreateRule), rule(col.UpdateRule), rule(col.DeleteRule),
	)
}
