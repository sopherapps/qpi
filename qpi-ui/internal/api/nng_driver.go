package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"go.nanomsg.org/mangos/v3"
	"go.nanomsg.org/mangos/v3/protocol/pull"
	"go.nanomsg.org/mangos/v3/protocol/push"
	_ "go.nanomsg.org/mangos/v3/transport/tlstcp"

	"qpi/internal/config"
	"qpi/internal/db"
	"qpi/internal/scheduler"
)

var (
	// activeDrivers stores active cancel functions for goroutines bound to
	// connected drivers, keyed by driver id (RFC 0001 Phase 2).
	activeDrivers   = make(map[string]context.CancelFunc)
	activeDriversMu sync.Mutex
)

// driverEventRegistry maps inbound driver→UI event types to their handlers. It
// is the production counterpart to the spike registry in events_test.go: the
// server holds one handler per type it receives (RFC 0001 §7).
var driverEventRegistry = func() *EventRegistry {
	registry := NewEventRegistry()
	registry.Register(EventJobResult, handleDriverJobResult)
	return registry
}()

// StartDriverDistribution starts the dispatch/listen goroutines for a connected
// driver if not already running, mirroring StartQPUDistribution for QPUs.
func StartDriverDistribution(app core.App, cfg *config.AppConfig, driverID, qpuID string, inPort, outPort int) {
	activeDriversMu.Lock()
	defer activeDriversMu.Unlock()
	if _, running := activeDrivers[driverID]; !running {
		ctx, cancel := context.WithCancel(context.Background())
		activeDrivers[driverID] = cancel
		go runDriverDispatcher(ctx, app, driverID, qpuID, inPort)
		go runDriverListener(ctx, app, driverID, qpuID, outPort)
		log.Printf("[QPi] Driver goroutines started for %s (in:%d out:%d)", driverID, inPort, outPort)
	}
}

// StopDriverDistribution cancels the goroutines for a specific driver.
func StopDriverDistribution(driverID string) {
	activeDriversMu.Lock()
	defer activeDriversMu.Unlock()
	if cancel, exists := activeDrivers[driverID]; exists {
		cancel()
		delete(activeDrivers, driverID)
		log.Printf("[QPi] Driver goroutines stopped for %s", driverID)
	}
}

// runDriverDispatcher pushes pending jobs for the driver's QPU as JobDispatch
// events over an NNG PUSH socket on inPort. It copies runDispatcher, differing
// only in that each job travels inside the event envelope (RFC 0001 §6) and the
// pipe hook flips both the driver's and its QPU's online/offline status.
func runDriverDispatcher(ctx context.Context, app core.App, driverID, qpuID string, inPort int) {
	cfg, err := config.GetConfigFromApp(app)
	if err != nil {
		log.Printf("[DriverDispatcher %s] failed to get config: %v", driverID, err)
		return
	}

	sock, err := push.NewSocket()
	if err != nil {
		log.Printf("[DriverDispatcher %s] socket error: %v", driverID, err)
		return
	}
	defer sock.Close()

	l, err := getListener(sock, inPort, cfg.GetTlsConfig())
	if err != nil {
		log.Printf("[DriverDispatcher %s] %v", driverID, err)
		return
	}

	sock.SetPipeEventHook(func(event mangos.PipeEvent, pipe mangos.Pipe) {
		switch event {
		case mangos.PipeEventAttached:
			log.Printf("[DriverDispatcher %s] driver attached: %s", driverID, pipe.Address())
			markDriverStatus(app, cfg, driverID, qpuID, "online")
		case mangos.PipeEventDetached:
			log.Printf("[DriverDispatcher %s] driver disconnected: %s", driverID, pipe.Address())
			markDriverStatus(app, cfg, driverID, qpuID, "offline")
		}
	})

	addr := l.Address()
	if err := l.Listen(); err != nil {
		log.Printf("[DriverDispatcher %s] listen error on %s: %v", driverID, addr, err)
		return
	}
	log.Printf("[DriverDispatcher %s] PUSH listening on %s", driverID, addr)

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

		event, err := NewEvent(driverID, EventJobDispatch, DispatchPayload{JobID: job.ID, Payload: job.Payload})
		if err != nil {
			log.Printf("[DriverDispatcher %s] cannot build dispatch for job %s: %v", driverID, job.ID, err)
			continue
		}
		payload, err := json.Marshal(event)
		if err != nil {
			log.Printf("[DriverDispatcher %s] cannot marshal dispatch for job %s: %v", driverID, job.ID, err)
			continue
		}

		if err := sock.Send(payload); err != nil {
			select {
			case <-ctx.Done():
				return
			default:
			}
			log.Printf("[DriverDispatcher %s] send error: %v — requeueing", driverID, err)

			updateData := map[string]any{"status": "pending"}
			var requeuedJob db.QuantumJob
			if updateErr := db.FindAndUpdateOne(app, cfg.CollectionQuantumJobs, job.ID, &requeuedJob, updateData); updateErr != nil {
				log.Printf("[DriverDispatcher %s] failed to requeue job %s: %v", driverID, job.ID, updateErr)
			}

			select {
			case <-ctx.Done():
				return
			case <-time.After(cfg.DispatchPollInterval):
			}
			continue
		}

		updateData := map[string]any{"status": "running"}
		var runningJob db.QuantumJob
		if err := db.FindAndUpdateOne(app, cfg.CollectionQuantumJobs, job.ID, &runningJob, updateData); err != nil {
			log.Printf("[DriverDispatcher %s] DB update error: %v", driverID, err)
		} else {
			log.Printf("[DriverDispatcher %s] dispatched job %s", driverID, job.ID)
		}
	}
}

