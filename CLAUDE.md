# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Development Commands

```bash
# Nix dev shell (recommended)
nix develop

# Build
make build

# Run
make run
# or: make run ARGS="--vault ~/notes"

# Tests
make test
make test-integration

# Lint
go env GOPATH  # sanity check you’re in the dev shell
make lint

# Docker
make docker

# Focused tests
# go test ./internal/index/... -run TestSearchNotes
# go test ./internal/panel/...
```

Nix dev shell (`nix develop`) provides: go, gopls, delve, golangci-lint, neovim, sqlite.

## Architecture

Kopr is a terminal-first knowledge management system that embeds Neovim inside a Bubble Tea TUI. Notes are plain Markdown files in a user-defined vault directory.

### Dual-mode operation
- **Local mode**: runs a `tea.Program` directly in the user’s terminal.
- **SSH mode** (`--serve` flag): starts a Wish SSH server (`internal/ssh`), each session gets its own `app.App`

### Core data flow
```
SSH / Local Terminal
       │
   app.App (Bubble Tea)       ← internal/app: coordinates all panels
       │ PTY
   editor.Editor              ← internal/editor: Neovim in a PTY + VT emulator
       │ msgpack RPC
   Neovim process             ← communicates via Unix socket
       │
   index.DB + Watcher         ← internal/index: SQLite FTS5, fsnotify
```

### Key packages

- **`internal/app`** — Central `App` model owns all subcomponents. Leader key state machine in `keymap.go`. Layout computed in `layout.go` (three-column: tree | editor | info).
- **`internal/editor`** — Wraps Neovim: spawns it in a PTY (`pty.go`), reads output into `charmbracelet/x/vt` terminal emulator (`vt.go`), controls via msgpack RPC (`rpc.go`). Key events are converted to PTY bytes in `input.go`. Managed nvim config via `profile.go`.
- **`internal/index`** — SQLite with FTS5 for full-text search. Hash-based change detection in `indexer.go`. `fsnotify` watcher for incremental reindex.
- **`internal/vault`** — File CRUD, daily/inbox note creation, template expansion, and vault-wide wiki-link rewriting helpers (used on rename).
- **`internal/panel`** — Tree browser (multi-select + clipboard cut/copy/paste), info/backlinks, finder (fuzzy search overlay), prompt (incl. confirm), status bar (mode/file/errors/clipboard), which-key popup. All implement Bubble Tea's `Init/Update/View`.
- **`internal/config`** — TOML config at `~/.config/kopr/config.toml` (XDG-aware). First-run setup wizard in `setup.go`.
- **`internal/session`** — Persists panel state to `<vault>/.kopr/state.json`.
- **`internal/markdown`** — Goldmark-based parser: frontmatter, headings, wiki links. Deterministic CommonMark formatter.

### Communication pattern
Panels communicate with `App` via typed Bubble Tea messages (e.g., `FileSelectedMsg`, `NoteClosedMsg`). No shared mutable state across packages. All Neovim RPC happens in `Update()` or commands — never in `View()`.

## Coding Conventions

- **Fail fast, loud errors (project policy)** — avoid silent error handling. In production code, check and propagate errors. If an error is intentionally ignored, use an explicit discard (`_ = err`) with a comment explaining why.
- **CGO disabled** — `modernc.org/sqlite` is a pure-Go SQLite driver. `CGO_ENABLED=0` throughout.
- **Value receivers** for `Update`/`View`, **pointer receivers** for mutating setters.
- **XDG-aware paths** — config at `~/.config/kopr/`, data at `~/.local/share/kopr/`. Vault metadata at `<vault>/.kopr/`.
- **Tests** use stdlib `testing` only — no external frameworks. Common patterns: `t.TempDir()` for filesystem isolation, `index.OpenMemory()` for in-memory SQLite, `t.Setenv()` for XDG overrides, table-driven tests.
