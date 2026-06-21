# Change log

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project follows versions of format `{year}.{month}.{patch_number}`.

## [Unreleased]

### Added

- TLS connection between the server (qpi-ui) and the driver (qpi-driver)
- `qpi-driver`: added the `--ca-file` and `--ca-fingerprint` params to the qpi-driver all
- `qpi-ui (dashboard)`: updated the code snippet shown to the user on QPU creation to include `--ca-fingerprint`.
- Comprehensive Cypress E2E test suite covering all dashboard sections:
  - **Auth & Navigation** — login error flow, role-based navigation, hash routing, back/forward sync, logout
  - **QPU Registry** — admin QPU registration (with token and command verification), toggle online/offline, regular user restrictions
  - **Jobs Console** — default form state, job submission and results, QPU dropdown filtering, empty state
  - **Bookings** — booking a time slot, validation (end before start), cancel with confirmation, visibility (user vs admin)
  - **Admin Panel** — user quota allocations, time request approval/rejection, broadcast announcements, notification badge, approval quota updates
  - **Overview & Header** — metrics row accuracy, quick-action navigation, recent jobs table, notifications panel (dismiss individual/clear all), notification targeting (broadcast vs targeted), notification dismiss isolation (per-user), header page title sync
  - **Settings & Request Time** — profile settings (email, quota, role badge), request time modal submission, validation (empty reason/seconds)
  - **Error & Edge Cases** — empty states (no jobs, no selected job, no QPUs), network failure handling (`alert()` messages), unauthorized access to `/#admin`
- Backend unit tests for `OnQPUTimeRequestUpdateRequest` hook:
  - Approval adds requested seconds to user quota; rejection leaves it unchanged
  - Non-superusers are forbidden from updating time requests
  - Already-processed (approved/rejected) requests cannot be modified

## [0.0.8] - 2026-06-17

### Added

- Added READMEs for all client packages (Go, JS, Python) and the hardware driver

## [0.0.7] - 2026-06-17

### Added

- Added logo to CLI and README

### Fixed

- Fixed error with 'make package' failing due to missing dashboard built files

## [0.0.6] - 2026-06-17

### Fixed

- Fixed failing tests on GitHub CI and reduced pocketbase's verbosity.

## [0.0.5] - 2026-06-17

### Changed

- Fixed GitHub Actions matrix for tests sleeping.

## [0.0.4] - 2026-06-17

### Changed

- `qpi-driver`: [BREAKING] Changed the format of the `element_type` in quantify.device.yml to include
  `path (str)`, `args (tuple)` and `kwargs (dict)`
- `qpi-driver`: Unskipped the e2e errors for 'quantify' executor
- `qpi-driver`: Added a log file for the driver at `data/{executor}-driver.log` during e2e tests 

### Fixed

- `qpi-driver`: Failing e2e errors for 'qblox' executor. Specifically:
  - Fixed 4ns grid rounding misalignment on custom durations for `Delay` operations.
  - Added support for OpenQASM `Delay` instructions by mapping them to `IdlePulse`.
  - Added concurrent anchoring (`ref_pt="start"`) for parallel multi-qubit Qiskit instructions (e.g., `Measure`, `Delay`, `Barrier`).
  - Handled invalid `-1` hardware acquisition dummy data thresholds during Qblox and Quantify dummy measurements.
- `qpi`: Resolved Apple Silicon macOS codesign binary integrity crashes during the E2E suite due to dynamically installed `q1asm_macos`.


## [0.0.3] - 2026-06-16

### Changed

- `qpi-ui`: Refactored the hooks.go files to make them easier to read
  

## [0.0.2] - 2026-06-16

### Added

- `qpi-ui`: Centralized API payload and database collection schemas as Go structs in the new `qpi/internal/schema` package (including `User`, `APIToken`, `QPU`, `TimeSlot`, `QuantumJob`, `QPUTimeRequest`, `Notification`, and corresponding request/response payloads).
- `qpi-ui`: Added `*FromRecord` helper mapping functions in the `schema` package to safely construct database model structs from PocketBase `*core.Record` objects.
- `qpi-ui`: Added `qpi_addr` dynamically computed field to the `/api/op/qpus/create` JSON response.
- `qpi-ui/internal/dashboard`: Updated the QPU registry tab to show a success modal upon QPU registration, including copy-to-clipboard icons for both the raw access token and a copyable `qpi-driver` start command.
- `qpi-driver`: Added support for the Qblox Scheduler (`qblox-scheduler`) package via a new `QbloxExecutor` (`qblox`).
- `qpi-driver`: Added `qblox` optional-dependencies group to `pyproject.toml` and a compatibility layer at `qpi_driver/compat/qblox.py` to gracefully handle cases where `qblox-scheduler` is not installed.
- `qpi-driver`: Created automated test suite at `qpi_driver/tests/test_qblox.py` and integrated `test-py-qblox` test target into `GitHub CI` matrix.
- `qpi-client/go`: Added `QpiAddr` field to the `QpuRecord` struct.
- `qpi-ui`: Added `FindAndDeleteOne` and `FindOneByFilter` helpers to the database query layer (`internal/db/queries.go`) to support cleaner repository queries.
- `qpi-ui`: Added validation-tagged API DTO models (`QPUCreateRequest`, `QPUCreateResponse`, `QPUToggleResponse`, `DispatchPayload`, `JobResultUpdate`) under `internal/api/schema.go`.

