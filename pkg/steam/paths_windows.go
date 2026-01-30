//go:build windows

package steam

import (
	"golang.org/x/sys/windows/registry"
)

// getBaseDir returns the Steam base directory on Windows using the registry.
func getBaseDir() (string, error) {
	// Try 64-bit registry first
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Wow6432Node\Valve\Steam`, registry.QUERY_VALUE)
	if err != nil {
		// Fall back to 32-bit registry
		key, err = registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Valve\Steam`, registry.QUERY_VALUE)
		if err != nil {
			return "", ErrSteamNotFound
		}
	}
	defer key.Close()

	steamPath, _, err := key.GetStringValue("InstallPath")
	if err != nil {
		return "", ErrSteamNotFound
	}

	return steamPath, nil
}

// IsSteamRunning checks if Steam is currently running on Windows.
func IsSteamRunning() (bool, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Valve\Steam\ActiveProcess`, registry.QUERY_VALUE)
	if err != nil {
		// Key doesn't exist, Steam probably not running
		return false, nil
	}
	defer key.Close()

	pid, _, err := key.GetIntegerValue("pid")
	if err != nil {
		return false, nil
	}

	return pid != 0, nil
}
