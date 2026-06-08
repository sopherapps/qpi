// Package main is the entry point for the QPI orchestrator service, bootstrapping PocketBase,
// binding custom command line flags, and registering schemas, HTTP handlers, and recovery background tasks.
package main

import (
	"log"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"

	"qpi/internal/api"
	"qpi/internal/config"
	"qpi/internal/scheduler"
	"qpi/internal/schema"
)

func main() {
	app := pocketbase.New()

	// Bind custom persistent CLI flags to configuration variables
	config.BindFlags(app.RootCmd)

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

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
