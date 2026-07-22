package api

import (
	"encoding/json"
	"time"

	"qpi/internal/db"
)

// EventType identifies one of the fixed set of typed messages exchanged with a
// driver over NNG. The set is defined per QPI-UI version and known at compile
// time; QPI-UI holds a server-side handler for each type (RFC 0001 §4, §7).
type EventType string

const (
	// EventJobDispatch is pushed from QPI-UI to a driver to run a job.
	EventJobDispatch EventType = "JobDispatch"
	// EventJobResult is emitted by a driver back to QPI-UI with a job's outcome.
	EventJobResult EventType = "JobResult"
)

// AllEventTypes lists every event type QPI-UI knows about in this version.
// Registration validates a custom driver's chosen events against this list.
var AllEventTypes = []EventType{EventJobDispatch, EventJobResult}

// Event is the single envelope carried on the wire in either direction between
// QPI-UI and a driver (RFC 0001 §6). Payload is left as raw JSON because its
// shape depends on Type and is validated by the handler that receives it.
type Event struct {
	ID      string          `json:"id"`
	Driver  string          `json:"driver"`
	Type    EventType       `json:"type"`
	Ts      string          `json:"ts"`
	Payload json.RawMessage `json:"payload"`
}

// SetDefaults assigns a fresh id and timestamp when they are missing.
func (e *Event) SetDefaults() {
	if e.ID == "" {
		e.ID = generateEventID()
	}
	if e.Ts == "" {
		e.Ts = time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	}
}

// ToMap converts the envelope to a map of field values.
func (e *Event) ToMap() map[string]any {
	return map[string]any{
		"id":      e.ID,
		"driver":  e.Driver,
		"type":    e.Type,
		"ts":      e.Ts,
		"payload": e.Payload,
	}
}

type GeneralDTO interface {
	// SetDefaults sets the default values for the DTO.
	SetDefaults()

	// ToMap converts the DTO to a map of field values
	ToMap() map[string]any
}

type ResponseDTO[T db.DbModel] interface {
	// RefreshFromDbModel refreshes this DTO's field values from a database model
	RefreshFromDbModel(v T) error
}

// ConnectRequest represents the JSON payload passed to /api/op/qpus/connect.
type ConnectRequest struct {
	Name         string         `json:"name"`
	AccessToken  string         `json:"access_token" validate:"required"`
	ExecutorType string         `json:"executor_type,omitempty"`
	DeviceConfig map[string]any `json:"device_config,omitempty"`
}

func (cr *ConnectRequest) SetDefaults() {
}

// ToMap converts the DTO to a map of field values
func (cr *ConnectRequest) ToMap() map[string]any {
	return map[string]any{
		"name":          cr.Name,
		"access_token":  cr.AccessToken,
		"executor_type": cr.ExecutorType,
		"device_config": cr.DeviceConfig,
	}
}

// ConnectResponse represents the JSON payload returned by /api/op/qpus/connect.
type ConnectResponse struct {
	Status         string `json:"status"`
	NNGCommandPort int    `json:"nng_command_port"`
	NNGResultPort  int    `json:"nng_result_port"`
	TLSHash        string `json:"tls_hash"`
	AuthToken      string `json:"auth_token"`
	NNGHost        string `json:"nng_host"`
}

func (cr *ConnectResponse) SetDefaults() {
}

func (cr *ConnectResponse) ToMap() map[string]any {
	return map[string]any{
		"status":           cr.Status,
		"nng_command_port": cr.NNGCommandPort,
		"nng_result_port":  cr.NNGResultPort,
		"auth_token":       cr.AuthToken,
		"nng_host":         cr.NNGHost,
	}
}

// ResultPayload represents the NNG incoming message format for job execution results.
type ResultPayload struct {
	JobID   string         `json:"job_id"`
	Results map[string]any `json:"results"`
}

func (rp *ResultPayload) SetDefaults() {
}

