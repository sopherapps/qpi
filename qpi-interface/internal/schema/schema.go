// Package schema handles the automatic creation and verification of database schemas
// for the QPI orchestrator backend upon bootstrap.
package schema

import (
	"fmt"
	"log"

	"github.com/pocketbase/pocketbase/core"
	"qpi/internal/config"
)

// EnsureSchema bootstraps the database collections required by the QPI control stack.
// It creates the QPUs, Time Slots, and Quantum Jobs collections if they do not exist.
func EnsureSchema(app core.App) error {
	if err := ensureQPUsCollection(app); err != nil {
		return fmt.Errorf("qpus collection: %w", err)
	}
	if err := ensureTimeSlotsCollection(app); err != nil {
		return fmt.Errorf("time_slots collection: %w", err)
	}
	if err := ensureQuantumJobsCollection(app); err != nil {
		return fmt.Errorf("quantum_jobs collection: %w", err)
	}
	log.Println("[QPi] Schema OK")
	return nil
}

// ensureQPUsCollection creates the collection storing QPU hardware properties, statuses, and ports.
func ensureQPUsCollection(app core.App) error {
	if _, err := app.FindCollectionByNameOrId(config.CollectionQPUs); err == nil {
		return nil // already exists
	}
	col := core.NewBaseCollection(config.CollectionQPUs)
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
func ensureTimeSlotsCollection(app core.App) error {
	if _, err := app.FindCollectionByNameOrId(config.CollectionTimeSlots); err == nil {
		return nil
	}
	col := core.NewBaseCollection(config.CollectionTimeSlots)
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
func ensureQuantumJobsCollection(app core.App) error {
	if _, err := app.FindCollectionByNameOrId(config.CollectionQuantumJobs); err == nil {
		return nil
	}
	col := core.NewBaseCollection(config.CollectionQuantumJobs)

	usersCol, _ := app.FindCollectionByNameOrId("users")
	if usersCol != nil {
		col.Fields.Add(&core.RelationField{
			Name:         "user_id",
			CollectionId: usersCol.Id,
			MaxSelect:    1,
		})
	}

	qpuCol, _ := app.FindCollectionByNameOrId(config.CollectionQPUs)
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
		Values:    []string{"pending", "running", "completed", "failed"},
		MaxSelect: 1,
		Required:  true,
	})
	col.Fields.Add(&core.DateField{Name: "finished_at"})
	col.Fields.Add(&core.JSONField{Name: "results"})
	col.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
	col.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
	return app.Save(col)
}
