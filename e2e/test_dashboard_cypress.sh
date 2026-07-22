#!/bin/bash
# test_dashboard_cypress.sh — E2E tests for the React dashboard using Cypress.
#
# Usage:
#   ./test_dashboard_cypress.sh

set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
source "${DIR}/lib.sh"

echo ""
echo "========================================================================"
echo "[e2e] Running Cypress E2E Dashboard tests"
echo "========================================================================"
echo ""

# Ensure dashboard dependencies are installed, including Cypress, and compiled
echo "[e2e] Preparing React dashboard..."
(cd "${PROJECT_ROOT}/qpi-ui/internal/dashboard" && npm ci)

echo "[e2e] Compiling static assets..."
(cd "${PROJECT_ROOT}/qpi-ui/internal/dashboard" && npm run build)

build_server
install_driver

# --enable-driver-framework is on so the Drivers registration page (RFC 0001
# Phase 1) is exercised alongside the legacy dashboard specs; the flag is
# additive and does not change existing QPU/job behaviour.
start_pocketbase --enable-driver-framework
seed_database
start_driver "mock"

# Allow driver time to register and start polling
sleep 2

status=0
echo "[e2e] Executing Cypress spec tests..."
(cd "${PROJECT_ROOT}/qpi-ui/internal/dashboard" && npx cypress run "$@") || status=$?

stop_driver
stop_pocketbase

if [ "$status" -ne 0 ]; then
    echo "[e2e] Cypress E2E Dashboard FAILED"
    exit 1
fi

echo "[e2e] Cypress E2E Dashboard PASSED"
