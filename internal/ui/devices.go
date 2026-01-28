package ui

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/lobinuxsoft/bazzite-devkit/internal/device"
)

// Device represents a remote device
type Device struct {
	Name      string
	Host      string
	Port      int
	User      string
	KeyFile   string
	Password  string
	Connected bool
	Client    *device.Client
}

// NetworkDevice represents a device found on the network
type NetworkDevice struct {
	IP       string
	Hostname string
	HasSSH   bool
}

var deviceList *widget.List
var devices []*Device

func init() {
	devices = make([]*Device, 0)
}

// createDevicesTab creates the devices management tab
func createDevicesTab() fyne.CanvasObject {
	// Device list
	deviceList = widget.NewList(
		func() int { return len(devices) },
		func() fyne.CanvasObject {
			status := widget.NewLabel("Status")
			status.Alignment = fyne.TextAlignTrailing
			return container.NewBorder(
				nil, nil,
				widget.NewIcon(theme.ComputerIcon()),
				status,
				widget.NewLabel("Device Name"),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(devices) {
				return
			}
			dev := devices[id]
			box := obj.(*fyne.Container)
			nameLabel := box.Objects[0].(*widget.Label)
			statusLabel := box.Objects[2].(*widget.Label)

			nameLabel.SetText(fmt.Sprintf("%s  (%s@%s)", dev.Name, dev.User, dev.Host))
			if dev.Connected {
				statusLabel.SetText("Connected")
			} else {
				statusLabel.SetText("Disconnected")
			}
		},
	)

	deviceList.OnSelected = func(id widget.ListItemID) {
		if id < len(devices) {
			State.SelectedDevice = devices[id]
		}
	}

	// Buttons
	scanBtn := widget.NewButtonWithIcon("Scan Network", theme.SearchIcon(), func() {
		showScanNetworkWindow()
	})

	addBtn := widget.NewButtonWithIcon("Add Manual", theme.ContentAddIcon(), func() {
		showAddDeviceWindow()
	})

	connectBtn := widget.NewButtonWithIcon("Connect", theme.LoginIcon(), func() {
		if State.SelectedDevice != nil {
			go connectToDevice(State.SelectedDevice)
		}
	})

	disconnectBtn := widget.NewButtonWithIcon("Disconnect", theme.LogoutIcon(), func() {
		if State.SelectedDevice != nil && State.SelectedDevice.Connected {
			disconnectDevice(State.SelectedDevice)
		}
	})

	removeBtn := widget.NewButtonWithIcon("Remove", theme.DeleteIcon(), func() {
		if State.SelectedDevice != nil {
			removeDevice(State.SelectedDevice)
		}
	})

	buttons := container.NewHBox(scanBtn, addBtn, connectBtn, disconnectBtn, removeBtn)

	return container.NewBorder(
		buttons,
		nil, nil, nil,
		deviceList,
	)
}

// showScanNetworkWindow shows a window to scan and select network devices
func showScanNetworkWindow() {
	scanWindow := fyne.CurrentApp().NewWindow("Scan Network")
	scanWindow.Resize(fyne.NewSize(500, 400))

	var foundDevices []NetworkDevice
	var networkList *widget.List
	scanningLabel := widget.NewLabel("Click 'Scan' to find devices with SSH...")
	progressBar := widget.NewProgressBarInfinite()
	progressBar.Hide()

	networkList = widget.NewList(
		func() int { return len(foundDevices) },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewIcon(theme.ComputerIcon()),
				widget.NewLabel("IP Address"),
				widget.NewLabel("Hostname"),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(foundDevices) {
				return
			}
			dev := foundDevices[id]
			box := obj.(*fyne.Container)
			ipLabel := box.Objects[1].(*widget.Label)
			hostLabel := box.Objects[2].(*widget.Label)
			ipLabel.SetText(dev.IP)
			if dev.Hostname != "" {
				hostLabel.SetText(fmt.Sprintf("(%s)", dev.Hostname))
			} else {
				hostLabel.SetText("")
			}
		},
	)

	var selectedNetDevice *NetworkDevice
	networkList.OnSelected = func(id widget.ListItemID) {
		if id < len(foundDevices) {
			selectedNetDevice = &foundDevices[id]
		}
	}

	scanBtn := widget.NewButtonWithIcon("Scan", theme.SearchIcon(), func() {
		progressBar.Show()
		scanningLabel.SetText("Scanning network for SSH devices...")
		foundDevices = []NetworkDevice{}
		networkList.Refresh()

		go func() {
			found := scanNetworkForSSH()
			foundDevices = found
			progressBar.Hide()
			scanningLabel.SetText(fmt.Sprintf("Found %d devices with SSH", len(found)))
			networkList.Refresh()
		}()
	})

	selectBtn := widget.NewButtonWithIcon("Select & Configure", theme.ConfirmIcon(), func() {
		if selectedNetDevice != nil {
			scanWindow.Close()
			showAddDeviceWindowWithIP(selectedNetDevice.IP, selectedNetDevice.Hostname)
		}
	})

	topBar := container.NewVBox(
		container.NewHBox(scanBtn, selectBtn),
		scanningLabel,
		progressBar,
	)

	scanWindow.SetContent(container.NewBorder(
		topBar,
		nil, nil, nil,
		networkList,
	))

	scanWindow.Show()
}

