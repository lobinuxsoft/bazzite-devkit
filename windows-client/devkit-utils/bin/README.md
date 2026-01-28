# steam-shortcut-manager Binaries

This directory should contain the compiled `steam-shortcut-manager` binaries.

## Required Binaries

- `steam-shortcut-manager-linux` - Linux AMD64 binary (runs on the target device)
- `steam-shortcut-manager.exe` - Windows binary (optional, for local testing)

## Building the Binaries

### Prerequisites

- Go 1.19 or later

### Build Commands

The source code is included as a submodule at `steam-shortcut-manager/` in the repo root.

```bash
# From the repo root, navigate to the submodule
cd steam-shortcut-manager

# Build for Linux (target device - Bazzite, etc.)
GOOS=linux GOARCH=amd64 go build -o ../windows-client/devkit-utils/bin/steam-shortcut-manager-linux

# Build for Windows (optional, for local testing)
GOOS=windows GOARCH=amd64 go build -o ../windows-client/devkit-utils/bin/steam-shortcut-manager.exe
```

### On Windows (PowerShell)

```powershell
cd steam-shortcut-manager

# Build for Linux
$env:GOOS="linux"; $env:GOARCH="amd64"
go build -o ../windows-client/devkit-utils/bin/steam-shortcut-manager-linux

# Build for Windows
$env:GOOS="windows"; $env:GOARCH="amd64"
go build -o ../windows-client/devkit-utils/bin/steam-shortcut-manager.exe

# Reset environment
Remove-Item Env:GOOS
Remove-Item Env:GOARCH
```

## Verification

After building, verify the binaries:

```bash
# Check Linux binary (on Linux or WSL)
file steam-shortcut-manager-linux
# Expected: ELF 64-bit LSB executable, x86-64

# Test the binary
./steam-shortcut-manager-linux --help
```

## Usage by devkit_utils

The `devkit_utils` Python module automatically selects the correct binary based on the platform:

- On Windows: Uses `steam-shortcut-manager.exe`
- On Linux: Uses `steam-shortcut-manager-linux`

The binary is called with these commands:

- `add <name> <exe> --start-dir <dir> [--tags <tags>]` - Add a shortcut
- `remove <name>` - Remove a shortcut
- `list -o json` - List all shortcuts as JSON

## Note on Deployment

When deploying to a target Bazzite/Linux device via rsync, ensure:

1. The `steam-shortcut-manager-linux` binary is included in the rsync transfer
2. The binary has execute permissions: `chmod +x steam-shortcut-manager-linux`
