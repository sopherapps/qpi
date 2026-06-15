package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"qpi/internal/config"

	"github.com/pocketbase/pocketbase/core"
)

type DbModel interface {
	// ToRecord converts this model into a pocketbase record
	ToRecord(app core.App) (*core.Record, error)

	// Refresh self using the values from a pocketbase record
	RefreshFromRecord(record *core.Record) error
}

// getOrCreateRecord loads an existing record by id from the database, or initializes a new one.
func getOrCreateRecord(app core.App, collectionName string, id string, col *core.Collection) (*core.Record, error) {
	if id != "" {
		record, err := app.FindRecordById(collectionName, id)
		if err == nil {
			return record, nil
		}
	}
	record := core.NewRecord(col)
	if id != "" {
		record.Set("id", id)
	}
	return record, nil
}

// User represents a user record in the database.
type User struct {
	ID         string  `json:"id" db:"id"`
	Email      string  `json:"email" db:"email"`
	Username   string  `json:"username" db:"username"`
	QPUSeconds float64 `json:"qpu_seconds" db:"qpu_seconds"`
}

// ToRecord converts this model into a pocketbase record
func (u *User) ToRecord(app core.App) (*core.Record, error) {
	if u == nil {
		return nil, nil
	}

	col_name := "users"
	col, err := app.FindCollectionByNameOrId(col_name)
	if err != nil {
		return nil, fmt.Errorf("error finding collection %s: %w", col_name, err)
	}

	record, err := getOrCreateRecord(app, col_name, u.ID, col)
	if err != nil {
		return nil, err
	}
	record.Set("email", u.Email)
	record.Set("username", u.Username)
	record.Set("qpu_seconds", u.QPUSeconds)

	return record, nil
}

// RefreshFromRecord updates this model using the values from a pocketbase record
func (u *User) RefreshFromRecord(record *core.Record) error {
	if u == nil || record == nil {
		return errors.New("cannot refresh from nil record")
	}

	u.ID = record.Id
	u.Email = record.Email()
	u.Username = record.GetString("username")
	u.QPUSeconds = record.GetFloat("qpu_seconds")

	return nil
}

// APIToken represents an API token record in the database.
type APIToken struct {
	ID        string `json:"id" db:"id"`
	Token     string `json:"token" db:"token" required:"true" maxSelect:"1"`
	User      string `json:"user" db:"user" type:"relation" required:"true" maxSelect:"1" collection:"users"`
	ExpiresAt string `json:"expires_at,omitempty" db:"expires_at" type:"date"`
	Created   string `json:"created" db:"created" type:"autodate" onCreate:"true"`
	Name      string `json:"name" db:"name"`
}

// ToRecord converts this model into a pocketbase record
func (at *APIToken) ToRecord(app core.App) (*core.Record, error) {
	if at == nil {
		return nil, nil
	}
	cfg, err := config.GetConfigFromApp(app)
	if err != nil {
		return nil, err
	}

	col_name := cfg.CollectionAPITokens
	col, err := app.FindCollectionByNameOrId(col_name)
	if err != nil {
		return nil, fmt.Errorf("error finding collection %s: %w", col_name, err)
	}

	record, err := getOrCreateRecord(app, col_name, at.ID, col)
	if err != nil {
		return nil, err
	}
	record.Set("token", at.Token)
	record.Set("user", at.User)
	record.Set("expires_at", at.ExpiresAt)
	record.Set("name", at.Name)

	return record, nil
}

// RefreshFromRecord updates this model using the values from a pocketbase record
func (at *APIToken) RefreshFromRecord(record *core.Record) error {
	if at == nil || record == nil {
		return errors.New("cannot refresh from nil record")
	}

	at.ID = record.Id
	at.Token = record.GetString("token")
	at.User = record.GetString("user")
	at.ExpiresAt = record.GetString("expires_at")
	at.Name = record.GetString("name")

	return nil
}

