#!/usr/bin/env bash
set -e

# Interactive systemd installer for qpi-driver
echo "=========================================="
echo "    QPI Driver Systemd Installer          "
echo "=========================================="

if [ "$EUID" -ne 0 ]; then 
  echo "Please run as root (or with sudo) so we can create the systemd service."
  exit 1
fi


# Detect the real user who ran sudo
if [ -n "$SUDO_USER" ]; then
    REAL_USER="$SUDO_USER"
    REAL_HOME=$(getent passwd "$SUDO_USER" | cut -d: -f6)
    if [ -z "$REAL_HOME" ]; then
        REAL_HOME=$(eval echo "~$SUDO_USER")
    fi
else
    REAL_USER=$(whoami)
    REAL_HOME=$HOME
fi

echo "Installing for user: $REAL_USER (Home: $REAL_HOME)"
echo ""

# 1. Prompt for configuration if not provided via environment
while [ -z "$QPI_TOKEN" ]; do read -p "Enter QPI Access Token: " QPI_TOKEN; done
while [ -z "$QPI_ADDR" ]; do read -p "Enter QPI Server Address (e.g. https://qpi.sopherapps.se): " QPI_ADDR; done
while [ -z "$CA_FINGERPRINT" ]; do read -p "Enter CA Fingerprint: " CA_FINGERPRINT; done
while [ -z "$QPU_NAME" ]; do read -p "Enter QPU Name (e.g. rigetti-aspen-1): " QPU_NAME; done

# Optional
[ -z "$EXECUTOR" ] && read -p "Enter Executor (mock, qiskit_aer, quantify, qblox, presto, bluefors_gen1) [mock]: " EXECUTOR
EXECUTOR=${EXECUTOR:-mock}

# bluefors_gen1 is a monitor, not an executor, but reuses the EXECUTOR
# variable to pick the extra + service shape below. It needs its own Control
# API config instead of the QPU/executor settings above.
if [ "$EXECUTOR" = "bluefors_gen1" ]; then
    [ -z "$BLUEFORS_BASE_URL" ] && read -p "Enter Bluefors Control API base URL (e.g. http://localhost:49099): " BLUEFORS_BASE_URL
    [ -z "$BLUEFORS_CHANNELS" ] && read -p "Enter Bluefors channels to poll, optionally suffixed with :unit (e.g. mapper.bf.tmc:K,mapper.bf.pmc:mbar): " BLUEFORS_CHANNELS
fi

# The version of qpi-driver to install.
# This should match the qpi-ui version if provided via environment variable.
QPI_DRIVER_VERSION="${QPI_DRIVER_VERSION:-}"
QPI_DATA_DIR="${QPI_DATA_DIR:-"/var/qpi-driver/${QPU_NAME}"}"
QPI_CA_FILE="${QPI_CA_FILE:-"${QPI_DATA_DIR}/qpi.ca.pem"}"
QPI_QUANTIFY_DEVICE_CONFIG="${QPI_QUANTIFY_DEVICE_CONFIG:-"${QPI_DATA_DIR}/quantify.device.yml"}"
QPI_QUANTIFY_HARDWARE_CONFIG="${QPI_QUANTIFY_HARDWARE_CONFIG:-"${QPI_DATA_DIR}/quantify.hardware.json"}"

# 2. Locate or install uv
if sudo -u "$REAL_USER" command -v uv >/dev/null 2>&1; then
    UV_PATH=$(sudo -u "$REAL_USER" command -v uv)
elif [ -f "$REAL_HOME/.local/bin/uv" ]; then
    UV_PATH="$REAL_HOME/.local/bin/uv"
else
    echo "Installing 'uv' for fast python package management..."
    sudo -u "$REAL_USER" bash -c "curl -LsSf https://astral.sh/uv/install.sh | sh"
    UV_PATH="$REAL_HOME/.local/bin/uv"
    [ -f "$REAL_HOME/.local/bin/env" ] && source "$REAL_HOME/.local/bin/env" || true
fi

# Ensure data directory exists and is owned by the real user
echo "Creating data directory at $QPI_DATA_DIR..."
mkdir -p "$QPI_DATA_DIR"
chown -R "$REAL_USER" "$QPI_DATA_DIR"

# 3. Install qpi-driver using uv tool
echo "Installing qpi-driver via uv tool..."

if [ -z "$QPI_DRIVER_VERSION" ]; then
    VERSION_SUFFIX=""
