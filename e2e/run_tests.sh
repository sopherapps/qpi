#!/bin/bash
set -e

# Get the directory of this script
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
PROJECT_ROOT="$( dirname "$DIR" )"

cd "$PROJECT_ROOT"

# Global variables for background PIDs
PB_PID=""
DRIVER_PID=""

# Cleanup background processes on exit
cleanup() {
    echo "[e2e] Cleaning up background processes..."
    if [ -n "$DRIVER_PID" ]; then
        echo "[e2e] Stopping Python driver (PID $DRIVER_PID)..."
        kill "$DRIVER_PID" 2>/dev/null || true
    fi
    if [ -n "$PB_PID" ]; then
        echo "[e2e] Stopping PocketBase server (PID $PB_PID)..."
        kill "$PB_PID" 2>/dev/null || true
    fi
    rm -rf bin/pb_data bin/data
}
trap cleanup EXIT

# Build Go orchestrator
echo "[e2e] Building Go orchestrator..."
mkdir -p bin
(cd qpi-interface && go build -o ../bin/qpi .)

# Detect Python interpreter.
# If the developer has already activated a virtual environment, we respect that environment
# and use the active python in PATH. Otherwise, we look for the local .venv folder,
# and finally fall back to the system python.
PYTHON="python"
if [ -n "$VIRTUAL_ENV" ]; then
    PYTHON="python"
elif [ -d ".venv" ]; then
    PYTHON="./.venv/bin/python"
fi

# Install python driver package
echo "[e2e] Installing Python qpi-driver package..."
$PYTHON -m pip install -e ./qpi-driver[cli,aer]

# NOTE regarding pytest vs bash script:
# We use this bash script for orchestration rather than pytest and pytest fixtures because
# this integration test involves running and managing processes across multiple languages:
# Go (PocketBase binary 'qpi') and Python (driver and test runner).
# Bash provides a lightweight, dependency-free, and natural process-management language
# to start PocketBase, wait for readiness, seed, run the python driver, verify, and clean
# them all up reliably on exit via standard traps.
run_e2e_for_executor() {
    local executor=$1
    echo ""
    echo "========================================================================"
    echo "[e2e] Running E2E tests for executor: $executor"
    echo "========================================================================"
    echo ""

    # Clear old data
    rm -rf bin/pb_data bin/data

    # Create superuser (this also initializes the schema and migrations safely and synchronously)
    echo "[e2e] Initializing database and creating superuser..."
    ./bin/qpi superuser upsert admin@example.com supersecretpassword1234 --dir bin/pb_data

    # Start PocketBase server in background
    echo "[e2e] Starting PocketBase server..."
    ./bin/qpi serve --dir bin/pb_data --dev &
    PB_PID=$!

    # Wait for PocketBase to start responding on port 8090
    echo "[e2e] Waiting for PocketBase to be ready..."
    READY=0
    for i in {1..30}; do
        if curl -s http://127.0.0.1:8090/ >/dev/null; then
            echo "[e2e] PocketBase is ready."
            READY=1
            break
        fi
        sleep 1
    done

    if [ "$READY" -ne 1 ]; then
        echo "[e2e] Timeout waiting for PocketBase to start"
        exit 1
    fi

    # Seed database
    echo "[e2e] Seeding database..."
    ADMIN_EMAIL=admin@example.com ADMIN_PASSWORD=supersecretpassword1234 $PYTHON e2e/seed.py

    # Start Python driver via CLI
    echo "[e2e] Starting Python driver with executor: $executor..."
    REGISTRATION_TOKEN=my-super-secret-token-12345 $PYTHON -m qpi_driver.cli start --executor "$executor" --data-dir bin/data &
    DRIVER_PID=$!

    # Run verification script
    echo "[e2e] Running verification script..."
    ADMIN_EMAIL=admin@example.com ADMIN_PASSWORD=supersecretpassword1234 $PYTHON e2e/verify.py
    
    VERIFY_STATUS=$?

    # Stop background processes for this run
    echo "[e2e] Stopping Python driver (PID $DRIVER_PID)..."
    kill "$DRIVER_PID" 2>/dev/null || true
    wait "$DRIVER_PID" 2>/dev/null || true
    DRIVER_PID=""

    echo "[e2e] Stopping PocketBase server (PID $PB_PID)..."
    kill "$PB_PID" 2>/dev/null || true
    wait "$PB_PID" 2>/dev/null || true
    PB_PID=""

    rm -rf bin/pb_data bin/data

    if [ $VERIFY_STATUS -ne 0 ]; then
        echo "[e2e] Verification FAILED for executor: $executor"
        exit 1
    fi
    echo "[e2e] Verification PASSED for executor: $executor"
}

# Run for mock
run_e2e_for_executor "mock"

# Run for qiskit_aer
run_e2e_for_executor "qiskit_aer"

echo ""
echo "[e2e] All E2E Tests completed successfully!"
