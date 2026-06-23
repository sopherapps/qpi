// Package main is the entry point for the QPI orchestrator service, bootstrapping PocketBase,
// binding custom command line flags, and registering dbs, HTTP handlers, and recovery background tasks.
package main

import (
	"context"
	"embed"
	"log"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"

	"qpi/internal/api"
	"qpi/internal/config"
	"qpi/internal/db"
	"qpi/internal/scheduler"
)

//go:embed all:internal/dashboard/dist
var dashboardFS embed.FS

var Version = "v0.0.15"

func main() {
	app := pocketbase.New()

	// Bind custom persistent CLI flags to configuration variables
	config.BindFlags(app.RootCmd)

	// set the current version of the application
	app.RootCmd.Version = Version

	appCtx, cancelAppCtx := context.WithCancel(context.Background())
	defer cancelAppCtx()

	// Bootstrap: create collections on first boot
	app.OnBootstrap().Bind(&hook.Handler[*core.BootstrapEvent]{
		Func: func(e *core.BootstrapEvent) error {
			// Populate and save AppConfig to the App store
			cfg, err := config.NewFromFlags(app.RootCmd)
			if err != nil {
				return err
			}

			cfg.StartTlsRenewalWorker(appCtx)

			config.SaveConfigOnApp(e.App, cfg)

			if err := e.Next(); err != nil {
				return err
			}
			return db.EnsureSchema(e.App)
		},
	})

	// Register custom HTTP routes & background tasks
	app.OnServe().Bind(&hook.Handler[*core.ServeEvent]{
		Func: func(e *core.ServeEvent) error {
			// Initialize the TLS
			err := api.SetupServer(e)
			if err != nil {
				return err
			}

			// Register api register handler routes
			api.RegisterRoutes(e, dashboardFS)

			// Start the global recovery engine
			go scheduler.RunRecoveryEngine(e.App)

			return e.Next()
		},
	})

	// for database
	app.OnRecordCreate().Bind(&hook.Handler[*core.RecordEvent]{
		Func: db.RegisterCollectionHooks(app, db.CollectionHookMap{
			config.DefaultTimeSlotsCollection: db.OnTimeSlotUpsert,
			config.DefaultQpusCollection:      db.OnQpuCreate,
		}),
	})

	app.OnRecordUpdate().Bind(&hook.Handler[*core.RecordEvent]{
		Func: db.RegisterCollectionHooks(app, db.CollectionHookMap{
			config.DefaultTimeSlotsCollection: db.OnTimeSlotUpsert,
			config.DefaultQpusCollection:      db.OnQpuUpdate,
		}),
	})

	// For requests
	app.OnRecordCreateRequest().Bind(&hook.Handler[*core.RecordRequestEvent]{
		Func: api.RegisterRequestHooks(app, api.RequestHookMap{
			config.DefaultTimeSlotsCollection:       api.OnTimeSlotCreateRequest,
			config.DefaultQPUTimeRequestsCollection: api.OnQPUTimeRequestCreateRequest,
		}),
	})

	app.OnRecordUpdateRequest().Bind(&hook.Handler[*core.RecordRequestEvent]{
		Func: api.RegisterRequestHooks(app, api.RequestHookMap{
			config.DefaultTimeSlotsCollection:       api.OnTimeSlotUpdateRequest,
			config.DefaultQPUTimeRequestsCollection: api.OnQPUTimeRequestUpdateRequest,
			config.DefaultQpusCollection:            api.OnQpuUpdateRequest,
		}),
	})

	app.OnRecordDeleteRequest().Bind(&hook.Handler[*core.RecordRequestEvent]{
		Func: api.RegisterRequestHooks(app, api.RequestHookMap{
			config.DefaultTimeSlotsCollection: api.OnTimeSlotDeleteRequest,
		}),
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
