package db

import (
	"fmt"
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
