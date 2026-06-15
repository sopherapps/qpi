// Package schema handles the automatic creation and verification of database schemas
// for the QPI orchestrator backend upon bootstrap.
package schema

import (
	"errors"
	"fmt"
	"log"

	"qpi/internal/config"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

// -- Database collection structs ---------------------------------------------

// User represents a user record in the database.
type User struct {
	ID         string  `json:"id" db:"id"`
	Email      string  `json:"email" db:"email"`
	Username   string  `json:"username" db:"username"`
	QPUSeconds float64 `json:"qpu_seconds" db:"qpu_seconds"`
}

// APIToken represents an API token record in the database.
type APIToken struct {
	ID        string `json:"id" db:"id"`
	Token     string `json:"token" db:"token"`
	User      string `json:"user" db:"user"`
	ExpiresAt string `json:"expires_at,omitempty" db:"expires_at"`
	Name      string `json:"name" db:"name"`
}

// QPU represents a QPU record in the database.
type QPU struct {
	ID             string `json:"id" db:"id"`
	Name           string `json:"name" db:"name"`
	AccessToken    string `json:"access_token" db:"access_token"`
	Status         string `json:"status" db:"status"`
	NNGCommandPort int    `json:"nng_command_port" db:"nng_command_port"`
	NNGResultPort  int    `json:"nng_result_port" db:"nng_result_port"`
	NumQubits      int    `json:"num_qubits" db:"num_qubits"`
	ExecutorType   string `json:"executor_type" db:"executor_type"`
	DeviceConfig   any    `json:"device_config" db:"device_config"`
	Enabled        bool   `json:"enabled" db:"enabled"`
	Created        string `json:"created" db:"created"`
	Updated        string `json:"updated" db:"updated"`
	QpiAddr        string `json:"qpi_addr,omitempty" db:"-"`
}

// TimeSlot represents a calendar booking slot in the database.
type TimeSlot struct {
	ID        string `json:"id" db:"id"`
	StartTime string `json:"start_time" db:"start_time"`
	EndTime   string `json:"end_time" db:"end_time"`
	BookedBy  string `json:"booked_by" db:"booked_by"`
}

// QuantumJob represents a job record in the database.
type QuantumJob struct {
	ID         string  `json:"id" db:"id"`
	UserID     string  `json:"user_id" db:"user_id"`
	QPUTarget  string  `json:"qpu_target" db:"qpu_target"`
	Payload    any     `json:"payload" db:"payload"`
	Status     string  `json:"status" db:"status"`
	FinishedAt string  `json:"finished_at,omitempty" db:"finished_at"`
	Duration   float64 `json:"duration,omitempty" db:"duration"`
	Results    any     `json:"results,omitempty" db:"results"`
	Created    string  `json:"created" db:"created"`
	Updated    string  `json:"updated" db:"updated"`
}

// QPUTimeRequest represents a user request for QPU seconds in the database.
type QPUTimeRequest struct {
	ID              string  `json:"id" db:"id"`
	User            string  `json:"user" db:"user"`
	Seconds         float64 `json:"seconds" db:"seconds"`
	Status          string  `json:"status" db:"status"`
	RequestedReason string  `json:"requested_reason,omitempty" db:"requested_reason"`
	RejectionReason string  `json:"rejection_reason,omitempty" db:"rejection_reason"`
	HandledBy       string  `json:"handled_by,omitempty" db:"handled_by"`
}

// Notification represents an admin notification in the database.
type Notification struct {
	ID          string   `json:"id" db:"id"`
	Title       string   `json:"title" db:"title"`
	Description string   `json:"description" db:"description"`
	TargetUsers []string `json:"target_users,omitempty" db:"target_users"`
	DismissedBy []string `json:"dismissed_by,omitempty" db:"dismissed_by"`
	StartTime   string   `json:"start_time,omitempty" db:"start_time"`
	EndTime     string   `json:"end_time,omitempty" db:"end_time"`
	Created     string   `json:"created" db:"created"`
	Updated     string   `json:"updated" db:"updated"`
}

// -- API Payload Structs ----------------------------------------------------

// ConnectRequest represents the JSON payload passed to /api/op/qpus/connect.
type ConnectRequest struct {
	Name         string         `json:"name"`
	AccessToken  string         `json:"access_token"`
	ExecutorType string         `json:"executor_type,omitempty"`
	DeviceConfig map[string]any `json:"device_config,omitempty"`
}

// ConnectResponse represents the JSON payload returned by /api/op/qpus/connect.
type ConnectResponse struct {
	Status         string `json:"status"`
	NNGCommandPort int    `json:"nng_command_port"`
	NNGResultPort  int    `json:"nng_result_port"`
	AuthToken      string `json:"auth_token"`
}

// ResultPayload represents the NNG incoming message format for job execution results.
type ResultPayload struct {
	JobID   string         `json:"job_id"`
	Results map[string]any `json:"results"`
}

// CircuitPayload represents a single quantum circuit within a job submission.
type CircuitPayload struct {
	Circuit         string      `json:"circuit"`
	ParameterValues [][]float64 `json:"parameter_values,omitempty"`
	Shots           *int        `json:"shots,omitempty"`
}

// JobSubmitRequest represents the JSON payload for POST /api/jobs.
type JobSubmitRequest struct {
	Circuits   []CircuitPayload `json:"circuits"`
	Shots      int              `json:"shots"`
	MeasLevel  *int             `json:"meas_level,omitempty"`
	MeasReturn string           `json:"meas_return,omitempty"`
	QPUTarget  string           `json:"qpu_target,omitempty"`
}

// UserUpdateRequest represents the JSON payload for PATCH /api/admin/users/{id}.
type UserUpdateRequest struct {
	QpuSeconds *float64 `json:"qpu_seconds,omitempty"`
	APITokens  []string `json:"api_tokens,omitempty"`
}

// TokenCreateRequest represents the JSON payload for POST /api/tokens.
type TokenCreateRequest struct {
	Name      string `json:"name,omitempty"`
	ExpiresAt string `json:"expires_at,omitempty"` // ISO 8601 date string
}

// TokenCreateResponse represents the JSON payload returned by POST /api/tokens.
type TokenCreateResponse struct {
	ID        string `json:"id"`
	Token     string `json:"token"`
	Name      string `json:"name"`
	ExpiresAt string `json:"expires_at,omitempty"`
	Created   string `json:"created"`
}

// TokenUpdateRequest represents the JSON payload for PATCH /api/tokens/{id}.
type TokenUpdateRequest struct {
	Name      *string `json:"name,omitempty"`
	ExpiresAt *string `json:"expires_at,omitempty"`
}

// DismissRequest represents the JSON payload for POST /api/notifications/{id}/dismiss.
type DismissRequest struct {
	UserID string `json:"user_id,omitempty"`
}

// QPUToggleRequest represents the JSON payload for POST /api/op/qpu/toggle.
type QPUToggleRequest struct {
	ID      string `json:"id"`
	Enabled bool   `json:"enabled"`
}

// EnsureSchema bootstraps the database collections required by the QPI control stack.
// It creates the users, QPUs, Time Slots, and Quantum Jobs collections if they do not exist,
// configuring authentication options and properties based on the loaded AppConfig.
func EnsureSchema(app core.App) error {
	cfg, err := config.GetConfigFromApp(app)
	if err != nil {
		return err
	}

	if err := ensureUsersCollection(app, cfg); err != nil {
		return fmt.Errorf("users collection: %w", err)
	}
	if err := ensureAPITokensCollection(app, cfg); err != nil {
		return fmt.Errorf("api_tokens collection: %w", err)
	}
	if err := ensureQPUsCollection(app, cfg); err != nil {
		return fmt.Errorf("qpus collection: %w", err)
	}
	if err := ensureTimeSlotsCollection(app, cfg); err != nil {
		return fmt.Errorf("time_slots collection: %w", err)
	}
	if err := ensureQuantumJobsCollection(app, cfg); err != nil {
		return fmt.Errorf("quantum_jobs collection: %w", err)
	}
	if err := ensureQPUTimeRequestsCollection(app, cfg); err != nil {
		return fmt.Errorf("qpu_time_requests collection: %w", err)
	}
	if err := ensureNotificationsCollection(app, cfg); err != nil {
		return fmt.Errorf("notifications collection: %w", err)
	}

	log.Println("[QPi] Schema OK")
	return nil
}

// ensureUsersCollection modifies the default users collection to disable Email/Password auth if cfg.DisableEmailPasswordAuth
// and registers any specified OAuth2 providers.
func ensureUsersCollection(app core.App, cfg *config.AppConfig) error {
	log.Printf("Migrating users collection")

	collection, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		// Create users collection if it doesn't exist
		collection = core.NewAuthCollection("users")

		// restrict the rules for record owners
		collection.ListRule = types.Pointer("id = @request.auth.id")
		collection.ViewRule = types.Pointer("id = @request.auth.id")
		collection.UpdateRule = types.Pointer("id = @request.auth.id")
		collection.DeleteRule = types.Pointer("id = @request.auth.id")
	}

	// Ensure extra fields exist (idempotent migration)
	hasQpuSeconds := false
	for _, f := range collection.Fields {
		if f.GetName() == "qpu_seconds" {
			hasQpuSeconds = true
		}
	}
	if !hasQpuSeconds {
		collection.Fields.Add(
			&core.NumberField{
				Name: "qpu_seconds",
				Min:  types.Pointer(0.0),
			},
		)
	}

	// Disable Email/Password authentication if configured
	if cfg.DisableEmailPasswordAuth {
		collection.PasswordAuth.Enabled = false
	} else {
		collection.PasswordAuth.Enabled = true
	}

	// Configure OAuth2 providers if specified
	if len(cfg.OAuth2Providers) > 0 {
		collection.OAuth2.Enabled = true

		for _, providerCfg := range cfg.OAuth2Providers {
			log.Printf("Configuring OAuth2 provider %s for users collection", providerCfg.Name)

			// Idempotent update/append
			found := false
			for i, existing := range collection.OAuth2.Providers {
				if existing.Name == providerCfg.Name {
					collection.OAuth2.Providers[i] = providerCfg
					found = true
					break
				}
			}
			if !found {
				collection.OAuth2.Providers = append(collection.OAuth2.Providers, providerCfg)
			}
		}
	}

	return app.Save(collection)
}

