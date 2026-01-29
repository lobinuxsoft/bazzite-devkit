// Package shortcuts provides Steam shortcut management functions
// using the steam-shortcut-manager library and binary
package shortcuts

import (
	"fmt"
	"strings"

	"github.com/lobinuxsoft/bazzite-devkit/internal/device"
	"github.com/shadowblip/steam-shortcut-manager/pkg/remote"
	"github.com/shadowblip/steam-shortcut-manager/pkg/shortcut"
	"github.com/shadowblip/steam-shortcut-manager/pkg/steam"
)

// ArtworkConfig holds the artwork URLs to download
type ArtworkConfig struct {
	GridPortrait  string // 600x900 portrait grid (e.g. {appid}p.png)
	GridLandscape string // 920x430 landscape grid (e.g. {appid}.png)
	HeroImage     string // 1920x620 hero banner (e.g. {appid}_hero.png)
	LogoImage     string // Logo with transparency (e.g. {appid}_logo.png)
	IconImage     string // Square icon (e.g. {appid}_icon.png)
}

// RemoteConfig holds the SSH connection parameters
type RemoteConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	KeyFile  string
}

// AddShortcut adds a Steam shortcut on a remote device
func AddShortcut(cfg *RemoteConfig, name, exe, startDir, launchOpts string, tags []string) error {
	return AddShortcutWithArtwork(cfg, name, exe, startDir, launchOpts, tags, nil, "")
}

// AddShortcutWithArtwork adds a Steam shortcut with custom artwork on a remote device.
// If binaryPath is provided, it will use the remote binary to apply artwork via Steam CEF API.
// If binaryPath is empty, artwork application will be skipped.
func AddShortcutWithArtwork(cfg *RemoteConfig, name, exe, startDir, launchOpts string, tags []string, artwork *ArtworkConfig, binaryPath string) error {
	// Create and connect remote client
	client := remote.NewClient(&remote.Config{
		Host:     cfg.Host,
		Port:     cfg.Port,
		User:     cfg.User,
		Password: cfg.Password,
		KeyFile:  cfg.KeyFile,
	})

	if err := client.Connect(); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer client.Close()

	// Set remote clients for library packages
	shortcut.SetRemoteClient(client)
	steam.SetRemoteClient(client)

	// Get all Steam users on the remote device
	users, err := steam.GetRemoteUsers()
	if err != nil {
		return fmt.Errorf("failed to get Steam users: %w", err)
	}

	if len(users) == 0 {
		return fmt.Errorf("no Steam users found on remote device")
	}

	// Format exe and startDir with quotes (Steam expects quoted paths)
	quotedExe := fmt.Sprintf("\"%s\"", exe)
	quotedStartDir := fmt.Sprintf("\"%s\"", startDir)

	// Calculate appID for artwork naming using quoted exe (matches Steam's internal calculation)
	appID := shortcut.CalculateAppID(quotedExe, name)
	fmt.Printf("[DEBUG] Calculated AppID for '%s' (exe: %s): %d\n", name, quotedExe, appID)

	// Add shortcut for all users
	for _, user := range users {
		shortcutsPath, err := steam.GetRemoteShortcutsPath(user)
		if err != nil {
			continue
		}

		// Load existing shortcuts or create new
		var shortcuts *shortcut.Shortcuts
		if steam.RemoteHasShortcuts(user) {
			shortcuts, err = shortcut.Load(shortcutsPath)
			if err != nil {
				return fmt.Errorf("failed to load shortcuts for user %s: %w", user, err)
			}
		} else {
			shortcuts = shortcut.NewShortcuts()
		}

		// Create new shortcut with quoted paths
		newShortcut := shortcut.NewShortcut(name, quotedExe, func(s *shortcut.Shortcut) {
			s.AllowDesktopConfig = 1
			s.AllowOverlay = 1
			s.StartDir = quotedStartDir
			s.LaunchOptions = launchOpts
			s.Appid = int64(appID)

			// Add tags
			s.Tags = map[string]interface{}{}
			for i, tag := range tags {
				s.Tags[fmt.Sprintf("%d", i)] = tag
			}
		})

		// Add to shortcuts collection
		if err := shortcuts.Add(newShortcut); err != nil {
			return fmt.Errorf("failed to add shortcut for user %s: %w", user, err)
		}

		// Save shortcuts
		if err := shortcut.Save(shortcuts, shortcutsPath); err != nil {
			return fmt.Errorf("failed to save shortcuts for user %s: %w", user, err)
		}

		// Verify the saved shortcut by re-reading it
		verifyShortcuts, err := shortcut.Load(shortcutsPath)
		if err != nil {
			fmt.Printf("[DEBUG] Failed to re-read shortcuts for verification: %v\n", err)
		} else {
			if savedSC, err := verifyShortcuts.LookupByName(name); err == nil {
				fmt.Printf("[DEBUG] VERIFICATION - Saved shortcut:\n")
				fmt.Printf("  AppName: %s\n", savedSC.AppName)
				fmt.Printf("  Exe: %s\n", savedSC.Exe)
				fmt.Printf("  StartDir: %s\n", savedSC.StartDir)
				fmt.Printf("  Appid (from file): %d\n", savedSC.Appid)
				fmt.Printf("  Expected Appid: %d\n", appID)
				if savedSC.Appid != int64(appID) {
					fmt.Printf("[WARNING] AppID mismatch! File has %d, expected %d\n", savedSC.Appid, appID)
				}
			} else {
				fmt.Printf("[DEBUG] Could not find saved shortcut by name: %v\n", err)
			}
		}
	}

	// Apply artwork using the remote binary if provided
	if artwork != nil && binaryPath != "" {
		fmt.Printf("[DEBUG] Applying artwork for AppID %d using remote binary: %s\n", appID, binaryPath)
		if err := applyArtworkViaBinary(client, binaryPath, appID, artwork); err != nil {
			fmt.Printf("[WARNING] Failed to apply artwork via binary: %v\n", err)
		}
	} else if artwork != nil {
		fmt.Printf("[WARNING] Artwork config provided but no binary path, skipping artwork application\n")
	}

	return nil
}

