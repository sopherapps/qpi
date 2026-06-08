// Package config manages application-wide configuration parameters for the QPI orchestrator,
// including database collection names, NNG connection ports, and job timeout intervals.
// Variables can be set via command-line arguments or fallback environment variables.
package config

import (
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"
)

var (
	// CollectionQPUs is the name of the database collection storing QPU hardware status and ports.
	CollectionQPUs = "qpus"

	// CollectionTimeSlots is the name of the database collection storing session reservations.
	CollectionTimeSlots = "time_slots"

	// CollectionQuantumJobs is the name of the database collection storing quantum job payloads and results.
	CollectionQuantumJobs = "quantum_jobs"

	// IdleThreshold is the maximum duration to wait for a time slot booker before falling back to drop-in queue.
	IdleThreshold = 5 * time.Second

	// RecoveryInterval is the interval at which the background recovery engine checks for stale running jobs.
	RecoveryInterval = 10 * time.Second

	// JobTimeout is the maximum time a job is allowed to remain in 'running' state before being reset.
	JobTimeout = 20 * time.Second

	// DispatchPollInterval is the sleep duration between job queue polling attempts.
	DispatchPollInterval = 1 * time.Second

	// PortRangeStart is the start of the port range allocated for dynamic dynamic NNG channels.
	PortRangeStart = 6000

	// PortRangeEnd is the end of the port range allocated for dynamic NNG channels.
	PortRangeEnd = 7000
)

// BindFlags registers custom command-line flags on the root Cobra command, mapping them to the package variables.
// The default values are resolved dynamically from the corresponding environment variables.
func BindFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&CollectionQPUs, "qpus-collection", getEnvString("QPI_QPUS_COLLECTION", "qpus"), "Collection name for QPUs")
	cmd.PersistentFlags().StringVar(&CollectionTimeSlots, "timeslots-collection", getEnvString("QPI_TIMESLOTS_COLLECTION", "time_slots"), "Collection name for Time Slots")
	cmd.PersistentFlags().StringVar(&CollectionQuantumJobs, "jobs-collection", getEnvString("QPI_JOBS_COLLECTION", "quantum_jobs"), "Collection name for Quantum Jobs")
	cmd.PersistentFlags().DurationVar(&IdleThreshold, "idle-threshold", getEnvDuration("QPI_IDLE_THRESHOLD", 5*time.Second), "Idle fallback threshold")
	cmd.PersistentFlags().DurationVar(&RecoveryInterval, "recovery-interval", getEnvDuration("QPI_RECOVERY_INTERVAL", 10*time.Second), "Stale job recovery check interval")
	cmd.PersistentFlags().DurationVar(&JobTimeout, "job-timeout", getEnvDuration("QPI_JOB_TIMEOUT", 20*time.Second), "Stale job execution timeout")
	cmd.PersistentFlags().DurationVar(&DispatchPollInterval, "dispatch-poll-interval", getEnvDuration("QPI_DISPATCH_POLL_INTERVAL", 1*time.Second), "Dispatch poll interval")
	cmd.PersistentFlags().IntVar(&PortRangeStart, "port-range-start", getEnvInt("QPI_PORT_RANGE_START", 6000), "NNG port range start")
	cmd.PersistentFlags().IntVar(&PortRangeEnd, "port-range-end", getEnvInt("QPI_PORT_RANGE_END", 7000), "NNG port range end")
}

// getEnvString retrieves the environment variable value by key, falling back to defaultValue if empty.
func getEnvString(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// getEnvInt retrieves the environment variable value by key and parses it as int, falling back to defaultValue on failure.
func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			return val
		}
	}
	return fallback
}

// getEnvDuration retrieves the environment variable value by key and parses it as time.Duration, falling back on failure.
func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if val, err := time.ParseDuration(v); err == nil {
			return val
		}
	}
	return fallback
}
