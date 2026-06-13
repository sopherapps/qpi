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

// ---------------------------------------------------------------------------
// Client
// ---------------------------------------------------------------------------

// Client is an HTTP client for the QPI orchestrator API.
type Client struct {
	// BaseURL is the root URL of the orchestrator (e.g. "http://localhost:8090").
	BaseURL string
	// APIToken is sent via the X-API-Token header when non-empty.
	APIToken string
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