// QPU represents a QPU record in the database.
type QPU struct {
	ID             string `json:"id" db:"id"`
	Name           string `json:"name" db:"name" required:"true" primaryKey:"true"`
	AccessToken    string `json:"access_token" db:"access_token" required:"true"`
	Status         string `json:"status" db:"status" type:"select" required:"true" maxSelect:"1" values:"offline,online,maintenance"`
	NNGCommandPort int    `json:"nng_command_port" db:"nng_command_port"`
	NNGResultPort  int    `json:"nng_result_port" db:"nng_result_port"`
	NumQubits      int    `json:"num_qubits" db:"num_qubits" min:"1.0"`
	ExecutorType   string `json:"executor_type" db:"executor_type"`
	DeviceConfig   any    `json:"device_config" db:"device_config" type:"json"`
	Enabled        bool   `json:"enabled" db:"enabled"`
	Created        string `json:"created" db:"created"`
	Updated        string `json:"updated" db:"updated"`
	QpiAddr        string `json:"qpi_addr,omitempty" db:""`
}

// ToRecord converts this model into a pocketbase record
func (q *QPU) ToRecord(app core.App) (*core.Record, error) {
	if q == nil {
		return nil, nil
	}
	cfg, err := config.GetConfigFromApp(app)
	if err != nil {
		return nil, err
	}

	col_name := cfg.CollectionQPUs
	col, err := app.FindCollectionByNameOrId(col_name)
	if err != nil {
		return nil, fmt.Errorf("error finding collection %s: %w", col_name, err)
	}

	record, err := getOrCreateRecord(app, col_name, q.ID, col)
	if err != nil {
		return nil, err
	}
	record.Set("name", q.Name)
	record.Set("access_token", q.AccessToken)
	record.Set("status", q.Status)
	record.Set("nng_command_port", q.NNGCommandPort)
	record.Set("nng_result_port", q.NNGResultPort)
	record.Set("num_qubits", q.NumQubits)
	record.Set("executor_type", q.ExecutorType)
	record.Set("enabled", q.Enabled)
	record.Set("created", q.Created)
	record.Set("updated", q.Updated)

	if q.DeviceConfig != nil {
		deviceConfigJSON, err := json.Marshal(q.DeviceConfig)
		if err != nil {
			return nil, fmt.Errorf("error marshaling device config: %w", err)
		}
		record.Set("device_config", deviceConfigJSON)
	}

	return record, nil
}

// RefreshFromRecord updates this model using the values from a pocketbase record
func (q *QPU) RefreshFromRecord(record *core.Record) error {
	if q == nil || record == nil {
		return errors.New("cannot refresh from nil record")
	}

	q.ID = record.Id
	q.Name = record.GetString("name")
	q.AccessToken = record.GetString("access_token")
	q.Status = record.GetString("status")
	q.NNGCommandPort = record.GetInt("nng_command_port")
	q.NNGResultPort = record.GetInt("nng_result_port")
	q.NumQubits = record.GetInt("num_qubits")
	q.ExecutorType = record.GetString("executor_type")
	q.DeviceConfig = record.Get("device_config")
	q.Enabled = record.GetBool("enabled")
	q.Created = record.GetString("created")
	q.Updated = record.GetString("updated")

	return nil
}

// TimeSlot represents a calendar booking slot in the database.
type TimeSlot struct {
	ID        string `json:"id" db:"id"`
	StartTime string `json:"start_time" type:"date" db:"start_time" required:"true"`
	EndTime   string `json:"end_time" type:"date" db:"end_time" required:"true"`
	BookedBy  string `json:"booked_by" db:"booked_by" type:"relation" required:"true" maxSelect:"1" collection:"users"`
}

// ToRecord converts this model into a pocketbase record
func (ts *TimeSlot) ToRecord(app core.App) (*core.Record, error) {
	if ts == nil {
		return nil, nil
	}
	cfg, err := config.GetConfigFromApp(app)
	if err != nil {
		return nil, err
	}

	col_name := cfg.CollectionTimeSlots
	col, err := app.FindCollectionByNameOrId(col_name)
	if err != nil {
		return nil, fmt.Errorf("error finding collection %s: %w", col_name, err)
	}

	record, err := getOrCreateRecord(app, col_name, ts.ID, col)
	if err != nil {
		return nil, err
	}
	record.Set("start_time", ts.StartTime)
	record.Set("end_time", ts.EndTime)
	record.Set("booked_by", ts.BookedBy)

	return record, nil
}