// ToMap converts the DTO to a map of field values
func (rp *ResultPayload) ToMap() map[string]any {
	return map[string]any{
		"job_id":  rp.JobID,
		"results": rp.Results,
	}
}

// CircuitPayload represents a single quantum circuit within a job submission.
type CircuitPayload struct {
	Circuit         string      `json:"circuit" validate:"required"`
	ParameterValues [][]float64 `json:"parameter_values,omitempty"`
	Shots           *int        `json:"shots,omitempty"`
}

func (cp *CircuitPayload) SetDefaults() {
}

// ToMap converts the DTO to a map of field values
func (cp *CircuitPayload) ToMap() map[string]any {
	return map[string]any{
		"circuit":          cp.Circuit,
		"parameter_values": cp.ParameterValues,
		"shots":            cp.Shots,
	}
}

// JobSubmitRequest represents the JSON payload for POST /api/jobs.
type JobSubmitRequest struct {
	Circuits   []CircuitPayload `json:"circuits" validate:"gt=0"`
	Shots      int              `json:"shots"`
	MeasLevel  *int             `json:"meas_level,omitempty"`
	MeasReturn string           `json:"meas_return,omitempty"`
	QPUTarget  string           `json:"qpu_target,omitempty"`
}

func (js *JobSubmitRequest) SetDefaults() {
	if js.Shots == 0 {
		js.Shots = 1024
	}
	if js.MeasLevel == nil {
		defaultMeasLevel := 2
		js.MeasLevel = &defaultMeasLevel
	}
	if js.MeasReturn == "" {
		js.MeasReturn = "single"
	}
}

// ToMap converts the DTO to a map of field values
func (js *JobSubmitRequest) ToMap() map[string]any {
	return map[string]any{
		"circuits":    js.Circuits,
		"shots":       js.Shots,
		"meas_level":  js.MeasLevel,
		"meas_return": js.MeasReturn,
		"qpu_target":  js.QPUTarget,
	}
}

// TokenCreateRequest represents the JSON payload for POST /api/tokens.
type TokenCreateRequest struct {
	Name      string `json:"name,omitempty"`
	ExpiresAt string `json:"expires_at,omitempty"` // ISO 8601 date string
}

func (tcr *TokenCreateRequest) SetDefaults() {
}

// ToMap converts the DTO to a map of field values
func (tcr *TokenCreateRequest) ToMap() map[string]any {
	return map[string]any{
		"name":       tcr.Name,
		"expires_at": tcr.ExpiresAt,
	}
}

// TokenCreateResponse represents the JSON payload returned by POST /api/tokens.
type TokenCreateResponse struct {
	ID        string `json:"id"`
	Token     string `json:"token,omitempty"`
	Name      string `json:"name"`
	ExpiresAt string `json:"expires_at,omitempty"`
	Created   string `json:"created"`
}

// RefreshFromDbModel refreshes this DTO's field values from a database model
func (tcr *TokenCreateResponse) RefreshFromDbModel(v *db.APIToken) error {
	// we never set the token from the database model as it is hashed
	tcr.ID = v.ID
	tcr.Name = v.Name
	tcr.ExpiresAt = v.ExpiresAt
	tcr.Created = v.Created

	return nil
}

func (tcr *TokenCreateResponse) SetDefaults() {
}

// ToMap converts the DTO to a map of field values
func (tcr *TokenCreateResponse) ToMap() map[string]any {
	return map[string]any{
		"id":         tcr.ID,
		"token":      tcr.Token,
		"name":       tcr.Name,
		"expires_at": tcr.ExpiresAt,
		"created":    tcr.Created,
	}
}

// TokenUpdateRequest represents the JSON payload for PATCH /api/tokens/{id}.
type TokenUpdateRequest struct {
	Name      *string `json:"name,omitempty"`
	ExpiresAt *string `json:"expires_at,omitempty"`
}

func (tur *TokenUpdateRequest) SetDefaults() {
}

