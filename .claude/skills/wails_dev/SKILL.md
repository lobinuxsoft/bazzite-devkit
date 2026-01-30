---
name: wails_dev
description: Wails framework development (Go + Web).
---
# Wails v2 (CapyDeploy Hub)

- **Docs**: https://wails.io/docs/introduction
- **Architecture**: Go backend + Web frontend in single binary
- **Template**: Svelte 5 + Tailwind + shadcn-svelte

## Project Structure
```
apps/hub/                   # Wails desktop app
├── app.go                  # Main App struct with Wails bindings
├── main.go                 # Wails app initialization
├── wails.json              # Wails config
├── frontend/               # Svelte app
│   ├── src/
│   │   ├── lib/
│   │   │   ├── components/ # UI components
│   │   │   ├── stores/     # Svelte stores
│   │   │   └── types.ts    # TypeScript types
│   │   └── routes/         # SvelteKit routes
│   ├── wailsjs/            # Auto-generated bindings
│   └── package.json
└── build/                  # Build output
```

## CLI Commands
```bash
cd apps/hub
wails dev              # Dev mode with hot reload
wails build            # Production build (current platform)
wails generate module  # Regenerate frontend bindings after Go changes
```

**IMPORTANT**: Cross-compile NOT supported. Build on target OS.

## Go -> Frontend Communication

### Bindings (Frontend calls Go)
```go
// app.go - Public methods are auto-exposed
func (a *App) DiscoverAgents() ([]protocol.AgentInfo, error) { ... }
```
```typescript
// frontend - Auto-generated in wailsjs/go/main/App
import { DiscoverAgents } from '$lib/wailsjs/go/main/App';
const agents = await DiscoverAgents();
```

### Events (Go pushes to Frontend)
```go
// Go side
runtime.EventsEmit(a.ctx, "agent:discovered", agentInfo)
runtime.EventsEmit(a.ctx, "transfer:progress", progress)
```
```typescript
// Frontend side
import { EventsOn } from '$lib/wailsjs/runtime/runtime';
EventsOn("agent:discovered", (data) => { ... });
EventsOn("transfer:progress", (progress) => { ... });
```

## Frontend -> Go Communication
- Call bound functions directly (returns Promise)
- Handle errors with try/catch
- Events for real-time updates (progress, status changes)

## Build Configuration (wails.json)
```json
{
  "name": "CapyDeploy Hub",
  "frontend:install": "bun install",
  "frontend:build": "bun run build",
  "frontend:dev:watcher": "bun run dev",
  "frontend:dev:serverUrl": "auto"
}
```

## Platform Specifics
- **Windows**: Requires WebView2 (auto-installed on Win10+)
- **Linux**: Requires webkit2gtk-4.0
- **Cross-compile**: NOT supported. Build on target OS.
