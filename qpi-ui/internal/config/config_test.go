package config

import (
	"os"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestNewFromFlags_Yaml(t *testing.T) {
	// Create a temporary YAML config file
	tmpFile, err := os.CreateTemp("", "qpi_config_*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	yamlContent := `
jobsCollection: "yaml_quantum_jobs"
portRangeStart: 6005
portRangeEnd: 7005
idleThreshold: "12s"
disableEmailPasswordAuth: true
oauth2Providers:
  - name: "github"
    clientId: "test_client_id"
    clientSecret: "test_client_secret"
    displayName: "GitHub Custom"
    pkce: true
`
	if _, err := tmpFile.Write([]byte(yamlContent)); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Bind flags to a dummy cobra command
	cmd := &cobra.Command{}
	BindFlags(cmd)

	// Set the config-file flag using the temp file
	if err := cmd.PersistentFlags().Set("config-file", tmpFile.Name()); err != nil {
		t.Fatalf("failed to set config-file flag: %v", err)
	}

	// Load configuration
	cfg, err := NewFromFlags(cmd)
	if err != nil {
		t.Fatalf("failed to load config from flags: %v", err)
	}

	// Verify loaded values
	if cfg.CollectionQuantumJobs != "yaml_quantum_jobs" {
		t.Errorf("expected CollectionQuantumJobs to be 'yaml_quantum_jobs', got '%s'", cfg.CollectionQuantumJobs)
	}
	if cfg.PortRangeStart != 6005 {
		t.Errorf("expected PortRangeStart to be 6005, got %d", cfg.PortRangeStart)
	}
	if cfg.PortRangeEnd != 7005 {
		t.Errorf("expected PortRangeEnd to be 7005, got %d", cfg.PortRangeEnd)
	}
	if cfg.IdleThreshold != 12*time.Second {
		t.Errorf("expected IdleThreshold to be 12s, got %v", cfg.IdleThreshold)
	}
	if !cfg.DisableEmailPasswordAuth {
		t.Errorf("expected DisableEmailPasswordAuth to be true, got false")
	}
	if len(cfg.OAuth2Providers) != 1 {
		t.Fatalf("expected 1 OAuth2 provider, got %d", len(cfg.OAuth2Providers))
	}

	p := cfg.OAuth2Providers[0]
	if p.Name != "github" {
		t.Errorf("expected provider name 'github', got '%s'", p.Name)
	}
	if p.ClientId != "test_client_id" {
		t.Errorf("expected ClientId 'test_client_id', got '%s'", p.ClientId)
	}
	if p.ClientSecret != "test_client_secret" {
		t.Errorf("expected ClientSecret 'test_client_secret', got '%s'", p.ClientSecret)
	}
	if p.DisplayName != "GitHub Custom" {
		t.Errorf("expected DisplayName 'GitHub Custom', got '%s'", p.DisplayName)
	}
	if p.PKCE == nil || !*p.PKCE {
		t.Errorf("expected PKCE to be true")
	}
}

func TestNewFromFlags_Json(t *testing.T) {
	// Create a temporary JSON config file
	tmpFile, err := os.CreateTemp("", "qpi_config_*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	jsonContent := `{
		"jobsCollection": "json_quantum_jobs",
		"portRangeStart": 8000,
		"oauth2Providers": [
			{
				"name": "google",
				"clientId": "google_id",
				"clientSecret": "google_secret"
			}
		]
	}`
	if _, err := tmpFile.Write([]byte(jsonContent)); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Bind flags to a dummy cobra command
	cmd := &cobra.Command{}
	BindFlags(cmd)

	// Set the config-file flag using the temp file
	if err := cmd.PersistentFlags().Set("config-file", tmpFile.Name()); err != nil {
		t.Fatalf("failed to set config-file flag: %v", err)
	}

	// Load configuration
	cfg, err := NewFromFlags(cmd)
	if err != nil {
		t.Fatalf("failed to load config from flags: %v", err)
	}

	// Verify loaded values
	if cfg.CollectionQuantumJobs != "json_quantum_jobs" {
		t.Errorf("expected CollectionQuantumJobs to be 'json_quantum_jobs', got '%s'", cfg.CollectionQuantumJobs)
	}
	if cfg.PortRangeStart != 8000 {
		t.Errorf("expected PortRangeStart to be 8000, got %d", cfg.PortRangeStart)
	}
	if len(cfg.OAuth2Providers) != 1 {
		t.Fatalf("expected 1 OAuth2 provider, got %d", len(cfg.OAuth2Providers))
	}

	p := cfg.OAuth2Providers[0]
	if p.Name != "google" {
		t.Errorf("expected provider name 'google', got '%s'", p.Name)
	}
	if p.ClientId != "google_id" {
		t.Errorf("expected ClientId 'google_id', got '%s'", p.ClientId)
	}
}
