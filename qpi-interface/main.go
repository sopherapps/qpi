// Package main is the entry point for the QPI orchestrator service, bootstrapping PocketBase,
// binding custom command line flags, and registering schemas, HTTP handlers, and recovery background tasks.
package main

import (
	"errors"
	"log"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"

	"qpi/internal/api"
	"qpi/internal/config"
	"qpi/internal/scheduler"
	"qpi/internal/schema"
)

var Version = "v0.0.1"

func main() {
	app := pocketbase.New()

	// Bind custom persistent CLI flags to configuration variables
	config.BindFlags(app.RootCmd)

	// set the current version of the application
	app.RootCmd.Version = Version

	// Bootstrap: create collections on first boot
	app.OnBootstrap().Bind(&hook.Handler[*core.BootstrapEvent]{
		Func: func(e *core.BootstrapEvent) error {
			// Populate and save AppConfig to the App store
			cfg := config.NewFromFlags(app.RootCmd)
			config.SaveConfigOnApp(e.App, cfg)

			if err := e.Next(); err != nil {
				return err
			}
			return schema.EnsureSchema(e.App)
		},
	})

	// Register custom HTTP routes & background tasks
	app.OnServe().Bind(&hook.Handler[*core.ServeEvent]{
		Func: func(e *core.ServeEvent) error {
			// Populate and save AppConfig to the App store to ensure it is always available
			cfg := config.NewFromFlags(app.RootCmd)
			config.SaveConfigOnApp(e.App, cfg)

			// Register api register handler routes
			api.RegisterRoutes(e)

			// Start the global recovery engine
			go scheduler.RunRecoveryEngine(e.App)

			return e.Next()
		},
	})

	// 1. Database-level validations for time slots (Interval order & Overlap check)
	app.OnRecordCreate("time_slots").Bind(&hook.Handler[*core.RecordEvent]{
		Func: func(e *core.RecordEvent) error {
			if err := validateTimeSlot(e.App, e.Record); err != nil {
				return err
			}
			return e.Next()
		},
	})

	app.OnRecordUpdate("time_slots").Bind(&hook.Handler[*core.RecordEvent]{
		Func: func(e *core.RecordEvent) error {
			if err := validateTimeSlot(e.App, e.Record); err != nil {
				return err
			}
			return e.Next()
		},
	})

	// 2. Request-level validations for time slots (Auto-population & Past-time check)
	app.OnRecordCreateRequest("time_slots").Bind(&hook.Handler[*core.RecordRequestEvent]{
		Func: func(e *core.RecordRequestEvent) error {
			if e.HasSuperuserAuth() {
				return e.Next()
			}
			if e.Auth != nil {
				e.Record.Set("booked_by", e.Auth.Id)
			}
			start := e.Record.GetDateTime("start_time").Time()
			if start.Before(time.Now()) {
				return e.Error(400, "Cannot book a slot in the past.", nil)
			}
			if err := validateTimeSlot(e.App, e.Record); err != nil {
				return e.Error(400, err.Error(), nil)
			}
			return e.Next()
		},
	})

	app.OnRecordUpdateRequest("time_slots").Bind(&hook.Handler[*core.RecordRequestEvent]{
		Func: func(e *core.RecordRequestEvent) error {
			if e.HasSuperuserAuth() {
				return e.Next()
			}
			originalStart := e.Record.Original().GetDateTime("start_time").Time()
			if originalStart.Before(time.Now()) {
				return e.Error(400, "Cannot modify a booking slot that has already started.", nil)
			}
			newStart := e.Record.GetDateTime("start_time").Time()
			if newStart.Before(time.Now()) {
				return e.Error(400, "Cannot reschedule a booking slot to a start time in the past.", nil)
			}
			if err := validateTimeSlot(e.App, e.Record); err != nil {
				return e.Error(400, err.Error(), nil)
			}
			return e.Next()
		},
	})

	app.OnRecordDeleteRequest("time_slots").Bind(&hook.Handler[*core.RecordRequestEvent]{
		Func: func(e *core.RecordRequestEvent) error {
			if e.HasSuperuserAuth() {
				return e.Next()
			}
			start := e.Record.GetDateTime("start_time").Time()
			if start.Before(time.Now()) {
				return e.Error(400, "Cannot delete/cancel a booking slot that has already started.", nil)
			}
			return e.Next()
		},
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

// validateTimeSlot asserts start_time is before end_time and prevents globally overlapping bookings.
func validateTimeSlot(app core.App, record *core.Record) error {
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
