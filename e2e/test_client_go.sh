#!/bin/bash
# test_client_go.sh — E2E smoke test for the Go qpi-client SDK.
#
# Usage:
#   ./test_client_go.sh

set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
source "${DIR}/lib.sh"

echo ""
echo "========================================================================"
echo "[e2e] Running Go client E2E smoke test"
echo "========================================================================"
echo ""

build_orchestrator
install_driver

start_pocketbase
seed_database
start_driver "mock"

# Allow driver time to register and start polling
sleep 2

status=0
run_verify --client-go || status=$?

stop_driver
stop_pocketbase

if [ "$status" -ne 0 ]; then
    echo "[e2e] Go client E2E FAILED"
    exit 1
fi

echo "[e2e] Go client E2E PASSED"
