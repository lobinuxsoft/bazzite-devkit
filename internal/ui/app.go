package ui

import (
	"fmt"
	"image/color"
	"os/exec"
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/lobinuxsoft/bazzite-devkit/internal/config"
)

// AppState holds the global application state
type AppState struct {
	Devices        []*Device
	SelectedDevice *Device
	Window         fyne.Window
}

var State = &AppState{
	Devices: make([]*Device, 0),
}

// Connection status widgets
var (
	connectionStatusLabel *widget.Label
	connectionDot         *canvas.Circle
)

// Setup initializes the main UI
func Setup(w fyne.Window) {
	State.Window = w
	State.Devices = devices // Load saved devices

	// Create connection status indicator (top right)
	connectionDot = canvas.NewCircle(color.RGBA{128, 128, 128, 255}) // Gray when disconnected
	connectionDot.StrokeWidth = 1
	connectionDot.StrokeColor = color.RGBA{40, 40, 40, 255}

	// Use a min size rect to ensure the dot has proper size
	dotSpacer := canvas.NewRectangle(color.Transparent)
	dotSpacer.SetMinSize(fyne.NewSize(14, 14))
	dotContainer := container.NewStack(dotSpacer, connectionDot)

	connectionStatusLabel = widget.NewLabel("Not connected")
	connectionStatusLabel.TextStyle = fyne.TextStyle{Italic: true}

	statusIndicator := container.NewHBox(
		dotContainer,
		connectionStatusLabel,
	)

	// Create tabs for different sections
	tabs := container.NewAppTabs(
		container.NewTabItem("Devices", createDevicesTab()),
		container.NewTabItem("Upload Game", createUploadTab()),
		container.NewTabItem("Installed Games", createGamesTab()),
		container.NewTabItem("Settings", createSettingsTab()),
	)
	tabs.SetTabLocation(container.TabLocationTop)

	// Top bar with tabs on left and status on right
	topBar := container.NewBorder(
		nil, nil,
		nil,
		container.NewHBox(layout.NewSpacer(), statusIndicator),
		nil,
	)

	// Main layout
	mainContent := container.NewBorder(
		topBar,
		nil,
		nil, nil,
		tabs,
	)

	w.SetContent(mainContent)
}

// UpdateConnectionStatus updates the connection status indicator
func UpdateConnectionStatus() {
	if connectionStatusLabel == nil || connectionDot == nil {
		return
	}

	if State.SelectedDevice != nil && State.SelectedDevice.Connected {
		dev := State.SelectedDevice
		connectionStatusLabel.SetText(fmt.Sprintf("%s (%s:%d)", dev.Name, dev.Host, dev.Port))
		connectionDot.FillColor = color.RGBA{0, 200, 0, 255} // Green when connected
		connectionDot.Refresh()
	} else {
		connectionStatusLabel.SetText("Not connected")
		connectionDot.FillColor = color.RGBA{128, 128, 128, 255} // Gray when disconnected
		connectionDot.Refresh()
	}
}

// formatBytes formats bytes into human readable string
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// createSettingsTab creates the settings tab
func createSettingsTab() fyne.CanvasObject {
	// Default paths
	steamPathEntry := widget.NewEntry()
	steamPathEntry.SetPlaceHolder("~/.steam/steam")

	gamePathEntry := widget.NewEntry()
	gamePathEntry.SetPlaceHolder("~/devkit-games")

	// SteamGridDB API key
	apiKeyEntry := widget.NewPasswordEntry()
	apiKeyEntry.SetPlaceHolder("Your SteamGridDB API key")

	// Load existing API key
	if apiKey, err := config.GetSteamGridDBAPIKey(); err == nil && apiKey != "" {
		apiKeyEntry.SetText(apiKey)
	}

	apiKeyHelp := widget.NewRichTextFromMarkdown("Get your API key from [steamgriddb.com/profile/preferences/api](https://www.steamgriddb.com/profile/preferences/api)")

	form := widget.NewForm(
		widget.NewFormItem("Steam Path", steamPathEntry),
		widget.NewFormItem("Games Path", gamePathEntry),
	)

	apiKeyForm := widget.NewForm(
		widget.NewFormItem("API Key", apiKeyEntry),
	)

	saveBtn := widget.NewButton("Save Settings", func() {
		// Save API key
		if err := config.SetSteamGridDBAPIKey(apiKeyEntry.Text); err != nil {
			dialog.ShowError(fmt.Errorf("failed to save API key: %w", err), State.Window)
			return
		}
		dialog.ShowInformation("Saved", "Settings saved successfully", State.Window)
	})

	// Cache management
	cacheSizeLabel := widget.NewLabel("Calculating...")
	updateCacheSize := func() {
		size, err := GetCacheSize()
		if err != nil {
			cacheSizeLabel.SetText("Unable to calculate")
		} else {
			cacheSizeLabel.SetText(formatBytes(size))
		}
	}
	go updateCacheSize()

	clearCacheBtn := widget.NewButton("Clear Cache", func() {
		dialog.ShowConfirm("Clear Cache",
			"This will delete all cached SteamGridDB images.\nAre you sure?",
			func(ok bool) {
				if ok {
					if err := ClearImageCache(); err != nil {
						dialog.ShowError(fmt.Errorf("failed to clear cache: %w", err), State.Window)
						return
					}
					dialog.ShowInformation("Cache Cleared", "Image cache has been cleared", State.Window)
					go updateCacheSize()
				}
			}, State.Window)
	})

	openCacheFolderBtn := widget.NewButton("Open Cache Folder", func() {
		cacheDir, err := GetImageCacheDir()
		if err != nil {
			dialog.ShowError(fmt.Errorf("failed to get cache directory: %w", err), State.Window)
			return
		}
		// Open folder in file explorer
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "windows":
			cmd = exec.Command("explorer", cacheDir)
		case "darwin":
			cmd = exec.Command("open", cacheDir)
		default: // linux and others
			cmd = exec.Command("xdg-open", cacheDir)
		}
		if err := cmd.Start(); err != nil {
			dialog.ShowError(fmt.Errorf("failed to open folder: %w", err), State.Window)
		}
	})

	cacheForm := widget.NewForm(
		widget.NewFormItem("Cache Size", cacheSizeLabel),
	)

	cacheButtons := container.NewHBox(clearCacheBtn, openCacheFolderBtn)

	return container.NewVBox(
		widget.NewLabelWithStyle("Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		form,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("SteamGridDB Integration", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("SteamGridDB allows you to select custom artwork for your games."),
		apiKeyHelp,
		apiKeyForm,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Image Cache", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Cached images are stored locally for faster loading."),
		cacheForm,
		cacheButtons,
		widget.NewSeparator(),
		saveBtn,
	)
}
