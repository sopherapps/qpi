package config

import (
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"
)

var (
	CollectionQPUs         = "qpus"
	CollectionTimeSlots    = "time_slots"
	CollectionQuantumJobs  = "quantum_jobs"
	IdleThreshold          = 5 * time.Second
	RecoveryInterval       = 10 * time.Second
	JobTimeout             = 20 * time.Second
	DispatchPollInterval   = 1 * time.Second
	PortRangeStart         = 6000
	PortRangeEnd           = 7000
)

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

func getEnvString(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			return val
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if val, err := time.ParseDuration(v); err == nil {
			return val
		}
	}
	return fallback
}
