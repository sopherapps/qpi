package api

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"go.nanomsg.org/mangos/v3"
	"go.nanomsg.org/mangos/v3/protocol"
	"go.nanomsg.org/mangos/v3/protocol/pull"
	"go.nanomsg.org/mangos/v3/protocol/push"
	_ "go.nanomsg.org/mangos/v3/transport/tlstcp"

	"qpi/internal/config"
	"qpi/internal/db"
	"qpi/internal/scheduler"
)

// runDispatcher starts an NNG PUSH socket on the cmdPort (secured with TLS certFile and keyFile), polling for pending quantum jobs
// from the scheduler and pushing them to the registered python driver node.
func runDispatcher(ctx context.Context, app core.App, qpuID string, cmdPort int) {
	cfg, err := config.GetConfigFromApp(app)
	if err != nil {
		log.Printf("[Dispatcher %s] failed to get config: %v", qpuID, err)
		return
	}

	sock, err := push.NewSocket()
	if err != nil {
		log.Printf("[Dispatcher %s] socket error: %v", qpuID, err)
		return
	}
	defer sock.Close()

	tlsConfig := cfg.GetTlsConfig()
	l, err := getListener(sock, cmdPort, tlsConfig)
	if err != nil {
		log.Printf("[Dispatcher %s] %v", qpuID, err)
		return
	}

	addr := l.Address()
	if err := l.Listen(); err != nil {
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

		payload, _ := json.Marshal(DispatchPayload{
			JobID:   job.ID,
			Payload: job.Payload,
		})

		if err := sock.Send(payload); err != nil {
			select {
			case <-ctx.Done():
				return
			default:
			}
			log.Printf("[Dispatcher %s] send error: %v — requeueing", qpuID, err)

			updateData := map[string]any{"status": "pending"}
			var requeuedJob db.QuantumJob
			if updateErr := db.FindAndUpdateOne(app, cfg.CollectionQuantumJobs, job.ID, &requeuedJob, updateData); updateErr != nil {
				log.Printf("[Dispatcher %s] failed to requeue job %s: %v", qpuID, job.ID, updateErr)
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
			log.Printf("[Dispatcher %s] DB update error: %v", qpuID, err)
		} else {
			log.Printf("[Dispatcher %s] dispatched job %s", qpuID, job.ID)
		}
	}
}

// runResultListener starts an NNG PULL socket on the resPort (secured with TLS cert and key), waiting for job execution
// results sent back by the hardware driver node and saving them to the database.
func runResultListener(ctx context.Context, app core.App, qpuID string, resPort int) {
	cfg, err := config.GetConfigFromApp(app)
	if err != nil {
		log.Printf("[Listener %s] failed to get config: %v", qpuID, err)
		return
	}

	sock, err := pull.NewSocket()
	if err != nil {
		log.Printf("[Listener %s] socket error: %v", qpuID, err)
		return
	}
	defer sock.Close()

	tlsConfig := cfg.GetTlsConfig()
	l, err := getListener(sock, resPort, tlsConfig)
	if err != nil {
		log.Printf("[Listener %s] %v", qpuID, err)
		return
	}

	addr := l.Address()
	if err := l.Listen(); err != nil {
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

		var result ResultPayload
		if err := json.Unmarshal(msg, &result); err != nil {
			log.Printf("[Listener %s] JSON parse error: %v", qpuID, err)
			continue
		}

		var job db.QuantumJob
		err = db.FindOne(app, cfg.CollectionQuantumJobs, result.JobID, &job)
		if err != nil {
			log.Printf("[Listener %s] job %s not found: %v", qpuID, result.JobID, err)
			continue
		}

		// Calculate execution duration from last update timestamp
		// We parse the Updated field as a time to compute the duration
		var executionDuration time.Duration
		if job.Updated != "" {
			if updatedTime, parseErr := time.Parse("2006-01-02 15:04:05.000Z", job.Updated); parseErr == nil {
				executionDuration = time.Since(updatedTime)
			}
		}
		durationSeconds := executionDuration.Seconds()

		// Deduct QPU seconds from the user's balance
		if job.UserID != "" {
			deductData := map[string]any{"qpu_seconds-": durationSeconds}
			var user db.User
			if updateErr := db.FindAndUpdateOne(app, "users", job.UserID, &user, deductData); updateErr != nil {
				if errors.Is(updateErr, db.ErrNotFound) {
					log.Printf("[Listener %s] user %s not found for QPU seconds deduction", qpuID, job.UserID)
				} else {
					log.Printf("[Listener %s] failed to deduct QPU seconds for user %s: %v", qpuID, job.UserID, updateErr)
				}
			}
		}

		// Determine final job status based on result contents
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
		updateData := jobUpdate.ToMap()

		var updatedJob db.QuantumJob
		if err := db.FindAndUpdateOne(app, cfg.CollectionQuantumJobs, result.JobID, &updatedJob, updateData); err != nil {
			log.Printf("[Listener %s] DB save error for job %s: %v", qpuID, result.JobID, err)
		} else {
			log.Printf("[Listener %s] job %s %s", qpuID, result.JobID, finalStatus)
		}
	}
}

// getListener gets a TLS Listener at the given port
func getListener(sock protocol.Socket, port int, tlsConfig *tls.Config) (mangos.Listener, error) {
	addr := fmt.Sprintf("tls+tcp://0.0.0.0:%d", port)
	l, err := sock.NewListener(addr, map[string]any{mangos.OptionTLSConfig: tlsConfig})
	if err != nil {
		return nil, fmt.Errorf("Listerner error: %w", err)
	}

	return l, nil
}
