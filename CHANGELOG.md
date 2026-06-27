# Change log

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project follows versions of format `{year}.{month}.{patch_number}`.

## [Unreleased]

### Fixed

- `qpi-ui`: Fixed an issue in the admin dashboard where dismissed system notifications reappeared on page refresh. Dismissals are now correctly persisted via proxy user API requests.
- `qpi-driver`: Fixed a `panic: nng is not fork-reentrant safe` error in multiprocessing environments by deferring the NNG TLSConfig initialization until after the worker processes have forked.

## [0.0.33] - 2026-06-27

### Added

- `qpi-ui`: Added `--ip-addr` (or `QPI_IP_ADDR`, or `ipAddr` in config) to explicitly specify the public IP for binding TLS sockets. The provided IP is now properly encoded in the X509 certificate's SAN IP block.

### Changed

- `qpi-driver`: Updated NNG setup logic. The driver now establishes connections using the explicit NNG IP address returned by the server via `ConnectResponse`, decoupling it from the HTTP QPI address.

### Fixed

- `qpi-ui`: Removed the `fetchHostIPs()` autodiscovery logic which caused unintended behavior when deployed behind proxies.
- `qpi-driver`: Fixed a race condition where the result sender process could attempt to read the CA certificate from disk before the main process had downloaded it.

## [0.0.32] - 2026-06-26

### Fixed

- `qpi`: Fixed various linting errors across the Go and React UI codebases.

## [0.0.31] - 2026-06-26

### Changed

- `qpi-ui`: Made the metric cards on the Overview dashboard (Active QPUs, Queue Status, Next Booking) clickable so they quickly route to their respective tabs.
- `qpi-ui`: Clarified the "Load Example" button text and icon in the Jobs Console to read "Load Bell State Example".

### Fixed

- `qpi-ui`: Fixed "authentication required" error that occurred when superusers attempted to submit a quantum job. Superusers are now transparently issued a proxy `users` record with unlimited QPU seconds to satisfy relational constraints.

## [0.0.30] - 2026-06-26

### Added

- `qpi-ui`: Light mode support for the dashboard UI with a theme toggle. Dark mode remains the default.
- `qpi-ui`: Added an admin option to delete QPUs from the QPU Registry, complete with a confirmation modal.
- `qpi-ui`: Added a user profile dropdown menu in the dashboard top bar for quick access to settings and signing out.
- `qpi-ui`: Synchronized auth sessions across tabs and between the `/_/` admin UI and `/dashboard`, automatically signing users in/out when state changes globally.

### Changed

- `qpi-ui`: Restricted the "Create QPU" and "Toggle Status" buttons in the QPU Registry tab to administrators only, while still allowing standard users to view available QPUs.
- `qpi-ui`: Conditionally hide the username and password login fields if `passwordAuth` is disabled in the PocketBase users collection.

### Fixed

- `qpi-ui`: Fixed the QPU Registry cards to properly display the Executor Driver (`executor_type`).

## [0.0.29] - 2026-06-25

### Fixed

- `ci`: Fixed failing python test step in CI.

## [0.0.28] - 2026-06-25

### Fixed

- `ci`: Fixed failing lint step in CI by updating `uv sync` flags.

## [0.0.27] - 2026-06-25

### Fixed

- `qpi-ui`: Fixed `CHANGELOG.md` versioning mismatch and correctly restored `0.0.25` entries. Bumping version to `0.0.27` due to tag immutability on `0.0.26`.

## [0.0.26] - 2026-06-25

### Fixed

- `qpi-ui`: Reverted the hiding of the "QPU Registry" dashboard tab for standard non-admin users so that they can see existing QPUs (but cannot register or toggle them).

## [0.0.25] - 2026-06-25

### Added

- `qpi-driver`: Added `install-systemd.sh` script to automate installation of the driver as a systemd background service.
- `qpi-ui`: Added an admin-only endpoint `GET /api/op/version` to retrieve the server's version.
- `qpi-ui`: Added a dynamic version label to the dashboard sidebar (visible only to admins).
- `qpi-ui`: Updated the QPU Registration success modal to generate and display a copyable `install-systemd.sh` execution snippet.
- `ci`: Added a dedicated E2E testing job (`test-systemd-installer`) in GitHub Actions to validate the systemd installation script via a Docker container.

