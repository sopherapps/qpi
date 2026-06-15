// Package api manages HTTP REST endpoints and low-level NNG message channels
// for registering QPUs, dispatching jobs, and listening for job execution results.
package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"go.nanomsg.org/mangos/v3"
	"go.nanomsg.org/mangos/v3/protocol/pull"
	"go.nanomsg.org/mangos/v3/protocol/push"
	_ "go.nanomsg.org/mangos/v3/transport/tcp"

	"qpi/internal/config"
	"qpi/internal/scheduler"
)

var (
	// activeQPUs stores active cancel functions for goroutines bound to registered QPUs.
	activeQPUs   = make(map[string]context.CancelFunc)
	activeQPUsMu sync.Mutex
)

// connectRequest represents the JSON payload passed to /api/op/qpus/connect.
type connectRequest struct {
	Name         string         `json:"name"`
	AccessToken  string         `json:"access_token"`
	ExecutorType string         `json:"executor_type,omitempty"`
	DeviceConfig map[string]any `json:"device_config,omitempty"`
}

// connectResponse represents the JSON payload returned by /api/op/qpus/connect.
type connectResponse struct {
	Status         string `json:"status"`
	NNGCommandPort int    `json:"nng_command_port"`
	NNGResultPort  int    `json:"nng_result_port"`
	AuthToken      string `json:"auth_token"`
}

// resultPayload represents the NNG incoming message format for job execution results.
type resultPayload struct {
	JobID   string         `json:"job_id"`
	Results map[string]any `json:"results"`
}

// circuitPayload represents a single quantum circuit within a job submission.
type circuitPayload struct {
	Circuit         string      `json:"circuit"`
	ParameterValues [][]float64 `json:"parameter_values,omitempty"`
	Shots           *int        `json:"shots,omitempty"`
}

// jobSubmitRequest represents the JSON payload for POST /api/jobs.
type jobSubmitRequest struct {
	Circuits   []circuitPayload `json:"circuits"`
	Shots      int              `json:"shots"`
	MeasLevel  *int             `json:"meas_level,omitempty"`
	MeasReturn string           `json:"meas_return,omitempty"`
	QPUTarget  string           `json:"qpu_target,omitempty"`
}

// userUpdateRequest represents the JSON payload for PATCH /api/admin/users/{id}.
type userUpdateRequest struct {
	QpuSeconds *float64 `json:"qpu_seconds,omitempty"`
	APITokens  []string `json:"api_tokens,omitempty"`
}

// tokenCreateRequest represents the JSON payload for POST /api/tokens.
type tokenCreateRequest struct {
	Name      string `json:"name,omitempty"`
	ExpiresAt string `json:"expires_at,omitempty"` // ISO 8601 date string
}

// tokenCreateResponse represents the JSON payload returned by POST /api/tokens.
type tokenCreateResponse struct {
	ID        string `json:"id"`
	Token     string `json:"token"`
	Name      string `json:"name"`
	ExpiresAt string `json:"expires_at,omitempty"`
	Created   string `json:"created"`
}

// tokenUpdateRequest represents the JSON payload for PATCH /api/tokens/{id}.
type tokenUpdateRequest struct {
	Name      *string `json:"name,omitempty"`
	ExpiresAt *string `json:"expires_at,omitempty"`
}

// dismissRequest represents the JSON payload for POST /api/notifications/{id}/dismiss.
type dismissRequest struct {
	UserID string `json:"user_id,omitempty"`
}

// HashToken returns a SHA-256 hex digest of the raw token value.
func HashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

// generateAPIToken creates a new random API token string.
func generateAPIToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based entropy if crypto/rand fails
		for i := range b {
			b[i] = byte(time.Now().UnixNano() % 256)
			time.Sleep(1 * time.Nanosecond)
		}
	}
	return "qpi_" + hex.EncodeToString(b)
}

// RegisterRoutes sets up custom HTTP routes for QPU interactions.
func RegisterRoutes(e *core.ServeEvent) {
	e.Router.POST("/api/op/qpus/connect", func(re *core.RequestEvent) error {
		return handleQPUConnect(re)
	})
	e.Router.POST("/api/op/qpus/create", func(re *core.RequestEvent) error {
		return handleQPUCreate(re)
	})
	e.Router.POST("/api/op/qpu/toggle", func(re *core.RequestEvent) error {
		return handleQPUToggle(re)
	})

	// Job CRUD routes
	e.Router.POST("/api/jobs", func(re *core.RequestEvent) error {
		return handleJobSubmit(re)
	})
	e.Router.GET("/api/jobs", func(re *core.RequestEvent) error {
		return handleJobList(re)
	})
	e.Router.GET("/api/jobs/{id}", func(re *core.RequestEvent) error {
		return handleJobGet(re)
	})
	e.Router.POST("/api/jobs/{id}/cancel", func(re *core.RequestEvent) error {
		return handleJobCancel(re)
	})

	// Admin-only user management route
	e.Router.PATCH("/api/admin/users/{id}", func(re *core.RequestEvent) error {
		return handleUserUpdate(re)
	})

	// QPU discovery routes (public — no auth required)
	e.Router.GET("/api/qpus", func(re *core.RequestEvent) error {
		return handleQPUList(re)
	})
	e.Router.GET("/api/qpus/{name}", func(re *core.RequestEvent) error {
		return handleQPUGet(re)
	})

	// API token CRUD routes (owner-only)
	e.Router.POST("/api/tokens", func(re *core.RequestEvent) error {
		return handleTokenCreate(re)
	})
	e.Router.GET("/api/tokens", func(re *core.RequestEvent) error {
		return handleTokenList(re)
	})
	e.Router.GET("/api/tokens/{id}", func(re *core.RequestEvent) error {
		return handleTokenGet(re)
	})
	e.Router.PATCH("/api/tokens/{id}", func(re *core.RequestEvent) error {
		return handleTokenUpdate(re)
	})
	e.Router.DELETE("/api/tokens/{id}", func(re *core.RequestEvent) error {
		return handleTokenDelete(re)
	})

	// Notification dismiss route (authenticated users only)
	e.Router.POST("/api/notifications/{id}/dismiss", func(re *core.RequestEvent) error {
		return handleNotificationDismiss(re)
	})
}

