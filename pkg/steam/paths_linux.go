//go:build !windows

package steam

import (
	"os"
	"path/filepath"
)

// getBaseDir returns the Steam base directory on Linux/Unix systems.
func getBaseDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	// Primary location: ~/.steam/steam
	steamDir := filepath.Join(home, ".steam", "steam")
	if _, err := os.Stat(steamDir); err == nil {
		return steamDir, nil
	}

	// Fallback: ~/.local/share/Steam
	steamDir = filepath.Join(home, ".local", "share", "Steam")
	if _, err := os.Stat(steamDir); err == nil {
		return steamDir, nil
	}

	// Flatpak location
	steamDir = filepath.Join(home, ".var", "app", "com.valvesoftware.Steam", ".steam", "steam")
	if _, err := os.Stat(steamDir); err == nil {
		return steamDir, nil
	}

	return "", ErrSteamNotFound
}

// IsSteamRunning checks if Steam is currently running on Linux.
func IsSteamRunning() (bool, error) {
	// Check for Steam's lock file
	home, err := os.UserHomeDir()
	if err != nil {
		return false, err
	}

	lockFile := filepath.Join(home, ".steam", "steam.pid")
	if _, err := os.Stat(lockFile); err == nil {
		// Lock file exists, Steam might be running
		// For a more accurate check, we'd need to verify the PID
		return true, nil
	}

	return false, nil
}
