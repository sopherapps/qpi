#!/bin/bash
# test_driver.sh — E2E tests for the qpi-driver (mock, qiskit_aer, qblox, quantify executors).
#
# Usage:
#   ./test_driver.sh [mock|qiskit_aer|qblox|quantify]
#   ./test_driver.sh          # runs all mock, qiskit_aer, qblox, and quantify

set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
source "${DIR}/lib.sh"

# Build once
build_server

run_driver_e2e() {
    local executor=$1
    echo ""
    echo "========================================================================"
    echo "[e2e] Running driver E2E for executor: $executor"
    echo "========================================================================"
    echo ""

    install_driver "$executor"

    start_pocketbase
    seed_database "$executor"
    start_driver "$executor"

    # Allow driver time to register and start polling
    sleep 2

    local status=0
    run_verify --driver || status=$?

    if [ "$status" -eq 0 ]; then
        echo "[e2e] Testing QPU offline transition on connection drop..."
        stop_driver
        sleep 2
        local online_count
        online_count=$(curl -s "http://127.0.0.1:8090/api/qpus" | grep -o "online" | wc -l || true)
        if [ "$online_count" -ne 0 ]; then
            echo "[e2e] ✗ FAILED: QPU is still online after driver was stopped!"
            status=1
        else
            echo "[e2e] ✓ QPU successfully transitioned to offline when driver stopped."
        fi
    else
        stop_driver
    fi

    stop_pocketbase

    if [ "$status" -ne 0 ]; then
        echo "[e2e] Driver E2E FAILED for executor: $executor"
        echo "=== Driver Logs ==="
        cat "${PROJECT_ROOT}/data/${executor}-driver.log" || true
        exit 1
    fi
    echo "[e2e] Driver E2E PASSED for executor: $executor"
    echo "=== Driver Logs ==="
    cat "${PROJECT_ROOT}/data/${executor}-driver.log" || true
}

if [ -n "$1" ]; then
    EXECUTOR="$1"
    if [ "$EXECUTOR" = "aer" ]; then
        EXECUTOR="qiskit_aer"
    fi
    run_driver_e2e "$EXECUTOR"
else
    run_driver_e2e "mock"
    run_driver_e2e "qiskit_aer"
    run_driver_e2e "qblox"
    run_driver_e2e "quantify"
fi

echo ""
echo "[e2e] All driver E2E tests completed successfully!"
