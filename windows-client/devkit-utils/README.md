# devkit-utils

Scripts and supporting utility module uploaded to the devkit device by the devkit client.

## Bazzite/Universal Devkit Modifications

This fork has been modified to work on non-SteamOS Linux distributions (Bazzite, etc.) by using `steam-shortcut-manager` instead of Steam's IPC pipe for shortcut management.

### Key Changes

1. **`devkit_utils/__init__.py`** - Added steam-shortcut-manager integration functions:
   - `add_steam_shortcut()` - Add a shortcut to Steam
   - `remove_steam_shortcut()` - Remove a shortcut from Steam
   - `list_steam_shortcuts()` - List all Steam shortcuts
   - `shortcut_manager_available()` - Check if the binary is available
   - Modified `validate_steam_client()` to only check Steam is installed (not running)

2. **`devkit_utils/resolve.py`** - Rewritten to use steam-shortcut-manager instead of IPC

3. **`steam-client-create-shortcut`** - Rewritten to use steam-shortcut-manager

4. **`steamos-delete`** - Updated to use new shortcut functions

### Required Binaries

Place the following binaries in the `bin/` directory:

- `steam-shortcut-manager-linux` - Linux AMD64 binary (for target device)
- `steam-shortcut-manager.exe` - Windows binary (optional, for local testing)

See `bin/README.md` for build instructions.

## Script Categories

### Works on All Systems (SteamOS + Bazzite)

- `steam-client-create-shortcut` - Creates Steam shortcuts for devkit games
- `steamos-delete` - Deletes devkit titles
- `steamos-list-games` - Lists installed devkit games
- `steamos-prepare-upload` - Prepares system for game upload

### SteamOS Only (uses Steam IPC pipe)

These scripts require SteamOS's special Steam client IPC:

- `steam-devkit-rpc` - Generic RPC interface to Steam client commands
- `steamos-dump-controller-config` - Exports controller configuration
- `steamos-set-steam-client` - Configures which Steam client to use
- `steamos-get-status` - Reports system and Steam client status

## Directory Structure

```
devkit-utils/
├── devkit_utils/
│   ├── __init__.py          # Core utility functions
│   └── resolve.py           # Shortcut synchronization
├── bin/
│   ├── steam-shortcut-manager-linux  # Linux binary (required)
│   ├── steam-shortcut-manager.exe    # Windows binary (optional)
│   └── README.md
├── steam-client-create-shortcut      # Shortcut creation script
├── steamos-delete                    # Title deletion script
├── steamos-list-games               # Game listing script
├── steamos-prepare-upload           # Upload preparation script
├── steam-devkit-rpc                 # SteamOS-only RPC
├── steamos-dump-controller-config   # SteamOS-only config dump
├── steamos-set-steam-client         # SteamOS-only client switch
├── steamos-get-status               # Status reporting
└── README.md
```

## Usage

The devkit client automatically uploads these scripts to the target device via rsync. The scripts are then executed remotely to manage devkit games.

### Example: Creating a Shortcut

```python
import devkit_utils

# Add a shortcut
devkit_utils.add_steam_shortcut(
    name='Devkit: MyGame',
    exe='/home/user/devkit-game/MyGame/launch.sh',
    start_dir='/home/user/devkit-game/MyGame',
    tags=['devkit']
)

# List shortcuts
shortcuts = devkit_utils.list_steam_shortcuts()
for sc in shortcuts:
    print(sc['AppName'])

# Remove a shortcut
devkit_utils.remove_steam_shortcut('Devkit: MyGame')
```

### Example: Synchronizing Shortcuts

```python
from devkit_utils.resolve import resolve_shortcuts

# Sync disk state with Steam shortcuts
resolve_shortcuts()
```