// runDriverListener receives events emitted by a driver over an NNG PULL socket
// on outPort and routes each through driverEventRegistry. It copies
// runResultListener, differing only in that it parses the event envelope and
// dispatches by type instead of assuming a bare result (RFC 0001 §4, §6).
func runDriverListener(ctx context.Context, app core.App, driverID, qpuID string, outPort int) {
	cfg, err := config.GetConfigFromApp(app)
	if err != nil {
		log.Printf("[DriverListener %s] failed to get config: %v", driverID, err)
		return
	}

	sock, err := pull.NewSocket()
	if err != nil {
		log.Printf("[DriverListener %s] socket error: %v", driverID, err)
		return
	}
	defer sock.Close()

	l, err := getListener(sock, outPort, cfg.GetTlsConfig())
	if err != nil {
		log.Printf("[DriverListener %s] %v", driverID, err)
		return
	}

	addr := l.Address()
	if err := l.Listen(); err != nil {
		log.Printf("[DriverListener %s] listen error on %s: %v", driverID, addr, err)
		return
	}
	log.Printf("[DriverListener %s] PULL listening on %s", driverID, addr)

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
			log.Printf("[DriverListener %s] recv error: %v", driverID, err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(cfg.DispatchPollInterval):
			}
			continue
		}

		var event Event
		if err := json.Unmarshal(msg, &event); err != nil {
			log.Printf("[DriverListener %s] envelope parse error: %v", driverID, err)
			continue
		}

		// A handler that rejects an event just logs and drops it; the loop
		// keeps listening (RFC 0001 §4).
		_ = driverEventRegistry.Dispatch(ctx, app, qpuID, &event)
	}
}

// markDriverStatus flips a driver's online/offline status and mirrors it onto
// the driver's QPU. A QPU driver's connection is what makes the QPU available,
// so the QPU status tracks the driver's pipe events the way the legacy
// dispatcher tracked the QPU's own connection (RFC 0001 §5).
func markDriverStatus(app core.App, cfg *config.AppConfig, driverID, qpuID, status string) {
	driverUpdate := map[string]any{
		"status":    status,
		"last_seen": time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
	}
	var driver db.Driver
	if err := db.FindAndUpdateOne(app, cfg.CollectionDrivers, driverID, &driver, driverUpdate); err != nil {
		log.Printf("[DriverDispatcher %s] failed to mark driver %s: %v", driverID, status, err)
	}

	var qpu db.QPU
	if err := db.FindAndUpdateOne(app, cfg.CollectionQPUs, qpuID, &qpu, map[string]any{"status": status}); err != nil {
		log.Printf("[DriverDispatcher %s] failed to mark QPU %s %s: %v", driverID, qpuID, status, err)
	}
}

// handleDriverJobResult applies a JobResult event to the calling driver's QPU,
// the event-framework counterpart of the body of runResultListener.
func handleDriverJobResult(ctx context.Context, app core.App, qpuID string, event *Event) error {
	var result ResultPayload
	if err := json.Unmarshal(event.Payload, &result); err != nil {
		return fmt.Errorf("cannot parse JobResult payload: %w", err)
	}
	return applyJobResult(app, qpuID, result)
}

// applyJobResult updates a finished job and deducts the QPU-seconds it used,
// mirroring the persistence the legacy result listener performs (RFC 0001 §8).
func applyJobResult(app core.App, qpuID string, result ResultPayload) error {
	cfg, err := config.GetConfigFromApp(app)
	if err != nil {
		return err
	}

	var job db.QuantumJob
	if err := db.FindOne(app, cfg.CollectionQuantumJobs, result.JobID, &job); err != nil {
		return fmt.Errorf("job %s not found: %w", result.JobID, err)
	}

	var executionDuration time.Duration
	if job.Updated != "" {
		if updatedTime, parseErr := time.Parse("2006-01-02 15:04:05.000Z", job.Updated); parseErr == nil {
			executionDuration = time.Since(updatedTime)
		}
	}
	durationSeconds := executionDuration.Seconds()

	if job.UserID != "" {
		deductData := map[string]any{"qpu_seconds-": durationSeconds}
		var user db.User
		if updateErr := db.FindAndUpdateOne(app, "users", job.UserID, &user, deductData); updateErr != nil {
			if errors.Is(updateErr, db.ErrNotFound) {
				log.Printf("[DriverListener %s] user %s not found for QPU seconds deduction", qpuID, job.UserID)
			} else {
				log.Printf("[DriverListener %s] failed to deduct QPU seconds for user %s: %v", qpuID, job.UserID, updateErr)
			}
		}
	}

	finalStatus := "completed"
	if _, hasError := result.Results["error"]; hasError {
		finalStatus = "failed"
	}

	resultsJSON, _ := json.Marshal(result.Results)
	jobUpdate := &JobResultUpdate{
		Status:     finalStatus,
		FinishedAt: time.Now().UTC().Format("2006-01-02 15:04:05.000Z"),
		Results:    string(resultsJSON),
		Duration:   durationSeconds,
	}

	var updatedJob db.QuantumJob
	if err := db.FindAndUpdateOne(app, cfg.CollectionQuantumJobs, result.JobID, &updatedJob, jobUpdate.ToMap()); err != nil {
		return fmt.Errorf("cannot save result for job %s: %w", result.JobID, err)
	}
	log.Printf("[DriverListener %s] job %s %s", qpuID, result.JobID, finalStatus)
	return nil
}
