---
name: software_design
description: SOLID/Patterns for Go applications.
---
# Architecture (CapyDeploy)

- **Comp > Inherit**: Go has no inheritance. Use composition and embedding.
- **Package Design**: Each package = one responsibility. `internal/` for private, `pkg/` for shared.

## Hub-Agent Architecture
```
┌─────────────────┐         ┌─────────────────┐
│   Hub (Desktop) │  mDNS   │ Agent (Handheld)│
│   Wails + UI    │◄───────►│ HTTP/WebSocket  │
│                 │  HTTP   │                 │
│  - Discovery    │────────►│ - File receiver │
│  - File upload  │   WS    │ - Shortcuts     │
│  - Artwork      │◄───────►│ - Steam control │
└─────────────────┘         └─────────────────┘
```

## SOLID in Go

- **SRP**: One package = one purpose. `config/` handles config, `discovery/` handles mDNS.
- **OCP**: Use interfaces to extend behavior without modifying existing code.
- **LSP**: Interface implementations must be substitutable.
- **ISP**: Small, focused interfaces. `io.Reader` not `io.ReadWriteCloser` when only reading.
- **DIP**: Accept interfaces in function params. Inject dependencies, don't create inside.

## Patterns

- **Repository**: Wrap data access (config.Store, device.Repository).
- **Factory**: Functions returning interfaces (`NewClient() Client`).
- **Strategy**: Interface + multiple implementations (platform-specific paths).
- **Observer**: Channels or callback functions for async notifications.
- **Builder**: For complex object construction with many optional params.

## Project Structure (CapyDeploy)
```
apps/
├── hub/                # Wails desktop app
└── agent/              # Daemon for handhelds

pkg/                    # Shared packages (Hub + Agent)
├── protocol/           # Message types, interfaces
├── discovery/          # mDNS client/server
├── steam/              # Steam integration
├── transfer/           # Chunked file upload
├── config/             # App configuration
└── steamgriddb/        # Artwork API

internal/               # Hub-only packages
├── device/             # Agent HTTP/WS client
├── shortcuts/          # Local shortcut management
└── embedded/           # Embedded agent binary
```

## Error Handling
- Define custom error types for domain errors.
- Use `errors.Is/As` for error checking.
- Wrap errors with context at each layer.
- Log at the top level, return errors from lower levels.