// resolveUserAuth resolves the authenticated user from the request.
// It checks re.Auth first, then falls back to API token auth via
// X-API-Token header or Authorization: Bearer <token> header.
// API tokens are stored in the api_tokens collection with a user relation.
func resolveUserAuth(re *core.RequestEvent) (*core.Record, error) {
	// 1. Check if already authenticated via PocketBase session/JWT
	if re.Auth != nil && re.Auth.Collection().Name == "users" {
		return re.Auth, nil
	}

	// 2. Check for API token in headers
	var token string
	if t := re.Request.Header.Get("X-API-Token"); t != "" {
		token = t
	} else if authHeader := re.Request.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
		token = strings.TrimPrefix(authHeader, "Bearer ")
	}

	if token != "" {
		cfg, err := config.GetConfigFromApp(re.App)
		if err != nil {
			return nil, err
		}
		// Look up hashed token in the api_tokens collection and expand the user relation
		tokenRec, err := re.App.FindFirstRecordByFilter(
			cfg.CollectionAPITokens,
			"token = {:token} && (expires_at = '' || expires_at >= {:now})",
			dbx.Params{"token": HashToken(token), "now": time.Now().UTC().Format("2006-01-02 15:04:05.000Z")},
		)
		if err == nil {
			userID := tokenRec.GetString("user")
			if userID != "" {
				user, err := re.App.FindRecordById("users", userID)
				if err == nil {
					return user, nil
				}
			}
		}
	}

	return nil, errors.New("authentication required")
}

// handleJobSubmit handles POST /api/jobs — creates a new quantum job.
func handleJobSubmit(re *core.RequestEvent) error {
	cfg, err := config.GetConfigFromApp(re.App)
	if err != nil {
		return re.Error(http.StatusInternalServerError, "failed to retrieve configuration", err)
	}

	user, err := resolveUserAuth(re)
	if err != nil {
		return re.Error(http.StatusUnauthorized, "authentication required", err)
	}

	// Check QPU seconds balance
	qpuSeconds := user.GetFloat("qpu_seconds")
	if qpuSeconds <= 0 {
		return re.Error(http.StatusForbidden, "insufficient QPU seconds balance", nil)
	}

	// Parse and validate request body
	var req jobSubmitRequest
	if err := re.BindBody(&req); err != nil {
		return re.Error(http.StatusBadRequest, "invalid request body", err)
	}

	if len(req.Circuits) == 0 {
		return re.Error(http.StatusBadRequest, "at least one circuit is required", nil)
	}
	for _, c := range req.Circuits {
		if c.Circuit == "" {
			return re.Error(http.StatusBadRequest, "circuit string must not be empty", nil)
		}
	}

	// Set defaults
	if req.Shots == 0 {
		req.Shots = 1024
	}
	if req.MeasLevel == nil {
		defaultMeasLevel := 2
		req.MeasLevel = &defaultMeasLevel
	}
	if req.MeasReturn == "" {
		req.MeasReturn = "single"
	}

	// Resolve QPU target
	qpuTargetID := req.QPUTarget
	if qpuTargetID == "" {
		// Find first online QPU
		qpus, err := re.App.FindRecordsByFilter(
			cfg.CollectionQPUs,
			"status = 'online'",
			"+created", 1, 0,
		)
		if err != nil || len(qpus) == 0 {
			return re.Error(http.StatusServiceUnavailable, "no online QPU available", nil)
		}
		qpuTargetID = qpus[0].Id
	}

	// Create the quantum job record
	jobsCol, err := re.App.FindCollectionByNameOrId(cfg.CollectionQuantumJobs)
	if err != nil {
		return re.Error(http.StatusInternalServerError, "jobs collection not found", err)
	}

	payloadJSON, err := json.Marshal(req)
	if err != nil {
		return re.Error(http.StatusInternalServerError, "failed to marshal payload", err)
	}

	record := core.NewRecord(jobsCol)
	record.Set("user_id", user.Id)
	record.Set("qpu_target", qpuTargetID)
	record.Set("duration", 0)
	record.Set("status", "pending")
	record.Set("payload", string(payloadJSON))

	if err := re.App.Save(record); err != nil {
		return re.Error(http.StatusInternalServerError, "failed to create job", err)
	}

	return re.JSON(http.StatusCreated, map[string]string{"job_id": record.Id})
}