// applyArtworkViaBinary executes the steam-shortcut-manager binary on the remote device
// to apply artwork using the Steam CEF API
func applyArtworkViaBinary(client *remote.Client, binaryPath string, appID uint64, artwork *ArtworkConfig) error {
	// Build the command with flags
	var args []string
	args = append(args, "steamgriddb", "apply")
	args = append(args, fmt.Sprintf("--app-id=%d", appID))

	if artwork.GridPortrait != "" {
		args = append(args, fmt.Sprintf("--grid-portrait=%q", artwork.GridPortrait))
	}
	if artwork.GridLandscape != "" {
		args = append(args, fmt.Sprintf("--grid-landscape=%q", artwork.GridLandscape))
	}
	if artwork.HeroImage != "" {
		args = append(args, fmt.Sprintf("--hero=%q", artwork.HeroImage))
	}
	if artwork.LogoImage != "" {
		args = append(args, fmt.Sprintf("--logo=%q", artwork.LogoImage))
	}
	if artwork.IconImage != "" {
		args = append(args, fmt.Sprintf("--icon=%q", artwork.IconImage))
	}

	// Build full command
	cmd := fmt.Sprintf("%q %s", binaryPath, strings.Join(args, " "))
	fmt.Printf("[DEBUG] Executing remote command: %s\n", cmd)

	// Execute on remote device
	output, err := client.RunCommand(cmd)
	if err != nil {
		return fmt.Errorf("command failed: %w (output: %s)", err, output)
	}

	fmt.Printf("[DEBUG] Remote command output:\n%s\n", output)
	return nil
}

// EnsureBinaryExists checks if the steam-shortcut-manager binary exists on the remote device
// at the specified path. Returns true if it exists, false otherwise.
func EnsureBinaryExists(client *device.Client, remotePath string) bool {
	cmd := fmt.Sprintf("test -x %q && echo 'exists'", remotePath)
	output, err := client.RunCommand(cmd)
	if err != nil {
		return false
	}
	return strings.TrimSpace(output) == "exists"
}

// UploadBinary uploads the steam-shortcut-manager binary to the remote device
func UploadBinary(client *device.Client, binaryData []byte, remotePath string) error {
	fmt.Printf("[DEBUG] Uploading steam-shortcut-manager binary to %s (%d bytes)\n", remotePath, len(binaryData))

	// Write the binary file
	if err := client.WriteFile(remotePath, binaryData, 0755); err != nil {
		return fmt.Errorf("failed to write binary: %w", err)
	}

	// Verify it's executable
	cmd := fmt.Sprintf("chmod +x %q && test -x %q && echo 'ok'", remotePath, remotePath)
	output, err := client.RunCommand(cmd)
	if err != nil || strings.TrimSpace(output) != "ok" {
		return fmt.Errorf("failed to set executable permissions")
	}

	fmt.Printf("[DEBUG] Binary uploaded and verified successfully\n")
	return nil
}

