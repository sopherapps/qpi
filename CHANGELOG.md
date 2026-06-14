# Change log

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project follows versions of format `{year}.{month}.{patch_number}`.

## [Unreleased]

### Added

- `qpi-interface`: Added authenticated CRUD rules and validation hooks for the `qpu_time_requests` collection, supporting user requests, admin approvals/rejections, automatic QPU seconds crediting, and handled request immutability.
- `qpi-interface`: Added authenticated CRUD rules and validation hooks for `time_slots` collection, implementing interval order, overlap checks, auto-population of owner, past booking/update/delete restrictions, and admin bypass capability.
- `qpi-interface`: Added admin-only `PATCH /api/admin/users/{id}` endpoint for superusers to update `qpu_seconds` and `api_tokens` on any user record.
- `qpi-client/py`: `QPIBackend.run()` now supports `parameter_values` kwarg for parameterized circuit execution, automatically binding parameters and forwarding ordered values to the API payload.
- `qpi-driver/tests`: Added `@pytest.mark.skipif` decorators to CLI and quantify tests so they gracefully skip when optional dependencies (`typer`, `quantify_scheduler`, `qblox_instruments`) are not installed.
- `Makefile`: Added granular `test-py-base`, `test-py-cli`, `test-py-aer`, and `test-py-quantify` targets for testing each `pyproject.toml` extra in isolation.
- `qpi-driver`: Added an abstract `process_result()` method to the `Executor` interface, letting executors handle their own data processing (e.g. state discrimination, IQ memory formatting) directly in the worker process.
- `qpi-driver`: Implemented state discrimination, average/single IQ memory formatting, and raw trace handling in `MockExecutor`, `QiskitAerExecutor`, and `QuantifyExecutor`.
- `qpi-driver`: Support for `ThresholdedAcquisition` protocol in `QuantifyExecutor` when threshold/rotation parameters are defined on device elements, automatically falling back to software discrimination via `SSBIntegrationComplex`.

### Changed

- `qpi-interface`: Default `qpu_seconds` for new users changed from `1000` to `0`. Users must now be granted QPU time explicitly by an admin via the `PATCH /api/admin/users/{id}` endpoint. The `OnRecordCreate` hook that previously set the default has been removed.
- `qpi-driver`: Renamed the `translator` process to `result sender` and simplified it to forward processed dicts via NNG PUSH directly from a queue, eliminating intermediate `.pkl` filesystem serialization overhead.