// handleJobList handles GET /api/jobs — lists jobs for the authenticated user.
func handleJobList(re *core.RequestEvent) error {
	cfg, err := config.GetConfigFromApp(re.App)
	if err != nil {
		return re.Error(http.StatusInternalServerError, "failed to retrieve configuration", err)
	}

	user, err := resolveUserAuth(re)
	if err != nil {
		return re.Error(http.StatusUnauthorized, "authentication required", err)
	}

	records, err := re.App.FindRecordsByFilter(
		cfg.CollectionQuantumJobs,
		"user_id = {:userId}",
		"-created", 0, 0,
		dbx.Params{"userId": user.Id},
	)
	if err != nil {
		return re.Error(http.StatusInternalServerError, "failed to query jobs", err)
	}

	// Serialize records to a simple JSON array
	jobs := make([]map[string]any, 0, len(records))
	for _, r := range records {
		jobs = append(jobs, map[string]any{
			"id":          r.Id,
			"user_id":     r.GetString("user_id"),
			"qpu_target":  r.GetString("qpu_target"),
			"status":      r.GetString("status"),
			"payload":     r.Get("payload"),
			"results":     r.Get("results"),
			"finished_at": r.GetString("finished_at"),
			"created":     r.GetString("created"),
			"updated":     r.GetString("updated"),
		})
	}

	return re.JSON(http.StatusOK, jobs)
}

// handleJobGet handles GET /api/jobs/{id} — retrieves a single job by ID.
func handleJobGet(re *core.RequestEvent) error {
	cfg, err := config.GetConfigFromApp(re.App)
	if err != nil {
		return re.Error(http.StatusInternalServerError, "failed to retrieve configuration", err)
	}

	user, err := resolveUserAuth(re)
	if err != nil {
		return re.Error(http.StatusUnauthorized, "authentication required", err)
	}

	jobID := re.Request.PathValue("id")
	record, err := re.App.FindRecordById(cfg.CollectionQuantumJobs, jobID)
	if err != nil {
		return re.Error(http.StatusNotFound, "job not found", err)
	}

	if record.GetString("user_id") != user.Id {
		return re.Error(http.StatusForbidden, "access denied", nil)
	}

	return re.JSON(http.StatusOK, map[string]any{
		"id":          record.Id,
		"user_id":     record.GetString("user_id"),
		"qpu_target":  record.GetString("qpu_target"),
		"status":      record.GetString("status"),
		"payload":     record.Get("payload"),
		"results":     record.Get("results"),
		"duration":    record.Get("duration"),
		"finished_at": record.GetString("finished_at"),
		"created":     record.GetString("created"),
		"updated":     record.GetString("updated"),
	})
}

// handleJobCancel handles POST /api/jobs/{id}/cancel — cancels a pending or running job.
func handleJobCancel(re *core.RequestEvent) error {
	cfg, err := config.GetConfigFromApp(re.App)
	if err != nil {
		return re.Error(http.StatusInternalServerError, "failed to retrieve configuration", err)
	}

	user, err := resolveUserAuth(re)
	if err != nil {
		return re.Error(http.StatusUnauthorized, "authentication required", err)
	}

	jobID := re.Request.PathValue("id")
	record, err := re.App.FindRecordById(cfg.CollectionQuantumJobs, jobID)
	if err != nil {
		return re.Error(http.StatusNotFound, "job not found", err)
	}

	if record.GetString("user_id") != user.Id {
		return re.Error(http.StatusForbidden, "access denied", nil)
	}

	status := record.GetString("status")
	switch status {
	case "pending", "running":
		record.Set("status", "cancelled")
		if err := re.App.Save(record); err != nil {
			return re.Error(http.StatusInternalServerError, "failed to cancel job", err)
		}
		return re.JSON(http.StatusOK, map[string]string{"status": "cancelled"})
	case "completed", "failed", "cancelled":
		return re.Error(http.StatusBadRequest, fmt.Sprintf("job is already %s", status), nil)
	default:
		return re.Error(http.StatusBadRequest, fmt.Sprintf("unexpected job status: %s", status), nil)
	}
}