// RefreshFromRecord updates this model using the values from a pocketbase record
func (ts *TimeSlot) RefreshFromRecord(record *core.Record) error {
	if ts == nil || record == nil {
		return errors.New("cannot refresh from nil record")
	}

	ts.ID = record.Id
	ts.StartTime = record.GetString("start_time")
	ts.EndTime = record.GetString("end_time")
	ts.BookedBy = record.GetString("booked_by")

	return nil
}

// QuantumJob represents a job record in the database.
type QuantumJob struct {
	ID         string  `json:"id" db:"id"`
	UserID     string  `json:"user_id" db:"user_id" type:"relation" maxSelect:"1" collection:"users"`
	QPUTarget  string  `json:"qpu_target" db:"qpu_target" type:"relation" maxSelect:"1" collection:"qpus"`
	Payload    any     `json:"payload" db:"payload" type:"json"`
	Status     string  `json:"status" db:"status" type:"select" maxSelect:"1" values:"pending,running,completed,failed,cancelled" required:"true"`
	FinishedAt string  `json:"finished_at,omitempty" db:"finished_at" type:"date"`
	Duration   float64 `json:"duration,omitempty" db:"duration" min:"0.0"`
	Results    any     `json:"results,omitempty" db:"results" type:"json"`
	Created    string  `json:"created" db:"created" type:"autodate" onCreate:"true"`
	Updated    string  `json:"updated" db:"updated" type:"autodate" onCreate:"true" onUpdate:"true"`
}

// ToRecord converts this model into a pocketbase record
func (qj *QuantumJob) ToRecord(app core.App) (*core.Record, error) {
	if qj == nil {
		return nil, nil
	}
	cfg, err := config.GetConfigFromApp(app)
	if err != nil {
		return nil, err
	}

	col_name := cfg.CollectionQuantumJobs
	col, err := app.FindCollectionByNameOrId(col_name)
	if err != nil {
		return nil, fmt.Errorf("error finding collection %s: %w", col_name, err)
	}

	record, err := getOrCreateRecord(app, col_name, qj.ID, col)
	if err != nil {
		return nil, err
	}
	record.Set("user_id", qj.UserID)
	record.Set("qpu_target", qj.QPUTarget)
	record.Set("status", qj.Status)
	record.Set("finished_at", qj.FinishedAt)
	record.Set("duration", qj.Duration)
	record.Set("created", qj.Created)
	record.Set("updated", qj.Updated)

	if qj.Payload != nil {
		payloadJSON, err := json.Marshal(qj.Payload)
		if err != nil {
			return nil, fmt.Errorf("error marshaling payload: %w", err)
		}
		record.Set("payload", payloadJSON)
	}

	if qj.Results != nil {
		resultsJSON, err := json.Marshal(qj.Results)
		if err != nil {
			return nil, fmt.Errorf("error marshaling results: %w", err)
		}
		record.Set("results", resultsJSON)
	}

	return record, nil
}

// RefreshFromRecord updates this model using the values from a pocketbase record
func (qj *QuantumJob) RefreshFromRecord(record *core.Record) error {
	if qj == nil || record == nil {
		return errors.New("cannot refresh from nil record")
	}

	qj.ID = record.Id
	qj.UserID = record.GetString("user_id")
	qj.QPUTarget = record.GetString("qpu_target")
	qj.Status = record.GetString("status")
	qj.FinishedAt = record.GetString("finished_at")
	qj.Duration = record.GetFloat("duration")
	qj.Created = record.GetString("created")
	qj.Updated = record.GetString("updated")
	qj.Payload = record.Get("payload")
	qj.Results = record.Get("payload")

	return nil
}

// QPUTimeRequest represents a user request for QPU seconds in the database.
type QPUTimeRequest struct {
	ID              string  `json:"id" db:"id"`
	User            string  `json:"user" db:"user" type:"relation" maxSelect:"1" collection:"users"`
	Seconds         float64 `json:"seconds" db:"seconds" min:"0.0" required:"true"`
	Status          string  `json:"status" db:"status" type:"select" maxSelect:"1" values:"pending,approved,rejected" required:"true"`
	RequestedReason string  `json:"requested_reason,omitempty" db:"requested_reason"`
	RejectionReason string  `json:"rejection_reason,omitempty" db:"rejection_reason"`
	HandledBy       string  `json:"handled_by,omitempty" db:"handled_by"`
}

