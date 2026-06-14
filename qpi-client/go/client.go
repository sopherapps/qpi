// Package qpiclient provides a Go client SDK for the QPI quantum computing
// platform orchestrator REST API.
//
// The [Client] type wraps all four API endpoints (submit, get, list, cancel)
// and adds a convenience [Client.WaitForJob] poller.  Only the Go standard
// library is used—there are no third-party dependencies.
//
// # Quick start
//
//	client := qpiclient.NewClient("http://localhost:8090", "my-api-token")
//
//	id, err := client.SubmitJob(ctx, qpiclient.JobSubmitRequest{
//	    Circuits: []qpiclient.CircuitPayload{{Circuit: "OPENQASM 3.0; ..."}},
//	    Shots:    1024,
//	})
//	if err != nil { log.Fatal(err) }
//
//	job, err := client.WaitForJob(ctx, id, 5*time.Second)
//	if err != nil { log.Fatal(err) }
//	fmt.Println(job.Status, job.Results)
package qpiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Request / response types
// ---------------------------------------------------------------------------

// CircuitPayload describes a single circuit within a job submission.
type CircuitPayload struct {
	// Circuit is an OpenQASM 3 string.
	Circuit string `json:"circuit"`
	// ParameterValues holds optional parameter bind sets.
	ParameterValues [][]float64 `json:"parameter_values,omitempty"`
	// Shots is an optional per-circuit shot override.
	Shots *int `json:"shots,omitempty"`
}

// JobSubmitRequest is the body of POST /api/jobs.
type JobSubmitRequest struct {
	Circuits   []CircuitPayload `json:"circuits"`
	Shots      int              `json:"shots,omitempty"`
	MeasLevel  *int             `json:"meas_level,omitempty"`
	MeasReturn string           `json:"meas_return,omitempty"`
	QPUTarget  string           `json:"qpu_target,omitempty"`
}

// JobRecord is the server representation of a job.
type JobRecord struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Payload any    `json:"payload"`
	Results any    `json:"results"`
	Created string `json:"created"`
	Updated string `json:"updated"`
}

// QpuRecord describes a QPU.
type QpuRecord struct {
	ID                string `json:"id,omitempty"`
	Name              string `json:"name"`
	RegistrationToken string `json:"registration_token,omitempty"`
	ExecutorType      string `json:"executor_type,omitempty"`
	DeviceConfig      any    `json:"device_config,omitempty"`
}

// NotificationRecord describes a notification.
type NotificationRecord struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Message string `json:"message"`
	Active  bool   `json:"active"`
}

