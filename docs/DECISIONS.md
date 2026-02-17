# Decisions

This file is the project’s lightweight decision log. Append entries; don’t rewrite history.

- 2026-02-17: Adopt **fail fast + loud** error handling. Avoid silent ignores in production code; either return/wrap errors or explicitly discard with a comment.
- 2026-02-17: Use basename-based wiki-link resolution and backlinks; enforce basename uniqueness within a vault.