// ToMap converts the DTO to a map of field values
func (tur *TokenUpdateRequest) ToMap() map[string]any {
	res := make(map[string]any, 2)
	if tur.Name != nil {
		res["name"] = *tur.Name
	}
	if tur.ExpiresAt != nil {
		res["expires_at"] = *tur.ExpiresAt
	}
	return res
}

// DismissRequest represents the JSON payload for POST /api/notifications/{id}/dismiss.
type DismissRequest struct {
	UserID string `json:"user_id,omitempty"`
}

func (dr *DismissRequest) SetDefaults() {
}

// ToMap converts the DTO to a map of field values
func (dr *DismissRequest) ToMap() map[string]any {
	return map[string]any{
		"user_id": dr.UserID,
	}
}

// QPUToggleRequest represents the JSON payload for POST /api/op/qpu/toggle.
type QPUToggleRequest struct {
	ID      string `json:"id" validate:"required"`
	Enabled bool   `json:"enabled"`
}

func (qtr *QPUToggleRequest) SetDefaults() {
}

// ToMap converts the DTO to a map of field values
func (qtr *QPUToggleRequest) ToMap() map[string]any {
	return map[string]any{
		"id":      qtr.ID,
		"enabled": qtr.Enabled,
	}
}

// QPUCreateRequest represents the JSON payload for POST /api/op/qpus/create.
type QPUCreateRequest struct {
	Name         string `json:"name" validate:"required"`
	ExecutorType string `json:"executor_type,omitempty"`
	NumQubits    int    `json:"num_qubits,omitempty"`
	Enabled      *bool  `json:"enabled,omitempty"`
}

func (qcr *QPUCreateRequest) SetDefaults() {
}

// ToMap converts the DTO to a map of field values
func (qcr *QPUCreateRequest) ToMap() map[string]any {
	return map[string]any{
		"name":          qcr.Name,
		"executor_type": qcr.ExecutorType,
		"num_qubits":    qcr.NumQubits,
		"enabled":       qcr.Enabled,
	}
}

// QPUCreateResponse represents the JSON payload returned by POST /api/op/qpus/create.
type QPUCreateResponse struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	AccessToken   string `json:"access_token"`
	ExecutorType  string `json:"executor_type"`
	Status        string `json:"status"`
	Enabled       bool   `json:"enabled"`
	QpiAddr       string `json:"qpi_addr"`
	CaFingerprint string `json:"ca_fingerprint"`
	DriverVersion string `json:"driver_version"`
}

// RefreshFromDbModel refreshes this DTO's field values from a database model
func (qcr *QPUCreateResponse) RefreshFromDbModel(v *db.QPU) error {
	qcr.ID = v.ID
	qcr.Name = v.Name
	qcr.ExecutorType = v.ExecutorType
	qcr.Status = v.Status
	qcr.Enabled = v.Enabled
	return nil
}

func (qcr *QPUCreateResponse) SetDefaults() {
}

// ToMap converts the DTO to a map of field values
func (qcr *QPUCreateResponse) ToMap() map[string]any {
	return map[string]any{
		"id":            qcr.ID,
		"name":          qcr.Name,
		"access_token":  qcr.AccessToken,
		"executor_type": qcr.ExecutorType,
		"status":        qcr.Status,
		"enabled":       qcr.Enabled,
		"qpi_addr":      qcr.QpiAddr,
	}
}

// QPUToggleResponse represents the JSON payload returned by POST /api/op/qpu/toggle.
type QPUToggleResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	Status  string `json:"status"`
}

// RefreshFromDbModel refreshes this DTO's field values from a database model
func (qtr *QPUToggleResponse) RefreshFromDbModel(v *db.QPU) error {
	qtr.ID = v.ID
	qtr.Name = v.Name
	qtr.Enabled = v.Enabled
	qtr.Status = v.Status
	return nil
}

