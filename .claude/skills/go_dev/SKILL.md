---
name: go_dev
description: Go 1.23 backend development for Wails apps.
---
# Go 1.23 Backend (CapyDeploy)

- **Ver**: Go 1.23+
- **Files**: `snake_case.go`; Packages: `lowercase`
- **Role**: Backend logic, Wails bindings (Hub), daemon services (Agent)

## Project Structure
```
apps/
├── hub/              # Wails UI app (desktop)
│   ├── app.go        # Wails bindings
│   └── main.go       # Entry point
└── agent/            # Daemon for handhelds
    └── main.go       # HTTP/WebSocket server

pkg/                  # Shared packages (Hub + Agent)
├── protocol/         # Message types, WebSocket messages
├── discovery/        # mDNS client/server
├── steam/            # Steam paths, shortcuts, users
├── transfer/         # Chunked file upload
├── config/           # App configuration
└── steamgriddb/      # Artwork API client

internal/             # Hub-only packages
├── device/           # Agent client (HTTP/WS)
├── shortcuts/        # Local shortcut management
└── embedded/         # Embedded agent binary
```

## Go Idioms
- **Errors**: Return `error` as last value. Wrap with `fmt.Errorf("context: %w", err)`
- **Defer**: Use for cleanup (Close, Unlock). Defer immediately after resource acquisition.
- **Interfaces**: Accept interfaces, return structs. Keep interfaces small.
- **Concurrency**: Channels for communication. `sync.Mutex` for shared state. Context for cancellation.

## Wails Bindings (Hub)
- **Expose functions**: Public methods on `App` struct are auto-bound.
- **Return types**: Use simple types or structs (serialized to JSON).
- **Errors**: Return `error` as second value, frontend receives as rejected promise.
- **Events**: Use `runtime.EventsEmit()` for Go->Frontend notifications.
- **Context**: Store `context.Context` from `startup()` for runtime calls.

```go
// Example binding
func (a *App) GetAgents() ([]protocol.AgentInfo, error) {
    return a.discovery.ListAgents()
}

// Example event emission
runtime.EventsEmit(a.ctx, "transfer:progress", transfer.Progress{Percent: 50})
```

## HTTP/WebSocket (Agent Communication)
- **Discovery**: Use mDNS to find agents on local network.
- **REST API**: Agent exposes HTTP endpoints for file upload, status.
- **WebSocket**: Real-time updates (progress, status changes).
- **Timeouts**: Always set connection/operation timeouts.
- **Errors**: Handle network errors gracefully. Return user-friendly messages.

## Performance
- Use `strings.Builder` for string concatenation.
- Preallocate slices when size is known: `make([]T, 0, capacity)`.
- Profile with `pprof` before optimizing.
- Avoid blocking Wails event loop with long operations (use goroutines).
