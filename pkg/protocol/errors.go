package protocol

import (
	"errors"
	"fmt"
)

// Error codes for protocol errors.
const (
	ErrCodeUnknown          = "UNKNOWN"
	ErrCodeInvalidRequest   = "INVALID_REQUEST"
	ErrCodeUploadNotFound   = "UPLOAD_NOT_FOUND"
	ErrCodeUploadFailed     = "UPLOAD_FAILED"
	ErrCodeShortcutNotFound = "SHORTCUT_NOT_FOUND"
	ErrCodeShortcutExists   = "SHORTCUT_EXISTS"
	ErrCodeSteamNotRunning  = "STEAM_NOT_RUNNING"
	ErrCodeSteamNotFound    = "STEAM_NOT_FOUND"
	ErrCodePermissionDenied = "PERMISSION_DENIED"
	ErrCodeDiskFull         = "DISK_FULL"
	ErrCodeTimeout          = "TIMEOUT"
	ErrCodeAgentBusy        = "AGENT_BUSY"
)

// Sentinel errors for common protocol errors.
var (
	ErrUploadNotFound   = errors.New("upload not found")
	ErrUploadFailed     = errors.New("upload failed")
	ErrShortcutNotFound = errors.New("shortcut not found")
	ErrShortcutExists   = errors.New("shortcut already exists")
	ErrSteamNotRunning  = errors.New("steam is not running")
	ErrSteamNotFound    = errors.New("steam installation not found")
	ErrPermissionDenied = errors.New("permission denied")
	ErrDiskFull         = errors.New("disk full")
	ErrTimeout          = errors.New("operation timed out")
	ErrAgentBusy        = errors.New("agent is busy")
	ErrInvalidRequest   = errors.New("invalid request")
)

// ProtocolError wraps an error with a code for transmission.
type ProtocolError struct {
	Code    string
	Message string
	Err     error
}

func (e *ProtocolError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *ProtocolError) Unwrap() error {
	return e.Err
}

// NewProtocolError creates a new protocol error.
func NewProtocolError(code, message string, err error) *ProtocolError {
	return &ProtocolError{Code: code, Message: message, Err: err}
}

// ToErrorResponse converts a ProtocolError to an ErrorResponse.
func (e *ProtocolError) ToErrorResponse() ErrorResponse {
	details := ""
	if e.Err != nil {
		details = e.Err.Error()
	}
	return ErrorResponse{
		Code:    e.Code,
		Message: e.Message,
		Details: details,
	}
}

// ErrorFromCode creates a ProtocolError from an error code.
func ErrorFromCode(code string, err error) *ProtocolError {
	msg := "unknown error"
	switch code {
	case ErrCodeInvalidRequest:
		msg = "invalid request"
	case ErrCodeUploadNotFound:
		msg = "upload not found"
	case ErrCodeUploadFailed:
		msg = "upload failed"
	case ErrCodeShortcutNotFound:
		msg = "shortcut not found"
	case ErrCodeShortcutExists:
		msg = "shortcut already exists"
	case ErrCodeSteamNotRunning:
		msg = "steam is not running"
	case ErrCodeSteamNotFound:
		msg = "steam installation not found"
	case ErrCodePermissionDenied:
		msg = "permission denied"
	case ErrCodeDiskFull:
		msg = "insufficient disk space"
	case ErrCodeTimeout:
		msg = "operation timed out"
	case ErrCodeAgentBusy:
		msg = "agent is busy with another operation"
	}
	return NewProtocolError(code, msg, err)
}
