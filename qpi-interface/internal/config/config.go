// Package config manages application-wide configuration parameters for the QPI orchestrator,
// including database collection names, NNG connection ports, and job timeout intervals.
// Configuration is saved on the pocketbase app store for concurrent-safe thread access.
// It supports a strict configuration precedence: CLI Flag > Env Var > Config File > Default.
package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const appStoreConfigKey = "custom_config"

// AppConfig stores application-wide configuration parameters for the QPI orchestrator.
type AppConfig struct {
	CollectionQPUs           string
	CollectionTimeSlots      string
	CollectionQuantumJobs    string
	IdleThreshold            time.Duration
	RecoveryInterval         time.Duration
	JobTimeout               time.Duration
	DispatchPollInterval     time.Duration
	PortRangeStart           int
	PortRangeEnd             int
	DisableEmailPasswordAuth bool
	OAuth2Providers          []core.OAuth2ProviderConfig
}

// SaveConfigOnApp saves the config on the app instance store.
func SaveConfigOnApp(app core.App, config *AppConfig) {
	app.Store().Set(appStoreConfigKey, config)
}

// GetConfigFromApp retrieves the config from the app instance store.
func GetConfigFromApp(app core.App) (*AppConfig, error) {
	value := app.Store().Get(appStoreConfigKey)
	config, ok := value.(*AppConfig)
	if !ok {
		return nil, fmt.Errorf("failed to retrieve AppConfig from app store")
	}
	return config, nil
}

// Package local flags populated by Cobra flag bindings.
var (
	flagConfigFile               string
	flagCollectionQPUs           string
	flagCollectionTimeSlots      string
	flagCollectionQuantumJobs    string
	flagIdleThreshold            time.Duration
	flagRecoveryInterval         time.Duration
	flagJobTimeout               time.Duration
	flagDispatchPollInterval     time.Duration
	flagPortRangeStart           int
	flagPortRangeEnd             int
	flagDisableEmailPasswordAuth bool
	flagOAuth2Providers          string // JSON array string
)

// BindFlags registers custom flags on the Cobra command.
func BindFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&flagConfigFile, "config-file", getEnvString("QPI_CONFIG_FILE", ""), "Path to QPI JSON configuration file")
	cmd.PersistentFlags().StringVar(&flagCollectionQPUs, "qpus-collection", "qpus", "Collection name for QPUs")
	cmd.PersistentFlags().StringVar(&flagCollectionTimeSlots, "timeslots-collection", "time_slots", "Collection name for Time Slots")
	cmd.PersistentFlags().StringVar(&flagCollectionQuantumJobs, "jobs-collection", "quantum_jobs", "Collection name for Quantum Jobs")
	cmd.PersistentFlags().DurationVar(&flagIdleThreshold, "idle-threshold", 5*time.Second, "Idle fallback threshold")
	cmd.PersistentFlags().DurationVar(&flagRecoveryInterval, "recovery-interval", 10*time.Second, "Stale job recovery check interval")
	cmd.PersistentFlags().DurationVar(&flagJobTimeout, "job-timeout", 20*time.Second, "Stale job execution timeout")
	cmd.PersistentFlags().DurationVar(&flagDispatchPollInterval, "dispatch-poll-interval", 1*time.Second, "Dispatch poll interval")
	cmd.PersistentFlags().IntVar(&flagPortRangeStart, "port-range-start", 6000, "NNG port range start")
	cmd.PersistentFlags().IntVar(&flagPortRangeEnd, "port-range-end", 7000, "NNG port range end")
	cmd.PersistentFlags().BoolVar(&flagDisableEmailPasswordAuth, "disable-email-password-auth", false, "Disable email/password auth on users collection")
	cmd.PersistentFlags().StringVar(&flagOAuth2Providers, "oauth2-providers", "", "JSON array of OAuth2 providers: '[{\"name\":\"github\",\"clientId\":\"...\",\"clientSecret\":\"...\",\"authURL\":\"...\",\"tokenURL\":\"...\",\"userInfoURL\":\"...\",\"displayName\":\"...\",\"pkce\":true}]'")
}

