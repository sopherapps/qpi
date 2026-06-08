#!/bin/bash
set -e

# Get the directory of this script
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
PROJECT_ROOT="$( dirname "$DIR" )"

cd "$PROJECT_ROOT"

# Cleanup background processes on exit
cleanup() {
    echo "[e2e] Cleaning up background processes..."
    if [ -n "$PB_PID" ]; then
        echo "[e2e] Stopping PocketBase server (PID $PB_PID)..."
        kill "$PB_PID" 2>/dev/null || true
    fi
    if [ -n "$DRIVER_PID" ]; then
        echo "[e2e] Stopping Python driver (PID $DRIVER_PID)..."
        kill "$DRIVER_PID" 2>/dev/null || true
    fi
    rm -rf bin/pb_data
}
trap cleanup EXIT

# Clear old data
rm -rf bin/pb_data

# Build Go orchestrator
echo "[e2e] Building Go orchestrator..."
mkdir -p bin
go build -o bin/qpi main.go

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

# Detect Python interpreter
PYTHON="python"
if [ -d ".venv" ]; then
    PYTHON="./.venv/bin/python"
fi

# Seed database
echo "[e2e] Seeding database..."
ADMIN_EMAIL=admin@example.com ADMIN_PASSWORD=supersecretpassword1234 $PYTHON e2e/seed.py

# Start Python driver
echo "[e2e] Starting Python driver..."
REGISTRATION_TOKEN=my-super-secret-token-12345 $PYTHON driver.py &
DRIVER_PID=$!

# Run verification script
echo "[e2e] Running verification script..."
ADMIN_EMAIL=admin@example.com ADMIN_PASSWORD=supersecretpassword1234 $PYTHON e2e/verify.py

echo "[e2e] E2E Tests completed successfully!"