// handleUserUpdate handles PATCH /api/admin/users/{id} — admin-only endpoint to update user fields.
// api_tokens are stored in a separate collection; this endpoint syncs them idempotently.
func handleUserUpdate(re *core.RequestEvent) error {
	cfg, err := config.GetConfigFromApp(re.App)
	if err != nil {
		return re.Error(http.StatusInternalServerError, "failed to retrieve configuration", err)
	}

	// Only superusers (admins) can access this endpoint
	if !re.HasSuperuserAuth() {
		return re.Error(http.StatusForbidden, "admin access required", nil)
	}

	userID := re.Request.PathValue("id")
	record, err := re.App.FindRecordById("users", userID)
	if err != nil {
		return re.Error(http.StatusNotFound, "user not found", err)
	}

	var req userUpdateRequest
	if err := re.BindBody(&req); err != nil {
		return re.Error(http.StatusBadRequest, "invalid request body", err)
	}

	if req.QpuSeconds != nil {
		record.Set("qpu_seconds", *req.QpuSeconds)
	}

	// Sync api_tokens to the separate collection (append-only; does NOT delete existing)
	if req.APITokens != nil {
		tokensCol, err := re.App.FindCollectionByNameOrId(cfg.CollectionAPITokens)
		if err != nil {
			return re.Error(http.StatusInternalServerError, "api_tokens collection not found", err)
		}

		for _, tokenValue := range req.APITokens {
			// Skip if this token already exists for the user
			_, err := re.App.FindFirstRecordByFilter(
				cfg.CollectionAPITokens,
				"token = {:token} && user = {:userId}",
				dbx.Params{"token": HashToken(tokenValue), "userId": userID},
			)
			if err == nil {
				continue // already exists
			}
			tokenRec := core.NewRecord(tokensCol)
			tokenRec.Set("token", HashToken(tokenValue))
			tokenRec.Set("user", userID)
			if err := re.App.Save(tokenRec); err != nil {
				log.Printf("Warning: failed to create api_token record: %v", err)
			}
		}
	}

	if err := re.App.Save(record); err != nil {
		return re.Error(http.StatusInternalServerError, "failed to update user", err)
	}

	// Return current token metadata for the user (hashes are never exposed)
	tokenRecs, _ := re.App.FindRecordsByFilter(
		cfg.CollectionAPITokens,
		"user = {:userId}",
		"", 0, 0,
		dbx.Params{"userId": userID},
	)
	tokenMeta := make([]map[string]any, 0, len(tokenRecs))
	for _, t := range tokenRecs {
		tokenMeta = append(tokenMeta, map[string]any{
			"id":         t.Id,
			"name":       t.GetString("name"),
			"expires_at": t.GetString("expires_at"),
			"created":    t.GetString("created"),
		})
	}

	return re.JSON(http.StatusOK, map[string]any{
		"id":          record.Id,
		"qpu_seconds": record.GetFloat("qpu_seconds"),
		"api_tokens":  tokenMeta,
	})
}

// handleTokenCreate handles POST /api/tokens — creates a new API token for the authenticated user.
// Returns the raw token exactly once; only the hash is stored in the database.
func handleTokenCreate(re *core.RequestEvent) error {
	cfg, err := config.GetConfigFromApp(re.App)
	if err != nil {
		return re.Error(http.StatusInternalServerError, "failed to retrieve configuration", err)
	}

	user, err := resolveUserAuth(re)
	if err != nil {
		return re.Error(http.StatusUnauthorized, "authentication required", err)
	}

	var req tokenCreateRequest
	if err := re.BindBody(&req); err != nil {
		return re.Error(http.StatusBadRequest, "invalid request body", err)
	}

	tokensCol, err := re.App.FindCollectionByNameOrId(cfg.CollectionAPITokens)
	if err != nil {
		return re.Error(http.StatusInternalServerError, "api_tokens collection not found", err)
	}

	rawToken := generateAPIToken()
	tokenRec := core.NewRecord(tokensCol)
	tokenRec.Set("token", HashToken(rawToken))
	tokenRec.Set("user", user.Id)
	if req.Name != "" {
		tokenRec.Set("name", req.Name)
	}
	if req.ExpiresAt != "" {
		tokenRec.Set("expires_at", req.ExpiresAt)
	}

	if err := re.App.Save(tokenRec); err != nil {
		return re.Error(http.StatusInternalServerError, "failed to create token", err)
	}

	return re.JSON(http.StatusCreated, tokenCreateResponse{
		ID:        tokenRec.Id,
		Token:     rawToken,
		Name:      tokenRec.GetString("name"),
		ExpiresAt: tokenRec.GetString("expires_at"),
		Created:   tokenRec.GetString("created"),
	})
}

// handleTokenList handles GET /api/tokens — lists metadata for tokens belonging to the authenticated user.
// Never exposes the raw token hash.
func handleTokenList(re *core.RequestEvent) error {
	cfg, err := config.GetConfigFromApp(re.App)
	if err != nil {
		return re.Error(http.StatusInternalServerError, "failed to retrieve configuration", err)
	}

	user, err := resolveUserAuth(re)
	if err != nil {
		return re.Error(http.StatusUnauthorized, "authentication required", err)
	}

	records, err := re.App.FindRecordsByFilter(
		cfg.CollectionAPITokens,
		"user = {:userId}",
		"-created", 0, 0,
		dbx.Params{"userId": user.Id},
	)
	if err != nil {
		return re.Error(http.StatusInternalServerError, "failed to query tokens", err)
	}

	tokens := make([]map[string]any, 0, len(records))
	for _, r := range records {
		tokens = append(tokens, map[string]any{
			"id":         r.Id,
			"name":       r.GetString("name"),
			"expires_at": r.GetString("expires_at"),
			"created":    r.GetString("created"),
			"updated":    r.GetString("updated"),
		})
	}
	return re.JSON(http.StatusOK, tokens)
}

// handleTokenGet handles GET /api/tokens/{id} — retrieves metadata for a single token.
// Only the owner (or a superuser) may access it.
func handleTokenGet(re *core.RequestEvent) error {
	cfg, err := config.GetConfigFromApp(re.App)
	if err != nil {
		return re.Error(http.StatusInternalServerError, "failed to retrieve configuration", err)
	}

	user, err := resolveUserAuth(re)
	if err != nil {
		return re.Error(http.StatusUnauthorized, "authentication required", err)
	}

	tokenID := re.Request.PathValue("id")
	record, err := re.App.FindRecordById(cfg.CollectionAPITokens, tokenID)
	if err != nil {
		return re.Error(http.StatusNotFound, "token not found", err)
	}

	if record.GetString("user") != user.Id && !re.HasSuperuserAuth() {
		return re.Error(http.StatusForbidden, "access denied", nil)
	}

	return re.JSON(http.StatusOK, map[string]any{
		"id":         record.Id,
		"name":       record.GetString("name"),
		"expires_at": record.GetString("expires_at"),
		"created":    record.GetString("created"),
		"updated":    record.GetString("updated"),
	})
}