// TimeSlotRecord describes a booking slot.
type TimeSlotRecord struct {
	ID        string `json:"id,omitempty"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	BookedBy  string `json:"booked_by,omitempty"`
}

// TimeRequestRecord describes a QPU time request.
type TimeRequestRecord struct {
	ID              string `json:"id,omitempty"`
	Seconds         int    `json:"seconds"`
	RequestedReason string `json:"requested_reason,omitempty"`
	Status          string `json:"status,omitempty"`
	RejectionReason string `json:"rejection_reason,omitempty"`
}

// UserRecord describes a user.
type UserRecord struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	QpuSeconds int    `json:"qpu_seconds"`
}

// ---------------------------------------------------------------------------
// Client
// ---------------------------------------------------------------------------

// Client is an HTTP client for the QPI orchestrator API.
type Client struct {
	// BaseURL is the root URL of the orchestrator (e.g. "http://localhost:8090").
	BaseURL string
	// APIToken is sent via the X-API-Token header when non-empty.
	APIToken string
	// BearerToken is sent via the Authorization header when non-empty.
	BearerToken string
	// HTTPClient is the underlying *http.Client used for requests.
	// If nil, http.DefaultClient is used.
	HTTPClient *http.Client
}

// NewClient returns a Client configured with the given base URL and API token.
// Pass an empty string for apiToken to rely on cookie/JWT auth instead.
func NewClient(baseURL, apiToken string) *Client {
	return &Client{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		APIToken:   apiToken,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// httpClient returns the effective *http.Client.
func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

// SubmitJob submits a quantum job and returns its server-assigned ID.
func (c *Client) SubmitJob(ctx context.Context, req JobSubmitRequest) (string, error) {
	var resp struct {
		ID    string `json:"id"`
		JobID string `json:"job_id"`
	}
	if err := c.doJSON(ctx, http.MethodPost, "/api/jobs", req, &resp); err != nil {
		return "", fmt.Errorf("submit job: %w", err)
	}
	id := resp.ID
	if id == "" {
		id = resp.JobID
	}
	if id == "" {
		return "", fmt.Errorf("submit job: server response did not contain a job ID")
	}
	return id, nil
}

// GetJob retrieves the full record for a single job.
func (c *Client) GetJob(ctx context.Context, jobID string) (*JobRecord, error) {
	var rec JobRecord
	if err := c.doJSON(ctx, http.MethodGet, "/api/jobs/"+jobID, nil, &rec); err != nil {
		return nil, fmt.Errorf("get job %s: %w", jobID, err)
	}
	return &rec, nil
}

// ListJobs returns all jobs belonging to the authenticated user.
func (c *Client) ListJobs(ctx context.Context) ([]JobRecord, error) {
	// The server may return a bare JSON array or {"jobs": [...]}.
	raw, err := c.doRaw(ctx, http.MethodGet, "/api/jobs", nil)
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}

	// Try bare array first.
	var list []JobRecord
	if err := json.Unmarshal(raw, &list); err == nil {
		return list, nil
	}

	// Try wrapped form.
	var wrapped struct {
		Jobs []JobRecord `json:"jobs"`
	}
	if err := json.Unmarshal(raw, &wrapped); err != nil {
		return nil, fmt.Errorf("list jobs: decode response: %w", err)
	}
	return wrapped.Jobs, nil
}

// CancelJob requests cancellation of a job and returns the updated record.
func (c *Client) CancelJob(ctx context.Context, jobID string) (*JobRecord, error) {
	var rec JobRecord
	if err := c.doJSON(ctx, http.MethodPost, "/api/jobs/"+jobID+"/cancel", nil, &rec); err != nil {
		return nil, fmt.Errorf("cancel job %s: %w", jobID, err)
	}
	return &rec, nil
}

// WaitForJob polls GetJob at the given interval until the job reaches a
// terminal state (completed, failed, cancelled) or the context is cancelled.
func (c *Client) WaitForJob(ctx context.Context, jobID string, interval time.Duration) (*JobRecord, error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		rec, err := c.GetJob(ctx, jobID)
		if err != nil {
			return nil, err
		}
		switch rec.Status {
		case "completed", "failed", "cancelled":
			return rec, nil
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("wait for job %s: %w", jobID, ctx.Err())
		case <-ticker.C:
			// next iteration
		}
	}
}

// ---------------------------------------------------------------------------
// Internal HTTP helpers
// ---------------------------------------------------------------------------

// doJSON performs an HTTP request with JSON encoding/decoding.
func (c *Client) doJSON(ctx context.Context, method, path string, body any, dest any) error {
	raw, err := c.doRaw(ctx, method, path, body)
	if err != nil {
		return err
	}
	if dest != nil {
		if err := json.Unmarshal(raw, dest); err != nil {
			return fmt.Errorf("decode response from %s %s: %w", method, path, err)
		}
	}
	return nil
}

// doRaw performs an HTTP request and returns the raw response body.
func (c *Client) doRaw(ctx context.Context, method, path string, body any) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encode request body for %s %s: %w", method, path, err)
		}
		bodyReader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request %s %s: %w", method, path, err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIToken != "" {
		req.Header.Set("X-API-Token", c.APIToken)
	}
	if c.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.BearerToken)
	}

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body from %s %s: %w", method, path, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       string(data),
		}
	}

	return data, nil
}

// ---------------------------------------------------------------------------
// APIError
// ---------------------------------------------------------------------------

// APIError is returned when the QPI server responds with a non-2xx status code.
type APIError struct {
	StatusCode int
	Status     string
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("qpi api error %s: %s", e.Status, e.Body)
}

// ListQpus lists all online QPUs.
func (c *Client) ListQpus(ctx context.Context) ([]QpuRecord, error) {
	var list []QpuRecord
	if err := c.doJSON(ctx, http.MethodGet, "/api/qpus", nil, &list); err != nil {
		return nil, fmt.Errorf("list qpus: %w", err)
	}
	return list, nil
}

// GetQpu retrieves a single QPU by name.
func (c *Client) GetQpu(ctx context.Context, name string) (*QpuRecord, error) {
	var qpu QpuRecord
	if err := c.doJSON(ctx, http.MethodGet, "/api/qpus/"+name, nil, &qpu); err != nil {
		return nil, fmt.Errorf("get qpu %s: %w", name, err)
	}
	return &qpu, nil
}

// RegisterQpu registers a new QPU (admin-only).
func (c *Client) RegisterQpu(ctx context.Context, req QpuRecord) (*QpuRecord, error) {
	var resp QpuRecord
	if err := c.doJSON(ctx, http.MethodPost, "/api/op/qpu/register", req, &resp); err != nil {
		return nil, fmt.Errorf("register qpu: %w", err)
	}
	return &resp, nil
}

// ToggleQpu toggles QPU driver state (admin-only).
func (c *Client) ToggleQpu(ctx context.Context, id string, enabled bool) (any, error) {
	var resp any
	payload := map[string]any{"id": id, "enabled": enabled}
	if err := c.doJSON(ctx, http.MethodPost, "/api/op/qpu/toggle", payload, &resp); err != nil {
		return nil, fmt.Errorf("toggle qpu %s: %w", id, err)
	}
	return resp, nil
}

// ListNotifications lists notifications visible to the authenticated user.
func (c *Client) ListNotifications(ctx context.Context) ([]NotificationRecord, error) {
	raw, err := c.doRaw(ctx, http.MethodGet, "/api/collections/notifications/records", nil)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	// PocketBase lists are wrapped in items
	var resp struct {
		Items []NotificationRecord `json:"items"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		// Try bare array
		var bare []NotificationRecord
		if err2 := json.Unmarshal(raw, &bare); err2 == nil {
			return bare, nil
		}
		return nil, fmt.Errorf("list notifications: decode: %w", err)
	}
	return resp.Items, nil
}

