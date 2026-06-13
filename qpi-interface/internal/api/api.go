// Package api manages HTTP REST endpoints and low-level NNG message channels
// for registering QPUs, dispatching jobs, and listening for job execution results.
package api

import (
	"context"
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

// registerRequest represents the JSON payload passed to /api/qpu/register.
type registerRequest struct {
	Name              string         `json:"name"`
	RegistrationToken string         `json:"registration_token"`
	ExecutorType      string         `json:"executor_type,omitempty"`
	DeviceConfig      map[string]any `json:"device_config,omitempty"`
}

// registerResponse represents the JSON payload returned by /api/qpu/register.
type registerResponse struct {
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
	N_Qubits   int              `json:"n_qubits,omitempty"`
	MeasLevel  *int             `json:"meas_level,omitempty"`
	MeasReturn string           `json:"meas_return,omitempty"`
	QPUTarget  string           `json:"qpu_target,omitempty"`
}

// userUpdateRequest represents the JSON payload for PATCH /api/admin/users/{id}.
type userUpdateRequest struct {
	QpuSeconds *float64 `json:"qpu_seconds,omitempty"`
	APITokens  []string `json:"api_tokens,omitempty"`
}

// RegisterRoutes sets up custom HTTP routes for QPU interactions.
func RegisterRoutes(e *core.ServeEvent) {
	e.Router.POST("/api/qpu/register", func(re *core.RequestEvent) error {
		return handleQPURegister(re)
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
		// Look up token in the api_tokens collection and expand the user relation
		tokenRec, err := re.App.FindFirstRecordByFilter(
			cfg.CollectionAPITokens,
			"token = {:token}",
			dbx.Params{"token": token},
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

	// Sync api_tokens to the separate collection
	if req.APITokens != nil {
		tokensCol, err := re.App.FindCollectionByNameOrId(cfg.CollectionAPITokens)
		if err != nil {
			return re.Error(http.StatusInternalServerError, "api_tokens collection not found", err)
		}

		// Delete existing tokens for this user
		existing, err := re.App.FindRecordsByFilter(
			cfg.CollectionAPITokens,
			"user = {:userId}",
			"", 0, 0,
			dbx.Params{"userId": userID},
		)
		if err == nil {
			for _, t := range existing {
				_ = re.App.Delete(t)
			}
		}

		// Create new token records
		for _, tokenValue := range req.APITokens {
			tokenRec := core.NewRecord(tokensCol)
			tokenRec.Set("token", tokenValue)
			tokenRec.Set("user", userID)
			if err := re.App.Save(tokenRec); err != nil {
				log.Printf("Warning: failed to create api_token record: %v", err)
			}
		}
	}

	if err := re.App.Save(record); err != nil {
		return re.Error(http.StatusInternalServerError, "failed to update user", err)
	}

	// Return current tokens for the user
	var currentTokens []string
	tokenRecs, _ := re.App.FindRecordsByFilter(
		cfg.CollectionAPITokens,
		"user = {:userId}",
		"", 0, 0,
		dbx.Params{"userId": userID},
	)
	for _, t := range tokenRecs {
		currentTokens = append(currentTokens, t.GetString("token"))
	}

	return re.JSON(http.StatusOK, map[string]any{
		"id":          record.Id,
		"qpu_seconds": record.GetFloat("qpu_seconds"),
		"api_tokens":  currentTokens,
	})
}

// handleQPURegister registers a new hardware driver node, allocating dynamic command/result ports
// and starting parallel dispatcher and result listener routines.
func handleQPURegister(re *core.RequestEvent) error {
	cfg, err := config.GetConfigFromApp(re.App)
	if err != nil {
		return re.Error(http.StatusInternalServerError, "failed to retrieve custom configuration", err)
	}

	var req registerRequest
	if err := re.BindBody(&req); err != nil {
		return re.Error(http.StatusBadRequest, "invalid request body", err)
	}

	// Find QPU by registration token
	qpu, err := re.App.FindFirstRecordByData(cfg.CollectionQPUs, "registration_token", req.RegistrationToken)
	if err != nil {
		return re.Error(http.StatusUnauthorized, "invalid registration token", err)
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
	qpuID := qpu.Id
	activeQPUsMu.Lock()
	if _, running := activeQPUs[qpuID]; !running {
		ctx, cancel := context.WithCancel(context.Background())
		activeQPUs[qpuID] = cancel
		go runDispatcher(re.App, qpuID, cmdPort)
		go runResultListener(re.App, qpuID, resPort)
		log.Printf("[QPi] Goroutines started for QPU %s (cmd:%d res:%d)", qpuID, cmdPort, resPort)
		_ = ctx // ctx is held for future cancellation via cancel()
	}
	activeQPUsMu.Unlock()

	// Generate auth token for the QPU record
	token, err := qpu.NewStaticAuthToken(0) // 0 = use app default duration
	if err != nil {
		// Non-fatal: fall back to the registration token as a simple identifier
		token = req.RegistrationToken
	}

	return re.JSON(http.StatusOK, registerResponse{
		Status:         "success",
		NNGCommandPort: cmdPort,
		NNGResultPort:  resPort,
		AuthToken:      token,
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

// runDispatcher starts an NNG PUSH socket on the cmdPort, polling for pending quantum jobs
// from the scheduler and pushing them to the registered python driver node.
func runDispatcher(app core.App, qpuID string, cmdPort int) {
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

	for {
		job := scheduler.FetchNextJob(app, qpuID)
		if job == nil {
			time.Sleep(cfg.DispatchPollInterval)
			continue
		}

		payload, _ := json.Marshal(map[string]any{
			"job_id":  job.Id,
			"payload": job.Get("payload"),
		})

		if err := sock.Send(payload); err != nil {
			log.Printf("[Dispatcher %s] send error: %v — requeueing", qpuID, err)
			job.Set("status", "pending")
			_ = app.Save(job)
			time.Sleep(cfg.DispatchPollInterval)
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
func runResultListener(app core.App, qpuID string, resPort int) {
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

	for {
		msg, err := sock.Recv()
		if err != nil {
			if err == mangos.ErrClosed {
				return
			}
			log.Printf("[Listener %s] recv error: %v", qpuID, err)
			time.Sleep(cfg.DispatchPollInterval)
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
