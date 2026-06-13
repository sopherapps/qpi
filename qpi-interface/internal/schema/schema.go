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

		// add extra fields in addition to the default ones
		collection.Fields.Add(
			&core.JSONField{
				Name: "api_tokens",
			},
		)
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

// ensureQPUsCollection creates the collection storing QPU hardware properties, statuses, and ports.
func ensureQPUsCollection(app core.App, cfg *config.AppConfig) error {
	if _, err := app.FindCollectionByNameOrId(cfg.CollectionQPUs); err == nil {
		return nil // already exists
	}
	col := core.NewBaseCollection(cfg.CollectionQPUs)
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
	return app.Save(col)
}

// ensureTimeSlotsCollection creates the collection storing calendar slot reservations for users.
func ensureTimeSlotsCollection(app core.App, cfg *config.AppConfig) error {
	if _, err := app.FindCollectionByNameOrId(cfg.CollectionTimeSlots); err == nil {
		return nil
	}
	col := core.NewBaseCollection(cfg.CollectionTimeSlots)
	col.Fields.Add(&core.DateField{Name: "start_time", Required: true})
	col.Fields.Add(&core.DateField{Name: "end_time", Required: true})

	// booked_by → relation to users collection (optional)
	usersCol, err := app.FindCollectionByNameOrId("users")
	if err == nil {
		col.Fields.Add(&core.RelationField{
			Name:         "booked_by",
			CollectionId: usersCol.Id,
			MaxSelect:    1,
		})
	}
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
