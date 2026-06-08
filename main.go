package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
	"go.nanomsg.org/mangos/v3"
	"go.nanomsg.org/mangos/v3/protocol/pull"
	"go.nanomsg.org/mangos/v3/protocol/push"
	_ "go.nanomsg.org/mangos/v3/transport/tcp"
)

// ── Constants ─────────────────────────────────────────────────────────────────

const (
	collectionQPUs         = "qpus"
	collectionTimeSlots    = "time_slots"
	collectionQuantumJobs  = "quantum_jobs"
	idleThreshold          = 5 * time.Second
	recoveryInterval       = 10 * time.Second
	jobTimeout             = 20 * time.Second
	dispatchPollInterval   = 1 * time.Second
	portRangeStart         = 6000
	portRangeEnd           = 7000
)

// ── Goroutine registry ────────────────────────────────────────────────────────

var (
	activeQPUs   = make(map[string]context.CancelFunc) // qpuID -> cancel
	activeQPUsMu sync.Mutex
)

// ── Entry point ───────────────────────────────────────────────────────────────

func main() {
	app := pocketbase.New()

	// Bootstrap: create collections on first boot
	app.OnBootstrap().Bind(&hook.Handler[*core.BootstrapEvent]{
		Func: func(e *core.BootstrapEvent) error {
			if err := e.Next(); err != nil {
				return err
			}
			return ensureSchema(e.App)
		},
	})

	// Register custom HTTP routes
	app.OnServe().Bind(&hook.Handler[*core.ServeEvent]{
		Func: func(e *core.ServeEvent) error {
			e.Router.POST("/api/qpu/register", func(re *core.RequestEvent) error {
				return handleQPURegister(re)
			})
			// Start the global recovery engine
			go runRecoveryEngine(e.App)
			return e.Next()
		},
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

// ── Schema bootstrap ──────────────────────────────────────────────────────────

func ensureSchema(app core.App) error {
	if err := ensureQPUsCollection(app); err != nil {
		return fmt.Errorf("qpus collection: %w", err)
	}
	if err := ensureTimeSlotsCollection(app); err != nil {
		return fmt.Errorf("time_slots collection: %w", err)
	}
	if err := ensureQuantumJobsCollection(app); err != nil {
		return fmt.Errorf("quantum_jobs collection: %w", err)
	}
	log.Println("[QPi] Schema OK")
	return nil
}

func ensureQPUsCollection(app core.App) error {
	if _, err := app.FindCollectionByNameOrId(collectionQPUs); err == nil {
		return nil // already exists
	}
	col := core.NewBaseCollection(collectionQPUs)
	col.Fields.Add(&core.TextField{Name: "name", Required: true})
	col.Fields.Add(&core.TextField{Name: "registration_token", Required: true})
	col.Fields.Add(&core.SelectField{
		Name:      "status",
		Values:    []string{"offline", "online", "maintenance"},
		MaxSelect: 1,
		Required:  true,
	})
	col.Fields.Add(&core.NumberField{Name: "nng_command_port"})
	col.Fields.Add(&core.NumberField{Name: "nng_result_port"})
	return app.Save(col)
}

func ensureTimeSlotsCollection(app core.App) error {
	if _, err := app.FindCollectionByNameOrId(collectionTimeSlots); err == nil {
		return nil
	}
	col := core.NewBaseCollection(collectionTimeSlots)
	col.Fields.Add(&core.DateField{Name: "start_time", Required: true})
	col.Fields.Add(&core.DateField{Name: "end_time", Required: true})

	// booked_by → relation to users collection (optional)
	usersCol, err := app.FindCollectionByNameOrId("users")
	if err == nil {
		col.Fields.Add(&core.RelationField{
			Name:         "booked_by",
			CollectionId: usersCol.Id,
			MaxSelect:    1,
		})
	}
	return app.Save(col)
}

func ensureQuantumJobsCollection(app core.App) error {
	if _, err := app.FindCollectionByNameOrId(collectionQuantumJobs); err == nil {
		return nil
	}
	col := core.NewBaseCollection(collectionQuantumJobs)

	usersCol, _ := app.FindCollectionByNameOrId("users")
	if usersCol != nil {
		col.Fields.Add(&core.RelationField{
			Name:         "user_id",
			CollectionId: usersCol.Id,
			MaxSelect:    1,
		})
	}

	qpuCol, _ := app.FindCollectionByNameOrId(collectionQPUs)
	if qpuCol != nil {
		col.Fields.Add(&core.RelationField{
			Name:         "qpu_target",
			CollectionId: qpuCol.Id,
			MaxSelect:    1,
		})
	}

	col.Fields.Add(&core.JSONField{Name: "payload"})
	col.Fields.Add(&core.SelectField{
		Name:      "status",
		Values:    []string{"pending", "running", "completed", "failed"},
		MaxSelect: 1,
		Required:  true,
	})
	col.Fields.Add(&core.DateField{Name: "finished_at"})
	col.Fields.Add(&core.JSONField{Name: "results"})
	col.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
	col.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
	return app.Save(col)
}

// ── Registration handler ──────────────────────────────────────────────────────

type registerRequest struct {
	Name              string `json:"name"`
	RegistrationToken string `json:"registration_token"`
}

type registerResponse struct {
	Status         string `json:"status"`
	NNGCommandPort int    `json:"nng_command_port"`
	NNGResultPort  int    `json:"nng_result_port"`
	AuthToken      string `json:"auth_token"`
}

func handleQPURegister(re *core.RequestEvent) error {
	var req registerRequest
	if err := re.BindBody(&req); err != nil {
		return re.Error(http.StatusBadRequest, "invalid request body", err)
	}

	// Find QPU by registration token
	qpu, err := re.App.FindFirstRecordByData(collectionQPUs, "registration_token", req.RegistrationToken)
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

// ── Dispatcher ────────────────────────────────────────────────────────────────

func runDispatcher(app core.App, qpuID string, cmdPort int) {
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
		job := fetchNextJob(app, qpuID)
		if job == nil {
			time.Sleep(dispatchPollInterval)
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
			time.Sleep(dispatchPollInterval)
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

// fetchNextJob implements the session-based booking + opportunistic FIFO algorithm.
func fetchNextJob(app core.App, qpuID string) *core.Record {
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000Z")

	// Is there an active time slot right now?
	slots, _ := app.FindRecordsByFilter(
		collectionTimeSlots,
		"start_time <= {:now} && end_time >= {:now}",
		"-start_time", 1, 0,
		dbx.Params{"now": now},
	)

	var bookerID string
	if len(slots) > 0 {
		bookerID = slots[0].GetString("booked_by")
	}

	if bookerID != "" {
		// Priority 1: booked user's oldest pending job for this QPU
		jobs, _ := app.FindRecordsByFilter(
			collectionQuantumJobs,
			"status = 'pending' && qpu_target = {:qpu} && user_id = {:user}",
			"+created", 1, 0,
			dbx.Params{"qpu": qpuID, "user": bookerID},
		)
		if len(jobs) > 0 {
			return jobs[0]
		}

		// Priority 2: idle fallback — check booker's last completed job time
		lastJobs, _ := app.FindRecordsByFilter(
			collectionQuantumJobs,
			"status = 'completed' && qpu_target = {:qpu} && user_id = {:user}",
			"-finished_at", 1, 0,
			dbx.Params{"qpu": qpuID, "user": bookerID},
		)
		if len(lastJobs) > 0 {
			finishedAt := lastJobs[0].GetDateTime("finished_at").Time()
			if time.Since(finishedAt) < idleThreshold {
				// Booker is still active; wait for their next job
				return nil
			}
		}
	}

	// Priority 3: general drop-in — oldest pending job for this QPU
	jobs, _ := app.FindRecordsByFilter(
		collectionQuantumJobs,
		"status = 'pending' && qpu_target = {:qpu}",
		"+created", 1, 0,
		dbx.Params{"qpu": qpuID},
	)
	if len(jobs) > 0 {
		return jobs[0]
	}
	return nil
}

// ── Result Listener ───────────────────────────────────────────────────────────

type resultPayload struct {
	JobID   string         `json:"job_id"`
	Results map[string]any `json:"results"`
}

func runResultListener(app core.App, qpuID string, resPort int) {
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
			time.Sleep(dispatchPollInterval)
			continue
		}

		var result resultPayload
		if err := json.Unmarshal(msg, &result); err != nil {
			log.Printf("[Listener %s] JSON parse error: %v", qpuID, err)
			continue
		}

		job, err := app.FindRecordById(collectionQuantumJobs, result.JobID)
		if err != nil {
			log.Printf("[Listener %s] job %s not found: %v", qpuID, result.JobID, err)
			continue
		}

		job.Set("status", "completed")
		job.Set("finished_at", time.Now().UTC().Format("2006-01-02 15:04:05.000Z"))
		resultsJSON, _ := json.Marshal(result.Results)
		job.Set("results", string(resultsJSON))

		if err := app.Save(job); err != nil {
			log.Printf("[Listener %s] DB save error for job %s: %v", qpuID, result.JobID, err)
		} else {
			log.Printf("[Listener %s] job %s completed", qpuID, result.JobID)
		}
	}
}

// ── Recovery Engine ───────────────────────────────────────────────────────────

func runRecoveryEngine(app core.App) {
	ticker := time.NewTicker(recoveryInterval)
	defer ticker.Stop()
	log.Println("[Recovery] Engine started")

	for range ticker.C {
		cutoff := time.Now().UTC().Add(-jobTimeout).Format("2006-01-02 15:04:05.000Z")
		staleJobs, err := app.FindRecordsByFilter(
			collectionQuantumJobs,
			"status = 'running' && updated <= {:cutoff}",
			"+updated", 100, 0,
			dbx.Params{"cutoff": cutoff},
		)
		if err != nil {
			log.Printf("[Recovery] query error: %v", err)
			continue
		}
		for _, job := range staleJobs {
			job.Set("status", "pending")
			if err := app.Save(job); err != nil {
				log.Printf("[Recovery] failed to reset job %s: %v", job.Id, err)
			} else {
				log.Printf("[Recovery] reset stale job %s to pending", job.Id)
			}
		}
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// findFreePorts finds multiple available TCP ports in the configured range,
// excluding ports currently allocated in the database.
func findFreePorts(app core.App, count int) ([]int, error) {
	allocated := make(map[int]bool)
	records, err := app.FindRecordsByFilter(collectionQPUs, "nng_command_port > 0 || nng_result_port > 0", "", 0, 0)
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
	for port := portRangeStart; port < portRangeEnd; port++ {
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
	return nil, fmt.Errorf("could not find %d free ports in range %d-%d", count, portRangeStart, portRangeEnd)
}