### Changed

- `qpi-ui`: Integrated the centralized `schema` structs into all custom REST controllers and handlers inside the `api` package, replacing duplicate local private struct definitions.
- `qpi-driver`: [Breaking] Removed deprecated `-H`/`--host` and `-P`/`--port` options from CLI and `run_driver` in favor of `--qpi-addr` / `-a` (env: `QPI_ADDR`, default: `http://127.0.0.1:8090`).
- `qpi-client`: Updated Go/Python/JS client E2E test suites to use the `QPI_ACCESS_TOKEN` environment variable.
- `qpi-ui`: Refactored all HTTP REST handlers (`handleNotificationDismiss`, `handleTokenDelete`, `handleQPUConnect`, `handleQPUToggle`) to use database models and generic queries instead of raw `core.Record` objects.
- `qpi-ui`: Removed reflection from `internal/db/queries.go` by refactoring methods to accept a pre-allocated model destination interface, improving performance.
- `qpi-ui`: Separated access token lookup and status validation in `handleQPUConnect` to correctly return `401 Unauthorized` for invalid tokens and `403 Forbidden` for disabled QPUs.
- `qpi-ui/internal/dashboard`: Updated `App.tsx` quantum job submission callback to extract `id` instead of `job_id` from the backend response.

## [0.0.1] - 2026-06-14

### Added

- `qpi-ui`: Added `notifications` collection with admin-only CRUD, user visibility rules, broadcast/targeted targeting, time-window filtering, and per-user dismiss support via `POST /api/notifications/{id}/dismiss`.
- `e2e/verify.py`: Added the `test_notifications_crud` E2E test to verify broadcast/targeted visibility, time-window filtering, per-user dismiss, and admin-only CUD enforcement.
- `qpi-ui`: Added `enabled` boolean field to `qpus` collection to allow administrators to toggle QPU drivers on and off.
- `qpi-ui`: Added an update event hook on the `qpus` collection that cancels/stops dispatcher and listener goroutines (and sets status to `"offline"`) when `enabled` is set to `false`, and starts goroutines (and sets status to `"online"`) when `enabled` is set to `true`.
- `qpi-ui`: Enforced `enabled` check in the `/api/op/qpu/register` route to reject registration of disabled QPUs with a `403 Forbidden` response.
- `e2e/verify.py`: Added the `test_qpu_toggle_switch` E2E test to verify the QPU disabled/enabled lifecycle, goroutine lifecycle, and registration blocking.

- `qpi-ui`: Added authenticated CRUD rules and validation hooks for the `qpu_time_requests` collection, supporting user requests, admin approvals/rejections, automatic QPU seconds crediting, and handled request immutability.
- `qpi-ui`: Added authenticated CRUD rules and validation hooks for `time_slots` collection, implementing interval order, overlap checks, auto-population of owner, past booking/update/delete restrictions, and admin bypass capability.
- `qpi-ui`: Added admin-only `PATCH /api/admin/users/{id}` endpoint for superusers to update `qpu_seconds` and `api_tokens` on any user record.
- `qpi-client/py`: `QPIBackend.run()` now supports `parameter_values` kwarg for parameterized circuit execution, automatically binding parameters and forwarding ordered values to the API payload.
- `qpi-driver/tests`: Added `@pytest.mark.skipif` decorators to CLI and quantify tests so they gracefully skip when optional dependencies (`typer`, `quantify_scheduler`, `qblox_instruments`) are not installed.
- `Makefile`: Added granular `test-py-base`, `test-py-cli`, `test-py-aer`, and `test-py-quantify` targets for testing each `pyproject.toml` extra in isolation.
- `qpi-driver`: Added an abstract `process_result()` method to the `Executor` interface, letting executors handle their own data processing (e.g. state discrimination, IQ memory formatting) directly in the worker process.
- `qpi-driver`: Implemented state discrimination, average/single IQ memory formatting, and raw trace handling in `MockExecutor`, `QiskitAerExecutor`, and `QuantifyExecutor`.
- `qpi-driver`: Support for `ThresholdedAcquisition` protocol in `QuantifyExecutor` when threshold/rotation parameters are defined on device elements, automatically falling back to software discrimination via `SSBIntegrationComplex`.

### Changed

- `qpi-ui`: Default `qpu_seconds` for new users changed from `1000` to `0`. Users must now be granted QPU time explicitly by an admin via the `PATCH /api/admin/users/{id}` endpoint. The `OnRecordCreate` hook that previously set the default has been removed.
- `qpi-driver`: Renamed the `translator` process to `result sender` and simplified it to forward processed dicts via NNG PUSH directly from a queue, eliminating intermediate `.pkl` filesystem serialization overhead.