### Changed

- Global: Renamed all instances of "Orchestrator" to "Server" (and "orchestrator" to "server") across documentation, code, and CI scripts.
- Global: Renamed all instances of "Hardware Driver" to "QPU Driver" (and "hardware driver" to "QPU driver") across the project.
- `qpi-ui`: Simplified the `README.md` introduction with a shorter description, a simpler mermaid diagram, and pulled the Quick Start section to the top.

### Fixed

- `qpi-ui`: Fixed a bug in the dashboard (`App.tsx` and `Sidebar.tsx`) where the "QPU Registry" tab was still visible to standard non-admin users.
- `qpi-ui`: Fixed a double-hashing bug in `handleQPUCreate` that caused driver connection snippet tests to fail with `401 Unauthorized`.

## [0.0.24] - 2026-06-23

### Fixed

- Updated the CHANGELOG appropriately.

## [0.0.23] - 2026-06-23

### Fixed

- `qpi-ui`: Used GoReleaser NFPM overrides to separate Debian/RPM and Alpine `init` script configurations, preventing `dpkg` installation crashes (`Default-Start contains no runlevels`) and eliminating improper `systemd` dependencies in `.apk` packages.


## [0.0.22] - 2026-06-23

### Fixed

- `qpi-ui`: Added `draft: true` to all intermediate `softprops/action-gh-release` asset upload steps to prevent them from prematurely publishing the GitHub release and triggering immutable release errors on subsequent jobs.

## [0.0.21] - 2026-06-23

### Added
- `qpi-ui`: Added macOS ARM64 (Apple Silicon) native installer packaging to the CI pipeline.

### Fixed

- `qpi-ui`: Fixed directory pathing error during the macOS binary build step in GitHub Actions.
- `qpi-ui`: Fixed macOS pkg output path evaluation and eliminated a GitHub release asset race condition between parallel macOS runners.

## [0.0.20] - 2026-06-23

### Fixed

- `qpi-ui`: Configured GoReleaser to create a `draft` release and automated publishing at the end of the pipeline to avoid GitHub's immutable release asset errors during Windows MSI and macOS PKG uploads.

## [0.0.19] - 2026-06-23

### Fixed

- `github-actions`: Updated Node.js version from 20 to 22 in CI jobs to resolve deprecation warnings.

## [0.0.18] - 2026-06-23

### Fixed

- `qpi-ui`: Added `wixl` to the apt-get install step to fix missing command during Windows MSI packaging.

## [0.0.17] - 2026-06-23

### Fixed

- `qpi-ui`: Fixed GoReleaser LICENSE path and removed invalid NFPM contents entry.

## [0.0.16] - 2026-06-23

### Fixed

- `qpi-ui`: Fixed GoReleaser v2 syntax errors and NFPM script names.

## [0.0.15] - 2026-06-23

### Changed

- `qpi-ui`: Upgraded the packaging to support 'rpm', 'apk', macOS and windows installers.

## [0.0.14] - 2026-06-23

### Fixed

- `qpi-ui`: Fixed the loading of flags which were not taking effect even when supplied.

## [0.0.13] - 2026-06-21

### Fixed

- Fixed broken links and typos in docs website.

## [0.0.12] - 2026-06-21

### Fixed

- Fixed failing deployment of documentation site in GitHub actions on push to new tag.

## [0.0.11] - 2026-06-21

### Fixed

- Fixed failing deployment of documentation site in GitHub actions

## [0.0.10] - 2026-06-21

### Added

- Documentation site configuration (`mkdocs.yml`) with automated deployments (`docs.yml`) to GitHub Pages via MkDocs Material.

## [0.0.9] - 2026-06-21

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

- Added READMEs for all client packages (Go, JS, Python) and the QPU driver

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


