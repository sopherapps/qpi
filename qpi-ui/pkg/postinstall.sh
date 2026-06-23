#!/bin/sh
set -e

# --- Systemd Handling (Debian/RPM) ---
if [ -d /run/systemd/system ]; then
    systemctl daemon-reload
    systemctl enable qpi.service
    systemctl start qpi.service

# --- OpenRC Handling (Alpine) ---
elif command -v rc-update >/dev/null 2>&1; then
    # Add to the default runlevel so it boots automatically
    rc-update add qpi default
    # Start the service immediately
    rc-service qpi start
fi