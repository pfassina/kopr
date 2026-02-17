# Architecture

Kopr is a terminal-first knowledge management system that embeds Neovim inside a Bubble Tea TUI. Notes are plain Markdown files in a user-defined vault directory.

## Execution modes

- **Local mode**: runs a `tea.Program` directly in the user’s terminal.
- **SSH mode** (`--serve`): runs a Wish SSH server; each session gets its own `app.App`.

## Core data flow

```
SSH / Local Terminal
       │
   app.App (Bubble Tea)
       │ PTY
   editor.Editor (Neovim in PTY + VT emulator)
       │ msgpack RPC
   Neovim process
       │
   index.DB + Watcher (SQLite FTS5 + fsnotify)
```

## Key invariants

- **Fail fast and loud**: no silent error handling in production code.
- **Basename uniqueness**: note filenames (basenames) are treated as unique within a vault.
- **Backlinks by basename**: backlinks match link targets by basename.
- **No RPC from `View()`**: Neovim RPC must never be invoked from Bubble Tea `View()`.

## Package map

- `internal/app`: root Bubble Tea model; routes messages; layout; leader keys; note lifecycle.
- `internal/editor`: starts embedded Neovim, VT rendering, RPC control.
- `internal/index`: SQLite schema, indexing pipeline, watcher, search/backlinks.
- `internal/vault`: filesystem operations and helpers (templates/daily/inbox/link rewrite).
- `internal/panel`: UI panels (tree/info/finder/status/prompt/which-key).
- `internal/config`: TOML config + first-run setup.
- `internal/session`: persists UI state under `<vault>/.kopr/`.
