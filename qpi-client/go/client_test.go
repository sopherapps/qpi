package qpiclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// newTestServer spins up an httptest.Server that replies with the provided
// status code and JSON body for every request.
func newTestServer(t *testing.T, status int, body any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(body)
	}))
}

// ---------------------------------------------------------------------------
// SubmitJob
// ---------------------------------------------------------------------------

func TestSubmitJob_Success(t *testing.T) {
	server := newTestServer(t, http.StatusCreated, map[string]string{"job_id": "job-123"})
	defer server.Close()

	client := NewClient(server.URL, "token")
	id, err := client.SubmitJob(context.Background(), JobSubmitRequest{
		Circuits: []CircuitPayload{{Circuit: "OPENQASM 3.0;"}},
		Shots:    1024,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "job-123" {
		t.Errorf("expected job-123, got %s", id)
	}
}

func TestSubmitJob_IDField(t *testing.T) {
	server := newTestServer(t, http.StatusCreated, map[string]string{"id": "job-456"})
	defer server.Close()

	client := NewClient(server.URL, "")
	id, err := client.SubmitJob(context.Background(), JobSubmitRequest{
		Circuits: []CircuitPayload{{Circuit: "qasm"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "job-456" {
		t.Errorf("expected job-456, got %s", id)
	}
}

func TestSubmitJob_MissingID(t *testing.T) {
	server := newTestServer(t, http.StatusCreated, map[string]string{"other": "data"})
	defer server.Close()

	client := NewClient(server.URL, "")
	_, err := client.SubmitJob(context.Background(), JobSubmitRequest{
		Circuits: []CircuitPayload{{Circuit: "qasm"}},
	})
	if err == nil {
		t.Fatal("expected error when job ID is missing")
	}
}

func TestSubmitJob_ServerError(t *testing.T) {
	server := newTestServer(t, http.StatusForbidden, map[string]string{"error": "no qpu seconds"})
	defer server.Close()

	client := NewClient(server.URL, "")
	_, err := client.SubmitJob(context.Background(), JobSubmitRequest{
		Circuits: []CircuitPayload{{Circuit: "qasm"}},
	})
	if err == nil {
		t.Fatal("expected error on 403")
	}
	// doJSON wraps the APIError, so use errors.As to unwrap.
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError wrapped in error chain, got %T", err)
	}
	if apiErr.StatusCode != 403 {
		t.Errorf("expected status 403, got %d", apiErr.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// GetJob
// ---------------------------------------------------------------------------

func TestGetJob_Success(t *testing.T) {
	server := newTestServer(t, http.StatusOK, map[string]any{
		"id":       "job-123",
		"status":   "completed",
		"duration": 1.23,
		"results": map[string]any{
			"counts": map[string]int{"0x0": 512},
		},
	})
	defer server.Close()

	client := NewClient(server.URL, "")
	job, err := client.GetJob(context.Background(), "job-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if job.ID != "job-123" {
		t.Errorf("expected job-123, got %s", job.ID)
	}
	if job.Status != "completed" {
		t.Errorf("expected completed, got %s", job.Status)
	}
	if job.Duration != 1.23 {
		t.Errorf("expected duration 1.23, got %f", job.Duration)
	}
}

// ---------------------------------------------------------------------------
// ListJobs
// ---------------------------------------------------------------------------

func TestListJobs_BareArray(t *testing.T) {
	server := newTestServer(t, http.StatusOK, []map[string]any{
		{"id": "j1", "status": "pending"},
		{"id": "j2", "status": "completed"},
	})
	defer server.Close()

	client := NewClient(server.URL, "")
	jobs, err := client.ListJobs(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(jobs) != 2 {
		t.Errorf("expected 2 jobs, got %d", len(jobs))
	}
	if jobs[0].ID != "j1" {
		t.Errorf("expected j1, got %s", jobs[0].ID)
	}
}

func TestListJobs_Wrapped(t *testing.T) {
	server := newTestServer(t, http.StatusOK, map[string]any{
		"jobs": []map[string]any{
			{"id": "j3", "status": "running"},
		},
	})
	defer server.Close()

	client := NewClient(server.URL, "")
	jobs, err := client.ListJobs(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(jobs) != 1 {
		t.Errorf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].ID != "j3" {
		t.Errorf("expected j3, got %s", jobs[0].ID)
	}
}

// ---------------------------------------------------------------------------
// CancelJob
// ---------------------------------------------------------------------------

func TestCancelJob_Success(t *testing.T) {
	server := newTestServer(t, http.StatusOK, map[string]string{"status": "cancelled"})
	defer server.Close()

	client := NewClient(server.URL, "")
	job, err := client.CancelJob(context.Background(), "job-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if job.Status != "cancelled" {
		t.Errorf("expected cancelled, got %s", job.Status)
	}
}

// ---------------------------------------------------------------------------
// WaitForJob
// ---------------------------------------------------------------------------

func TestWaitForJob_AlreadyTerminal(t *testing.T) {
	server := newTestServer(t, http.StatusOK, map[string]string{"id": "j1", "status": "completed"})
	defer server.Close()

	client := NewClient(server.URL, "")
	job, err := client.WaitForJob(context.Background(), "j1", 10*time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if job.Status != "completed" {
		t.Errorf("expected completed, got %s", job.Status)
	}
}

func TestWaitForJob_PollsUntilTerminal(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount < 3 {
			_ = json.NewEncoder(w).Encode(map[string]string{"id": "j1", "status": "running"})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "j1", "status": "completed"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	job, err := client.WaitForJob(context.Background(), "j1", 10*time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if job.Status != "completed" {
		t.Errorf("expected completed, got %s", job.Status)
	}
	if callCount < 3 {
		t.Errorf("expected at least 3 calls, got %d", callCount)
	}
}

func TestWaitForJob_ContextCancelled(t *testing.T) {
	server := newTestServer(t, http.StatusOK, map[string]string{"id": "j1", "status": "running"})
	defer server.Close()

	client := NewClient(server.URL, "")
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := client.WaitForJob(ctx, "j1", 20*time.Millisecond)
	if err == nil {
		t.Fatal("expected error when context is cancelled")
	}
}

// ---------------------------------------------------------------------------
// Auth header
// ---------------------------------------------------------------------------

func TestAuthHeader_SentWhenTokenProvided(t *testing.T) {
	var receivedToken string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedToken = r.Header.Get("X-API-Token")
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "j1", "status": "completed"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "my-secret-token")
	_, _ = client.GetJob(context.Background(), "j1")

	if receivedToken != "my-secret-token" {
		t.Errorf("expected token 'my-secret-token', got %q", receivedToken)
	}
}

func TestAuthHeader_NotSentWhenEmpty(t *testing.T) {
	var receivedToken string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedToken = r.Header.Get("X-API-Token")
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "j1", "status": "completed"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	_, _ = client.GetJob(context.Background(), "j1")

	if receivedToken != "" {
		t.Errorf("expected no token, got %q", receivedToken)
	}
}

// ---------------------------------------------------------------------------
// APIError
// ---------------------------------------------------------------------------

func TestAPIError_ErrorString(t *testing.T) {
	err := &APIError{StatusCode: 404, Status: "Not Found", Body: "job missing"}
	expected := "qpi api error Not Found: job missing"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

// ---------------------------------------------------------------------------
// Request payload
// ---------------------------------------------------------------------------

func TestSubmitJob_RequestBody(t *testing.T) {
	var receivedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		_ = json.NewEncoder(w).Encode(map[string]string{"job_id": "j1"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	_, _ = client.SubmitJob(context.Background(), JobSubmitRequest{
		Circuits: []CircuitPayload{
			{Circuit: "qasm1", ParameterValues: [][]float64{{0.1, 0.2}}, Shots: intPtr(512)},
		},
		Shots:      1024,
		MeasLevel:  intPtr(1),
		MeasReturn: "avg",
		QPUTarget:  "qpu-01",
	})

	circuits, ok := receivedBody["circuits"].([]any)
	if !ok || len(circuits) != 1 {
		t.Fatalf("expected 1 circuit, got %+v", receivedBody["circuits"])
	}
	if receivedBody["shots"] != float64(1024) {
		t.Errorf("expected shots 1024, got %v", receivedBody["shots"])
	}
	if receivedBody["meas_level"] != float64(1) {
		t.Errorf("expected meas_level 1, got %v", receivedBody["meas_level"])
	}
	if receivedBody["meas_return"] != "avg" {
		t.Errorf("expected meas_return avg, got %v", receivedBody["meas_return"])
	}
	if receivedBody["qpu_target"] != "qpu-01" {
		t.Errorf("expected qpu_target qpu-01, got %v", receivedBody["qpu_target"])
	}
}

func intPtr(i int) *int {
	return &i
}

func TestNewClientMethods(t *testing.T) {
	ctx := context.Background()

	t.Run("ListQpus", func(t *testing.T) {
		server := newTestServer(t, http.StatusOK, []QpuRecord{{ID: "q1", Name: "qpu-1"}})
		defer server.Close()
		client := NewClient(server.URL, "")
		res, err := client.ListQpus(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(res) != 1 || res[0].Name != "qpu-1" {
			t.Errorf("unexpected: %+v", res)
		}
	})

	t.Run("GetQpu", func(t *testing.T) {
		server := newTestServer(t, http.StatusOK, QpuRecord{ID: "q1", Name: "qpu-1"})
		defer server.Close()
		client := NewClient(server.URL, "")
		res, err := client.GetQpu(ctx, "qpu-1")
		if err != nil {
			t.Fatal(err)
		}
		if res.ID != "q1" {
			t.Errorf("unexpected: %+v", res)
		}
	})

	t.Run("CreateQpu", func(t *testing.T) {
		server := newTestServer(t, http.StatusOK, QpuRecord{ID: "q1", Name: "qpu-1"})
		defer server.Close()
		client := NewClient(server.URL, "")
		res, err := client.CreateQpu(ctx, QpuRecord{Name: "qpu-1"})
		if err != nil {
			t.Fatal(err)
		}
		if res.ID != "q1" {
			t.Errorf("unexpected: %+v", res)
		}
	})

	t.Run("ConnectQpu", func(t *testing.T) {
		server := newTestServer(t, http.StatusOK, QpuRecord{ID: "q1", Name: "qpu-1"})
		defer server.Close()
		client := NewClient(server.URL, "")
		res, err := client.ConnectQpu(ctx, QpuRecord{Name: "qpu-1", AccessToken: "secret"})
		if err != nil {
			t.Fatal(err)
		}
		if res.ID != "q1" {
			t.Errorf("unexpected: %+v", res)
		}
	})

	t.Run("ToggleQpu", func(t *testing.T) {
		server := newTestServer(t, http.StatusOK, map[string]any{"success": true})
		defer server.Close()
		client := NewClient(server.URL, "")
		_, err := client.ToggleQpu(ctx, "q1", true)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("ListNotifications", func(t *testing.T) {
		server := newTestServer(t, http.StatusOK, map[string]any{"items": []NotificationRecord{{ID: "n1"}}})
		defer server.Close()
		client := NewClient(server.URL, "")
		res, err := client.ListNotifications(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(res) != 1 || res[0].ID != "n1" {
			t.Errorf("unexpected: %+v", res)
		}
	})

	t.Run("DismissNotification", func(t *testing.T) {
		server := newTestServer(t, http.StatusOK, map[string]any{"success": true})
		defer server.Close()
		client := NewClient(server.URL, "")
		_, err := client.DismissNotification(ctx, "n1")
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("BookingSlots", func(t *testing.T) {
		// list
		server1 := newTestServer(t, http.StatusOK, map[string]any{"items": []TimeSlotRecord{{ID: "s1"}}})
		client := NewClient(server1.URL, "")
		slots, err := client.ListTimeSlots(ctx)
		server1.Close()
		if err != nil {
			t.Fatal(err)
		}
		if len(slots) != 1 || slots[0].ID != "s1" {
			t.Errorf("unexpected list: %+v", slots)
		}

		// create
		server2 := newTestServer(t, http.StatusOK, TimeSlotRecord{ID: "s1"})
		client.BaseURL = server2.URL
		slot, err := client.CreateTimeSlot(ctx, TimeSlotRecord{StartTime: "start", EndTime: "end"})
		server2.Close()
		if err != nil {
			t.Fatal(err)
		}
		if slot.ID != "s1" {
			t.Errorf("unexpected create: %+v", slot)
		}

		// update
		server3 := newTestServer(t, http.StatusOK, TimeSlotRecord{ID: "s1", StartTime: "start2"})
		client.BaseURL = server3.URL
		slot, err = client.UpdateTimeSlot(ctx, "s1", TimeSlotRecord{StartTime: "start2"})
		server3.Close()
		if err != nil {
			t.Fatal(err)
		}
		if slot.StartTime != "start2" {
			t.Errorf("unexpected update: %+v", slot)
		}

		// delete
		server4 := newTestServer(t, http.StatusNoContent, nil)
		client.BaseURL = server4.URL
		err = client.DeleteTimeSlot(ctx, "s1")
		server4.Close()
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("TimeRequests", func(t *testing.T) {
		// list
		server1 := newTestServer(t, http.StatusOK, map[string]any{"items": []TimeRequestRecord{{ID: "tr1"}}})
		client := NewClient(server1.URL, "")
		reqs, err := client.ListTimeRequests(ctx)
		server1.Close()
		if err != nil {
			t.Fatal(err)
		}
		if len(reqs) != 1 || reqs[0].ID != "tr1" {
			t.Errorf("unexpected list: %+v", reqs)
		}

		// create
		server2 := newTestServer(t, http.StatusOK, TimeRequestRecord{ID: "tr1"})
		client.BaseURL = server2.URL
		req, err := client.CreateTimeRequest(ctx, TimeRequestRecord{Seconds: 100})
		server2.Close()
		if err != nil {
			t.Fatal(err)
		}
		if req.ID != "tr1" {
			t.Errorf("unexpected create: %+v", req)
		}

		// update
		server3 := newTestServer(t, http.StatusOK, TimeRequestRecord{ID: "tr1", Status: "approved"})
		client.BaseURL = server3.URL
		req, err = client.UpdateTimeRequest(ctx, "tr1", TimeRequestRecord{Status: "approved"})
		server3.Close()
		if err != nil {
			t.Fatal(err)
		}
		if req.Status != "approved" {
			t.Errorf("unexpected update: %+v", req)
		}
	})

	t.Run("AdminUserQuota", func(t *testing.T) {
		// list
		server1 := newTestServer(t, http.StatusOK, map[string]any{"items": []UserRecord{{ID: "u1"}}})
		client := NewClient(server1.URL, "")
		users, err := client.ListUsers(ctx)
		server1.Close()
		if err != nil {
			t.Fatal(err)
		}
		if len(users) != 1 || users[0].ID != "u1" {
			t.Errorf("unexpected list: %+v", users)
		}

		// allocate
		server2 := newTestServer(t, http.StatusOK, UserRecord{ID: "u1", QpuSeconds: 500})
		client.BaseURL = server2.URL
		user, err := client.AllocateQpuTime(ctx, "u1", 500)
		server2.Close()
		if err != nil {
			t.Fatal(err)
		}
		if user.QpuSeconds != 500 {
			t.Errorf("unexpected allocate: %+v", user)
		}
	})

	t.Run("AuthWithPassword", func(t *testing.T) {
		server := newTestServer(t, http.StatusOK, map[string]any{"token": "jwt-token-123"})
		defer server.Close()
		client := NewClient(server.URL, "")
		_, err := client.AuthWithPassword(ctx, "alice", "secret")
		if err != nil {
			t.Fatal(err)
		}
		if client.BearerToken != "jwt-token-123" {
			t.Errorf("expected bearer token to be set, got %q", client.BearerToken)
		}
	})
}

// Ensure the compiler catches any interface drift.
var _ error = (*APIError)(nil)
