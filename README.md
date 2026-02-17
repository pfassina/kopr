# kopr

A terminal-first knowledge management system. Kopr embeds Neovim inside a TUI, giving you an Obsidian-like experience entirely in the terminal.

Notes are plain Markdown files. Wiki links, backlinks, full-text search, daily notes, and templates all work out of the box. Connect over SSH for remote access.

## Features

- Embedded Neovim editor with managed config
- File tree and backlinks panels
- Full-text search (SQLite FTS5)
- Wiki links (`[[note]]`, `[[note#section]]`, `[[note|alias]]`)
- Daily notes and inbox capture
- Template system with variable expansion
- SSH server mode for remote access
- Session persistence
- Themes (catppuccin, nord, gruvbox, tokyo-night)

## Install

```bash
go install github.com/pfassina/kopr/cmd/kopr@latest
```

Requires Neovim >= 0.9.

## Usage

```bash
# Local mode
kopr --vault ~/notes

# SSH server mode
kopr --serve --vault ~/notes --listen :2222
```

## License

[MIT](LICENSE)
