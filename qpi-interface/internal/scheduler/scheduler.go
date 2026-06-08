// Package scheduler manages the job queue dispatcher algorithm and recovery loops,
// prioritizing booked session users and reverting hung quantum jobs.
package scheduler

import (
	"log"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"qpi/internal/config"
)

// FetchNextJob implements the session-based booking + opportunistic FIFO algorithm.
// It prioritizes the booked user's oldest pending job. If no slot is active, or if
// the booked user remains idle beyond config.IdleThreshold, it falls back to the oldest
// pending job from any user targetted at this QPU.
func FetchNextJob(app core.App, qpuID string) *core.Record {
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000Z")

	// Is there an active time slot right now?
	slots, _ := app.FindRecordsByFilter(
		config.CollectionTimeSlots,
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
			config.CollectionQuantumJobs,
			"status = 'pending' && qpu_target = {:qpu} && user_id = {:user}",
			"+created", 1, 0,
			dbx.Params{"qpu": qpuID, "user": bookerID},
		)
		if len(jobs) > 0 {
			return jobs[0]
		}

		// Priority 2: idle fallback — check booker's last completed job time
		lastJobs, _ := app.FindRecordsByFilter(
			config.CollectionQuantumJobs,
			"status = 'completed' && qpu_target = {:qpu} && user_id = {:user}",
			"-finished_at", 1, 0,
			dbx.Params{"qpu": qpuID, "user": bookerID},
		)
		if len(lastJobs) > 0 {
			finishedAt := lastJobs[0].GetDateTime("finished_at").Time()
			if time.Since(finishedAt) < config.IdleThreshold {
				// Booker is still active; wait for their next job
				return nil
			}
		}
	}

	// Priority 3: general drop-in — oldest pending job for this QPU
	jobs, _ := app.FindRecordsByFilter(
		config.CollectionQuantumJobs,
		"status = 'pending' && qpu_target = {:qpu}",
		"+created", 1, 0,
		dbx.Params{"qpu": qpuID},
	)
	if len(jobs) > 0 {
		return jobs[0]
	}
	return nil
}

// RunRecoveryEngine runs a background loop that identifies 'running' jobs
// that have exceeded config.JobTimeout and resets their status to 'pending'
// (e.g. if the hardware driver crashed or lost connection during simulation).
func RunRecoveryEngine(app core.App) {
	ticker := time.NewTicker(config.RecoveryInterval)
	defer ticker.Stop()
	log.Println("[Recovery] Engine started")

	for range ticker.C {
		cutoff := time.Now().UTC().Add(-config.JobTimeout).Format("2006-01-02 15:04:05.000Z")
		staleJobs, err := app.FindRecordsByFilter(
			config.CollectionQuantumJobs,
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
