package db

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
	"qpi/internal/config"
)

// OnQPUStatusChange is a callback to avoid import cycles. It is populated by the api package.
var OnQPUStatusChange func(app core.App, qpuID string, enabled bool, cmdPort, resPort int)

// RegisterHooks registers database-level validation and hooks.
func RegisterHooks(app core.App) {
	app.OnRecordCreate().Bind(&hook.Handler[*core.RecordEvent]{
		Func: func(e *core.RecordEvent) error {
			cfg, err := config.GetConfigFromApp(e.App)
			if err != nil {
				return e.Next()
			}
			switch e.Record.Collection().Name {
			case cfg.CollectionTimeSlots:
				if err := ValidateTimeSlot(e.App, e.Record); err != nil {
					return err
				}
			case cfg.CollectionQPUs:
				token := e.Record.GetString("access_token")
				if token != "" {
					e.Record.Set("access_token", HashToken(token))
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
				if err := ValidateTimeSlot(e.App, e.Record); err != nil {
					return err
				}
			case cfg.CollectionQPUs:
				// Hash access_token if it changed
				originalToken := e.Record.Original().GetString("access_token")
				newToken := e.Record.GetString("access_token")
				if newToken != "" && newToken != originalToken {
					e.Record.Set("access_token", HashToken(newToken))
				}

				originalEnabled := e.Record.Original().GetBool("enabled")
				newEnabled := e.Record.GetBool("enabled")

				if originalEnabled != newEnabled {
					qpuID := e.Record.Id
					if OnQPUStatusChange != nil {
						OnQPUStatusChange(e.App, qpuID, newEnabled, e.Record.GetInt("nng_command_port"), e.Record.GetInt("nng_result_port"))
					}
					if !newEnabled {
						e.Record.Set("status", "offline")
					} else {
						cmdPort := e.Record.GetInt("nng_command_port")
						resPort := e.Record.GetInt("nng_result_port")
						if cmdPort > 0 && resPort > 0 {
							e.Record.Set("status", "online")
						}
					}
				}
			}
			return e.Next()
		},
	})
}
