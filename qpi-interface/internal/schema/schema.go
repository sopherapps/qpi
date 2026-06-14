// Package schema handles the automatic creation and verification of database schemas
// for the QPI orchestrator backend upon bootstrap.
package schema

import (
	"fmt"
	"log"

	"qpi/internal/config"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

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
		for _, f := range col.Fields {
			switch f.GetName() {
			case "num_qubits":
				hasNumQubits = true
			case "executor_type":
				hasExecutorType = true
			case "device_config":
				hasDeviceConfig = true
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
		return app.Save(col)
	}

	// Create new collection — use name as the primary key
	col = core.NewBaseCollection(cfg.CollectionQPUs)
	col.Id = "name"
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
	col.Fields.Add(&core.NumberField{Name: "num_qubits", Min: types.Pointer(1.0)})
	col.Fields.Add(&core.TextField{Name: "executor_type"})
	col.Fields.Add(&core.JSONField{Name: "device_config"})
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
	if _, err := app.FindCollectionByNameOrId(cfg.CollectionQuantumJobs); err == nil {
		return nil
	}
	col := core.NewBaseCollection(cfg.CollectionQuantumJobs)

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
	col.Fields.Add(&core.JSONField{Name: "results"})
	col.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
	col.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
	return app.Save(col)
}
