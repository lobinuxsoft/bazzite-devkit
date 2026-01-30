---
name: frontend_dev
description: Svelte 5 + Tailwind + shadcn-svelte frontend development.
---
# Frontend Stack (CapyDeploy Hub)

- **Framework**: Svelte 5 (runes mode) + SvelteKit
- **Styling**: Tailwind CSS v4
- **Components**: shadcn-svelte
- **Build**: Vite
- **Language**: TypeScript

## Svelte 5 Runes

```svelte
<script lang="ts">
  // Reactive state
  let count = $state(0);

  // Derived values
  let doubled = $derived(count * 2);

  // Effects
  $effect(() => {
    console.log('Count changed:', count);
  });

  // Props
  let { title, onSubmit }: { title: string; onSubmit: () => void } = $props();
</script>
```

## Component Structure
```
apps/hub/frontend/src/
├── lib/
│   ├── components/
│   │   ├── ui/              # shadcn-svelte components
│   │   ├── DeviceList.svelte
│   │   ├── ArtworkSelector.svelte
│   │   ├── GameSetupList.svelte
│   │   └── Settings.svelte
│   ├── stores/              # Svelte stores for global state
│   │   ├── devices.ts
│   │   ├── games.ts
│   │   └── connection.ts
│   ├── wailsjs.ts           # Wails binding wrappers
│   └── types.ts             # TypeScript interfaces
├── routes/
│   ├── +layout.svelte       # App layout
│   └── +page.svelte         # Main page
└── app.css                  # Global styles
```

## shadcn-svelte Usage
```bash
cd apps/hub/frontend
bunx shadcn-svelte@latest add button card tabs dialog
```

```svelte
<script>
  import { Button } from '$lib/components/ui/Button.svelte';
  import Card from '$lib/components/ui/Card.svelte';
</script>

<Card>
  <h3>Agent</h3>
  <Button onclick={connect}>Connect</Button>
</Card>
```

## Tailwind Best Practices
- Use utility classes directly, avoid `@apply` in most cases
- Group related utilities: `class="flex items-center gap-2"`
- Use CSS variables from shadcn theme: `text-primary`, `bg-card`
- Responsive: `md:flex-row`, `lg:grid-cols-3`

## Wails Integration
```typescript
// lib/wailsjs.ts - Typed wrappers for Wails bindings
import { DiscoverAgents, UploadGame } from './wailsjs/go/main/App';
import { EventsOn, EventsOff } from './wailsjs/runtime/runtime';
import type { AgentInfo, UploadProgress } from './types';

export async function discoverAgents(): Promise<AgentInfo[]> {
  return await DiscoverAgents();
}

export function onTransferProgress(callback: (progress: UploadProgress) => void) {
  return EventsOn('transfer:progress', callback);
}
```

## Image Handling
```svelte
<!-- Native browser support - animated WebP/GIF work! -->
<img
  src={artworkUrl}
  alt={gameName}
  class="rounded-lg object-cover"
  loading="lazy"
/>

<!-- With error fallback -->
<img
  src={artworkUrl}
  alt={gameName}
  onerror={(e) => e.currentTarget.src = '/placeholder.png'}
/>
```

## State Management
```typescript
// stores/devices.ts
import { writable } from 'svelte/store';
import type { AgentInfo } from '$lib/types';

export const agents = writable<AgentInfo[]>([]);
export const connectedAgent = writable<AgentInfo | null>(null);
```