// ensureAPITokensCollection creates the collection storing API tokens with
// user relation, optional expiry, and metadata.
func ensureAPITokensCollection(app core.App, cfg *config.AppConfig) error {
	col, err := app.FindCollectionByNameOrId(cfg.CollectionAPITokens)
	if err == nil {
		// Already exists — idempotent field check
		hasToken := false
		hasUser := false
		hasExpiresAt := false
		hasName := false
		for _, f := range col.Fields {
			switch f.GetName() {
			case "token":
				hasToken = true
			case "user":
				hasUser = true
			case "expires_at":
				hasExpiresAt = true
			case "name":
				hasName = true
			}
		}
		usersCol, _ := app.FindCollectionByNameOrId("users")
		if !hasToken {
			col.Fields.Add(&core.TextField{Name: "token", Required: true})
		}
		if !hasUser && usersCol != nil {
			col.Fields.Add(&core.RelationField{
				Name:         "user",
				CollectionId: usersCol.Id,
				MaxSelect:    1,
				Required:     true,
			})
		}
		if !hasExpiresAt {
			col.Fields.Add(&core.DateField{Name: "expires_at"})
		}
		if !hasName {
			col.Fields.Add(&core.TextField{Name: "name"})
		}
		// Set API rules: owner-only access
		col.ListRule = types.Pointer("user = @request.auth.id")
		col.ViewRule = types.Pointer("user = @request.auth.id")
		col.CreateRule = types.Pointer("@request.auth.id != \"\" && user = @request.auth.id")
		col.UpdateRule = types.Pointer("user = @request.auth.id")
		col.DeleteRule = types.Pointer("user = @request.auth.id")
		return app.Save(col)
	}

	// Create new collection
	col = core.NewBaseCollection(cfg.CollectionAPITokens)
	col.Fields.Add(&core.TextField{Name: "token", Required: true})

	usersCol, _ := app.FindCollectionByNameOrId("users")
	if usersCol != nil {
		col.Fields.Add(&core.RelationField{
			Name:         "user",
			CollectionId: usersCol.Id,
			MaxSelect:    1,
			Required:     true,
		})
	}
	col.Fields.Add(&core.DateField{Name: "expires_at"})
	col.Fields.Add(&core.TextField{Name: "name"})

	// Set API rules: owner-only access
	col.ListRule = types.Pointer("user = @request.auth.id")
	col.ViewRule = types.Pointer("user = @request.auth.id")
	col.CreateRule = types.Pointer("@request.auth.id != \"\" && user = @request.auth.id")
	col.UpdateRule = types.Pointer("user = @request.auth.id")
	col.DeleteRule = types.Pointer("user = @request.auth.id")

	return app.Save(col)
}

