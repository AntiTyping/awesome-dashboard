# Awesome Dashboard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a single Go binary that serves a real-time web dashboard of colored tiles and provides a CLI for managing those tiles via SQLite + SSE.

**Architecture:** Single binary with Cobra subcommands (serve, add, update, remove, list). SQLite for persistence, SSE for live browser updates. CLI mutation commands write to SQLite then POST to the running server's notify endpoint to trigger a broadcast. Frontend is a single embedded HTML file.

**Tech Stack:** Go, Cobra, modernc.org/sqlite, google/uuid, net/http, embed

**Spec:** `docs/superpowers/specs/2026-10-26-awesome-dashboard-design.md`

## Global Constraints

- Pure Go SQLite driver (`modernc.org/sqlite`) — no CGO
- Module path: `github.com/AntiTyping/awesome-dashboard`
- Default DB path: `~/.awesome-dashboard/data.db`
- Default port: `8080`
- Both `--db` and `--port` are persistent flags on the root command, available to all subcommands
- IDs are full UUIDs stored in the DB; CLI displays 8-char prefixes and accepts any unique prefix
- Notification POST is fire-and-forget — CLI writes succeed even if the server is down
- No external JS/CSS dependencies in the frontend
- Git commits use `--author="Andy Pliszka <andy@isp44.com>"`

## Review Focus

1. **Ambiguous ID prefix on update/remove:** Two items whose UUIDs share the first 8+ characters — update/remove with that prefix should return a clear error, not silently pick one. *Test added to Task 1, step 2.*
2. **Invalid hex color on add/update:** Input like `"red"` or `"not-a-color"` — should reject with an error before writing to SQLite. *Test added to Task 4, step 5.*
3. **Update with no change flags:** `update <id>` with none of --title/--subtitle/--color specified — should error rather than silently no-op. *Test added to Task 4, step 5.*
4. **Concurrent SSE clients:** Multiple browsers connected to `/api/events` should all receive the refresh event when `/api/notify` is called. *Test added to Task 2, step 1.*
5. **SSE client disconnect cleanup:** A browser that disconnects should not cause a goroutine leak or a blocked broadcast. *Test added to Task 2, step 1.*

---

## File Structure

```
awesome-dashboard/
├── main.go                          # Task 1: Entry point
├── go.mod                           # Task 1: Module definition
├── cmd/
│   ├── root.go                      # Task 1: Root command, --db/--port persistent flags
│   ├── serve.go                     # Task 3: serve subcommand
│   ├── add.go                       # Task 4: add subcommand
│   ├── update.go                    # Task 4: update subcommand
│   ├── remove.go                    # Task 4: remove subcommand
│   └── list.go                      # Task 4: list subcommand
├── internal/
│   ├── store/
│   │   ├── store.go                 # Task 1: SQLite CRUD, schema migration
│   │   └── store_test.go            # Task 1: Store tests
│   ├── server/
│   │   ├── server.go                # Task 2: HTTP handlers, SSE broker
│   │   └── server_test.go           # Task 2: Server tests
│   └── notify/
│       ├── notify.go                # Task 4: POST notification helper
│       └── notify_test.go           # Task 4: Notify tests
├── web/
│   ├── web.go                       # Task 3: embed.FS exporting index.html
│   └── index.html                   # Task 3: Dashboard frontend
```

---

### Task 1: Project Scaffolding + Store Layer

**Files:**
- Create: `go.mod`, `main.go`, `cmd/root.go`, `internal/store/store.go`, `internal/store/store_test.go`

**Interfaces:**
- Consumes: nothing (foundation task)
- Produces:
  ```go
  // internal/store/store.go
  type Item struct {
      ID        string    `json:"id"`
      Title     string    `json:"title"`
      Subtitle  string    `json:"subtitle"`
      Color     string    `json:"color"`
      CreatedAt time.Time `json:"created_at"`
      UpdatedAt time.Time `json:"updated_at"`
  }

  type Store struct { /* unexported fields */ }

  func Open(dbPath string) (*Store, error)
  func (s *Store) Add(title, subtitle, color string) (*Item, error)
  func (s *Store) GetAll() ([]Item, error)
  func (s *Store) GetByPrefix(prefix string) (*Item, error)
  func (s *Store) Update(id string, title, subtitle, color *string) (*Item, error)
  func (s *Store) Delete(id string) error
  func (s *Store) Close() error

  // cmd/root.go
  var rootCmd *cobra.Command  // with persistent flags: --db (string), --port (int)
  func Execute()              // called from main.go
  ```

