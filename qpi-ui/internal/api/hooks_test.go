package api

import (
	"fmt"
	"testing"
	"time"

	"qpi/internal/config"
	"qpi/internal/db"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func TestOnQPUTimeRequestUpdateRequest_ApprovalAddsSeconds(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("failed to create test app: %v", err)
	}
	defer app.Cleanup()
	config.SaveConfigOnApp(app, &config.AppConfig{CollectionQPUs: config.DefaultQpusCollection, CollectionTimeSlots: config.DefaultTimeSlotsCollection, CollectionQuantumJobs: config.DefaultQuantumJobsCollection, CollectionAPITokens: config.DefaultAPITokensCollection, CollectionNotifications: config.DefaultNotificationsCollection, CollectionQPUTimeRequests: "qpu_time_requests"})
	if err := db.EnsureSchema(app); err != nil {
		t.Fatalf("failed to ensure schema: %v", err)
	}

	cfg := &config.AppConfig{
		CollectionQPUs:            config.DefaultQpusCollection,
		CollectionTimeSlots:       config.DefaultTimeSlotsCollection,
		CollectionQuantumJobs:     config.DefaultQuantumJobsCollection,
		CollectionAPITokens:       config.DefaultAPITokensCollection,
		CollectionNotifications:   config.DefaultNotificationsCollection,
		CollectionQPUTimeRequests: "qpu_time_requests",
	}
	_ = cfg

	// Create a regular user with initial qpu_seconds
	usersCol := getCollectionByName(t, app, "users")
	userRec := core.NewRecord(usersCol)
	userRec.Set("email", fmt.Sprintf("test_%d@example.com", time.Now().UnixNano()))
	userRec.Set("password", "testpassword1234")
	userRec.Set("qpu_seconds", 100.0)
	if err := app.Save(userRec); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Create a superuser admin
	adminsCol := getCollectionByName(t, app, "_superusers")
	adminRec := core.NewRecord(adminsCol)
	adminRec.Set("email", fmt.Sprintf("admin_%d@example.com", time.Now().UnixNano()))
	adminRec.Set("password", "adminpassword1234")
	if err := app.Save(adminRec); err != nil {
		t.Fatalf("failed to create admin: %v", err)
	}

	// Create a pending time request
	requestsCol := getCollectionByName(t, app, "qpu_time_requests")
	reqRec := core.NewRecord(requestsCol)
	reqRec.Set("user", userRec.Id)
	reqRec.Set("seconds", 250.0)
	reqRec.Set("reason", "Need more time")
	reqRec.Set("status", "pending")
	if err := app.Save(reqRec); err != nil {
		t.Fatalf("failed to create time request: %v", err)
	}

	// Simulate admin approving the request
	reqRec, err = app.FindRecordById("qpu_time_requests", reqRec.Id)
	if err != nil {
		t.Fatalf("FindRecordById failed: %v", err)
	}
	reqRec.Set("status", "approved")

	e := &core.RecordRequestEvent{
		RequestEvent: &core.RequestEvent{
			App:  app,
			Auth: adminRec,
		},
		Record: reqRec,
	}

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
	config.SaveConfigOnApp(app, &config.AppConfig{CollectionQPUs: config.DefaultQpusCollection, CollectionTimeSlots: config.DefaultTimeSlotsCollection, CollectionQuantumJobs: config.DefaultQuantumJobsCollection, CollectionAPITokens: config.DefaultAPITokensCollection, CollectionNotifications: config.DefaultNotificationsCollection, CollectionQPUTimeRequests: "qpu_time_requests"})
	if err := db.EnsureSchema(app); err != nil {
		t.Fatalf("failed to ensure schema: %v", err)
	}

	// Create a regular user with initial qpu_seconds
	usersCol := getCollectionByName(t, app, "users")
	userRec := core.NewRecord(usersCol)
	userRec.Set("email", fmt.Sprintf("test_%d@example.com", time.Now().UnixNano()))
	userRec.Set("password", "testpassword1234")
	userRec.Set("qpu_seconds", 500.0)
	if err := app.Save(userRec); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Create a superuser admin
	adminsCol := getCollectionByName(t, app, "_superusers")
	adminRec := core.NewRecord(adminsCol)
	adminRec.Set("email", fmt.Sprintf("admin_%d@example.com", time.Now().UnixNano()))
	adminRec.Set("password", "adminpassword1234")
	if err := app.Save(adminRec); err != nil {
		t.Fatalf("failed to create admin: %v", err)
	}

	// Create a pending time request
	requestsCol := getCollectionByName(t, app, "qpu_time_requests")
	reqRec := core.NewRecord(requestsCol)
	reqRec.Set("user", userRec.Id)
	reqRec.Set("seconds", 300.0)
	reqRec.Set("reason", "Testing rejection")
	reqRec.Set("status", "pending")
	if err := app.Save(reqRec); err != nil {
		t.Fatalf("failed to create time request: %v", err)
	}

	// Simulate admin rejecting the request
	reqRec, err = app.FindRecordById("qpu_time_requests", reqRec.Id)
	if err != nil {
		t.Fatalf("FindRecordById failed: %v", err)
	}
	reqRec.Set("status", "rejected")
	reqRec.Set("rejection_reason", "Insufficient justification")

	e := &core.RecordRequestEvent{
		RequestEvent: &core.RequestEvent{
			App:  app,
			Auth: adminRec,
		},
		Record: reqRec,
	}

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
	config.SaveConfigOnApp(app, &config.AppConfig{CollectionQPUs: config.DefaultQpusCollection, CollectionTimeSlots: config.DefaultTimeSlotsCollection, CollectionQuantumJobs: config.DefaultQuantumJobsCollection, CollectionAPITokens: config.DefaultAPITokensCollection, CollectionNotifications: config.DefaultNotificationsCollection, CollectionQPUTimeRequests: "qpu_time_requests"})
	if err := db.EnsureSchema(app); err != nil {
		t.Fatalf("failed to ensure schema: %v", err)
	}

	// Create a regular user
	usersCol := getCollectionByName(t, app, "users")
	userRec := core.NewRecord(usersCol)
	userRec.Set("email", fmt.Sprintf("regular_%d@example.com", time.Now().UnixNano()))
	userRec.Set("password", "regularpassword1234")
	if err := app.Save(userRec); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Create another regular user who will try to update
	otherUser := core.NewRecord(usersCol)
	otherUser.Set("email", fmt.Sprintf("other_%d@example.com", time.Now().UnixNano()))
	otherUser.Set("password", "otherpassword1234")
	if err := app.Save(otherUser); err != nil {
		t.Fatalf("failed to create other user: %v", err)
	}

	// Create a pending time request
	reqsCol := getCollectionByName(t, app, "qpu_time_requests")
	reqRec := core.NewRecord(reqsCol)
	reqRec.Set("user", userRec.Id)
	reqRec.Set("seconds", 100.0)
	reqRec.Set("reason", "Test")
	reqRec.Set("status", "pending")
	if err := app.Save(reqRec); err != nil {
		t.Fatalf("failed to create time request: %v", err)
	}

	reqRec, err = app.FindRecordById("qpu_time_requests", reqRec.Id)
	if err != nil {
		t.Fatalf("FindRecordById failed: %v", err)
	}
	reqRec.Set("status", "approved")

	e := &core.RecordRequestEvent{
		RequestEvent: &core.RequestEvent{
			App:  app,
			Auth: otherUser,
		},
		Record: reqRec,
	}

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
	config.SaveConfigOnApp(app, &config.AppConfig{CollectionQPUs: config.DefaultQpusCollection, CollectionTimeSlots: config.DefaultTimeSlotsCollection, CollectionQuantumJobs: config.DefaultQuantumJobsCollection, CollectionAPITokens: config.DefaultAPITokensCollection, CollectionNotifications: config.DefaultNotificationsCollection, CollectionQPUTimeRequests: "qpu_time_requests"})
	if err := db.EnsureSchema(app); err != nil {
		t.Fatalf("failed to ensure schema: %v", err)
	}

	// Create a regular user
	usersCol := getCollectionByName(t, app, "users")
	userRec := core.NewRecord(usersCol)
	userRec.Set("email", fmt.Sprintf("test_%d@example.com", time.Now().UnixNano()))
	userRec.Set("password", "testpassword1234")
	if err := app.Save(userRec); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Create a superuser admin
	adminsCol := getCollectionByName(t, app, "_superusers")
	adminRec := core.NewRecord(adminsCol)
	adminRec.Set("email", fmt.Sprintf("admin_%d@example.com", time.Now().UnixNano()))
	adminRec.Set("password", "adminpassword1234")
	if err := app.Save(adminRec); err != nil {
		t.Fatalf("failed to create admin: %v", err)
	}

	// Create an already-approved time request
	reqsCol := getCollectionByName(t, app, "qpu_time_requests")
	reqRec := core.NewRecord(reqsCol)
	reqRec.Set("user", userRec.Id)
	reqRec.Set("seconds", 100.0)
	reqRec.Set("reason", "Test")
	reqRec.Set("status", "approved")
	if err := app.Save(reqRec); err != nil {
		t.Fatalf("failed to create time request: %v", err)
	}

	// Try to change it to rejected
	reqRec, err = app.FindRecordById("qpu_time_requests", reqRec.Id)
	if err != nil {
		t.Fatalf("FindRecordById failed: %v", err)
	}
	reqRec.Set("status", "rejected")

	e := &core.RecordRequestEvent{
		RequestEvent: &core.RequestEvent{
			App:  app,
			Auth: adminRec,
		},
		Record: reqRec,
	}

	err = OnQPUTimeRequestUpdateRequest(e)
	if err == nil {
		t.Fatal("expected error when modifying already-processed request, got nil")
	}
}

// getCollectionByName gets the collection by name or errors out and fails the test
func getCollectionByName(t *testing.T, app *tests.TestApp, name string) *core.Collection {
	col, err := app.FindCollectionByNameOrId(name)
	if err != nil {
		t.Errorf("error getting %s collection: %v", name, err)
	}
	return col
}