// ensureQPUsCollection creates the collection storing QPU hardware properties, statuses, and ports.
func ensureQPUsCollection(app core.App, cfg *config.AppConfig) error {
	col, err := app.FindCollectionByNameOrId(cfg.CollectionQPUs)
	if err == nil {
		// Idempotent field check — add missing fields to existing collection
		hasNumQubits := false
		hasExecutorType := false
		hasDeviceConfig := false
		hasEnabled := false
		for _, f := range col.Fields {
			switch f.GetName() {
			case "num_qubits":
				hasNumQubits = true
			case "executor_type":
				hasExecutorType = true
			case "device_config":
				hasDeviceConfig = true
			case "enabled":
				hasEnabled = true
			}
		}
		if !hasNumQubits {
			col.Fields.Add(&core.NumberField{Name: "num_qubits", Min: types.Pointer(1.0)})
		}
		if !hasExecutorType {
			col.Fields.Add(&core.TextField{Name: "executor_type"})
		}
		if !hasDeviceConfig {
			col.Fields.Add(&core.JSONField{Name: "device_config"})
		}
		if !hasEnabled {
			col.Fields.Add(&core.BoolField{Name: "enabled"})
		}
		// Public read, superuser-only CUD
		col.ListRule = types.Pointer("")
		col.ViewRule = types.Pointer("")
		col.CreateRule = nil
		col.UpdateRule = nil
		col.DeleteRule = nil
		return app.Save(col)
	}

	// Create new collection — use name as the primary key
	col = core.NewBaseCollection(cfg.CollectionQPUs)
	col.Id = "name"
	col.Fields.Add(&core.TextField{Name: "name", Required: true})
	col.Fields.Add(&core.TextField{Name: "access_token", Required: true})
	col.Fields.Add(&core.SelectField{
		Name:      "status",
		Values:    []string{"offline", "online", "maintenance"},
		MaxSelect: 1,
		Required:  true,
	})
	col.Fields.Add(&core.NumberField{Name: "nng_command_port"})
	col.Fields.Add(&core.NumberField{Name: "nng_result_port"})
	col.Fields.Add(&core.NumberField{Name: "num_qubits", Min: types.Pointer(1.0)})
	col.Fields.Add(&core.TextField{Name: "executor_type"})
	col.Fields.Add(&core.JSONField{Name: "device_config"})
	col.Fields.Add(&core.BoolField{Name: "enabled"})

	// Public read, superuser-only CUD
	col.ListRule = types.Pointer("")
	col.ViewRule = types.Pointer("")
	col.CreateRule = nil
	col.UpdateRule = nil
	col.DeleteRule = nil

	return app.Save(col)
}

