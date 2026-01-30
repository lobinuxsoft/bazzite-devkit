// Package agent defines the interfaces that an Agent must implement.
package agent

import (
	"github.com/lobinuxsoft/capydeploy/pkg/protocol"
)

// BaseAgent provides basic agent information and health checks.
type BaseAgent interface {
	// GetInfo returns information about the agent.
	GetInfo() protocol.AgentInfo

	// Ping checks if the agent is responsive.
	Ping() error
}

// FileReceiver handles file uploads from the Hub.
type FileReceiver interface {
	// InitUpload initializes a new upload session.
	// Returns an upload ID for tracking and the offset to resume from (0 for new uploads).
	InitUpload(config protocol.UploadConfig, totalSize int64, fileCount int) (uploadID string, resumeFrom int64, err error)

	// UploadChunk receives a chunk of data for an active upload.
	UploadChunk(uploadID string, filePath string, chunk []byte, offset int64) error

	// CompleteUpload finalizes an upload and optionally creates a shortcut.
	CompleteUpload(uploadID string, createShortcut bool) error

	// CancelUpload cancels an active upload and cleans up.
	CancelUpload(uploadID string) error

	// GetUploadProgress returns the current progress of an upload.
	GetUploadProgress(uploadID string) (*protocol.UploadProgress, error)
}

// ShortcutManager handles Steam shortcut operations.
type ShortcutManager interface {
	// CreateShortcut creates a new Steam shortcut.
	CreateShortcut(userID uint32, cfg protocol.ShortcutConfig) error

	// DeleteShortcut removes a Steam shortcut by app ID or name.
	DeleteShortcut(userID uint32, appID uint32, name string) error

	// ListShortcuts returns all shortcuts for a user.
	ListShortcuts(userID uint32) ([]protocol.ShortcutInfo, error)

	// UpdateShortcut updates an existing shortcut.
	UpdateShortcut(userID uint32, appID uint32, cfg protocol.ShortcutConfig) error
}

// SteamController manages Steam process operations.
type SteamController interface {
	// RestartSteam restarts the Steam client.
	RestartSteam() error

	// GetSteamStatus returns whether Steam is running.
	GetSteamStatus() (running bool, err error)

	// GetSteamPath returns the Steam installation path.
	GetSteamPath() (string, error)
}

// ArtworkManager handles Steam artwork operations.
type ArtworkManager interface {
	// SetArtwork sets artwork for a shortcut.
	SetArtwork(userID uint32, appID uint32, artwork protocol.ArtworkConfig) error

	// GetArtwork returns the artwork paths for a shortcut.
	GetArtwork(userID uint32, appID uint32) (*protocol.ArtworkConfig, error)

	// DeleteArtwork removes all artwork for a shortcut.
	DeleteArtwork(userID uint32, appID uint32) error
}

// FullAgent combines all agent capabilities.
type FullAgent interface {
	BaseAgent
	FileReceiver
	ShortcutManager
	SteamController
	ArtworkManager
}
