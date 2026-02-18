# Decisions

This file is the project’s lightweight decision log. Append entries; don’t rewrite history.

- 2026-02-17: Adopt **fail fast + loud** error handling. Avoid silent ignores in production code; either return/wrap errors or explicitly discard with a comment.
- 2026-02-17: Use basename-based wiki-link resolution and backlinks; enforce basename uniqueness within a vault.
- 2026-02-17: Basename uniqueness and wiki-link/backlink resolution are **case-insensitive** (canonical key = strings.ToLower(filepath.Base(path))).
- 2026-02-17: UX: Errors and user-facing messages should be **front-and-center** (overlay/prompt), not only the status bar. Validation errors during prompts keep the prompt open and render the message inside the prompt.
- 2026-02-17: Add deterministic Markdown auto-format on save (configurable via `auto_format_on_save`).
- 2026-02-17: Normalize vault path to an absolute path at startup to avoid Neovim CWD/path double-prefix issues.
- 2026-02-17: On Neovim buffer write, immediately re-index the saved note and refresh the info panel backlinks for the currently open note.
- 2026-02-17: Resizing: force a full Bubble Tea terminal repaint (`tea.ClearScreen`) on `WindowSizeMsg` to avoid persistent blank UI after terminal resizes; also signal Neovim with SIGWINCH after PTY resize.
- 2026-02-18: Finder UX: when no results match the query, show an explicit hint that Enter will create a note and require a confirm prompt before creating; cancel returns to the finder with the query preserved.
