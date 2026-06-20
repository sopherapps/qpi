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
DATA_DIR="${PROJECT_ROOT}/data"

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
    local executor="${1:-mock}"
    echo "[e2e] Syncing Python qpi-driver package dependencies (executor=$executor)..."

    local uv_extras="--extra cli"
    local pip_extras="cli"

    case "$executor" in
        qiskit_aer)
            uv_extras="$uv_extras --extra aer"
            pip_extras="$pip_extras,aer"
            ;;
        quantify)
            uv_extras="$uv_extras --extra quantify"
            pip_extras="$pip_extras,quantify"
            ;;
        qblox)
            uv_extras="$uv_extras --extra qblox"
            pip_extras="$pip_extras,qblox"
            ;;
        # mock and any other executor: cli only
    esac

    if command -v uv >/dev/null 2>&1; then
        uv sync --project "${PROJECT_ROOT}/qpi-driver" $uv_extras
    else
        echo "[e2e] uv not found, falling back to pip..."
        pip install -e "${PROJECT_ROOT}/qpi-driver[$pip_extras]"
    fi

    if [ "$(uname)" = "Darwin" ]; then
        codesign --force --deep --sign - "${PROJECT_ROOT}/qpi-driver/.venv/lib/python3.12/site-packages/qblox_instruments/assemblers/q1asm_macos" 2>/dev/null || true
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
    # Kill any existing process on port 8090 to prevent connecting to stale servers
    local existing_pid
    existing_pid=$(lsof -ti:8090 2>/dev/null || true)
    if [ -n "$existing_pid" ]; then
        echo "[e2e] Killing existing process on port 8090 (PIDs: $existing_pid)..."
        echo "$existing_pid" | xargs kill -9 || true
        sleep 1
    fi

    echo "[e2e] Initializing database and creating superuser..."
    rm -rf "${PROJECT_ROOT}/bin/pb_data" "${PROJECT_ROOT}/bin/data"
    "${PROJECT_ROOT}/bin/qpi" superuser upsert admin@example.com supersecretpassword1234 --dir "${PROJECT_ROOT}/bin/pb_data"

    echo "[e2e] Starting PocketBase server..."
    mkdir -p "${DATA_DIR}"

    # Run from a temp directory to avoid picking up qpi.config.yml from project root
    (cd "$(mktemp -d)" && "${PROJECT_ROOT}/bin/qpi" serve --dir "${PROJECT_ROOT}/bin/pb_data" --dev > "${DATA_DIR}/pocketbase.log" 2>&1) &
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
    DRIVER_LOG="${DATA_DIR}/${executor}-driver.log"

    echo "[e2e] Starting Python driver with executor: $executor..."
    local py
    py="$(detect_python)"

    local extra_flags=""
    if [ "$executor" = "quantify" ] || [ "$executor" = "qblox" ]; then
        extra_flags="--quantify-hardware-config ${PROJECT_ROOT}/qpi-driver/tests/fixtures/quantify.hardware.json --quantify-device-config ${PROJECT_ROOT}/qpi-driver/tests/fixtures/quantify.device.yml"
    fi

    # Fetch the CA fingerprint from the orchestrator for TLS verification
    local ca_fingerprint
    ca_fingerprint=$(curl -s http://127.0.0.1:8090/api/pub/root-ca.pem | openssl x509 -outform DER | sha256sum | cut -d' ' -f1)
    if [ -z "$ca_fingerprint" ]; then
        echo "[e2e] Failed to fetch CA fingerprint from orchestrator"
        exit 1
    fi
    echo "[e2e] CA fingerprint: $ca_fingerprint"

    QPI_ACCESS_TOKEN=my-super-secret-token-12345 "$py" -u -m qpi_driver.cli start \
        --executor "$executor" \
        --data-dir "${PROJECT_ROOT}/bin/data" \
        --ca-fingerprint "$ca_fingerprint" \
        --is-dummy $extra_flags >"$DRIVER_LOG" 2>&1 &
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