- [ ] **Step 1: Initialize Go module and install dependencies**

Run:
```bash
go mod init github.com/AntiTyping/awesome-dashboard
go get github.com/spf13/cobra
go get modernc.org/sqlite
go get github.com/google/uuid
```

- [ ] **Step 2: Write `cmd/root.go` and `main.go`**

`cmd/root.go`: Define `rootCmd` with `Use: "awesome-dashboard"`. Add persistent string flag `--db` (default `~/.awesome-dashboard/data.db`) and persistent int flag `--port` (default `8080`). Export `Execute()` which calls `rootCmd.Execute()`.

`main.go`: Call `cmd.Execute()`.

- [ ] **Step 3: Verify the skeleton builds**

Run: `go build -o awesome-dashboard .`
Expected: binary created with no errors.

- [ ] **Step 4: Write failing store tests in `internal/store/store_test.go`**

```go
func TestOpen_CreatesSchema(t *testing.T)
// Open with ":memory:", query sqlite_master for "items" table, assert it exists

func TestAdd_And_GetAll(t *testing.T)
// Add two items, call GetAll, assert len==2, fields match inputs, IDs are valid UUIDs,
// CreatedAt/UpdatedAt are set, results ordered by CreatedAt

func TestGetByPrefix_Unique(t *testing.T)
// Add one item, call GetByPrefix with first 8 chars of its ID, assert match

func TestGetByPrefix_NotFound(t *testing.T)
// Empty store, call GetByPrefix("nonexist"), assert error

func TestGetByPrefix_Ambiguous(t *testing.T)
// Add two items, call GetByPrefix("") (empty string matches all), assert error indicates ambiguity

func TestUpdate_AllFields(t *testing.T)
// Add item, update all three fields, assert returned item has new values and UpdatedAt changed

func TestUpdate_PartialFields(t *testing.T)
// Add item, update only title (subtitle/color nil), assert only title changed

func TestDelete(t *testing.T)
// Add item, delete by full ID, GetAll returns empty

func TestDelete_NotFound(t *testing.T)
// Empty store, delete non-existent ID, assert error
```

- [ ] **Step 5: Run tests to verify they fail**

