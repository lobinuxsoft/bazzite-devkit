# Bazzite Devkit Service

A lightweight HTTP service that enables the SteamOS Devkit Client to connect to Bazzite (and other non-SteamOS Linux distributions).

## What it does

The service runs on port 32000 and emulates the SteamOS Devkit Service API:

- `/properties.json` - Returns device metadata and SSH login credentials
- `/login-name` - Returns the SSH username (legacy endpoint)
- `/register` - Registers SSH public keys for passwordless authentication

## Installation

### Quick Install

```bash
# Copy service files
sudo mkdir -p /opt/bazzite-devkit
sudo cp devkit_service.py /opt/bazzite-devkit/
sudo chmod +x /opt/bazzite-devkit/devkit_service.py

# Install systemd service (user service)
mkdir -p ~/.config/systemd/user/
cp bazzite-devkit.service ~/.config/systemd/user/

# Enable and start
systemctl --user daemon-reload
systemctl --user enable bazzite-devkit
systemctl --user start bazzite-devkit
```

### Manual Run

```bash
python3 devkit_service.py
```

Options:
- `--port PORT` - Listen on a different port (default: 32000)
- `--bind ADDRESS` - Bind to specific address (default: 0.0.0.0)
- `--debug` - Enable debug logging

## Firewall

If you have a firewall enabled, allow port 32000:

```bash
# For firewalld (Fedora/Bazzite)
sudo firewall-cmd --add-port=32000/tcp --permanent
sudo firewall-cmd --reload

# For ufw (Ubuntu)
sudo ufw allow 32000/tcp
```

## Verify it's working

```bash
# Check service status
systemctl --user status bazzite-devkit

# Test the endpoint
curl http://localhost:32000/properties.json
```

## Requirements

- Python 3.6+
- SSH server running (for file transfers)
- Network access from the development machine

## Troubleshooting

### Port 32000 unreachable

1. Check the service is running: `systemctl --user status bazzite-devkit`
2. Check firewall: `sudo firewall-cmd --list-ports`
3. Check the port is listening: `ss -tlnp | grep 32000`

### SSH connection fails

1. Ensure SSH is enabled: `sudo systemctl enable --now sshd`
2. Check your SSH key is registered: `cat ~/.ssh/authorized_keys`

### Devkit scripts not found

Make sure devkit-utils are installed in `~/devkit-utils/` (the client uploads these automatically on first connection).
