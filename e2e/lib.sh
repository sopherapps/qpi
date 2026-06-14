#!/bin/bash
# lib.sh — Shared E2E utilities for the QPi control stack.
#
# Usage:
#   source "$(dirname "$0")/lib.sh"
#
# Provides: build_orchestrator, build_js_client, detect_python, install_driver,
#           install_py_client, start_pocketbase, stop_pocketbase, seed_database,
#           start_driver, stop_driver, run_verify

set -e

# ---------------------------------------------------------------------------
# Paths
# ---------------------------------------------------------------------------
E2E_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
PROJECT_ROOT="$(dirname "$E2E_DIR")"

# ---------------------------------------------------------------------------
# State
# ---------------------------------------------------------------------------
PB_PID=""
DRIVER_PID=""

# ---------------------------------------------------------------------------
# Cleanup
# ---------------------------------------------------------------------------
cleanup() {
    echo "[e2e] Cleaning up background processes..."
    stop_driver
    stop_pocketbase
    rm -rf "${PROJECT_ROOT}/bin/pb_data" "${PROJECT_ROOT}/bin/data"
}

trap cleanup EXIT

# ---------------------------------------------------------------------------
# Build
# ---------------------------------------------------------------------------
build_orchestrator() {
    echo "[e2e] Building Go orchestrator..."
    mkdir -p "${PROJECT_ROOT}/bin"
    (cd "${PROJECT_ROOT}/qpi-ui" && go build -o ../bin/qpi .)
}

build_js_client() {
    echo "[e2e] Building JS client SDK..."
    (cd "${PROJECT_ROOT}/qpi-client/js" && npm ci && npm run build)
}

# ---------------------------------------------------------------------------
# Python detection
# ---------------------------------------------------------------------------
detect_python() {
    if [ -n "$VIRTUAL_ENV" ]; then
        PYTHON="python"
    elif [ -d "${PROJECT_ROOT}/qpi-driver/.venv" ]; then
        PYTHON="${PROJECT_ROOT}/qpi-driver/.venv/bin/python"
    elif [ -d "${PROJECT_ROOT}/.venv" ]; then
        PYTHON="${PROJECT_ROOT}/.venv/bin/python"
    else
        PYTHON="python"
    fi
    echo "$PYTHON"
}

# ---------------------------------------------------------------------------
# Install packages
# ---------------------------------------------------------------------------
install_driver() {
    echo "[e2e] Syncing Python qpi-driver package dependencies..."
    if command -v uv >/dev/null 2>&1; then
        uv sync --project "${PROJECT_ROOT}/qpi-driver" --extra cli --extra aer --extra quantify
    else
        local py
        py="$(detect_python)"
        "$py" -m pip install -e "${PROJECT_ROOT}/qpi-driver[cli,aer,quantify]"
    fi
}

install_py_client() {
    echo "[e2e] Installing Python qpi-client package..."
    local py
    py="$(detect_python)"
    if command -v uv >/dev/null 2>&1; then
        uv pip install --python "$py" -e "${PROJECT_ROOT}/qpi-client/py"
    else
        "$py" -m pip install -e "${PROJECT_ROOT}/qpi-client/py"
    fi
}

# ---------------------------------------------------------------------------
# PocketBase lifecycle
# ---------------------------------------------------------------------------
start_pocketbase() {
    echo "[e2e] Initializing database and creating superuser..."
    rm -rf "${PROJECT_ROOT}/bin/pb_data" "${PROJECT_ROOT}/bin/data"
    "${PROJECT_ROOT}/bin/qpi" superuser upsert admin@example.com supersecretpassword1234 --dir "${PROJECT_ROOT}/bin/pb_data"

    echo "[e2e] Starting PocketBase server..."
    "${PROJECT_ROOT}/bin/qpi" serve --dir "${PROJECT_ROOT}/bin/pb_data" --dev &
    PB_PID=$!

    echo "[e2e] Waiting for PocketBase to be ready..."
    local ready=0
    for i in {1..30}; do
        if curl -s http://127.0.0.1:8090/ >/dev/null; then
            echo "[e2e] PocketBase is ready."
            ready=1
            break
        fi
        sleep 1
    done

    if [ "$ready" -ne 1 ]; then
        echo "[e2e] Timeout waiting for PocketBase to start"
        exit 1
    fi
}

stop_pocketbase() {
    if [ -n "$PB_PID" ]; then
        echo "[e2e] Stopping PocketBase server (PID $PB_PID)..."
        kill "$PB_PID" 2>/dev/null || true
        wait "$PB_PID" 2>/dev/null || true
        PB_PID=""
    fi
}

# ---------------------------------------------------------------------------
# Seeding
# ---------------------------------------------------------------------------
seed_database() {
    echo "[e2e] Seeding database..."
    local py
    py="$(detect_python)"
    ADMIN_EMAIL=admin@example.com ADMIN_PASSWORD=supersecretpassword1234 "$py" "${E2E_DIR}/seed.py"
}

# ---------------------------------------------------------------------------
# Driver lifecycle
# ---------------------------------------------------------------------------
start_driver() {
    local executor="${1:-mock}"
    echo "[e2e] Starting Python driver with executor: $executor..."
    local py
    py="$(detect_python)"
    REGISTRATION_TOKEN=my-super-secret-token-12345 "$py" -m qpi_driver.cli start --executor "$executor" --data-dir "${PROJECT_ROOT}/bin/data" &
    DRIVER_PID=$!
}

stop_driver() {
    if [ -n "$DRIVER_PID" ]; then
        echo "[e2e] Stopping Python driver (PID $DRIVER_PID)..."
        kill "$DRIVER_PID" 2>/dev/null || true
        wait "$DRIVER_PID" 2>/dev/null || true
        DRIVER_PID=""
    fi
}

# ---------------------------------------------------------------------------
# Verification
# ---------------------------------------------------------------------------
run_verify() {
    local py
    py="$(detect_python)"
    echo "[e2e] Running verification script..."
    ADMIN_EMAIL=admin@example.com ADMIN_PASSWORD=supersecretpassword1234 "$py" "${E2E_DIR}/verify.py" "$@"
}
