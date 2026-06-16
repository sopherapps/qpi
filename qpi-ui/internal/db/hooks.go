package db

import (
	"qpi/internal/config"

	"github.com/pocketbase/pocketbase/core"
)

// CollectionHookMap is a map of collection names to hook functions.
type CollectionHookMap map[string]func(e *core.RecordEvent) error

// RegisterCollectionHooks returns a hook function for each default collection name in the hooks map
func RegisterCollectionHooks(app core.App, hooks CollectionHookMap) func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		cfg := config.MustGetConfigFromApp(app)
		col := e.Record.Collection().Name
		defaultColName := cfg.GetCollectionName(col)
		if fn, ok := hooks[defaultColName]; ok {
			return fn(e)
		}

		return e.Next()
	}
}

// OnTimeSlotUpsert runs on time slot creation or update
func OnTimeSlotUpsert(e *core.RecordEvent) error {
	if err := ValidateTimeSlot(e.App, e.Record); err != nil {
		return err
	}
	return e.Next()
}

// OnQpuCreate runs on QPU creation
func OnQpuCreate(e *core.RecordEvent) error {
	// Hash access_token if it exists
	token := e.Record.GetString("access_token")
	if token != "" {
		e.Record.Set("access_token", HashToken(token))
	}

	if !e.Record.GetBool("enabled") {
		e.Record.Set("status", "offline")
	}
	return e.Next()
}

// OnQpuUpdate runs on QPU update
func OnQpuUpdate(e *core.RecordEvent) error {
	// Hash access_token if it changed
	originalToken := e.Record.Original().GetString("access_token")
	newToken := e.Record.GetString("access_token")
	if newToken != "" && newToken != originalToken {
		e.Record.Set("access_token", HashToken(newToken))
	}

	originalEnabled := e.Record.Original().GetBool("enabled")
	newEnabled := e.Record.GetBool("enabled")

	if originalEnabled != newEnabled {
		if !newEnabled {
			e.Record.Set("status", "offline")
		} else {
			// check if it is connected on ports on this server
			cmdPort := e.Record.GetInt("nng_command_port")
			resPort := e.Record.GetInt("nng_result_port")
			if cmdPort > 0 && resPort > 0 {
				e.Record.Set("status", "online")
			}
		}
	}
	return e.Next()
}