Run: `go test ./internal/store/ -v`
Expected: compilation errors (store.go doesn't exist yet)

- [ ] **Step 6: Implement `internal/store/store.go`**

`Open(dbPath string) (*Store, error)`: Open SQLite via `database/sql` with `modernc.org/sqlite` driver. Create parent directory if needed. Run `CREATE TABLE IF NOT EXISTS items (...)` with the schema from the spec. Return `&Store{db}`.

`Add(title, subtitle, color string) (*Item, error)`: Generate UUID via `uuid.New()`, INSERT with `datetime('now')` for both timestamps, query back the inserted row, return it.

`GetAll() ([]Item, error)`: `SELECT * FROM items ORDER BY created_at`. Scan rows into `[]Item`.

`GetByPrefix(prefix string) (*Item, error)`: `SELECT * FROM items WHERE id LIKE ? || '%'`. If 0 rows, return `"item not found"` error. If >1 rows, return `"ambiguous prefix, matches N items"` error. Return the single match.

`Update(id string, title, subtitle, color *string) (*Item, error)`: Build SET clause dynamically from non-nil pointers. Always set `updated_at = datetime('now')`. UPDATE WHERE id = ?. If rows affected == 0, return error. Query back and return updated item.

`Delete(id string) error`: `DELETE FROM items WHERE id = ?`. If rows affected == 0, return `"item not found"` error.

`Close() error`: Close the underlying `*sql.DB`.

- [ ] **Step 7: Run tests to verify they pass**

Run: `go test ./internal/store/ -v`
Expected: all 9 tests PASS

- [ ] **Step 8: Commit**

```bash
git add main.go go.mod go.sum cmd/root.go internal/store/store.go internal/store/store_test.go
git commit --author="Andy Pliszka <andy@isp44.com>" -m "feat: add project scaffolding and SQLite store layer"
```

---

### Task 2: HTTP Server + SSE Broker

**Files:**
- Create: `internal/server/server.go`, `internal/server/server_test.go`

**Interfaces:**
- Consumes: `store.Store.GetAll() ([]Item, error)` from Task 1
- Produces:
  ```go
  // internal/server/server.go
  type Server struct { /* unexported fields */ }

  func New(store *store.Store) *Server
  func (s *Server) RegisterRoutes(mux *http.ServeMux)
  // Registers onto the provided mux:
  //   GET  /api/items  → JSON array of all items
  //   GET  /api/events → SSE stream, sends "data: refresh\n\n" on notify
  //   POST /api/notify → triggers SSE broadcast, returns 204
  ```

- [ ] **Step 1: Write failing tests in `internal/server/server_test.go`**

```go
func TestGetItems_Empty(t *testing.T)
// Create in-memory store, New(store), RegisterRoutes on a new mux, GET /api/items via httptest
// Assert: 200, Content-Type application/json, body is "[]"

func TestGetItems_WithData(t *testing.T)
// Add 2 items to store, GET /api/items
// Assert: 200, body is JSON array with 2 items, fields match

func TestGetItems_OrderedByCreatedAt(t *testing.T)
// Add items A then B, GET /api/items
// Assert: first element is A (earlier created_at)

func TestNotifyTriggersSSE(t *testing.T)
// Connect to GET /api/events in a goroutine, POST /api/notify
// Assert: SSE client receives "data: refresh"

func TestSSEMultipleClients(t *testing.T)
// Connect 3 clients to /api/events, POST /api/notify once
// Assert: all 3 receive "data: refresh"

func TestSSEClientDisconnect(t *testing.T)
// Connect client to /api/events, disconnect it, POST /api/notify
// Assert: no panic or hang, subsequent notify still works for remaining clients
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/server/ -v`
Expected: compilation errors (server.go doesn't exist yet)

- [ ] **Step 3: Implement `internal/server/server.go`**

**SSE broker** (unexported): Holds a `sync.RWMutex`-guarded map of `chan struct{}` for each connected client. `subscribe()` adds a channel, `unsubscribe()` removes it, `broadcast()` sends to all channels (non-blocking send to avoid blocking on a slow client — drop the event if the channel buffer is full).

**`New(store *store.Store) *Server`**: Store the store reference, initialize the broker.

**`RegisterRoutes(mux *http.ServeMux)`**: Register onto the provided mux:
- `GET /api/items`: Call `store.GetAll()`, marshal to JSON, write with `Content-Type: application/json`.
- `GET /api/events`: Set SSE headers (`Content-Type: text/event-stream`, `Cache-Control: no-cache`, `Connection: keep-alive`). Flush. Subscribe to broker. Loop: wait on channel or context done. On channel receive, write `data: refresh\n\n` and flush. On context done, unsubscribe and return.
- `POST /api/notify`: Call `broker.broadcast()`, return 204 No Content.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/server/ -v`
Expected: all 6 tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/server/server.go internal/server/server_test.go
git commit --author="Andy Pliszka <andy@isp44.com>" -m "feat: add HTTP server with SSE broker"
```

---

### Task 3: Frontend + Serve Command

**Files:**
- Create: `web/web.go`, `web/index.html`, `cmd/serve.go`

**Interfaces:**
- Consumes: `server.New(store) → *Server`, `Server.RegisterRoutes(mux)` from Task 2; `store.Open(dbPath)` from Task 1; `rootCmd` from Task 1 (to register subcommand)
- Produces:
  ```go
  // web/web.go
  var Content embed.FS  // embeds index.html via //go:embed index.html
  ```
  ```
  Subcommand: awesome-dashboard serve
  Serves embedded index.html at GET /
  All API routes from Task 2's Handler mounted on the same mux
  ```

- [ ] **Step 1: Write `web/index.html`**

Single HTML file with inline CSS and JS. Structure:

**CSS:** System font stack. CSS custom properties for light/dark themes. `@media (prefers-color-scheme: dark)` overrides. CSS grid on `.grid` container: `grid-template-columns: repeat(auto-fill, minmax(280px, 1fr))`. Card styling: rounded corners, subtle shadow, left border using `style="border-left-color: ${item.color}"`. Header bar with app name "Awesome Dashboard".

**JS:** On `DOMContentLoaded`: call `fetchItems()`, connect SSE. `fetchItems()`: `fetch('/api/items')`, parse JSON, render cards into `.grid` container. If array is empty, show centered "No items yet. Use the CLI to add one." message. SSE: `new EventSource('/api/events')`, on `message` event call `fetchItems()`.

**Card HTML template** (generated in JS):
```html
<div class="card" style="border-left-color: COLOR">
  <div class="title">TITLE</div>
  <div class="subtitle">SUBTITLE</div>
</div>
```

- [ ] **Step 2: Write `web/web.go`**

Package `web`. Single `//go:embed index.html` directive. Export `var Content embed.FS`.

- [ ] **Step 3: Write `cmd/serve.go`**

Cobra command: `Use: "serve"`, `Short: "Start the dashboard web server"`. In `RunE`:
1. Read `--db` and `--port` from root persistent flags
2. Call `store.Open(dbPath)`; defer `store.Close()`
3. Create `server.New(store)`
4. Create `http.NewServeMux()`, call `srv.RegisterRoutes(mux)` to register API routes
5. Add a route for `GET /` on the same mux that serves `index.html` from `web.Content` (use `http.FileServer(http.FS(web.Content))`)
6. Print `"Dashboard running at http://localhost:<port>"` before listening
7. `http.ListenAndServe(fmt.Sprintf(":%d", port), mux)`

Register the command with `rootCmd.AddCommand(serveCmd)` in an `init()` function.

- [ ] **Step 4: Build and verify in browser**

Run:
```bash
go build -o awesome-dashboard .
./awesome-dashboard serve
```
Expected: Server starts, prints URL. Open `http://localhost:8080` in browser. See the header and empty state message. Check browser DevTools → Network → EventStream for `/api/events` connection.

- [ ] **Step 5: Commit**

```bash
git add web/web.go web/index.html cmd/serve.go
git commit --author="Andy Pliszka <andy@isp44.com>" -m "feat: add dashboard frontend and serve command"
```

---

### Task 4: CLI Commands + Notification Bridge

**Files:**
- Create: `internal/notify/notify.go`, `internal/notify/notify_test.go`, `cmd/add.go`, `cmd/update.go`, `cmd/remove.go`, `cmd/list.go`

**Interfaces:**
- Consumes: `store.Open`, `store.Add`, `store.GetByPrefix`, `store.Update`, `store.Delete`, `store.GetAll` from Task 1; `rootCmd` from Task 1 (to register subcommands)
- Produces:
  ```go
  // internal/notify/notify.go
  func Send(port int) // fire-and-forget POST to http://localhost:<port>/api/notify
                      // ignores all errors (best-effort)
  ```
  ```
  Subcommands: add, update, remove, list
  Each writes to stdout on success: item details with 8-char ID prefix
  ```

- [ ] **Step 1: Write failing test for notify in `internal/notify/notify_test.go`**

```go
func TestSend_PostsToNotifyEndpoint(t *testing.T)
// Start httptest.NewServer that records whether POST /api/notify was called
// Call Send(server.Port), assert the POST was received

func TestSend_IgnoresErrors(t *testing.T)
// Call Send(99999) (nothing listening), assert no panic
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/notify/ -v`
Expected: compilation errors

- [ ] **Step 3: Implement `internal/notify/notify.go`**

`Send(port int)`: `http.Post(fmt.Sprintf("http://localhost:%d/api/notify", port), "", nil)`. Discard response and error. Use a short timeout (1 second) via `http.Client{Timeout: time.Second}` so the CLI doesn't hang if the server is unreachable.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/notify/ -v`
Expected: both tests PASS

- [ ] **Step 5: Write failing CLI command tests**

Test file location: `cmd/cmd_test.go` (single file for all command tests).

Helper: `setupTestCmd(t) (rootCmd *cobra.Command, dbPath string)` — creates a temp DB, sets `--db` flag to it, returns the configured root command.

```go
func TestAddCommand(t *testing.T)
// Execute: add --title "Project A" --subtitle "In progress" --color "#FF5733"
// Assert: stdout contains "Project A", stdout contains 8-char ID prefix
// Assert: store.GetAll() returns 1 item with matching fields

func TestAddCommand_InvalidColor(t *testing.T)
// Execute: add --title "X" --subtitle "Y" --color "red"
// Assert: command returns error containing "invalid color"

func TestAddCommand_MissingRequiredFlag(t *testing.T)
// Execute: add --title "X" --subtitle "Y" (no --color)
// Assert: command returns error

func TestUpdateCommand(t *testing.T)
// Pre-add an item, execute: update <8-char-prefix> --title "New Title"
// Assert: stdout shows updated item, store shows new title, subtitle/color unchanged

func TestUpdateCommand_NoFlags(t *testing.T)
// Pre-add an item, execute: update <8-char-prefix> (no change flags)
// Assert: command returns error containing "at least one"

func TestRemoveCommand(t *testing.T)
// Pre-add an item, execute: remove <8-char-prefix>
// Assert: stdout confirms removal, store.GetAll() returns empty

func TestListCommand_Empty(t *testing.T)
// Execute: list
// Assert: stdout contains "No items"

func TestListCommand_WithItems(t *testing.T)
// Pre-add 2 items, execute: list
// Assert: stdout contains both titles and 8-char ID prefixes
```

- [ ] **Step 6: Run tests to verify they fail**

Run: `go test ./cmd/ -v`
Expected: compilation errors

- [ ] **Step 7: Implement `cmd/add.go`**

Cobra command: `Use: "add"`. Required flags: `--title`, `--subtitle`, `--color` (all strings, mark required). In `RunE`: validate color matches `^#[0-9a-fA-F]{6}$` regex. Open store, call `store.Add(...)`, print item summary with 8-char ID prefix (`id[:8]`), call `notify.Send(port)`, close store.

- [ ] **Step 8: Implement `cmd/update.go`**

Cobra command: `Use: "update"`, `Args: cobra.ExactArgs(1)`. Optional flags: `--title`, `--subtitle`, `--color`. In `RunE`: check that at least one flag was explicitly set (use `cmd.Flags().Changed()`), return error if none. If `--color` was set, validate hex format. Open store, call `store.GetByPrefix(args[0])` to resolve prefix to full ID, call `store.Update(...)` with non-nil pointers for changed flags only, print updated item, call `notify.Send(port)`, close store.

- [ ] **Step 9: Implement `cmd/remove.go`**

Cobra command: `Use: "remove"`, `Args: cobra.ExactArgs(1)`. In `RunE`: Open store, call `store.GetByPrefix(args[0])` to resolve and confirm the item exists (print its title in the success message), call `store.Delete(item.ID)`, print confirmation, call `notify.Send(port)`, close store.

- [ ] **Step 10: Implement `cmd/list.go`**

Cobra command: `Use: "list"`. In `RunE`: Open store, call `store.GetAll()`. If empty, print `"No items."`. Otherwise, print a formatted table: 8-char ID prefix, title, subtitle, color. Close store.

- [ ] **Step 11: Run tests to verify they pass**

Run: `go test ./cmd/ -v`
Expected: all 8 tests PASS

- [ ] **Step 12: End-to-end manual verification**

```bash
go build -o awesome-dashboard .
./awesome-dashboard serve &
./awesome-dashboard add --title "Project Alpha" --subtitle "Phase 1" --color "#3498DB"
./awesome-dashboard add --title "Project Beta" --subtitle "Planning" --color "#E74C3C"
./awesome-dashboard list
./awesome-dashboard update <id-prefix> --title "Project Alpha v2"
./awesome-dashboard remove <id-prefix>
```

Verify: dashboard at `http://localhost:8080` updates in real-time after each mutation. Kill the server after.

- [ ] **Step 13: Commit**

```bash
git add internal/notify/notify.go internal/notify/notify_test.go cmd/add.go cmd/update.go cmd/remove.go cmd/list.go cmd/cmd_test.go
git commit --author="Andy Pliszka <andy@isp44.com>" -m "feat: add CLI commands (add, update, remove, list) with notification bridge"
```
