package api

import "qpi/internal/db"

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
}

func (cr *ConnectResponse) SetDefaults() {
}

func (cr *ConnectResponse) ToMap() map[string]any {
	return map[string]any{
		"status":           cr.Status,
		"nng_command_port": cr.NNGCommandPort,
		"nng_result_port":  cr.NNGResultPort,
		"auth_token":       cr.AuthToken,
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