// showAddDeviceWindow shows a separate window to add a device
func showAddDeviceWindow() {
	showAddDeviceWindowWithIP("", "")
}

// showAddDeviceWindowWithIP shows the add device window with pre-filled IP
func showAddDeviceWindowWithIP(ip, hostname string) {
	addWindow := fyne.CurrentApp().NewWindow("Add Device")
	addWindow.Resize(fyne.NewSize(500, 450))

	nameEntry := widget.NewEntry()
	if hostname != "" {
		nameEntry.SetText(hostname)
	} else {
		nameEntry.SetPlaceHolder("My Bazzite Device")
	}

	hostEntry := widget.NewEntry()
	if ip != "" {
		hostEntry.SetText(ip)
	} else {
		hostEntry.SetPlaceHolder("192.168.1.100")
	}

	portEntry := widget.NewEntry()
	portEntry.SetText("22")

	userEntry := widget.NewEntry()
	userEntry.SetText("deck")

	// Authentication method selection
	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("Enter your password")

	keyFileEntry := widget.NewEntry()
	keyFileEntry.SetPlaceHolder("~/.ssh/id_ed25519")

	// Detect existing SSH keys
	existingKeys := findExistingSSHKeys()
	keySelect := widget.NewSelect(existingKeys, func(selected string) {
		keyFileEntry.SetText(selected)
	})
	if len(existingKeys) > 0 {
		keySelect.SetSelected(existingKeys[0])
		keyFileEntry.SetText(existingKeys[0])
	}

	// Auth method containers
	passwordContainer := container.NewVBox(
		widget.NewLabel("Password:"),
		passwordEntry,
	)

	keyContainer := container.NewVBox(
		widget.NewLabel("Select SSH Key:"),
		keySelect,
		widget.NewLabel("Or enter path manually:"),
		keyFileEntry,
	)
	keyContainer.Hide()

	// Auth type selector
	authType := widget.NewRadioGroup([]string{"Password", "SSH Key"}, func(selected string) {
		if selected == "Password" {
			passwordContainer.Show()
			keyContainer.Hide()
		} else {
			passwordContainer.Hide()
			keyContainer.Show()
		}
	})
	authType.SetSelected("Password")

	// Basic info form
	basicForm := widget.NewForm(
		widget.NewFormItem("Name", nameEntry),
		widget.NewFormItem("Host/IP", hostEntry),
		widget.NewFormItem("Port", portEntry),
		widget.NewFormItem("User", userEntry),
	)

	saveBtn := widget.NewButtonWithIcon("Add Device", theme.ConfirmIcon(), func() {
		port := 22
		fmt.Sscanf(portEntry.Text, "%d", &port)

		name := nameEntry.Text
		if name == "" {
			name = hostEntry.Text
		}

		var password, keyFile string
		if authType.Selected == "Password" {
			password = passwordEntry.Text
		} else {
			keyFile = keyFileEntry.Text
		}

		dev := &Device{
			Name:     name,
			Host:     hostEntry.Text,
			Port:     port,
			User:     userEntry.Text,
			KeyFile:  keyFile,
			Password: password,
		}
		devices = append(devices, dev)
		State.Devices = devices
		deviceList.Refresh()
		addWindow.Close()
	})

	cancelBtn := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
		addWindow.Close()
	})

	buttons := container.NewHBox(cancelBtn, saveBtn)

	content := container.NewVBox(
		widget.NewLabelWithStyle("Configure SSH Connection", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		basicForm,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Authentication Method", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		authType,
		passwordContainer,
		keyContainer,
		widget.NewSeparator(),
		container.NewCenter(buttons),
	)

	addWindow.SetContent(container.NewPadded(content))
	addWindow.Show()
}

