package protocol

import "encoding/json"

// MessageType identifies the type of WebSocket message.
type MessageType string

const (
	// Requests from Hub to Agent
	MsgTypePing            MessageType = "ping"
	MsgTypeGetInfo         MessageType = "get_info"
	MsgTypeInitUpload      MessageType = "init_upload"
	MsgTypeUploadChunk     MessageType = "upload_chunk"
	MsgTypeCompleteUpload  MessageType = "complete_upload"
	MsgTypeCancelUpload    MessageType = "cancel_upload"
	MsgTypeCreateShortcut  MessageType = "create_shortcut"
	MsgTypeDeleteShortcut  MessageType = "delete_shortcut"
	MsgTypeListShortcuts   MessageType = "list_shortcuts"
	MsgTypeRestartSteam    MessageType = "restart_steam"
	MsgTypeGetSteamStatus  MessageType = "get_steam_status"

	// Responses from Agent to Hub
	MsgTypePong           MessageType = "pong"
	MsgTypeInfoResponse   MessageType = "info_response"
	MsgTypeUploadResponse MessageType = "upload_response"
	MsgTypeShortcutResponse MessageType = "shortcut_response"
	MsgTypeSteamResponse  MessageType = "steam_response"
	MsgTypeError          MessageType = "error"

	// Events from Agent to Hub
	MsgTypeUploadProgress MessageType = "upload_progress"
)

// Message is the envelope for all WebSocket communication.
type Message struct {
	ID      string          `json:"id"`
	Type    MessageType     `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// NewMessage creates a new message with the given type and payload.
func NewMessage(id string, msgType MessageType, payload any) (*Message, error) {
	var raw json.RawMessage
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		raw = data
	}
	return &Message{ID: id, Type: msgType, Payload: raw}, nil
}

// ParsePayload unmarshals the payload into the given type.
func (m *Message) ParsePayload(v any) error {
	if m.Payload == nil {
		return nil
	}
	return json.Unmarshal(m.Payload, v)
}

// Request payloads

// InitUploadRequest starts a new upload session.
type InitUploadRequest struct {
	Config     UploadConfig `json:"config"`
	TotalSize  int64        `json:"totalSize"`
	FileCount  int          `json:"fileCount"`
	ResumeFrom int64        `json:"resumeFrom,omitempty"`
}

// UploadChunkRequest sends a chunk of data.
type UploadChunkRequest struct {
	UploadID string `json:"uploadId"`
	Offset   int64  `json:"offset"`
	Data     []byte `json:"data"`
	FilePath string `json:"filePath"`
	IsLast   bool   `json:"isLast"`
}

// CompleteUploadRequest finalizes an upload.
type CompleteUploadRequest struct {
	UploadID       string `json:"uploadId"`
	CreateShortcut bool   `json:"createShortcut"`
}

// CancelUploadRequest cancels an active upload.
type CancelUploadRequest struct {
	UploadID string `json:"uploadId"`
}

// CreateShortcutRequest creates a Steam shortcut.
type CreateShortcutRequest struct {
	UserID   uint32         `json:"userId"`
	Shortcut ShortcutConfig `json:"shortcut"`
}

// DeleteShortcutRequest removes a Steam shortcut.
type DeleteShortcutRequest struct {
	UserID  uint32 `json:"userId"`
	AppID   uint32 `json:"appId,omitempty"`
	Name    string `json:"name,omitempty"`
}

// ListShortcutsRequest lists shortcuts for a user.
type ListShortcutsRequest struct {
	UserID uint32 `json:"userId"`
}

// Response payloads

// InfoResponse contains agent information.
type InfoResponse struct {
	Agent AgentInfo `json:"agent"`
}

// InitUploadResponse acknowledges upload initialization.
type InitUploadResponse struct {
	UploadID   string `json:"uploadId"`
	ResumeFrom int64  `json:"resumeFrom"`
}

// UploadChunkResponse acknowledges a chunk.
type UploadChunkResponse struct {
	UploadID    string `json:"uploadId"`
	BytesWritten int64 `json:"bytesWritten"`
	TotalWritten int64 `json:"totalWritten"`
}

// CompleteUploadResponse confirms upload completion.
type CompleteUploadResponse struct {
	UploadID string `json:"uploadId"`
	Success  bool   `json:"success"`
}

// ShortcutResponse contains shortcut operation result.
type ShortcutResponse struct {
	Success   bool           `json:"success"`
	Shortcuts []ShortcutInfo `json:"shortcuts,omitempty"`
}

// SteamStatusResponse contains Steam status.
type SteamStatusResponse struct {
	Running bool   `json:"running"`
	Path    string `json:"path,omitempty"`
}

// ErrorResponse contains error details.
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}