else
    VERSION_SUFFIX="==$QPI_DRIVER_VERSION"
fi

# Ensure the correct extras are added based on the executor
if [ "$EXECUTOR" = "qblox" ]; then
    sudo -u "$REAL_USER" "$UV_PATH" tool install --python 3.12 --prerelease allow "qpi-driver[cli,qblox]${VERSION_SUFFIX}"
elif [ "$EXECUTOR" = "quantify" ]; then
    sudo -u "$REAL_USER" "$UV_PATH" tool install --python 3.12 "qpi-driver[cli,quantify]${VERSION_SUFFIX}"
elif [ "$EXECUTOR" = "qiskit_aer" ]; then
    sudo -u "$REAL_USER" "$UV_PATH" tool install --python 3.12 "qpi-driver[cli,aer]${VERSION_SUFFIX}"
elif [ "$EXECUTOR" = "bluefors_gen1" ]; then
    sudo -u "$REAL_USER" "$UV_PATH" tool install --python 3.12 "qpi-driver[cli,bluefors_gen1]${VERSION_SUFFIX}"
else
    sudo -u "$REAL_USER" "$UV_PATH" tool install --python 3.12 "qpi-driver[cli]${VERSION_SUFFIX}"
fi

# 4. Create systemd unit file
SERVICE_FILE="/etc/systemd/system/${QPU_NAME}.qpi-driver.service"
echo "Creating systemd service at $SERVICE_FILE..."

# bluefors_gen1 is a monitor, so it runs through `qpi-driver monitor --kind`
# with its own Control API config, instead of `start --executor` (RFC 0001
# §7, Phase 3).
if [ "$EXECUTOR" = "bluefors_gen1" ]; then
    BLUEFORS_ENV_LINES="Environment=\"BLUEFORS_BASE_URL=$BLUEFORS_BASE_URL\"
Environment=\"BLUEFORS_CHANNELS=$BLUEFORS_CHANNELS\"
Environment=\"BLUEFORS_API_KEY=$BLUEFORS_API_KEY\""
    EXEC_START_CMD="$REAL_HOME/.local/bin/qpi-driver monitor \\
        --kind bluefors_gen1 \\
        --ca-fingerprint $CA_FINGERPRINT \\
        --qpi-addr $QPI_ADDR \\
        --name \"$QPU_NAME\""
else
    BLUEFORS_ENV_LINES=""
    EXEC_START_CMD="$REAL_HOME/.local/bin/qpi-driver start \\
        --ca-fingerprint $CA_FINGERPRINT \\
        --qpi-addr $QPI_ADDR \\
        --name \"$QPU_NAME\" \\
        --executor \"$EXECUTOR\""
fi

cat > "$SERVICE_FILE" <<EOF
[Unit]
Description=QPI Driver Service ($QPU_NAME)
After=network.target

[Service]
Type=simple

Environment="QPI_ACCESS_TOKEN=$QPI_TOKEN"
Environment="QPI_DATA_DIR=$QPI_DATA_DIR"
Environment="QPI_CA_FILE=$QPI_CA_FILE"
Environment="QPI_QUANTIFY_DEVICE_CONFIG=$QPI_QUANTIFY_DEVICE_CONFIG"
Environment="QPI_QUANTIFY_HARDWARE_CONFIG=$QPI_QUANTIFY_HARDWARE_CONFIG"
$BLUEFORS_ENV_LINES
# Standard Python output buffering disabled to ensure logs appear immediately in journalctl
Environment=PYTHONUNBUFFERED=1

ExecStart=$EXEC_START_CMD

Restart=on-failure
User=$REAL_USER

# Journalctl logging configuration
StandardOutput=journal
StandardError=journal
SyslogIdentifier=${QPU_NAME}.qpi-driver

[Install]
WantedBy=multi-user.target
EOF

# 5. Enable and start the service
echo "Reloading systemd daemon..."
systemctl daemon-reload

echo "Enabling and starting ${QPU_NAME}.qpi-driver.service..."
systemctl enable "${QPU_NAME}.qpi-driver.service"
systemctl start "${QPU_NAME}.qpi-driver.service"

echo "=========================================="
echo "Installation complete!"
echo "Service status:"
systemctl status "${QPU_NAME}.qpi-driver.service" --no-pager || true
echo "=========================================="
echo "To view logs, run: journalctl -u ${QPU_NAME}.qpi-driver.service -f"
