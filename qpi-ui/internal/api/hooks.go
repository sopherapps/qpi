package api

import (
	"fmt"
	"time"

	"qpi/internal/config"
	"qpi/internal/db"

	"github.com/pocketbase/pocketbase/core"
)

// RequestHookMap is a map of collection names to hook functions.
type RequestHookMap map[string]func(e *core.RecordRequestEvent) error

// RegisterRequestHooks returns a hook function for each default collection name in the hooks map
func RegisterRequestHooks(app core.App, hooks RequestHookMap) func(e *core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {
		cfg := config.MustGetConfigFromApp(app)
		col := e.Record.Collection().Name
		defaultColName := cfg.GetCollectionName(col)
		if fn, ok := hooks[defaultColName]; ok {
			return fn(e)
		}

		return e.Next()
	}
}

// OnTimeSlotCreateRequest occurs when a request is made to create a time slot.
func OnTimeSlotCreateRequest(e *core.RecordRequestEvent) error {
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

	return e.Next()
}

// OnTimeSlotUpdateRequest handles the event of receiving an update request for a time slot.
func OnTimeSlotUpdateRequest(e *core.RecordRequestEvent) error {
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
	return e.Next()
}

// OnTimeSlotDeleteRequest handles the event of receiving a delete request for a time slot.
func OnTimeSlotDeleteRequest(e *core.RecordRequestEvent) error {
	if e.HasSuperuserAuth() {
		return e.Next()
	}
	start := e.Record.GetDateTime("start_time").Time()
	if start.Before(time.Now()) {
		return e.Error(400, "Cannot delete/cancel a booking slot that has already started.", nil)
	}
	return e.Next()
}

// OnQPUTimeRequestCreateRequest occurs when a request is made to create a QPU time request.
func OnQPUTimeRequestCreateRequest(e *core.RecordRequestEvent) error {
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
	return e.Next()
}

// OnQPUTimeRequestUpdateRequest handles the event of receiving an update request for a QPU time request.
func OnQPUTimeRequestUpdateRequest(e *core.RecordRequestEvent) error {
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

	return e.Next()
}

// OnQpuUpdateRequest runs on QPU update request
func OnQpuUpdateRequest(e *core.RecordRequestEvent) error {
	if !e.HasSuperuserAuth() {
		return e.Error(403, "Only administrators can update QPU settings.", nil)
	}

	originalRecord := e.Record.Original()
	newRecord := e.Record

	originalEnabled := originalRecord.GetBool("enabled")
	newEnabled := newRecord.GetBool("enabled")
	qpuId := newRecord.Id

	if originalEnabled != newEnabled {
		if !newEnabled {
			StopQPUDistribution(qpuId)
		} else {
			// check if it is connected on ports on this server
			cmdPort := newRecord.GetInt("nng_command_port")
			if cmdPort <= 0 {
				cmdPort = originalRecord.GetInt("nng_command_port")
			}
			resPort := newRecord.GetInt("nng_result_port")
			if resPort <= 0 {
				resPort = originalRecord.GetInt("nng_result_port")
			}

			if cmdPort > 0 && resPort > 0 {
				StartQPUDistribution(e.App, qpuId, cmdPort, resPort)
			}
		}
	}
	return e.Next()
}

// OnBootstrap runs on application bootstrap
