package api
package api

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/sopherapps/qpi/qpi-ui/internal/config"
)

func TestOnQPUTimeRequestUpdateRequest_ApprovalAddsSeconds(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("failed to create test app: %v", err)
	}
	defer app.Cleanup()

	cfg := &config.AppConfig{
		CollectionQPUTimeRequests: "qpu_time_requests",
	}
	_ = cfg

	// Create a regular user with initial qpu_seconds
	userRec := core.NewRecord(app.FindCollectionByNameOrId("users"))
	userRec.Set("email", "test@example.com")
	userRec.Set("password", "testpassword1234")
	userRec.Set("qpu_seconds", 100.0)
	if err := app.Save(userRec); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Create a superuser admin
	adminRec := core.NewRecord(app.FindCollectionByNameOrId("_superusers"))
	adminRec.Set("email", "admin@example.com")
	adminRec.Set("password", "adminpassword1234")
	if err := app.Save(adminRec); err != nil {
		t.Fatalf("failed to create admin: %v", err)
	}

	// Create a pending time request
	reqRec := core.NewRecord(app.FindCollectionByNameOrId("qpu_time_requests"))
	reqRec.Set("user", userRec.Id)
	reqRec.Set("seconds", 250.0)
	reqRec.Set("reason", "Need more time")
	reqRec.Set("status", "pending")
	if err := app.Save(reqRec); err != nil {
		t.Fatalf("failed to create time request: %v", err)
	}

	// Simulate admin approving the request
	reqRec.Set("status", "approved")

	e := new(core.RecordRequestEvent)
	e.App = app
	e.Auth = adminRec
	e.Record = reqRec

	if err := OnQPUTimeRequestUpdateRequest(e); err != nil {
		t.Fatalf("approval failed: %v", err)
	}

	// Reload user and verify seconds increased
	updatedUser, err := app.FindRecordById("users", userRec.Id)
	if err != nil {
		t.Fatalf("failed to reload user: %v", err)
	}

	got := updatedUser.GetFloat("qpu_seconds")
	want := 350.0 // 100 + 250
	if got != want {
		t.Errorf("qpu_seconds after approval = %v, want %v", got, want)
	}
}

func TestOnQPUTimeRequestUpdateRequest_RejectionDoesNotChangeSeconds(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("failed to create test app: %v", err)
	}
	defer app.Cleanup()

	// Create a regular user with initial qpu_seconds
	userRec := core.NewRecord(app.FindCollectionByNameOrId("users"))
	userRec.Set("email", "test@example.com")
	userRec.Set("password", "testpassword1234")
	userRec.Set("qpu_seconds", 500.0)
	if err := app.Save(userRec); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Create a superuser admin
	adminRec := core.NewRecord(app.FindCollectionByNameOrId("_superusers"))
	adminRec.Set("email", "admin@example.com")
	adminRec.Set("password", "adminpassword1234")
	if err := app.Save(adminRec); err != nil {
		t.Fatalf("failed to create admin: %v", err)
	}

	// Create a pending time request
	reqRec := core.NewRecord(app.FindCollectionByNameOrId("qpu_time_requests"))
	reqRec.Set("user", userRec.Id)
	reqRec.Set("seconds", 300.0)
	reqRec.Set("reason", "Testing rejection")
	reqRec.Set("status", "pending")
	if err := app.Save(reqRec); err != nil {
		t.Fatalf("failed to create time request: %v", err)
	}

	// Simulate admin rejecting the request
	reqRec.Set("status", "rejected")
	reqRec.Set("rejection_reason", "Insufficient justification")

	e := new(core.RecordRequestEvent)
	e.App = app
	e.Auth = adminRec
	e.Record = reqRec

	if err := OnQPUTimeRequestUpdateRequest(e); err != nil {
		t.Fatalf("rejection failed: %v", err)
	}

	// Reload user and verify seconds unchanged
	updatedUser, err := app.FindRecordById("users", userRec.Id)
	if err != nil {
		t.Fatalf("failed to reload user: %v", err)
	}

	got := updatedUser.GetFloat("qpu_seconds")
	want := 500.0
	if got != want {
		t.Errorf("qpu_seconds after rejection = %v, want %v", got, want)
	}
}

func TestOnQPUTimeRequestUpdateRequest_NonSuperuserForbidden(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("failed to create test app: %v", err)
	}
	defer app.Cleanup()

	// Create a regular user
	userRec := core.NewRecord(app.FindCollectionByNameOrId("users"))
	userRec.Set("email", "regular@example.com")
	userRec.Set("password", "regularpassword1234")
	if err := app.Save(userRec); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Create another regular user who will try to update
	otherUser := core.NewRecord(app.FindCollectionByNameOrId("users"))
	otherUser.Set("email", "other@example.com")
	otherUser.Set("password", "otherpassword1234")
	if err := app.Save(otherUser); err != nil {
		t.Fatalf("failed to create other user: %v", err)
	}

	// Create a pending time request
	reqRec := core.NewRecord(app.FindCollectionByNameOrId("qpu_time_requests"))
	reqRec.Set("user", userRec.Id)
	reqRec.Set("seconds", 100.0)
	reqRec.Set("reason", "Test")
	reqRec.Set("status", "pending")
	if err := app.Save(reqRec); err != nil {
		t.Fatalf("failed to create time request: %v", err)
	}

	reqRec.Set("status", "approved")

	e := new(core.RecordRequestEvent)
	e.App = app
	e.Auth = otherUser
	e.Record = reqRec

	err = OnQPUTimeRequestUpdateRequest(e)
	if err == nil {
		t.Fatal("expected error for non-superuser update, got nil")
	}
}

func TestOnQPUTimeRequestUpdateRequest_CannotModifyProcessedRequest(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("failed to create test app: %v", err)
	}
	defer app.Cleanup()

	// Create a regular user
	userRec := core.NewRecord(app.FindCollectionByNameOrId("users"))
	userRec.Set("email", "test@example.com")
	userRec.Set("password", "testpassword1234")
	if err := app.Save(userRec); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Create a superuser admin
	adminRec := core.NewRecord(app.FindCollectionByNameOrId("_superusers"))
	adminRec.Set("email", "admin@example.com")
	adminRec.Set("password", "adminpassword1234")
	if err := app.Save(adminRec); err != nil {
		t.Fatalf("failed to create admin: %v", err)
	}

	// Create an already-approved time request
	reqRec := core.NewRecord(app.FindCollectionByNameOrId("qpu_time_requests"))
	reqRec.Set("user", userRec.Id)
	reqRec.Set("seconds", 100.0)
	reqRec.Set("reason", "Test")
	reqRec.Set("status", "approved")
	if err := app.Save(reqRec); err != nil {
		t.Fatalf("failed to create time request: %v", err)
	}

	// Try to change it to rejected
	reqRec.Set("status", "rejected")

	e := new(core.RecordRequestEvent)
	e.App = app
	e.Auth = adminRec
	e.Record = reqRec

	err = OnQPUTimeRequestUpdateRequest(e)
	if err == nil {
		t.Fatal("expected error when modifying already-processed request, got nil")
	}
}