// handleTokenUpdate handles PATCH /api/tokens/{id} — updates name or expiry for a token.
// Only the owner (or a superuser) may update it.
func handleTokenUpdate(re *core.RequestEvent) error {
	cfg, err := config.GetConfigFromApp(re.App)
	if err != nil {
		return re.Error(http.StatusInternalServerError, "failed to retrieve configuration", err)
	}

	user, err := resolveUserAuth(re)
	if err != nil {
		return re.Error(http.StatusUnauthorized, "authentication required", err)
	}

	tokenID := re.Request.PathValue("id")
	record, err := re.App.FindRecordById(cfg.CollectionAPITokens, tokenID)
	if err != nil {
		return re.Error(http.StatusNotFound, "token not found", err)
	}

	if record.GetString("user") != user.Id && !re.HasSuperuserAuth() {
		return re.Error(http.StatusForbidden, "access denied", nil)
	}

	var req tokenUpdateRequest
	if err := re.BindBody(&req); err != nil {
		return re.Error(http.StatusBadRequest, "invalid request body", err)
	}

	if req.Name != nil {
		record.Set("name", *req.Name)
	}
	if req.ExpiresAt != nil {
		record.Set("expires_at", *req.ExpiresAt)
	}

	if err := re.App.Save(record); err != nil {
		return re.Error(http.StatusInternalServerError, "failed to update token", err)
	}

	return re.JSON(http.StatusOK, map[string]any{
		"id":         record.Id,
		"name":       record.GetString("name"),
		"expires_at": record.GetString("expires_at"),
		"created":    record.GetString("created"),
		"updated":    record.GetString("updated"),
	})
}

// handleNotificationDismiss handles POST /api/notifications/{id}/dismiss —
// adds the authenticated user to the notification's dismissed_by relation.
func handleNotificationDismiss(re *core.RequestEvent) error {
	cfg, err := config.GetConfigFromApp(re.App)
	if err != nil {
		return re.Error(http.StatusInternalServerError, "failed to retrieve configuration", err)
	}

	user, err := resolveUserAuth(re)
	if err != nil {
		return re.Error(http.StatusUnauthorized, "authentication required", err)
	}

	notifID := re.Request.PathValue("id")
	record, err := re.App.FindRecordById(cfg.CollectionNotifications, notifID)
	if err != nil {
		return re.Error(http.StatusNotFound, "notification not found", err)
	}

	// Verify the user is allowed to see this notification (target_users empty or includes user)
	targetUsers := record.GetStringSlice("target_users")
	if len(targetUsers) > 0 {
		found := false
		for _, uid := range targetUsers {
			if uid == user.Id {
				found = true
				break
			}
		}
		if !found {
			return re.Error(http.StatusForbidden, "access denied", nil)
		}
	}

	// Add user to dismissed_by if not already present
	dismissedBy := record.GetStringSlice("dismissed_by")
	alreadyDismissed := false
	for _, uid := range dismissedBy {
		if uid == user.Id {
			alreadyDismissed = true
			break
		}
	}
	if !alreadyDismissed {
		dismissedBy = append(dismissedBy, user.Id)
		record.Set("dismissed_by", dismissedBy)
		if err := re.App.Save(record); err != nil {
			return re.Error(http.StatusInternalServerError, "failed to dismiss notification", err)
		}
	}

	return re.JSON(http.StatusOK, map[string]string{"status": "dismissed"})
}

