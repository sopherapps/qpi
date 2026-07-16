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

# The version of qpi-driver to install.
# This should match the qpi-ui version if provided via environment variable.
QPI_DRIVER_VERSION="${QPI_DRIVER_VERSION:-}"

# Detect the real user who ran sudo
if [ -n "$SUDO_USER" ]; then
    REAL_USER="$SUDO_USER"
    REAL_HOME=$(getent passwd "$SUDO_USER" | cut -d: -f6)
else
    REAL_USER=$(whoami)
    REAL_HOME=$HOME
fi

echo "Installing for user: $REAL_USER (Home: $REAL_HOME)"
echo ""

# 1. Prompt for configuration if not provided via environment
[ -z "$QPI_TOKEN" ] && read -p "Enter QPI Access Token: " QPI_TOKEN
[ -z "$QPI_ADDR" ] && read -p "Enter QPI Server Address (e.g. https://qpi.sopherapps.se): " QPI_ADDR
[ -z "$CA_FINGERPRINT" ] && read -p "Enter CA Fingerprint: " CA_FINGERPRINT
[ -z "$QPU_NAME" ] && read -p "Enter QPU Name (e.g. rigetti-aspen-1): " QPU_NAME
[ -z "$EXECUTOR" ] && read -p "Enter Executor (mock, qiskit_aer, quantify, qblox, presto) [mock]: " EXECUTOR
EXECUTOR=${EXECUTOR:-mock}

# 2. Install uv if not present
if ! sudo -u "$REAL_USER" command -v uv >/dev/null 2>&1; then
    echo "Installing 'uv' for fast python package management..."
    sudo -u "$REAL_USER" bash -c "curl -LsSf https://astral.sh/uv/install.sh | sh"
    source "$REAL_HOME/.local/bin/env"
fi

# 3. Install qpi-driver using uv tool
echo "Installing qpi-driver via uv tool..."

if [ -z "$QPI_DRIVER_VERSION" ]; then
    VERSION_SUFFIX=""
else
    VERSION_SUFFIX="==$QPI_DRIVER_VERSION"
fi

# Ensure the correct extras are added based on the executor
if [ "$EXECUTOR" = "qblox" ]; then
    sudo -u "$REAL_USER" "$REAL_HOME/.local/bin/uv" tool install --python 3.12 --prerelease allow "qpi-driver[cli,qblox]${VERSION_SUFFIX}"
elif [ "$EXECUTOR" = "quantify" ]; then
    sudo -u "$REAL_USER" "$REAL_HOME/.local/bin/uv" tool install --python 3.12 "qpi-driver[cli,quantify]${VERSION_SUFFIX}"
elif [ "$EXECUTOR" = "qiskit_aer" ]; then
    sudo -u "$REAL_USER" "$REAL_HOME/.local/bin/uv" tool install --python 3.12 "qpi-driver[cli,aer]${VERSION_SUFFIX}"
else
    sudo -u "$REAL_USER" "$REAL_HOME/.local/bin/uv" tool install --python 3.12 "qpi-driver[cli]${VERSION_SUFFIX}"
fi

# 4. Create systemd unit file
SERVICE_FILE="/etc/systemd/system/${QPU_NAME}.qpi-driver.service"
echo "Creating systemd service at $SERVICE_FILE..."

cat > "$SERVICE_FILE" <<EOF
[Unit]
Description=QPI Driver Service ($QPU_NAME)
After=network.target

[Service]
Type=simple

Environment="QPI_ACCESS_TOKEN=$QPI_TOKEN"
# Standard Python output buffering disabled to ensure logs appear immediately in journalctl
Environment=PYTHONUNBUFFERED=1

ExecStart=$REAL_HOME/.local/bin/qpi-driver start \\
        --ca-fingerprint $CA_FINGERPRINT \\
        --qpi-addr $QPI_ADDR \\
        --name "$QPU_NAME" \\
        --executor "$EXECUTOR"

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
