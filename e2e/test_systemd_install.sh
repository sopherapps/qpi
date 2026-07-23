#!/usr/bin/env bash
set -e

echo "Starting systemd container..."
CONTAINER_ID=$(docker run -d --platform linux/amd64 --privileged --cgroupns=host --tmpfs /tmp --tmpfs /run -v /sys/fs/cgroup:/sys/fs/cgroup:rw jrei/systemd-ubuntu:22.04)

# Ensure container is destroyed on exit
trap "echo 'Cleaning up...'; docker stop \$CONTAINER_ID >/dev/null; docker rm \$CONTAINER_ID >/dev/null" EXIT

# Give systemd a moment to initialize
sleep 3

# Resolve absolute path to project root
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "Container started: $CONTAINER_ID"
echo "Copying installer to container..."
# Copy from project root to ensure robustness regardless of execution directory
docker cp "$PROJECT_ROOT/qpi-driver/install-systemd.sh" $CONTAINER_ID:/install-systemd.sh
docker exec $CONTAINER_ID chmod +x /install-systemd.sh

# Need to install curl and sudo as they are missing in basic ubuntu image
docker exec $CONTAINER_ID apt-get update
docker exec $CONTAINER_ID apt-get install -y curl sudo

echo "Running installer..."
docker exec -e QPI_TOKEN="mock_token" \
            -e QPI_ADDR="http://mock" \
            -e CA_FINGERPRINT="mock_fingerprint" \
            -e QPU_NAME="mock_qpu" \
            -e OPERATION="process" \
            -e DEVICE="mock" \
            $CONTAINER_ID bash -c "/install-systemd.sh || true"

echo "Checking if service file was generated..."
if docker exec $CONTAINER_ID cat /etc/systemd/system/mock_qpu.qpi-driver.service | grep -q "QPI Driver Service"; then
    echo "SUCCESS: systemd service file generated correctly!"
else
    echo "FAILED: systemd service file not found or incorrect."
    exit 1
fi

echo "Checking if service can start..."
docker exec $CONTAINER_ID systemctl daemon-reload
docker exec $CONTAINER_ID systemctl start mock_qpu.qpi-driver

# We give it a moment to fail if it's going to
sleep 2

if docker exec $CONTAINER_ID systemctl status mock_qpu.qpi-driver | grep -q "active (running)"; then
    echo "SUCCESS: systemd service started successfully!"
else
    echo "FAILED: systemd service failed to start."
    docker exec $CONTAINER_ID systemctl status mock_qpu.qpi-driver || true
    docker exec $CONTAINER_ID journalctl -u mock_qpu.qpi-driver --no-pager || true
    exit 1
fi
