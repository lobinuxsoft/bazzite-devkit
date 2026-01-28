#!/bin/bash
# Bazzite Devkit Service Installer

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_DIR="/opt/bazzite-devkit"
SERVICE_DIR="$HOME/.config/systemd/user"

echo "=== Bazzite Devkit Service Installer ==="
echo

# Check if running as root (we don't want that for user service)
if [ "$EUID" -eq 0 ]; then
    echo "Please run without sudo. The service runs as your user."
    exit 1
fi

# Install service files
echo "[1/4] Installing service files..."
sudo mkdir -p "$INSTALL_DIR"
sudo cp "$SCRIPT_DIR/devkit_service.py" "$INSTALL_DIR/"
sudo chmod +x "$INSTALL_DIR/devkit_service.py"

# Install systemd user service
echo "[2/4] Installing systemd user service..."
mkdir -p "$SERVICE_DIR"
cat > "$SERVICE_DIR/bazzite-devkit.service" << EOF
[Unit]
Description=Bazzite Devkit Service
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/bin/python3 $INSTALL_DIR/devkit_service.py
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
EOF

# Enable and start service
echo "[3/4] Enabling and starting service..."
systemctl --user daemon-reload
systemctl --user enable bazzite-devkit
systemctl --user start bazzite-devkit

# Configure firewall if firewalld is present
echo "[4/4] Configuring firewall..."
if command -v firewall-cmd &> /dev/null; then
    sudo firewall-cmd --add-port=32000/tcp --permanent 2>/dev/null || true
    sudo firewall-cmd --reload 2>/dev/null || true
    echo "Firewall configured for port 32000"
else
    echo "No firewalld found, skipping firewall configuration"
    echo "If you have a firewall, manually allow port 32000/tcp"
fi

echo
echo "=== Installation Complete ==="
echo
echo "Service status:"
systemctl --user status bazzite-devkit --no-pager || true
echo
echo "Test the service:"
echo "  curl http://localhost:32000/properties.json"
echo
echo "Your Bazzite IP: $(hostname -I | awk '{print $1}')"
echo "Connect from Devkit Client using this IP"
