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
- **Wiki-link following is Go-side**: `gf` is remapped to an RPC notification; Kopr (not
  Neovim's native `gf`) parses the buffer and resolves the link. The cursor may be anywhere
  inside `[[...]]`, including on the brackets. The `.md` suffix is optional in links and
  matched case-insensitively; targets may contain spaces. Resolution is by basename
  (case-insensitive DB lookup), so this works identically in local and SSH mode.
- **Diagnostics are Go-side and event-driven**: `internal/markdown.Lint` runs on note open
  and after save (after format-on-save), never as-you-type. Inline hints go through
  Neovim's `vim.diagnostic` API over RPC; the list panel is a Bubble Tea panel carved out
  of the editor column. Broken-link warnings use the same basename resolution as `gf`.

## Package map

- `internal/app`: root Bubble Tea model; routes messages; layout; leader keys; note lifecycle.
- `internal/editor`: starts embedded Neovim, VT rendering, RPC control.
- `internal/index`: SQLite schema, indexing pipeline, watcher, search/backlinks.
- `internal/vault`: filesystem operations and helpers (templates/daily/inbox/link rewrite).
- `internal/panel`: UI panels (tree/info/finder/status/prompt/which-key/diagnostics).
- `internal/theme`: color palette shared by all panels; extracted from Neovim highlight groups.
- `internal/config`: TOML config + first-run setup.
- `internal/session`: persists UI state under `<vault>/.kopr/`.

## Theming

All TUI panels hold a `*theme.Theme` pointer set during `app.New()`. When Neovim's RPC connects, `App` applies the configured colorscheme, extracts highlight group colors via `nvim_get_hl`, and maps them onto the `Theme` struct in-place. Every panel's next `View()` call picks up the new colors automatically.

Config fields: `colorscheme` (passed to `:colorscheme <name>`) and `colorscheme_repo` (GitHub `owner/repo` to auto-clone into the managed plugin directory).
