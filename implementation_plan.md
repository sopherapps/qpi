# Implementation Plan: CRUD API Routes & qpi-client Package

Implement authenticated/authorized CRUD routes on the `qpi-interface` Go orchestrator for submitting, viewing, and cancelling quantum jobs, incorporating user QPU allocation tracking and Qiskit-standard batch/parameterized payloads. Develop a multi-language client library (`qpi-client`) containing a Python Qiskit provider (BackendV2 & JobV1), JavaScript/TypeScript, and Go SDKs.

## User Review Required

> [!IMPORTANT]
> **QPU Time Allocation (`qpu_seconds`)**:
> We will add a `qpu_seconds` number field to the `users` collection. 
> Only users with `qpu_seconds > 0` can submit jobs. 
> Every job execution subtracts the actual execution duration from the user's `qpu_seconds` balance.
> 
> **Qiskit Payload & Measurement Levels (`meas_level`, `meas_return`)**:
> Jobs will accept a list of circuits (each with QASM, optional parameters, and optional shots) and Qiskit-standard `meas_level` (int) and `meas_return` (string) options.
> - `meas_level = 2` (default): Performs state-discrimination and returns counts (binary states).
> - `meas_level = 1` or `0`: Returns raw/kerneled complex IQ values.
> - `meas_return = "single"` (default): Returns single-shot data.
> - `meas_return = "avg"`: Returns average data.
> 
> **Authentication Options (Browser Cookie/JWT & API Tokens)**:
> The custom routes will support two authentication methods:
> 1. Standard PocketBase cookie/JWT token validation (for browser dashboard dashboard integrations).
> 2. Custom API token verification via header (`X-API-Token` or `Authorization: Bearer <API-Token>`), matching keys stored in the user's `api_tokens` JSON list.

---

## Proposed Changes

### 1. qpi-interface: Authenticated CRUD & QPU Resource Allocation
#### [MODIFY] [schema.go](file:///Users/martinahindura/work/code/sopherapps/open-source/qpi/qpi-interface/internal/schema/schema.go)
* Add a `qpu_seconds` number field to the `users` collection, defaulting to `1000` seconds on creation.

#### [MODIFY] [api.go](file:///Users/martinahindura/work/code/sopherapps/open-source/qpi/qpi-interface/internal/api/api.go)
* Add new HTTP REST endpoints with request authentication verification:
  * Check standard `re.Auth` first, then fall back to checking `X-API-Token` or `Authorization` headers for API keys matched against the user's `api_tokens` JSON field.
  * **`POST /api/jobs`**: 
    * Accepts a list of `circuits` (each with `qasm`, optional `parameter_values`, and optional `shots`), and global options `shots`, `meas_level` (default 2), and `meas_return` (default "single").
    * Checks if `user.qpu_seconds > 0`. If not, returns `403 Forbidden`.
    * Validates and resolves target QPU.
    * Creates a `quantum_jobs` record with status `"pending"`, user ID set to the resolved user ID.
    * Returns the created job ID.
  * **`GET /api/jobs`**:
    * Retrieves all job records belonging to the authenticated user sorted by `created` descending.
  * **`GET /api/jobs/{id}`**:
    * Retrieves job by ID. Verifies user ownership.
  * **`POST /api/jobs/{id}/cancel`**:
    * Retrieves job by ID, verifies ownership, and cancels if status is `"pending"` or `"running"`.
* **Resource Deduction in NNG listener**:
  * Upon receiving the job execution result from the driver, calculate the duration since the job's `updated` timestamp.
  * Deduct the duration (in seconds) from the user's `qpu_seconds` balance.
  * If the job results contain an `error` field, set the job status to `failed` instead of `completed`.

### 2. Python Client Package (Qiskit Integration)
Located under `qpi-client/py/` (standalone package structure ready for PyPI publication).