// ensureTimeSlotsCollection creates/updates the collection storing calendar slot reservations for users.
func ensureTimeSlotsCollection(app core.App, cfg *config.AppConfig) error {
	col, err := app.FindCollectionByNameOrId(cfg.CollectionTimeSlots)
	if err != nil {
		col = core.NewBaseCollection(cfg.CollectionTimeSlots)
	}

	// Ensure fields
	hasStartTime := false
	hasEndTime := false
	hasBookedBy := false
	for _, f := range col.Fields {
		switch f.GetName() {
		case "start_time":
			hasStartTime = true
		case "end_time":
			hasEndTime = true
		case "booked_by":
			hasBookedBy = true
		}
	}

	if !hasStartTime {
		col.Fields.Add(&core.DateField{Name: "start_time", Required: true})
	}
	if !hasEndTime {
		col.Fields.Add(&core.DateField{Name: "end_time", Required: true})
	}

	usersCol, err := app.FindCollectionByNameOrId("users")
	if err == nil && !hasBookedBy {
		col.Fields.Add(&core.RelationField{
			Name:         "booked_by",
			CollectionId: usersCol.Id,
			MaxSelect:    1,
		})
	}

	// Set API Rules for user-level CRUD and administration
	col.ListRule = types.Pointer("@request.auth.id != \"\"")
	col.ViewRule = types.Pointer("@request.auth.id != \"\"")
	col.CreateRule = types.Pointer("@request.auth.id != \"\" && booked_by = @request.auth.id")
	col.UpdateRule = types.Pointer("booked_by = @request.auth.id")
	col.DeleteRule = types.Pointer("booked_by = @request.auth.id")

	return app.Save(col)
}