// findExistingSSHKeys looks for SSH keys in ~/.ssh/
func findExistingSSHKeys() []string {
	var keys []string
	home, err := os.UserHomeDir()
	if err != nil {
		return keys
	}

	sshDir := filepath.Join(home, ".ssh")
	keyNames := []string{"id_ed25519", "id_rsa", "id_ecdsa", "id_dsa"}

	for _, name := range keyNames {
		keyPath := filepath.Join(sshDir, name)
		if _, err := os.Stat(keyPath); err == nil {
			keys = append(keys, keyPath)
		}
	}

	return keys
}

// scanNetworkForSSH scans the local network for devices with SSH (port 22) open
func scanNetworkForSSH() []NetworkDevice {
	var found []NetworkDevice
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Get local IP to determine network range
	localIP := getLocalIP()
	if localIP == "" {
		return found
	}

	// Parse network range (assume /24)
	parts := strings.Split(localIP, ".")
	if len(parts) != 4 {
		return found
	}
	baseIP := strings.Join(parts[:3], ".")

	// Scan all IPs in range concurrently
	semaphore := make(chan struct{}, 50) // Limit concurrent connections

	for i := 1; i <= 254; i++ {
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if hasSSH(ip) {
				hostname := getHostname(ip)
				mu.Lock()
				found = append(found, NetworkDevice{
					IP:       ip,
					Hostname: hostname,
					HasSSH:   true,
				})
				mu.Unlock()
			}
		}(fmt.Sprintf("%s.%d", baseIP, i))
	}

	wg.Wait()
	return found
}

// getLocalIP returns the local IP address
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

// hasSSH checks if a host has SSH (port 22) open
func hasSSH(ip string) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:22", ip), 500*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// getHostname tries to resolve the hostname for an IP
func getHostname(ip string) string {
	names, err := net.LookupAddr(ip)
	if err != nil || len(names) == 0 {
		return ""
	}
	// Clean up the hostname (remove trailing dot)
	hostname := strings.TrimSuffix(names[0], ".")
	return hostname
}

// connectToDevice connects to the selected device
func connectToDevice(dev *Device) {
	client, err := device.NewClient(dev.Host, dev.Port, dev.User, dev.Password, dev.KeyFile)
	if err != nil {
		dialog.ShowError(err, State.Window)
		return
	}

	if err := client.Connect(); err != nil {
		dialog.ShowError(err, State.Window)
		return
	}

	dev.Client = client
	dev.Connected = true
	deviceList.Refresh()

	dialog.ShowInformation("Connected", fmt.Sprintf("Connected to %s", dev.Name), State.Window)
}

// disconnectDevice disconnects from the device
func disconnectDevice(dev *Device) {
	if dev.Client != nil {
		dev.Client.Close()
		dev.Client = nil
	}
	dev.Connected = false
	deviceList.Refresh()
}

// removeDevice removes a device from the list
func removeDevice(dev *Device) {
	disconnectDevice(dev)
	for i, d := range devices {
		if d == dev {
			devices = append(devices[:i], devices[i+1:]...)
			break
		}
	}
	State.Devices = devices
	State.SelectedDevice = nil
	deviceList.Refresh()
}