// ToRecord converts this model into a pocketbase record
func (qtr *QPUTimeRequest) ToRecord(app core.App) (*core.Record, error) {
	if qtr == nil {
		return nil, nil
	}
	cfg, err := config.GetConfigFromApp(app)
	if err != nil {
		return nil, err
	}

	col_name := cfg.CollectionQPUTimeRequests
	col, err := app.FindCollectionByNameOrId(col_name)
	if err != nil {
		return nil, fmt.Errorf("error finding collection %s: %w", col_name, err)
	}

	record, err := getOrCreateRecord(app, col_name, qtr.ID, col)
	if err != nil {
		return nil, err
	}
	record.Set("user", qtr.User)
	record.Set("seconds", qtr.Seconds)
	record.Set("status", qtr.Status)
	record.Set("requested_reason", qtr.RequestedReason)
	record.Set("rejection_reason", qtr.RejectionReason)
	record.Set("handled_by", qtr.HandledBy)

	return record, nil
}

// RefreshFromRecord updates this model using the values from a pocketbase record
func (qtr *QPUTimeRequest) RefreshFromRecord(record *core.Record) error {
	if qtr == nil || record == nil {
		return errors.New("cannot refresh from nil record")
	}

	qtr.ID = record.Id
	qtr.User = record.GetString("user")
	qtr.Seconds = record.GetFloat("seconds")
	qtr.Status = record.GetString("status")
	qtr.RequestedReason = record.GetString("requested_reason")
	qtr.RejectionReason = record.GetString("rejection_reason")
	qtr.HandledBy = record.GetString("handled_by")

	return nil
}

// Notification represents an admin notification in the database.
type Notification struct {
	ID          string   `json:"id" db:"id"`
	Title       string   `json:"title" db:"title" required:"true"`
	Description string   `json:"description" db:"description"`
	TargetUsers []string `json:"target_users,omitempty" db:"target_users" type:"relation" maxSelect:"0" collection:"users"`
	DismissedBy []string `json:"dismissed_by,omitempty" db:"dismissed_by" type:"relation" maxSelect:"0" collection:"users"`
	StartTime   string   `json:"start_time,omitempty" db:"start_time" type:"date"`
	EndTime     string   `json:"end_time,omitempty" db:"end_time" type:"date"`
	Created     string   `json:"created" db:"created" type:"autodate" onCreate:"true"`
	Updated     string   `json:"updated" db:"updated" type:"autodate" onCreate:"true" onUpdate:"true"`
}

// ToRecord converts this model into a pocketbase record
func (n *Notification) ToRecord(app core.App) (*core.Record, error) {
	if n == nil {
		return nil, nil
	}
	cfg, err := config.GetConfigFromApp(app)
	if err != nil {
		return nil, err
	}

	col_name := cfg.CollectionNotifications
	col, err := app.FindCollectionByNameOrId(col_name)
	if err != nil {
		return nil, fmt.Errorf("error finding collection %s: %w", col_name, err)
	}

	record, err := getOrCreateRecord(app, col_name, n.ID, col)
	if err != nil {
		return nil, err
	}
	record.Set("title", n.Title)
	record.Set("description", n.Description)
	record.Set("target_users", n.TargetUsers)
	record.Set("dismissed_by", n.DismissedBy)
	record.Set("start_time", n.StartTime)
	record.Set("end_time", n.EndTime)
	record.Set("created", n.Created)
	record.Set("updated", n.Updated)

	return record, nil
}

// RefreshFromRecord updates this model using the values from a pocketbase record
func (n *Notification) RefreshFromRecord(record *core.Record) error {
	if n == nil || record == nil {
		return errors.New("cannot refresh from nil record")
	}

	n.ID = record.Id
	n.Title = record.GetString("title")
	n.Description = record.GetString("description")
	n.TargetUsers = record.GetStringSlice("target_users")
	n.DismissedBy = record.GetStringSlice("dismissed_by")
	n.StartTime = record.GetString("start_time")
	n.EndTime = record.GetString("end_time")
	n.Created = record.GetString("created")
	n.Updated = record.GetString("updated")

	return nil
}