// ensureQuantumJobsCollection creates the collection storing jobs pending execution or containing results.
func ensureQuantumJobsCollection(app core.App, cfg *config.AppConfig) error {
	col, err := app.FindCollectionByNameOrId(cfg.CollectionQuantumJobs)
	if err == nil {
		// Collection exists — ensure rules are set
		col.ListRule = types.Pointer("user_id = @request.auth.id")
		col.ViewRule = types.Pointer("user_id = @request.auth.id")
		col.CreateRule = types.Pointer("@request.auth.id != \"\" && user_id = @request.auth.id")
		col.UpdateRule = nil
		col.DeleteRule = nil
		return app.Save(col)
	}
	col = core.NewBaseCollection(cfg.CollectionQuantumJobs)

	usersCol, _ := app.FindCollectionByNameOrId("users")
	if usersCol != nil {
		col.Fields.Add(&core.RelationField{
			Name:         "user_id",
			CollectionId: usersCol.Id,
			MaxSelect:    1,
		})
	}

	qpuCol, _ := app.FindCollectionByNameOrId(cfg.CollectionQPUs)
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
		Values:    []string{"pending", "running", "completed", "failed", "cancelled"},
		MaxSelect: 1,
		Required:  true,
	})
	col.Fields.Add(&core.DateField{Name: "finished_at"})
	col.Fields.Add(&core.NumberField{Name: "duration", Min: types.Pointer(0.0)})
	col.Fields.Add(&core.JSONField{Name: "results"})
	col.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
	col.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})

	// Owner-only read; authenticated create for self; no update/delete for regular users
	col.ListRule = types.Pointer("user_id = @request.auth.id")
	col.ViewRule = types.Pointer("user_id = @request.auth.id")
	col.CreateRule = types.Pointer("@request.auth.id != \"\" && user_id = @request.auth.id")
	col.UpdateRule = nil
	col.DeleteRule = nil

	return app.Save(col)
}

// ensureQPUTimeRequestsCollection creates/updates the collection storing QPU time requests by users.
func ensureQPUTimeRequestsCollection(app core.App, cfg *config.AppConfig) error {
	col, err := app.FindCollectionByNameOrId(cfg.CollectionQPUTimeRequests)
	if err != nil {
		col = core.NewBaseCollection(cfg.CollectionQPUTimeRequests)
	}

	// Ensure fields
	hasUser := false
	hasSeconds := false
	hasStatus := false
	hasRequestedReason := false
	hasRejectionReason := false
	hasHandledBy := false

	for _, f := range col.Fields {
		switch f.GetName() {
		case "user":
			hasUser = true
		case "seconds":
			hasSeconds = true
		case "status":
			hasStatus = true
		case "requested_reason":
			hasRequestedReason = true
		case "rejection_reason":
			hasRejectionReason = true
		case "handled_by":
			hasHandledBy = true
		}
	}

	usersCol, err := app.FindCollectionByNameOrId("users")
	if err == nil && !hasUser {
		col.Fields.Add(&core.RelationField{
			Name:         "user",
			CollectionId: usersCol.Id,
			MaxSelect:    1,
			Required:     true,
		})
	}

	if !hasSeconds {
		col.Fields.Add(&core.NumberField{
			Name:     "seconds",
			Required: true,
			Min:      types.Pointer(0.0),
		})
	}

	if !hasStatus {
		col.Fields.Add(&core.SelectField{
			Name:      "status",
			Values:    []string{"pending", "approved", "rejected"},
			MaxSelect: 1,
			Required:  true,
		})
	}

	if !hasRequestedReason {
		col.Fields.Add(&core.TextField{
			Name: "requested_reason",
		})
	}

	if !hasRejectionReason {
		col.Fields.Add(&core.TextField{
			Name: "rejection_reason",
		})
	}

	if !hasHandledBy {
		col.Fields.Add(&core.TextField{
			Name: "handled_by",
		})
	}

	// API Authorization Rules
	col.ListRule = types.Pointer("@request.auth.id != \"\" && user = @request.auth.id")
	col.ViewRule = types.Pointer("@request.auth.id != \"\" && user = @request.auth.id")
	col.CreateRule = types.Pointer("@request.auth.id != \"\" && user = @request.auth.id && status = \"pending\"")
	col.UpdateRule = nil // Disallowed for regular users; superusers bypass
	col.DeleteRule = types.Pointer("@request.auth.id != \"\" && user = @request.auth.id && status = \"pending\"")

	return app.Save(col)
}