func (qtr *QPUToggleResponse) SetDefaults() {
}

// ToMap converts the DTO to a map of field values
func (qtr *QPUToggleResponse) ToMap() map[string]any {
	return map[string]any{
		"id":      qtr.ID,
		"name":    qtr.Name,
		"enabled": qtr.Enabled,
		"status":  qtr.Status,
	}
}

// DriverCreateRequest represents the JSON payload for POST /api/op/drivers/create.
type DriverCreateRequest struct {
	Name     string   `json:"name" validate:"required"`
	QPU      string   `json:"qpu" validate:"required"`
	Kind     string   `json:"kind" validate:"required"`
	Language string   `json:"language" validate:"required"`
	Events   []string `json:"events,omitempty"`
	Enabled  *bool    `json:"enabled,omitempty"`
}

func (dcr *DriverCreateRequest) SetDefaults() {
}

// ToMap converts the DTO to a map of field values
func (dcr *DriverCreateRequest) ToMap() map[string]any {
	return map[string]any{
		"name":     dcr.Name,
		"qpu":      dcr.QPU,
		"kind":     dcr.Kind,
		"language": dcr.Language,
		"events":   dcr.Events,
		"enabled":  dcr.Enabled,
	}
}

// DriverSnippets holds the ready-to-use setup snippets shown once at
// registration, resolved from the driver's kind×language (RFC 0001 §3). An
// official build fills Systemd/ManualCLI/InstallAndRun; anything else fills
// Install/Stub instead.
type DriverSnippets struct {
	Systemd       string `json:"systemd,omitempty"`
	ManualCLI     string `json:"manual_cli,omitempty"`
	InstallAndRun string `json:"install_and_run,omitempty"`
	Install       string `json:"install,omitempty"`
	Stub          string `json:"stub,omitempty"`
}

// DriverCreateResponse represents the JSON payload returned by POST /api/op/drivers/create.
type DriverCreateResponse struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	QPU           string         `json:"qpu"`
	Kind          string         `json:"kind"`
	Language      string         `json:"language"`
	Events        []string       `json:"events"`
	Status        string         `json:"status"`
	Enabled       bool           `json:"enabled"`
	Token         string         `json:"token"`
	CaFingerprint string         `json:"ca_fingerprint"`
	QpiAddr       string         `json:"qpi_addr"`
	DriverVersion string         `json:"driver_version"`
	Snippets      DriverSnippets `json:"snippets"`
}

// RefreshFromDbModel refreshes this DTO's field values from a database model
func (dcr *DriverCreateResponse) RefreshFromDbModel(v *db.Driver) error {
	dcr.ID = v.ID
	dcr.Name = v.Name
	dcr.QPU = v.QPU
	dcr.Kind = v.Kind
	dcr.Language = v.Language
	dcr.Events = v.Events
	dcr.Status = v.Status
	dcr.Enabled = v.Enabled
	return nil
}

func (dcr *DriverCreateResponse) SetDefaults() {
}

// ToMap converts the DTO to a map of field values
func (dcr *DriverCreateResponse) ToMap() map[string]any {
	return map[string]any{
		"id":             dcr.ID,
		"name":           dcr.Name,
		"qpu":            dcr.QPU,
		"kind":           dcr.Kind,
		"language":       dcr.Language,
		"events":         dcr.Events,
		"status":         dcr.Status,
		"enabled":        dcr.Enabled,
		"token":          dcr.Token,
		"ca_fingerprint": dcr.CaFingerprint,
		"qpi_addr":       dcr.QpiAddr,
		"driver_version": dcr.DriverVersion,
		"snippets":       dcr.Snippets,
	}
}

// DriverConnectRequest represents the JSON payload for POST /api/op/drivers/connect.
// Uses "token" (not "access_token") to match the field DriverCreateResponse
// returns it under, since a driver's own DB field is "token".
type DriverConnectRequest struct {
	AccessToken string `json:"token" validate:"required"`
	Name        string `json:"name,omitempty"`
	Host        string `json:"host,omitempty"`
	Version     string `json:"version,omitempty"`
}