// RemoveShortcut removes a Steam shortcut from a remote device
func RemoveShortcut(cfg *RemoteConfig, name string) error {
	// Create and connect remote client
	client := remote.NewClient(&remote.Config{
		Host:     cfg.Host,
		Port:     cfg.Port,
		User:     cfg.User,
		Password: cfg.Password,
		KeyFile:  cfg.KeyFile,
	})

	if err := client.Connect(); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer client.Close()

	// Set remote clients for library packages
	shortcut.SetRemoteClient(client)
	steam.SetRemoteClient(client)

	// Get all Steam users
	users, err := steam.GetRemoteUsers()
	if err != nil {
		return fmt.Errorf("failed to get Steam users: %w", err)
	}

	// Remove shortcut for all users
	for _, user := range users {
		if !steam.RemoteHasShortcuts(user) {
			continue
		}

		shortcutsPath, err := steam.GetRemoteShortcutsPath(user)
		if err != nil {
			continue
		}

		shortcuts, err := shortcut.Load(shortcutsPath)
		if err != nil {
			continue
		}

		// Filter out the shortcut with the given name
		newShortcuts := shortcut.NewShortcuts()
		for _, sc := range shortcuts.Shortcuts {
			if sc.AppName == name {
				continue // Skip the one we're removing
			}
			newShortcuts.Add(&sc)
		}

		// Save the updated shortcuts
		if err := shortcut.Save(newShortcuts, shortcutsPath); err != nil {
			return fmt.Errorf("failed to save shortcuts for user %s: %w", user, err)
		}
	}

	return nil
}

// ListShortcuts returns all Steam shortcuts from a remote device
func ListShortcuts(cfg *RemoteConfig) ([]ShortcutInfo, error) {
	// Create and connect remote client
	client := remote.NewClient(&remote.Config{
		Host:     cfg.Host,
		Port:     cfg.Port,
		User:     cfg.User,
		Password: cfg.Password,
		KeyFile:  cfg.KeyFile,
	})

	if err := client.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer client.Close()

	// Set remote clients for library packages
	shortcut.SetRemoteClient(client)
	steam.SetRemoteClient(client)

	// Get all Steam users
	users, err := steam.GetRemoteUsers()
	if err != nil {
		return nil, fmt.Errorf("failed to get Steam users: %w", err)
	}

	var result []ShortcutInfo

	// Get shortcuts from all users
	for _, user := range users {
		if !steam.RemoteHasShortcuts(user) {
			continue
		}

		shortcutsPath, err := steam.GetRemoteShortcutsPath(user)
		if err != nil {
			continue
		}

		shortcuts, err := shortcut.Load(shortcutsPath)
		if err != nil {
			continue
		}

		for _, sc := range shortcuts.Shortcuts {
			result = append(result, ShortcutInfo{
				Name:          sc.AppName,
				Exe:           sc.Exe,
				StartDir:      sc.StartDir,
				LaunchOptions: sc.LaunchOptions,
				AppID:         sc.Appid,
			})
		}
	}

	return result, nil
}

// ShortcutInfo represents basic shortcut information
type ShortcutInfo struct {
	Name          string
	Exe           string
	StartDir      string
	LaunchOptions string
	AppID         int64
}

// ParseTags parses a comma-separated tag string into a slice
func ParseTags(tagsStr string) []string {
	if tagsStr == "" {
		return nil
	}
	tags := strings.Split(tagsStr, ",")
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			result = append(result, tag)
		}
	}
	return result
}

// RefreshSteamLibrary performs a soft restart of Steam to reload shortcuts
// In Gaming Mode (Big Picture), Steam will automatically relaunch
func RefreshSteamLibrary(cfg *RemoteConfig) error {
	// Create and connect remote client
	client := remote.NewClient(&remote.Config{
		Host:     cfg.Host,
		Port:     cfg.Port,
		User:     cfg.User,
		Password: cfg.Password,
		KeyFile:  cfg.KeyFile,
	})

	if err := client.Connect(); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer client.Close()

	// Soft restart Steam - in Gaming Mode it will automatically relaunch
	// We use steam -shutdown which gracefully closes Steam
	// On Bazzite/SteamOS Gaming Mode, the session manager will restart Steam automatically
	client.RunCommand(`steam -shutdown >/dev/null 2>&1 || true`)

	return nil
}