// ensureNotificationsCollection creates/updates the collection storing admin notifications
// that can target specific users or all users (empty target_users = broadcast).
func ensureNotificationsCollection(app core.App, cfg *config.AppConfig) error {
	col, err := app.FindCollectionByNameOrId(cfg.CollectionNotifications)
	if err != nil {
		col = core.NewBaseCollection(cfg.CollectionNotifications)
	}

	hasTitle := false
	hasDescription := false
	hasTargetUsers := false
	hasStartTime := false
	hasEndTime := false
	hasDismissedBy := false
	hasCreated := false
	hasUpdated := false

	for _, f := range col.Fields {
		switch f.GetName() {
		case "title":
			hasTitle = true
		case "description":
			hasDescription = true
		case "target_users":
			hasTargetUsers = true
		case "start_time":
			hasStartTime = true
		case "end_time":
			hasEndTime = true
		case "dismissed_by":
			hasDismissedBy = true
		case "created":
			hasCreated = true
		case "updated":
			hasUpdated = true
		}
	}

	if !hasTitle {
		col.Fields.Add(&core.TextField{Name: "title", Required: true})
	}
	if !hasDescription {
		col.Fields.Add(&core.TextField{Name: "description"})
	}
	if !hasCreated {
		col.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
	}
	if !hasUpdated {
		col.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
	}

	usersCol, err := app.FindCollectionByNameOrId("users")
	if err == nil {
		if !hasTargetUsers {
			col.Fields.Add(&core.RelationField{
				Name:         "target_users",
				CollectionId: usersCol.Id,
				MaxSelect:    0, // 0 = unlimited
			})
		}
		if !hasDismissedBy {
			col.Fields.Add(&core.RelationField{
				Name:         "dismissed_by",
				CollectionId: usersCol.Id,
				MaxSelect:    0, // 0 = unlimited
			})
		}
	}

	if !hasStartTime {
		col.Fields.Add(&core.DateField{Name: "start_time"})
	}
	if !hasEndTime {
		col.Fields.Add(&core.DateField{Name: "end_time"})
	}

	// Visibility rules:
	// - authenticated users only
	// - target_users empty (broadcast) OR current user is in target_users
	// - within start_time / end_time window (if set)
	// - not dismissed by current user
	visibilityRule := "@request.auth.id != \"\" && " +
		"(@request.auth.id ?= target_users.id || target_users:length = 0) && " +
		"(start_time = '' || start_time <= @now) && " +
		"(end_time = '' || end_time >= @now) && " +
		"(dismissed_by:length = 0 || dismissed_by.id ?!= @request.auth.id)"

	col.ListRule = types.Pointer(visibilityRule)
	col.ViewRule = types.Pointer(visibilityRule)
	// nil = disabled for regular users; superusers bypass API rules
	col.CreateRule = nil
	col.UpdateRule = nil
	col.DeleteRule = nil

	return app.Save(col)
}

// -- Record to Struct Mapper Helpers ----------------------------------------

// UserFromRecord converts a PocketBase core.Record to a User struct.
func UserFromRecord(r *core.Record) User {
	if r == nil {
		return User{}
	}
	return User{
		ID:         r.Id,
		Email:      r.Email(),
		Username:   r.GetString("username"),
		QPUSeconds: r.GetFloat("qpu_seconds"),
	}
}

// APITokenFromRecord converts a PocketBase core.Record to an APIToken struct.
func APITokenFromRecord(r *core.Record) APIToken {
	if r == nil {
		return APIToken{}
	}
	return APIToken{
		ID:        r.Id,
		Token:     r.GetString("token"),
		User:      r.GetString("user"),
		ExpiresAt: r.GetString("expires_at"),
		Name:      r.GetString("name"),
	}
}

