# Change log

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project follows versions of format `{year}.{month}.{patch_number}`.

## [Unreleased]

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


