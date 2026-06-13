// +build ignore

// Smoke test for the Go QPI client SDK.
//
// Usage:
//   QPI_BASE_URL=http://localhost:8090 QPI_API_TOKEN=xxx go run smoke_go.go
package main

import (
	"context"
	"fmt"
	"os"

	qpiclient "github.com/sopherapps/qpi/qpi-client/go"
)

func main() {
	baseURL := os.Getenv("QPI_BASE_URL")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8090"
	}
	token := os.Getenv("QPI_API_TOKEN")
	if token == "" {
		token = "test-api-token-abc-123"
	}

	client := qpiclient.NewClient(baseURL, token)
	ctx := context.Background()

	// List jobs
	jobs, err := client.ListJobs(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "list jobs failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Go client list_jobs returned %d jobs\n", len(jobs))

	// Get first job if any
	if len(jobs) > 0 {
		job, err := client.GetJob(ctx, jobs[0].ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "get job failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Go client get_job returned status=%s\n", job.Status)
	}

	fmt.Println("Go client smoke test passed")
}
