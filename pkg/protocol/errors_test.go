package protocol

import (
	"errors"
	"testing"
)

func TestProtocolError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *ProtocolError
		contains string
	}{
		{
			name: "error without wrapped error",
			err: &ProtocolError{
				Code:    ErrCodeUploadFailed,
				Message: "upload failed",
			},
			contains: "UPLOAD_FAILED: upload failed",
		},
		{
			name: "error with wrapped error",
			err: &ProtocolError{
				Code:    ErrCodeDiskFull,
				Message: "no space left",
				Err:     errors.New("underlying error"),
			},
			contains: "underlying error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got == "" {
				t.Error("Error() should not return empty string")
			}
			if !containsString(got, tt.contains) {
				t.Errorf("Error() = %q, should contain %q", got, tt.contains)
			}
		})
	}
}

func TestProtocolError_Unwrap(t *testing.T) {
	underlying := errors.New("underlying error")
	err := &ProtocolError{
		Code:    ErrCodeUploadFailed,
		Message: "upload failed",
		Err:     underlying,
	}

	unwrapped := err.Unwrap()
	if unwrapped != underlying {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, underlying)
	}
}

func TestProtocolError_Unwrap_Nil(t *testing.T) {
	err := &ProtocolError{
		Code:    ErrCodeUploadFailed,
		Message: "upload failed",
	}

	if err.Unwrap() != nil {
		t.Error("Unwrap() should return nil when no wrapped error")
	}
}

func TestNewProtocolError(t *testing.T) {
	underlying := errors.New("test error")
	err := NewProtocolError(ErrCodeTimeout, "operation timed out", underlying)

	if err.Code != ErrCodeTimeout {
		t.Errorf("Code = %q, want %q", err.Code, ErrCodeTimeout)
	}
	if err.Message != "operation timed out" {
		t.Errorf("Message = %q, want %q", err.Message, "operation timed out")
	}
	if err.Err != underlying {
		t.Error("Err should be the underlying error")
	}
}

func TestProtocolError_ToErrorResponse(t *testing.T) {
	underlying := errors.New("disk full")
	err := &ProtocolError{
		Code:    ErrCodeDiskFull,
		Message: "insufficient disk space",
		Err:     underlying,
	}

	resp := err.ToErrorResponse()

	if resp.Code != ErrCodeDiskFull {
		t.Errorf("Code = %q, want %q", resp.Code, ErrCodeDiskFull)
	}
	if resp.Message != "insufficient disk space" {
		t.Errorf("Message = %q, want %q", resp.Message, "insufficient disk space")
	}
	if resp.Details != "disk full" {
		t.Errorf("Details = %q, want %q", resp.Details, "disk full")
	}
}

func TestProtocolError_ToErrorResponse_NoWrappedError(t *testing.T) {
	err := &ProtocolError{
		Code:    ErrCodeAgentBusy,
		Message: "agent is busy",
	}

	resp := err.ToErrorResponse()

	if resp.Details != "" {
		t.Errorf("Details should be empty, got %q", resp.Details)
	}
}

func TestErrorFromCode(t *testing.T) {
	tests := []struct {
		code    string
		wantMsg string
	}{
		{ErrCodeInvalidRequest, "invalid request"},
		{ErrCodeUploadNotFound, "upload not found"},
		{ErrCodeUploadFailed, "upload failed"},
		{ErrCodeShortcutNotFound, "shortcut not found"},
		{ErrCodeShortcutExists, "shortcut already exists"},
		{ErrCodeSteamNotRunning, "steam is not running"},
		{ErrCodeSteamNotFound, "steam installation not found"},
		{ErrCodePermissionDenied, "permission denied"},
		{ErrCodeDiskFull, "insufficient disk space"},
		{ErrCodeTimeout, "operation timed out"},
		{ErrCodeAgentBusy, "agent is busy with another operation"},
		{ErrCodeUnknown, "unknown error"},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			err := ErrorFromCode(tt.code, nil)
			if err.Code != tt.code {
				t.Errorf("Code = %q, want %q", err.Code, tt.code)
			}
			if err.Message != tt.wantMsg {
				t.Errorf("Message = %q, want %q", err.Message, tt.wantMsg)
			}
		})
	}
}

func TestErrorFromCode_WithWrappedError(t *testing.T) {
	underlying := errors.New("test")
	err := ErrorFromCode(ErrCodeUploadFailed, underlying)

	if err.Err != underlying {
		t.Error("ErrorFromCode should preserve wrapped error")
	}
}

func TestSentinelErrors(t *testing.T) {
	// Verify sentinel errors are not nil
	sentinels := []error{
		ErrUploadNotFound,
		ErrUploadFailed,
		ErrShortcutNotFound,
		ErrShortcutExists,
		ErrSteamNotRunning,
		ErrSteamNotFound,
		ErrPermissionDenied,
		ErrDiskFull,
		ErrTimeout,
		ErrAgentBusy,
		ErrInvalidRequest,
	}

	for _, err := range sentinels {
		if err == nil {
			t.Error("Sentinel error should not be nil")
		}
		if err.Error() == "" {
			t.Error("Sentinel error message should not be empty")
		}
	}
}

func TestErrorCodes_Constants(t *testing.T) {
	codes := []string{
		ErrCodeUnknown,
		ErrCodeInvalidRequest,
		ErrCodeUploadNotFound,
		ErrCodeUploadFailed,
		ErrCodeShortcutNotFound,
		ErrCodeShortcutExists,
		ErrCodeSteamNotRunning,
		ErrCodeSteamNotFound,
		ErrCodePermissionDenied,
		ErrCodeDiskFull,
		ErrCodeTimeout,
		ErrCodeAgentBusy,
	}

	seen := make(map[string]bool)
	for _, code := range codes {
		if code == "" {
			t.Error("Error code should not be empty")
		}
		if seen[code] {
			t.Errorf("Duplicate error code: %q", code)
		}
		seen[code] = true
	}
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