// NewFromFlags builds an AppConfig from the parsed CLI flags, optional config file, and environment variables.
// It enforces the precedence: CLI Flag (explicit) > Env Var > Config File > Default.
func NewFromFlags(cmd *cobra.Command) *AppConfig {
	cfg := &AppConfig{}

	// 1. Set hardcoded defaults
	cfg.CollectionQPUs = "qpus"
	cfg.CollectionTimeSlots = "time_slots"
	cfg.CollectionQuantumJobs = "quantum_jobs"
	cfg.IdleThreshold = 5 * time.Second
	cfg.RecoveryInterval = 10 * time.Second
	cfg.JobTimeout = 20 * time.Second
	cfg.DispatchPollInterval = 1 * time.Second
	cfg.PortRangeStart = 6000
	cfg.PortRangeEnd = 7000
	cfg.DisableEmailPasswordAuth = false

	// 2. Overlay Config File (if specified via env or flag)
	configFile := flagConfigFile
	if envVal := os.Getenv("QPI_CONFIG_FILE"); envVal != "" {
		configFile = envVal
	}
	if configFile != "" {
		if data, err := os.ReadFile(configFile); err == nil {
			type oauth2ProviderConfigLocal struct {
				Name         string         `json:"name" yaml:"name"`
				ClientId     string         `json:"clientId" yaml:"clientId"`
				ClientSecret string         `json:"clientSecret" yaml:"clientSecret"`
				AuthURL      string         `json:"authURL" yaml:"authURL"`
				TokenURL     string         `json:"tokenURL" yaml:"tokenURL"`
				UserInfoURL  string         `json:"userInfoURL" yaml:"userInfoURL"`
				DisplayName  string         `json:"displayName" yaml:"displayName"`
				PKCE         *bool          `json:"pkce" yaml:"pkce"`
				Extra        map[string]any `json:"extra" yaml:"extra"`
			}
			var fileCfg struct {
				CollectionQPUs           *string                     `json:"qpusCollection" yaml:"qpusCollection"`
				CollectionTimeSlots      *string                     `json:"timeslotsCollection" yaml:"timeslotsCollection"`
				CollectionQuantumJobs    *string                     `json:"jobsCollection" yaml:"jobsCollection"`
				IdleThreshold            *string                     `json:"idleThreshold" yaml:"idleThreshold"`
				RecoveryInterval         *string                     `json:"recoveryInterval" yaml:"recoveryInterval"`
				JobTimeout               *string                     `json:"jobTimeout" yaml:"jobTimeout"`
				DispatchPollInterval     *string                     `json:"dispatchPollInterval" yaml:"dispatchPollInterval"`
				PortRangeStart           *int                        `json:"portRangeStart" yaml:"portRangeStart"`
				PortRangeEnd             *int                        `json:"portRangeEnd" yaml:"portRangeEnd"`
				DisableEmailPasswordAuth *bool                       `json:"disableEmailPasswordAuth" yaml:"disableEmailPasswordAuth"`
				OAuth2Providers          []oauth2ProviderConfigLocal `json:"oauth2Providers" yaml:"oauth2Providers"`
			}

			var parseErr error
			isYaml := strings.HasSuffix(configFile, ".yaml") || strings.HasSuffix(configFile, ".yml")
			if isYaml {
				parseErr = yaml.Unmarshal(data, &fileCfg)
			} else {
				parseErr = json.Unmarshal(data, &fileCfg)
			}

			if parseErr == nil {
				if fileCfg.CollectionQPUs != nil {
					cfg.CollectionQPUs = *fileCfg.CollectionQPUs
				}
				if fileCfg.CollectionTimeSlots != nil {
					cfg.CollectionTimeSlots = *fileCfg.CollectionTimeSlots
				}
				if fileCfg.CollectionQuantumJobs != nil {
					cfg.CollectionQuantumJobs = *fileCfg.CollectionQuantumJobs
				}
				if fileCfg.IdleThreshold != nil {
					if d, err := time.ParseDuration(*fileCfg.IdleThreshold); err == nil {
						cfg.IdleThreshold = d
					}
				}
				if fileCfg.RecoveryInterval != nil {
					if d, err := time.ParseDuration(*fileCfg.RecoveryInterval); err == nil {
						cfg.RecoveryInterval = d
					}
				}
				if fileCfg.JobTimeout != nil {
					if d, err := time.ParseDuration(*fileCfg.JobTimeout); err == nil {
						cfg.JobTimeout = d
					}
				}
				if fileCfg.DispatchPollInterval != nil {
					if d, err := time.ParseDuration(*fileCfg.DispatchPollInterval); err == nil {
						cfg.DispatchPollInterval = d
					}
				}
				if fileCfg.PortRangeStart != nil {
					cfg.PortRangeStart = *fileCfg.PortRangeStart
				}
				if fileCfg.PortRangeEnd != nil {
					cfg.PortRangeEnd = *fileCfg.PortRangeEnd
				}
				if fileCfg.DisableEmailPasswordAuth != nil {
					cfg.DisableEmailPasswordAuth = *fileCfg.DisableEmailPasswordAuth
				}
				if len(fileCfg.OAuth2Providers) > 0 {
					cfg.OAuth2Providers = make([]core.OAuth2ProviderConfig, len(fileCfg.OAuth2Providers))
					for i, p := range fileCfg.OAuth2Providers {
						cfg.OAuth2Providers[i] = core.OAuth2ProviderConfig{
							Name:         p.Name,
							ClientId:     p.ClientId,
							ClientSecret: p.ClientSecret,
							AuthURL:      p.AuthURL,
							TokenURL:     p.TokenURL,
							UserInfoURL:  p.UserInfoURL,
							DisplayName:  p.DisplayName,
							PKCE:         p.PKCE,
							Extra:        p.Extra,
						}
					}
				}
			} else {
				log.Printf("Warning: failed to parse config file: %v", parseErr)
			}
		} else {
			log.Printf("Warning: failed to read config file %s", configFile)
		}
	}

	// Helper resolution functions to check precedence: CLI Changed > Env Set > Config File / Default
	resolveString := func(flagName, envName, current string) string {
		if cmd != nil && cmd.Flags().Changed(flagName) {
			val, _ := cmd.Flags().GetString(flagName)
			return val
		}
		if envVal := os.Getenv(envName); envVal != "" {
			return envVal
		}
		return current
	}

	resolveDuration := func(flagName, envName string, current time.Duration) time.Duration {
		if cmd != nil && cmd.Flags().Changed(flagName) {
			val, _ := cmd.Flags().GetDuration(flagName)
			return val
		}
		if envVal := os.Getenv(envName); envVal != "" {
			if d, err := time.ParseDuration(envVal); err == nil {
				return d
			}
		}
		return current
	}

	resolveInt := func(flagName, envName string, current int) int {
		if cmd != nil && cmd.Flags().Changed(flagName) {
			val, _ := cmd.Flags().GetInt(flagName)
			return val
		}
		if envVal := os.Getenv(envName); envVal != "" {
			if val, err := strconv.Atoi(envVal); err == nil {
				return val
			}
		}
		return current
	}

	resolveBool := func(flagName, envName string, current bool) bool {
		if cmd != nil && cmd.Flags().Changed(flagName) {
			val, _ := cmd.Flags().GetBool(flagName)
			return val
		}
		if envVal := os.Getenv(envName); envVal != "" {
			if val, err := strconv.ParseBool(envVal); err == nil {
				return val
			}
		}
		return current
	}

	// 3 & 4. Overlay Env and CLI precedence
	cfg.CollectionQPUs = resolveString("qpus-collection", "QPI_QPUS_COLLECTION", cfg.CollectionQPUs)
	cfg.CollectionTimeSlots = resolveString("timeslots-collection", "QPI_TIMESLOTS_COLLECTION", cfg.CollectionTimeSlots)
	cfg.CollectionQuantumJobs = resolveString("jobs-collection", "QPI_JOBS_COLLECTION", cfg.CollectionQuantumJobs)
	cfg.IdleThreshold = resolveDuration("idle-threshold", "QPI_IDLE_THRESHOLD", cfg.IdleThreshold)
	cfg.RecoveryInterval = resolveDuration("recovery-interval", "QPI_RECOVERY_INTERVAL", cfg.RecoveryInterval)
	cfg.JobTimeout = resolveDuration("job-timeout", "QPI_JOB_TIMEOUT", cfg.JobTimeout)
	cfg.DispatchPollInterval = resolveDuration("dispatch-poll-interval", "QPI_DISPATCH_POLL_INTERVAL", cfg.DispatchPollInterval)
	cfg.PortRangeStart = resolveInt("port-range-start", "QPI_PORT_RANGE_START", cfg.PortRangeStart)
	cfg.PortRangeEnd = resolveInt("port-range-end", "QPI_PORT_RANGE_END", cfg.PortRangeEnd)
	cfg.DisableEmailPasswordAuth = resolveBool("disable-email-password-auth", "QPI_DISABLE_EMAIL_PASSWORD_AUTH", cfg.DisableEmailPasswordAuth)

	// Merge OAuth2 providers JSON from CLI or Env
	oauth2ProvidersRaw := ""
	if cmd != nil && cmd.Flags().Changed("oauth2-providers") {
		oauth2ProvidersRaw, _ = cmd.Flags().GetString("oauth2-providers")
	} else if envVal := os.Getenv("QPI_OAUTH2_PROVIDERS"); envVal != "" {
		oauth2ProvidersRaw = envVal
	}
	if oauth2ProvidersRaw != "" {
		var providers []core.OAuth2ProviderConfig
		if err := json.Unmarshal([]byte(oauth2ProvidersRaw), &providers); err == nil {
			for _, p := range providers {
				found := false
				for i, existing := range cfg.OAuth2Providers {
					if existing.Name == p.Name {
						cfg.OAuth2Providers[i] = p
						found = true
						break
					}
				}
				if !found {
					cfg.OAuth2Providers = append(cfg.OAuth2Providers, p)
				}
			}
		} else {
			log.Printf("Warning: failed to parse OAuth2 providers: %v", err)
		}
	}

	return cfg
}

func getEnvString(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