// QPUFromRecord converts a PocketBase core.Record to a QPU struct.
func QPUFromRecord(r *core.Record) QPU {
	if r == nil {
		return QPU{}
	}
	deviceConfig := r.Get("device_config")
	if deviceConfig == nil {
		deviceConfig = map[string]any{}
	}
	return QPU{
		ID:             r.Id,
		Name:           r.GetString("name"),
		AccessToken:    r.GetString("access_token"),
		Status:         r.GetString("status"),
		NNGCommandPort: r.GetInt("nng_command_port"),
		NNGResultPort:  r.GetInt("nng_result_port"),
		NumQubits:      r.GetInt("num_qubits"),
		ExecutorType:   r.GetString("executor_type"),
		DeviceConfig:   deviceConfig,
		Enabled:        r.GetBool("enabled"),
		Created:        r.GetString("created"),
		Updated:        r.GetString("updated"),
	}
}

// TimeSlotFromRecord converts a PocketBase core.Record to a TimeSlot struct.
func TimeSlotFromRecord(r *core.Record) TimeSlot {
	if r == nil {
		return TimeSlot{}
	}
	return TimeSlot{
		ID:        r.Id,
		StartTime: r.GetString("start_time"),
		EndTime:   r.GetString("end_time"),
		BookedBy:  r.GetString("booked_by"),
	}
}

// QuantumJobFromRecord converts a PocketBase core.Record to a QuantumJob struct.
func QuantumJobFromRecord(r *core.Record) QuantumJob {
	if r == nil {
		return QuantumJob{}
	}
	return QuantumJob{
		ID:         r.Id,
		UserID:     r.GetString("user_id"),
		QPUTarget:  r.GetString("qpu_target"),
		Payload:    r.Get("payload"),
		Status:     r.GetString("status"),
		FinishedAt: r.GetString("finished_at"),
		Duration:   r.GetFloat("duration"),
		Results:    r.Get("results"),
		Created:    r.GetString("created"),
		Updated:    r.GetString("updated"),
	}
}

// QPUTimeRequestFromRecord converts a PocketBase core.Record to a QPUTimeRequest struct.
func QPUTimeRequestFromRecord(r *core.Record) QPUTimeRequest {
	if r == nil {
		return QPUTimeRequest{}
	}
	return QPUTimeRequest{
		ID:              r.Id,
		User:            r.GetString("user"),
		Seconds:         r.GetFloat("seconds"),
		Status:          r.GetString("status"),
		RequestedReason: r.GetString("requested_reason"),
		RejectionReason: r.GetString("rejection_reason"),
		HandledBy:       r.GetString("handled_by"),
	}
}

// NotificationFromRecord converts a PocketBase core.Record to a Notification struct.
func NotificationFromRecord(r *core.Record) Notification {
	if r == nil {
		return Notification{}
	}
	return Notification{
		ID:          r.Id,
		Title:       r.GetString("title"),
		Description: r.GetString("description"),
		TargetUsers: r.GetStringSlice("target_users"),
		DismissedBy: r.GetStringSlice("dismissed_by"),
		StartTime:   r.GetString("start_time"),
		EndTime:     r.GetString("end_time"),
		Created:     r.GetString("created"),
		Updated:     r.GetString("updated"),
	}
}

// ValidateTimeSlot asserts start_time is before end_time and prevents globally overlapping bookings.
func ValidateTimeSlot(app core.App, record *core.Record) error {
	start := record.GetDateTime("start_time").Time()
	end := record.GetDateTime("end_time").Time()

	if start.After(end) || start.Equal(end) {
		return errors.New("start_time must be strictly before end_time")
	}

	// Query DB to check if there are globally overlapping slots
	query := app.DB().
		Select("count(*)").
		From("time_slots").
		Where(dbx.NewExp("start_time < {:end_time} AND end_time > {:start_time}", dbx.Params{
			"end_time":   end,
			"start_time": start,
		}))

	if record.Id != "" {
		query = query.AndWhere(dbx.NewExp("id != {:id}", dbx.Params{"id": record.Id}))
	}

	var count int
	if err := query.Row(&count); err != nil {
		return err
	}

	if count > 0 {
		return errors.New("The requested booking slot overlaps with an existing reservation.")
	}

	return nil
}
