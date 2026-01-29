// Package embedded provides embedded binaries for deployment
package embedded

import (
	_ "embed"
)

// SteamShortcutManager is the embedded Linux binary for steam-shortcut-manager
//
//go:embed steam-shortcut-manager
var SteamShortcutManager []byte

// SteamShortcutManagerName is the filename for the binary
const SteamShortcutManagerName = "steam-shortcut-manager"
