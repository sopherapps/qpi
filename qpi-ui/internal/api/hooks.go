package api

import (
	"fmt"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
	"qpi/internal/config"
	"qpi/internal/db"
)

func init() {
	db.OnQPUStatusChange = func(app core.App, qpuID string, enabled bool, cmdPort, resPort int) {
		if !enabled {
			StopQPUDistribution(qpuID)
		} else {
			if cmdPort > 0 && resPort > 0 {
				StartQPUDistribution(app, qpuID, cmdPort, resPort)
			}
		}
	}
}

// RegisterHooks registers API-level request validation hooks.
func RegisterHooks(app core.App) {
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
				if err := db.ValidateTimeSlot(e.App, e.Record); err != nil {
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
				if err := db.ValidateTimeSlot(e.App, e.Record); err != nil {
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
}