#### [NEW] [qpi-client/py/pyproject.toml](file:///Users/martinahindura/work/code/sopherapps/open-source/qpi/qpi-client/py/pyproject.toml)
* Configure python packaging using `setuptools` with dependencies: `requests`, `qiskit>=1.0.0`.

#### [NEW] [client.py](file:///Users/martinahindura/work/code/sopherapps/open-source/qpi/qpi-client/py/qpi_client/client.py)
* Implement low-level `QPIClient` wrapper supporting custom API token authentication via header.

#### [NEW] [provider.py](file:///Users/martinahindura/work/code/sopherapps/open-source/qpi/qpi-client/py/qpi_client/provider.py)
* Implement `QPIBackend` subclassing Qiskit's `BackendV2`. It transpiles/compiles Qiskit `QuantumCircuit` inputs to QASM, packages payloads (including parameter binds), and submits jobs.
* Implement `QPIJob` subclassing Qiskit's `JobV1`. It handles asynchronous polling of job status, results retrieval (including raw IQ memory reconstruction if `meas_level < 2`), and job cancellation.

### 3. JavaScript/TypeScript Client SDK
Located under `qpi-client/js/` (standard Node package ready for npm publish).

#### [NEW] [package.json](file:///Users/martinahindura/work/code/sopherapps/open-source/qpi/qpi-client/js/package.json)
* JS package file with dependencies.

#### [NEW] [client.ts](file:///Users/martinahindura/work/code/sopherapps/open-source/qpi/qpi-client/js/src/client.ts)
* Reusable API client class supporting both custom API token authentication and cookie-based authentication, with methods for job submission, listing, retrieval, and cancellation.

### 4. Go Client SDK
Located under `qpi-client/go/` (standard Go module importable by other Go projects).

#### [NEW] [go.mod](file:///Users/martinahindura/work/code/sopherapps/open-source/qpi/qpi-client/go/go.mod)
* Go module file.

#### [NEW] [client.go](file:///Users/martinahindura/work/code/sopherapps/open-source/qpi/qpi-client/go/client.go)
* Reusable Go package client wrapping HTTP requests to the orchestrator REST endpoints (supporting QASM string submission).

### 5. qpi-driver: Multi-Circuit & Parameterized Executions
#### [MODIFY] [base.py](file:///Users/martinahindura/work/code/sopherapps/open-source/qpi/qpi-driver/qpi_driver/executors/base.py)
* Update `JobPayload` to support a list of circuits (each with `qasm`, optional `parameter_values`, and optional `shots`), `meas_level` (default 2), and `meas_return` (default "single").

#### [MODIFY] [driver.py](file:///Users/martinahindura/work/code/sopherapps/open-source/qpi/qpi-driver/qpi_driver/driver.py)
* Update `xarray_to_qiskit_counts` to support multi-circuit datasets (dimensions `circuit_index` and `acq_index_{i}`) and handle raw complex IQ value serialization (as lists of `[real, imag]`) if `meas_level < 2` is requested. Implement average outcomes if `meas_return == "avg"`.

#### [MODIFY] [mock.py](file:///Users/martinahindura/work/code/sopherapps/open-source/qpi/qpi-driver/qpi_driver/executors/mock.py) & [qiskit_aer.py](file:///Users/martinahindura/work/code/sopherapps/open-source/qpi/qpi-driver/qpi_driver/executors/qiskit_aer.py)
* Update executors to loop over the payload's circuits, bind parameter values (if any), execute them, and return a combined `xr.Dataset` containing the extra `circuit_index` dimension.

---

## Verification Plan

### Automated Tests
* Build and start `qpi-interface` and `qpi-driver` mock services locally.
* Create a test script that:
  1. Registers a test user in PocketBase, assigns QPU seconds, and configures an API token in the database.
  2. Runs Python integration tests verifying successful Qiskit Backend connection using the API token, job execution with parameterized circuits, QPU seconds deduction verification, raw complex IQ memory retrieval (using `meas_level=1`), and cancellation.
  3. Runs basic checks for JS and Go clients.