func (dcr *DriverConnectRequest) SetDefaults() {
}

// ToMap converts the DTO to a map of field values
func (dcr *DriverConnectRequest) ToMap() map[string]any {
	return map[string]any{
		"token":   dcr.AccessToken,
		"name":    dcr.Name,
		"host":    dcr.Host,
		"version": dcr.Version,
	}
}

// DriverConnectResponse represents the JSON payload returned by POST /api/op/drivers/connect.
type DriverConnectResponse struct {
	Status     string `json:"status"`
	NNGInPort  int    `json:"nng_in_port"`
	NNGOutPort int    `json:"nng_out_port"`
	TLSHash    string `json:"tls_hash"`
	AuthToken  string `json:"auth_token"`
	NNGHost    string `json:"nng_host"`
}

func (dcr *DriverConnectResponse) SetDefaults() {
}

// ToMap converts the DTO to a map of field values
func (dcr *DriverConnectResponse) ToMap() map[string]any {
	return map[string]any{
		"status":       dcr.Status,
		"nng_in_port":  dcr.NNGInPort,
		"nng_out_port": dcr.NNGOutPort,
		"tls_hash":     dcr.TLSHash,
		"auth_token":   dcr.AuthToken,
		"nng_host":     dcr.NNGHost,
	}
}

// DriverToggleRequest represents the JSON payload for POST /api/op/drivers/toggle.
type DriverToggleRequest struct {
	ID      string `json:"id" validate:"required"`
	Enabled bool   `json:"enabled"`
}

func (dtr *DriverToggleRequest) SetDefaults() {
}

// ToMap converts the DTO to a map of field values
func (dtr *DriverToggleRequest) ToMap() map[string]any {
	return map[string]any{
		"id":      dtr.ID,
		"enabled": dtr.Enabled,
	}
}

// DriverToggleResponse represents the JSON payload returned by POST /api/op/drivers/toggle.
type DriverToggleResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	Status  string `json:"status"`
}

// RefreshFromDbModel refreshes this DTO's field values from a database model
func (dtr *DriverToggleResponse) RefreshFromDbModel(v *db.Driver) error {
	dtr.ID = v.ID
	dtr.Name = v.Name
	dtr.Enabled = v.Enabled
	dtr.Status = v.Status
	return nil
}

func (dtr *DriverToggleResponse) SetDefaults() {
}

// ToMap converts the DTO to a map of field values
func (dtr *DriverToggleResponse) ToMap() map[string]any {
	return map[string]any{
		"id":      dtr.ID,
		"name":    dtr.Name,
		"enabled": dtr.Enabled,
		"status":  dtr.Status,
	}
}

// DispatchPayload represents the NNG outgoing message format for job dispatch.
type DispatchPayload struct {
	JobID   string `json:"job_id"`
	Payload any    `json:"payload"`
}

func (dp *DispatchPayload) SetDefaults() {
}

// ToMap converts the DTO to a map of field values
func (dp *DispatchPayload) ToMap() map[string]any {
	return map[string]any{
		"job_id":  dp.JobID,
		"payload": dp.Payload,
	}
}

// JobResultUpdate holds the fields to update on a completed/failed quantum job.
type JobResultUpdate struct {
	Status     string  `json:"status"`
	FinishedAt string  `json:"finished_at"`
	Results    string  `json:"results"`
	Duration   float64 `json:"duration"`
}

func (jru *JobResultUpdate) SetDefaults() {
}

// ToMap converts the DTO to a map of field values
func (jru *JobResultUpdate) ToMap() map[string]any {
	return map[string]any{
		"status":      jru.Status,
		"finished_at": jru.FinishedAt,
		"results":     jru.Results,
		"duration":    jru.Duration,
	}
}