// handleTokenDelete handles DELETE /api/tokens/{id} — removes a token.
// Only the owner (or a superuser) may delete it.
func handleTokenDelete(re *core.RequestEvent) error {
	cfg, err := config.GetConfigFromApp(re.App)
	if err != nil {
		return re.Error(http.StatusInternalServerError, "failed to retrieve configuration", err)
	}

	user, err := resolveUserAuth(re)
	if err != nil {
		return re.Error(http.StatusUnauthorized, "authentication required", err)
	}

	tokenID := re.Request.PathValue("id")
	record, err := re.App.FindRecordById(cfg.CollectionAPITokens, tokenID)
	if err != nil {
		return re.Error(http.StatusNotFound, "token not found", err)
	}

	if record.GetString("user") != user.Id && !re.HasSuperuserAuth() {
		return re.Error(http.StatusForbidden, "access denied", nil)
	}

	if err := re.App.Delete(record); err != nil {
		return re.Error(http.StatusInternalServerError, "failed to delete token", err)
	}

	return re.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

// handleQPUCreate creates a new QPU record (admin-only), generating a random
// access token and returning it in the response.  The hashed token is stored.
// POST /api/op/qpus/create
func handleQPUCreate(re *core.RequestEvent) error {
	cfg, err := config.GetConfigFromApp(re.App)
	if err != nil {
		return re.Error(http.StatusInternalServerError, "failed to retrieve configuration", err)
	}

	if !re.HasSuperuserAuth() {
		return re.Error(http.StatusForbidden, "admin access required", nil)
	}

	var req struct {
		Name         string `json:"name"`
		ExecutorType string `json:"executor_type,omitempty"`
		NumQubits    int    `json:"num_qubits,omitempty"`
		Enabled      *bool  `json:"enabled,omitempty"`
	}
	if err := re.BindBody(&req); err != nil {
		return re.Error(http.StatusBadRequest, "invalid request body", err)
	}
	if req.Name == "" {
		return re.Error(http.StatusBadRequest, "name is required", nil)
	}

	qpuCol, err := re.App.FindCollectionByNameOrId(cfg.CollectionQPUs)
	if err != nil {
		return re.Error(http.StatusInternalServerError, "QPUs collection not found", err)
	}

	// Generate a random access token (raw token is returned once, hash is stored)
	rawToken := generateAPIToken()

	record := core.NewRecord(qpuCol)
	record.Set("name", req.Name)
	record.Set("access_token", HashToken(rawToken))
	record.Set("status", "offline")
	if req.ExecutorType != "" {
		record.Set("executor_type", req.ExecutorType)
	}
	if req.NumQubits > 0 {
		record.Set("num_qubits", req.NumQubits)
	}
	if req.Enabled != nil {
		record.Set("enabled", *req.Enabled)
	} else {
		record.Set("enabled", true)
	}

	if err := re.App.Save(record); err != nil {
		return re.Error(http.StatusInternalServerError, "failed to create QPU", err)
	}

	return re.JSON(http.StatusCreated, map[string]any{
		"id":            record.Id,
		"name":          record.GetString("name"),
		"access_token":  rawToken,
		"executor_type": record.GetString("executor_type"),
		"status":        record.GetString("status"),
		"enabled":       record.GetBool("enabled"),
	})
}

// handleQPUConnect connects a hardware driver node, allocating dynamic command/result ports
// and starting parallel dispatcher and result listener routines.
// POST /api/op/qpus/connect
func handleQPUConnect(re *core.RequestEvent) error {
	cfg, err := config.GetConfigFromApp(re.App)
	if err != nil {
		return re.Error(http.StatusInternalServerError, "failed to retrieve custom configuration", err)
	}

	var req connectRequest
	if err := re.BindBody(&req); err != nil {
		return re.Error(http.StatusBadRequest, "invalid request body", err)
	}

	// Find QPU by access token (hashed)
	hashedToken := HashToken(req.AccessToken)
	qpu, err := re.App.FindFirstRecordByData(cfg.CollectionQPUs, "access_token", hashedToken)
	if err != nil {
		return re.Error(http.StatusUnauthorized, "invalid access token", err)
	}

	// Check if QPU is enabled
	if qpu.Get("enabled") != nil && !qpu.GetBool("enabled") {
		return re.Error(http.StatusForbidden, "QPU is currently disabled by administrator", nil)
	}

	// Update name if provided
	if req.Name != "" {
		qpu.Set("name", req.Name)
	}

	// Allocate ports if not already done
	cmdPort := qpu.GetInt("nng_command_port")
	resPort := qpu.GetInt("nng_result_port")

	if cmdPort == 0 || resPort == 0 {
		ports, err := findFreePorts(re.App, 2)
		if err != nil {
			return re.Error(http.StatusInternalServerError, "cannot allocate NNG ports", err)
		}
		cmdPort = ports[0]
		resPort = ports[1]
		qpu.Set("nng_command_port", cmdPort)
		qpu.Set("nng_result_port", resPort)
	}

	qpu.Set("status", "online")
	if req.ExecutorType != "" {
		qpu.Set("executor_type", req.ExecutorType)
	}
	if req.DeviceConfig != nil {
		deviceJSON, _ := json.Marshal(req.DeviceConfig)
		qpu.Set("device_config", string(deviceJSON))
	}
	if err := re.App.Save(qpu); err != nil {
		return re.Error(http.StatusInternalServerError, "cannot save QPU record", err)
	}

	// Spin up orchestration goroutines if not already running
	StartQPUDistribution(re.App, qpu.Id, cmdPort, resPort)

	// Generate auth token for the QPU record
	token, err := qpu.NewStaticAuthToken(0) // 0 = use app default duration
	if err != nil {
		// Non-fatal: fall back to the access token as a simple identifier
		token = req.AccessToken
	}

	return re.JSON(http.StatusOK, connectResponse{
		Status:         "success",
		NNGCommandPort: cmdPort,
		NNGResultPort:  resPort,
		AuthToken:      token,
	})
}

type qpuToggleRequest struct {
	ID      string `json:"id"`
	Enabled bool   `json:"enabled"`
}

// handleQPUToggle handles POST /api/op/qpu/toggle — toggles a QPU's enabled status (admin-only).
func handleQPUToggle(re *core.RequestEvent) error {
	cfg, err := config.GetConfigFromApp(re.App)
	if err != nil {
		return re.Error(http.StatusInternalServerError, "failed to retrieve configuration", err)
	}

	// Only superusers (admins) can access this endpoint
	if !re.HasSuperuserAuth() {
		return re.Error(http.StatusForbidden, "admin access required", nil)
	}

	var req qpuToggleRequest
	if err := re.BindBody(&req); err != nil {
		return re.Error(http.StatusBadRequest, "invalid request body", err)
	}

	if req.ID == "" {
		return re.Error(http.StatusBadRequest, "QPU id is required", nil)
	}

	// Find QPU by ID or name
	qpu, err := re.App.FindRecordById(cfg.CollectionQPUs, req.ID)
	if err != nil {
		qpu, err = re.App.FindFirstRecordByData(cfg.CollectionQPUs, "name", req.ID)
		if err != nil {
			return re.Error(http.StatusNotFound, "QPU not found", err)
		}
	}

	qpu.Set("enabled", req.Enabled)
	if err := re.App.Save(qpu); err != nil {
		return re.Error(http.StatusInternalServerError, "failed to update QPU status", err)
	}

	return re.JSON(http.StatusOK, map[string]any{
		"id":      qpu.Id,
		"name":    qpu.GetString("name"),
		"enabled": qpu.GetBool("enabled"),
		"status":  qpu.GetString("status"),
	})
}

// handleQPUList handles GET /api/qpus — lists all online QPUs.
func handleQPUList(re *core.RequestEvent) error {
	cfg, err := config.GetConfigFromApp(re.App)
	if err != nil {
		return re.Error(http.StatusInternalServerError, "failed to retrieve configuration", err)
	}

	records, err := re.App.FindRecordsByFilter(
		cfg.CollectionQPUs,
		"status = 'online'",
		"+created", 0, 0,
	)
	if err != nil {
		return re.Error(http.StatusInternalServerError, "failed to query QPUs", err)
	}

	qpus := make([]map[string]any, 0, len(records))
	for _, r := range records {
		qpus = append(qpus, serializeQPU(r))
	}
	return re.JSON(http.StatusOK, qpus)
}

// handleQPUGet handles GET /api/qpus/{name} — retrieves a single QPU by name.
// Since the QPUs collection uses "name" as its primary key, we can look up
// directly by record ID.
func handleQPUGet(re *core.RequestEvent) error {
	cfg, err := config.GetConfigFromApp(re.App)
	if err != nil {
		return re.Error(http.StatusInternalServerError, "failed to retrieve configuration", err)
	}

	name := re.Request.PathValue("name")
	record, err := re.App.FindRecordById(cfg.CollectionQPUs, name)
	if err != nil {
		return re.Error(http.StatusNotFound, "QPU not found", err)
	}

	return re.JSON(http.StatusOK, serializeQPU(record))
}

// serializeQPU converts a QPU record to a public JSON representation.
func serializeQPU(r *core.Record) map[string]any {
	deviceConfig := r.Get("device_config")
	if deviceConfig == nil {
		deviceConfig = map[string]any{}
	}
	return map[string]any{
		"id":               r.Id,
		"name":             r.GetString("name"),
		"status":           r.GetString("status"),
		"num_qubits":       r.GetInt("num_qubits"),
		"executor_type":    r.GetString("executor_type"),
		"device_config":    deviceConfig,
		"nng_command_port": r.GetInt("nng_command_port"),
		"nng_result_port":  r.GetInt("nng_result_port"),
		"created":          r.GetString("created"),
		"updated":          r.GetString("updated"),
	}
}

// StartQPUDistribution starts the goroutines for a specific QPU if not already running.
func StartQPUDistribution(app core.App, qpuID string, cmdPort, resPort int) {
	activeQPUsMu.Lock()
	defer activeQPUsMu.Unlock()
	if _, running := activeQPUs[qpuID]; !running {
		ctx, cancel := context.WithCancel(context.Background())
		activeQPUs[qpuID] = cancel
		go runDispatcher(ctx, app, qpuID, cmdPort)
		go runResultListener(ctx, app, qpuID, resPort)
		log.Printf("[QPi] Goroutines started for QPU %s (cmd:%d res:%d)", qpuID, cmdPort, resPort)
	}
}

// StopQPUDistribution cancels the goroutines for a specific QPU.
func StopQPUDistribution(qpuID string) {
	activeQPUsMu.Lock()
	defer activeQPUsMu.Unlock()
	if cancel, exists := activeQPUs[qpuID]; exists {
		cancel()
		delete(activeQPUs, qpuID)
		log.Printf("[QPi] Goroutines stopped for QPU %s", qpuID)
	}
}

// runDispatcher starts an NNG PUSH socket on the cmdPort, polling for pending quantum jobs
// from the scheduler and pushing them to the registered python driver node.
func runDispatcher(ctx context.Context, app core.App, qpuID string, cmdPort int) {
	cfg, err := config.GetConfigFromApp(app)
	if err != nil {
		log.Printf("[Dispatcher %s] failed to get config: %v", qpuID, err)
		return
	}

	addr := fmt.Sprintf("tcp://0.0.0.0:%d", cmdPort)
	sock, err := push.NewSocket()
	if err != nil {
		log.Printf("[Dispatcher %s] socket error: %v", qpuID, err)
		return
	}
	defer sock.Close()

	if err := sock.Listen(addr); err != nil {
		log.Printf("[Dispatcher %s] listen error on %s: %v", qpuID, addr, err)
		return
	}
	log.Printf("[Dispatcher %s] PUSH listening on %s", qpuID, addr)

	go func() {
		<-ctx.Done()
		sock.Close()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		job := scheduler.FetchNextJob(app, qpuID)
		if job == nil {
			select {
			case <-ctx.Done():
				return
			case <-time.After(cfg.DispatchPollInterval):
			}
			continue
		}

		payload, _ := json.Marshal(map[string]any{
			"job_id":  job.Id,
			"payload": job.Get("payload"),
		})

		if err := sock.Send(payload); err != nil {
			select {
			case <-ctx.Done():
				return
			default:
			}
			log.Printf("[Dispatcher %s] send error: %v — requeueing", qpuID, err)
			job.Set("status", "pending")
			_ = app.Save(job)
			select {
			case <-ctx.Done():
				return
			case <-time.After(cfg.DispatchPollInterval):
			}
			continue
		}

		job.Set("status", "running")
		if err := app.Save(job); err != nil {
			log.Printf("[Dispatcher %s] DB update error: %v", qpuID, err)
		} else {
			log.Printf("[Dispatcher %s] dispatched job %s", qpuID, job.Id)
		}
	}
}

// runResultListener starts an NNG PULL socket on the resPort, waiting for job execution
// results sent back by the hardware driver node and saving them to the database.
func runResultListener(ctx context.Context, app core.App, qpuID string, resPort int) {
	cfg, err := config.GetConfigFromApp(app)
	if err != nil {
		log.Printf("[Listener %s] failed to get config: %v", qpuID, err)
		return
	}

	addr := fmt.Sprintf("tcp://0.0.0.0:%d", resPort)
	sock, err := pull.NewSocket()
	if err != nil {
		log.Printf("[Listener %s] socket error: %v", qpuID, err)
		return
	}
	defer sock.Close()

	if err := sock.Listen(addr); err != nil {
		log.Printf("[Listener %s] listen error on %s: %v", qpuID, addr, err)
		return
	}
	log.Printf("[Listener %s] PULL listening on %s", qpuID, addr)

	go func() {
		<-ctx.Done()
		sock.Close()
	}()

	for {
		msg, err := sock.Recv()
		if err != nil {
			if err == mangos.ErrClosed {
				return
			}
			select {
			case <-ctx.Done():
				return
			default:
			}
			log.Printf("[Listener %s] recv error: %v", qpuID, err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(cfg.DispatchPollInterval):
			}
			continue
		}

		var result resultPayload
		if err := json.Unmarshal(msg, &result); err != nil {
			log.Printf("[Listener %s] JSON parse error: %v", qpuID, err)
			continue
		}

		job, err := app.FindRecordById(cfg.CollectionQuantumJobs, result.JobID)
		if err != nil {
			log.Printf("[Listener %s] job %s not found: %v", qpuID, result.JobID, err)
			continue
		}

		// Calculate execution duration from last update timestamp
		executionDuration := time.Since(job.GetDateTime("updated").Time())
		durationSeconds := executionDuration.Seconds()

		// Deduct QPU seconds from the user's balance
		userID := job.GetString("user_id")
		if userID != "" {
			userRecord, userErr := app.FindRecordById("users", userID)
			if userErr == nil {
				userRecord.Set("qpu_seconds-", durationSeconds)
				if saveErr := app.Save(userRecord); saveErr != nil {
					log.Printf("[Listener %s] failed to deduct QPU seconds for user %s: %v", qpuID, userID, saveErr)
				}
			} else {
				log.Printf("[Listener %s] user %s not found for QPU seconds deduction: %v", qpuID, userID, userErr)
			}
		}

		// Determine final job status based on result contents
		finalStatus := "completed"
		if _, hasError := result.Results["error"]; hasError {
			finalStatus = "failed"
		}

		job.Set("status", finalStatus)
		job.Set("finished_at", time.Now().UTC().Format("2006-01-02 15:04:05.000Z"))
		resultsJSON, _ := json.Marshal(result.Results)
		job.Set("results", string(resultsJSON))
		job.Set("duration", durationSeconds)

		if err := app.Save(job); err != nil {
			log.Printf("[Listener %s] DB save error for job %s: %v", qpuID, result.JobID, err)
		} else {
			log.Printf("[Listener %s] job %s %s", qpuID, result.JobID, finalStatus)
		}
	}
}

// findFreePorts searches for free TCP ports within the configuration range,
// excluding ports currently reserved/allocated in the QPUs database table.
func findFreePorts(app core.App, count int) ([]int, error) {
	cfg, err := config.GetConfigFromApp(app)
	if err != nil {
		return nil, err
	}

	allocated := make(map[int]bool)
	filter := fmt.Sprintf("nng_command_port >= %d || nng_result_port >= %d", cfg.PortRangeStart, cfg.PortRangeStart)
	records, err := app.FindRecordsByFilter(cfg.CollectionQPUs, filter, "", 0, 0)
	if err == nil {
		for _, r := range records {
			cmd := r.GetInt("nng_command_port")
			res := r.GetInt("nng_result_port")
			if cmd > 0 {
				allocated[cmd] = true
			}
			if res > 0 {
				allocated[res] = true
			}
		}
	}

	var ports []int
	for port := cfg.PortRangeStart; port < cfg.PortRangeEnd; port++ {
		if allocated[port] {
			continue
		}
		addr := fmt.Sprintf(":%d", port)
		ln, err := net.Listen("tcp", addr)
		if err == nil {
			ln.Close()
			ports = append(ports, port)
			allocated[port] = true
			if len(ports) == count {
				return ports, nil
			}
		}
	}
	return nil, fmt.Errorf("could not find %d free ports in range %d-%d", count, cfg.PortRangeStart, cfg.PortRangeEnd)
}
