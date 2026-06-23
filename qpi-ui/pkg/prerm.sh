#!/bin/sh
set -e

# --- Systemd Handling (Debian/RPM) ---
if [ -d /run/systemd/system ]; then
    systemctl stop qpi.service || true

# --- OpenRC Handling (Alpine) ---
elif command -v rc-service >/dev/null 2>&1; then
    rc-service qpi stop || true
fi