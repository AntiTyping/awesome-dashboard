# Awesome Dashboard — Design Spec

## Overview

A single Go binary that serves a real-time web dashboard of colored tiles and provides a CLI for managing those tiles. Items are persisted in SQLite. The dashboard updates automatically via Server-Sent Events when the CLI mutates data.

## Goals

- Single self-contained binary for both the web server and CLI
- Live dashboard updates without manual page refresh
- Simple local-first tool with no authentication or external dependencies
- Minimal frontend with no JS framework or build step

## Architecture

### Single Binary, Subcommand CLI

The `awesome-dashboard` binary uses Cobra for subcommand routing:

| Command | Purpose |
|---------|---------|
| `serve` | Start the web server (default port 8080) |
| `add --title "X" --subtitle "Y" --color "#FF5733"` | Add a new item |
| `update <id> [--title] [--subtitle] [--color]` | Update an existing item |
| `remove <id>` | Remove an item |
| `list` | List all items |

### Components

- **CLI layer** — Cobra subcommands parse flags and call the store
- **Storage layer** — SQLite via `modernc.org/sqlite` (pure Go, no CGO)
- **Web server** — Go `net/http` with embedded static files (`embed`)
- **SSE broker** — In-process pub/sub that pushes refresh events to connected browsers
- **Notification bridge** — After CLI mutations, an HTTP POST to `localhost:<port>/api/notify` tells the running server to broadcast

### Update Flow

1. CLI writes to SQLite (add/update/remove)
2. CLI POSTs to `http://localhost:<port>/api/notify` (fire-and-forget, ignores errors)
3. Server broadcasts `data: refresh` to all SSE clients
4. Browser receives the event, fetches `/api/items`, re-renders tiles

The notification is best-effort. If the server isn't running, the CLI write still succeeds.

## Data Model

Single `items` table in SQLite:

| Column | Type | Notes |
|--------|------|-------|
| `id` | TEXT (UUID) | Primary key, generated on `add` |
| `title` | TEXT NOT NULL | Tile heading |
| `subtitle` | TEXT NOT NULL | Tile secondary text |
| `color` | TEXT NOT NULL | Hex color, e.g. `#FF5733` |
| `created_at` | DATETIME | Auto-set on insert |
| `updated_at` | DATETIME | Auto-set on insert and update |

Database location: `~/.awesome-dashboard/data.db` by default, overridable with `--db` flag on all commands. The directory and schema are created automatically on first use.

IDs are UUIDs. The CLI displays an 8-character prefix and accepts any unique prefix for `update` and `remove` commands.

## API

The `serve` command starts an HTTP server with these endpoints:

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/` | Serves the embedded HTML dashboard |
| GET | `/api/items` | Returns all items as JSON array (ordered by `created_at`) |
| GET | `/api/events` | SSE stream — pushes `refresh` events |
| POST | `/api/notify` | Internal — CLI calls this to trigger SSE broadcast |

### JSON Response Shape (`/api/items`)

```json
[
  {
    "id": "a1b2c3d4-...",
    "title": "My Project",
    "subtitle": "In progress",
    "color": "#FF5733",
    "created_at": "2026-10-26T12:00:00Z",
    "updated_at": "2026-10-26T12:00:00Z"
  }
]
```

### SSE Protocol (`/api/events`)

The server sends a simple `data: refresh\n\n` message whenever `/api/notify` is called. The browser handles reconnection natively via the `EventSource` API. No event IDs or typed events are needed.

## Frontend

A single `index.html` embedded in the binary. No external JS or CSS dependencies.

### Layout

- Responsive CSS grid: 1 column on mobile, 2 on tablet, 3-4 on desktop
- Minimal header with the app name

### Tile Card

- Left border or top accent bar colored with the item's `color`
- Title in bold
- Subtitle in lighter/smaller text
- Subtle shadow and rounded corners

### Behavior

- On load: fetch `/api/items` and render tiles
- Connect to `/api/events` (SSE) — on each `refresh` event, re-fetch and re-render
- Auto-reconnect on SSE disconnect (native `EventSource` behavior)
- Empty state: centered message "No items yet. Use the CLI to add one."

### Styling

- Light/clean design, system font stack
- Dark mode via `prefers-color-scheme` media query
- All CSS inline in the HTML file

## Project Structure

```
awesome-dashboard/
├── cmd/
│   ├── root.go          # Cobra root command, --db flag
│   ├── serve.go         # serve subcommand
│   ├── add.go           # add subcommand
│   ├── update.go        # update subcommand
│   ├── remove.go        # remove subcommand
│   └── list.go          # list subcommand
├── internal/
│   ├── store/
│   │   └── store.go     # SQLite CRUD, schema migration
│   ├── server/
│   │   ├── server.go    # HTTP handlers, SSE broker
│   │   └── server_test.go
│   └── notify/
│       └── notify.go    # POST to /api/notify (used by CLI)
├── web/
│   └── index.html       # Embedded frontend
├── main.go              # Entry point
├── go.mod
└── go.sum
```

## Dependencies

- `github.com/spf13/cobra` — CLI framework
- `modernc.org/sqlite` — Pure-Go SQLite driver
- `github.com/google/uuid` — ID generation
- Standard library for everything else (`net/http`, `embed`, `encoding/json`, `database/sql`)

## Testing Strategy

- **Store layer:** Unit tests against in-memory SQLite — CRUD operations, schema auto-creation, ID prefix matching
- **Server:** `httptest` for API endpoints — `/api/items` returns correct JSON, `/api/notify` triggers SSE broadcast
- **CLI integration:** Table-driven tests running add/update/remove/list against a temp database, asserting stdout output
- **No frontend tests** — single static page with minimal JS

## Configuration

| Flag | Default | Scope | Description |
|------|---------|-------|-------------|
| `--db` | `~/.awesome-dashboard/data.db` | All commands | Path to SQLite database |
| `--port` | `8080` | All commands | HTTP server port (used by `serve` to listen, by mutating commands to notify) |