// DismissNotification dismisses a notification.
func (c *Client) DismissNotification(ctx context.Context, id string) (any, error) {
	var resp any
	if err := c.doJSON(ctx, http.MethodPost, "/api/notifications/"+id+"/dismiss", nil, &resp); err != nil {
		return nil, fmt.Errorf("dismiss notification %s: %w", id, err)
	}
	return resp, nil
}

// ListTimeSlots lists all booking slots.
func (c *Client) ListTimeSlots(ctx context.Context) ([]TimeSlotRecord, error) {
	var resp struct {
		Items []TimeSlotRecord `json:"items"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/collections/time_slots/records", nil, &resp); err != nil {
		return nil, fmt.Errorf("list time slots: %w", err)
	}
	return resp.Items, nil
}

// CreateTimeSlot creates a new booking slot.
func (c *Client) CreateTimeSlot(ctx context.Context, slot TimeSlotRecord) (*TimeSlotRecord, error) {
	var resp TimeSlotRecord
	if err := c.doJSON(ctx, http.MethodPost, "/api/collections/time_slots/records", slot, &resp); err != nil {
		return nil, fmt.Errorf("create time slot: %w", err)
	}
	return &resp, nil
}

// UpdateTimeSlot updates an existing booking slot.
func (c *Client) UpdateTimeSlot(ctx context.Context, id string, slot TimeSlotRecord) (*TimeSlotRecord, error) {
	var resp TimeSlotRecord
	if err := c.doJSON(ctx, http.MethodPatch, "/api/collections/time_slots/records/"+id, slot, &resp); err != nil {
		return nil, fmt.Errorf("update time slot %s: %w", id, err)
	}
	return &resp, nil
}

// DeleteTimeSlot deletes a booking slot.
func (c *Client) DeleteTimeSlot(ctx context.Context, id string) error {
	if _, err := c.doRaw(ctx, http.MethodDelete, "/api/collections/time_slots/records/"+id, nil); err != nil {
		return fmt.Errorf("delete time slot %s: %w", id, err)
	}
	return nil
}

// ListTimeRequests lists QPU time requests.
func (c *Client) ListTimeRequests(ctx context.Context) ([]TimeRequestRecord, error) {
	var resp struct {
		Items []TimeRequestRecord `json:"items"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/collections/qpu_time_requests/records", nil, &resp); err != nil {
		return nil, fmt.Errorf("list time requests: %w", err)
	}
	return resp.Items, nil
}

// CreateTimeRequest creates a new QPU time request.
func (c *Client) CreateTimeRequest(ctx context.Context, req TimeRequestRecord) (*TimeRequestRecord, error) {
	var resp TimeRequestRecord
	if err := c.doJSON(ctx, http.MethodPost, "/api/collections/qpu_time_requests/records", req, &resp); err != nil {
		return nil, fmt.Errorf("create time request: %w", err)
	}
	return &resp, nil
}

// UpdateTimeRequest updates/handles a QPU time request (admin-only).
func (c *Client) UpdateTimeRequest(ctx context.Context, id string, req TimeRequestRecord) (*TimeRequestRecord, error) {
	var resp TimeRequestRecord
	if err := c.doJSON(ctx, http.MethodPatch, "/api/collections/qpu_time_requests/records/"+id, req, &resp); err != nil {
		return nil, fmt.Errorf("update time request %s: %w", id, err)
	}
	return &resp, nil
}

// ListUsers lists all registered users (admin-only).
func (c *Client) ListUsers(ctx context.Context) ([]UserRecord, error) {
	var resp struct {
		Items []UserRecord `json:"items"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/collections/users/records", nil, &resp); err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return resp.Items, nil
}

// AllocateQpuTime allocates QPU time to a user (admin-only).
func (c *Client) AllocateQpuTime(ctx context.Context, userID string, seconds int) (*UserRecord, error) {
	var resp UserRecord
	payload := map[string]int{"qpu_seconds": seconds}
	if err := c.doJSON(ctx, http.MethodPatch, "/api/admin/users/"+userID, payload, &resp); err != nil {
		return nil, fmt.Errorf("allocate qpu time: %w", err)
	}
	return &resp, nil
}

// AuthWithPassword authenticates as a regular user using email/password.
func (c *Client) AuthWithPassword(ctx context.Context, identity, password string) (any, error) {
	var resp struct {
		Token  string `json:"token"`
		Record any    `json:"record"`
	}
	payload := map[string]string{"identity": identity, "password": password}
	if err := c.doJSON(ctx, http.MethodPost, "/api/collections/users/auth-with-password", payload, &resp); err != nil {
		return nil, fmt.Errorf("auth with password: %w", err)
	}
	if resp.Token != "" {
		c.BearerToken = resp.Token
	}
	return resp, nil
}
