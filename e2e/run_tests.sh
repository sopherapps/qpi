#!/bin/bash
# run_tests.sh — Master E2E server.
#
# Delegates to per-component E2E scripts:
#   test_driver.sh    — qpi-driver (mock, qiskit_aer)
#   test_client_py.sh — Python qpi-client SDK smoke
#   test_client_js.sh — JavaScript qpi-client SDK smoke
#   test_client_go.sh — Go qpi-client SDK smoke
#
# Usage:
#   ./run_tests.sh              # run all E2E tests
#   ./run_tests.sh driver       # driver only (accepts optional executor arg)
#   ./run_tests.sh client-py    # Python client only
#   ./run_tests.sh client-js    # JS client only
#   ./run_tests.sh client-go    # Go client only

set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"

if [ -n "$1" ]; then
    case "$1" in
        driver)
            shift
            exec "${DIR}/test_driver.sh" "$@"
            ;;
        client-py)
            exec "${DIR}/test_client_py.sh"
            ;;
        client-js)
            exec "${DIR}/test_client_js.sh"
            ;;
        client-go)
            exec "${DIR}/test_client_go.sh"
            ;;
        *)
            echo "Unknown target: $1"
            echo "Usage: $0 [driver|client-py|client-js|client-go]"
            exit 1
            ;;
    esac
fi

# Run everything
"${DIR}/test_driver.sh"
"${DIR}/test_client_py.sh"
"${DIR}/test_client_js.sh"
"${DIR}/test_client_go.sh"

echo ""
echo "[e2e] All E2E tests completed successfully!"

