// Package main is the entry point for the QPI orchestrator service, bootstrapping PocketBase,
// binding custom command line flags, and registering schemas, HTTP handlers, and recovery background tasks.
package main

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"

	"qpi/internal/api"
	"qpi/internal/config"
	"qpi/internal/scheduler"
	"qpi/internal/schema"
)

//go:embed all:internal/dashboard/dist
var dashboardFS embed.FS

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

			// Serve static dashboard
			subFS, err := fs.Sub(dashboardFS, "internal/dashboard/dist")
			if err != nil {
				return err
			}
			e.Router.GET("/dashboard/{path...}", apis.Static(subFS, true))

			// Start the global recovery engine
			go scheduler.RunRecoveryEngine(e.App)

			return e.Next()
		},
	})

	// 1. Database-level validations for time slots (Interval order & Overlap check)
	app.OnRecordCreate().Bind(&hook.Handler[*core.RecordEvent]{
		Func: func(e *core.RecordEvent) error {
			cfg, err := config.GetConfigFromApp(e.App)
			if err != nil {
				return e.Next()
			}
			switch e.Record.Collection().Name {
			case cfg.CollectionTimeSlots:
				if err := validateTimeSlot(e.App, e.Record); err != nil {
					return err
				}
			case cfg.CollectionQPUs:
				token := e.Record.GetString("registration_token")
				if token != "" {
					e.Record.Set("registration_token", api.HashToken(token))
				}
				if !e.Record.GetBool("enabled") {
					e.Record.Set("status", "offline")
				}
			}
			return e.Next()
		},
	})

	app.OnRecordUpdate().Bind(&hook.Handler[*core.RecordEvent]{
		Func: func(e *core.RecordEvent) error {
			cfg, err := config.GetConfigFromApp(e.App)
			if err != nil {
				return e.Next()
			}
			switch e.Record.Collection().Name {
			case cfg.CollectionTimeSlots:
				if err := validateTimeSlot(e.App, e.Record); err != nil {
					return err
				}
			case cfg.CollectionQPUs:
				// Hash registration_token if it changed
				originalToken := e.Record.Original().GetString("registration_token")
				newToken := e.Record.GetString("registration_token")
				if newToken != "" && newToken != originalToken {
					e.Record.Set("registration_token", api.HashToken(newToken))
				}

				originalEnabled := e.Record.Original().GetBool("enabled")
				newEnabled := e.Record.GetBool("enabled")

				if originalEnabled != newEnabled {
					qpuID := e.Record.Id
					if !newEnabled {
						api.StopQPUDistribution(qpuID)
						e.Record.Set("status", "offline")
					} else {
						cmdPort := e.Record.GetInt("nng_command_port")
						resPort := e.Record.GetInt("nng_result_port")
						if cmdPort > 0 && resPort > 0 {
							api.StartQPUDistribution(e.App, qpuID, cmdPort, resPort)
							e.Record.Set("status", "online")
						}
					}
				}
			}
			return e.Next()
		},
	})

	// 2. Request-level validations
	app.OnRecordCreateRequest().Bind(&hook.Handler[*core.RecordRequestEvent]{
		Func: func(e *core.RecordRequestEvent) error {
			cfg, err := config.GetConfigFromApp(e.App)
			if err != nil {
				return e.Next()
			}
			switch e.Record.Collection().Name {
			case cfg.CollectionTimeSlots:
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
			case cfg.CollectionQPUTimeRequests:
				if e.HasSuperuserAuth() {
					return e.Next()
				}
				if e.Auth != nil {
					e.Record.Set("user", e.Auth.Id)
				} else {
					return e.Error(401, "Authentication required.", nil)
				}
				e.Record.Set("status", "pending")
				e.Record.Set("handled_by", "")
			}
			return e.Next()
		},
	})

	app.OnRecordUpdateRequest().Bind(&hook.Handler[*core.RecordRequestEvent]{
		Func: func(e *core.RecordRequestEvent) error {
			cfg, err := config.GetConfigFromApp(e.App)
			if err != nil {
				return e.Next()
			}
			switch e.Record.Collection().Name {
			case cfg.CollectionTimeSlots:
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
			case cfg.CollectionQPUTimeRequests:
				if !e.HasSuperuserAuth() {
					return e.Error(403, "Only administrators can update time requests.", nil)
				}

				originalStatus := e.Record.Original().GetString("status")
				newStatus := e.Record.GetString("status")

				originalUser := e.Record.Original().GetString("user")
				newUser := e.Record.GetString("user")
				if originalUser != newUser {
					return e.Error(400, "Cannot modify the user of a time request.", nil)
				}

				originalSeconds := e.Record.Original().GetFloat("seconds")
				newSeconds := e.Record.GetFloat("seconds")
				if originalSeconds != newSeconds {
					return e.Error(400, "Cannot modify the requested seconds.", nil)
				}

				if originalStatus != newStatus {
					if originalStatus == "approved" || originalStatus == "rejected" {
						return e.Error(400, "Cannot update a request that has already been processed.", nil)
					}

					if newStatus == "approved" {
						userId := e.Record.GetString("user")
						seconds := e.Record.GetFloat("seconds")

						userRec, err := e.App.FindRecordById("users", userId)
						if err != nil {
							return e.Error(400, fmt.Sprintf("Failed to find target user: %v", err), err)
						}

						currentSeconds := userRec.GetFloat("qpu_seconds")
						userRec.Set("qpu_seconds", currentSeconds+seconds)

						if err := e.App.Save(userRec); err != nil {
							return e.Error(500, fmt.Sprintf("Failed to credit QPU time to user: %v", err), err)
						}
					}

					adminId := "admin"
					if e.Auth != nil {
						adminId = e.Auth.Id
					}
					e.Record.Set("handled_by", adminId)
				}
			}
			return e.Next()
		},
	})

	app.OnRecordDeleteRequest().Bind(&hook.Handler[*core.RecordRequestEvent]{
		Func: func(e *core.RecordRequestEvent) error {
			cfg, err := config.GetConfigFromApp(e.App)
			if err != nil {
				return e.Next()
			}
			switch e.Record.Collection().Name {
			case cfg.CollectionTimeSlots:
				if e.HasSuperuserAuth() {
					return e.Next()
				}
				start := e.Record.GetDateTime("start_time").Time()
				if start.Before(time.Now()) {
					return e.Error(400, "Cannot delete/cancel a booking slot that has already started.", nil)
				}
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
