# Decisions

This file is the project’s lightweight decision log. Append entries; don’t rewrite history.

- 2026-02-17: Adopt **fail fast + loud** error handling. Avoid silent ignores in production code; either return/wrap errors or explicitly discard with a comment.
- 2026-02-17: Use basename-based wiki-link resolution and backlinks; enforce basename uniqueness within a vault.
- 2026-02-17: Add deterministic Markdown auto-format on save (configurable via `auto_format_on_save`).
- 2026-02-17: Normalize vault path to an absolute path at startup to avoid Neovim CWD/path double-prefix issues.
