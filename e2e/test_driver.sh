#!/bin/bash
# test_driver.sh — E2E tests for the qpi-driver (mock and qiskit_aer executors).
#
# Usage:
#   ./test_driver.sh [mock|qiskit_aer]
#   ./test_driver.sh          # runs both mock and qiskit_aer

set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
source "${DIR}/lib.sh"

# Build once
build_orchestrator
install_driver

run_driver_e2e() {
    local executor=$1
    echo ""
    echo "========================================================================"
    echo "[e2e] Running driver E2E for executor: $executor"
    echo "========================================================================"
    echo ""

    start_pocketbase
    seed_database
    start_driver "$executor"

    # Allow driver time to register and start polling
    sleep 2

    local status=0
    run_verify --driver || status=$?

    stop_driver
    stop_pocketbase

    if [ "$status" -ne 0 ]; then
        echo "[e2e] Driver E2E FAILED for executor: $executor"
        exit 1
    fi
    echo "[e2e] Driver E2E PASSED for executor: $executor"
}

if [ -n "$1" ]; then
    EXECUTOR="$1"
    if [ "$EXECUTOR" = "aer" ]; then
        EXECUTOR="qiskit_aer"
    fi
    if [ "$EXECUTOR" = "quantify" ]; then
        echo "[e2e] Live E2E tests are not supported for quantify (requires physical hardware config). Skipping..."
        exit 0
    fi
    run_driver_e2e "$EXECUTOR"
else
    run_driver_e2e "mock"
    run_driver_e2e "qiskit_aer"
fi

echo ""
echo "[e2e] All driver E2E tests completed successfully!"
