Below is a single, complete, copy-ready PRD that consolidates everything we discussed, including:
 • Go + TUI + embedded Neovim
 • Local + SSH dual mode
 • Space-leader UX
 • Obsidian-style live preview
 • Focus/Zen modes
 • Smart search → create
 • Templates
 • Hidden frontmatter
 • Inline images (when supported)
 • Markdown auto-formatting (Prettier / CommonMark compatible)

⸻

📘 Product Requirements Document (PRD)

Product Name (Working)

VimVault

A self-hosted, terminal-first knowledge system combining Obsidian-style note management with Neovim-grade editing.

⸻

1. Vision

1.1 North Star

Build the first purpose-built, self-hosted “Knowledge IDE” that:
 • Uses real Neovim for editing
 • Runs locally and over SSH
 • Stores notes as plain Markdown files
 • Provides first-class Zettelkasten workflows
 • Requires no plugin ecosystem
 • Is installable as a single Go-based container

Users should feel:

“This is Obsidian rebuilt around Neovim, for serious knowledge work.”

⸻

1.2 Core Value Proposition

Problem Solution
Plugin fragmentation Integrated system
Heavy Electron apps Lightweight TUI
SaaS lock-in File-based storage
Remote instability SSH-native design
Inconsistent UX Opinionated workflows

⸻

1. Target Users

Primary
 • Neovim users
 • Engineers / PMs / researchers
 • Self-hosters
 • Zettelkasten practitioners

Secondary
 • Writers
 • Academics
 • Knowledge workers

⸻

1. Non-Goals (v1–v2)
 • No real-time collaboration
 • No SaaS hosting
 • No mobile-first UI
 • No proprietary formats
 • No plugin marketplace

⸻

1. Design Principles
 1. Files Are Truth
Markdown on disk is authoritative.
 2. Keyboard First
Everything is accessible via shortcuts.
 3. Opinionated Defaults
Product ships complete.
 4. Minimal Dependencies
Prefer Go stdlib and vendored libraries.
 5. Remote-First Reliability
Designed for SSH/tmux first.

⸻

1. System Architecture

┌──────────────┐
│ SSH / Local  │
│ Terminal     │
└──────┬───────┘
       │
┌──────▼────────┐
│ VimVault TUI  │  ← Go + Bubble Tea
│ UI Shell      │
└──────┬────────┘
       │ PTY
┌──────▼────────┐
│ Neovim        │  ← Native editor
└──────┬────────┘
       │
┌──────▼────────┐
│ Indexer       │  ← Go + SQLite
│ File Watcher  │
└───────────────┘

⸻

1. Functional Requirements

⸻

6.1 Execution Modes

FR-1: Dual Mode

The system MUST support:

Mode Description
Local Run directly in terminal
Remote Run via SSH/tmux

UX and keymaps must be identical.

⸻

FR-2: Session Persistence
 • Restore layout, buffers, cursors
 • Stored in .vimvault/state.json
 • Auto-resume on reconnect

⸻

6.2 Storage

FR-3: Vault
 • Single root directory
 • Internal structure is user-defined

Example:

/vault/
  inbox/
  notes/
  daily/
  assets/

No enforced hierarchy.

⸻

FR-4: File Sync
 • inotify-based watching
 • Automatic reindex on change
 • External edits reflected immediately

⸻

6.3 Editor Integration

FR-5: Embedded Neovim
 • Run nvim as PTY child
 • Full native behavior
 • Optional managed config

⸻

FR-6: Editor Control API

Support:
 • Open file
 • Jump to section
 • Insert template
 • Follow link
 • Format document
 • Save/close buffers

⸻

6.4 TUI Shell

FR-7: Default Layout

┌─────────┬───────────────┬──────────┐
│ Tree    │ Neovim Editor │ Info     │
│         │ + Preview     │ Panel    │
└─────────┴───────────────┴──────────┘

⸻

FR-8: Panels

Panel Purpose
Tree File browser
Info Backlinks / outline / preview
Finder Global search
Status Mode / git / stats

Panels are toggleable.

⸻

6.5 Navigation & Search

FR-9: File Tree
 • Keyboard navigation
 • Create/delete/rename
 • Collapse/expand

⸻

FR-10: Global Fuzzy Finder

Shortcut: <Space><Space>

Searches:
 • Files
 • Notes
 • Headings
 • Tags
 • Commands
 • Templates

Powered by SQLite FTS + scoring.

⸻

6.6 Knowledge Model

FR-11: Wiki Links

Supported:

[[note]]
[[note#section]]
[[note|alias]]

 • Auto-complete
 • Auto-create if missing

⸻

FR-12: Backlinks
 • Live backlinks panel
 • Orphan detection
 • Reference count

⸻

FR-13: Metadata

YAML frontmatter indexed:

tags:

- research
status: draft

Queryable.

⸻

6.7 Templates

FR-14: Template System

Directory: /templates

Variables:
 • {{title}}
 • {{date}}
 • {{uuid}}
 • {{path}}

Insert via fuzzy search.

⸻

6.8 Markdown Preview

FR-15: Unified Edit + Preview

Default mode:
 • Single pane
 • Source + rendered view together
 • Live rendering

Implementation:
 • Neovim buffer
 • Goldmark renderer
 • Conceal for syntax

⸻

FR-16: Raw Mode

Toggle raw source view:

<Space> v r

⸻

FR-17: Frontmatter Handling
 • Folded/hidden by default
 • Treated as metadata panel

Toggle:

<Space> v m

⸻

6.9 Assets & Images

FR-18: Image Handling
 • Store in /assets
 • Relative paths
 • Auto-insert link
 • Preview if terminal supports graphics
 • Fallback: open externally

⸻

6.10 Markdown Formatting

FR-19: Auto-Formatter

Shortcut:

<Space> m f

or

:gq

Function:
 • Format entire document
 • Use industry-standard rules:
 • CommonMark
 • Prettier-compatible style
 • Stable output
 • No semantic changes

Implementation options:
 • Embedded formatter
 • or prettier-markdown compatible engine
 • or Pandoc-based formatter

Formatting must be deterministic.

⸻

6.11 Remote UX

FR-20: SSH Mode
 • Auto-launch app
 • tmux session by default
 • Reattach on reconnect

⸻

FR-21: Clipboard Bridge

Commands:

:Copy
:Paste
:CopyToLocal
:PasteFromLocal

Via helper CLI.

⸻

FR-22: Copy Mode
 • App-level scrollback
 • Vim navigation
 • Search/yank

⸻

6.12 Authentication

FR-23: Access Control

Primary: SSH keys

Optional:
 • SSH certificates
 • SSO gateway

No in-app auth in v1.

⸻

1. Interaction Model

⸻

7.1 Leader Key

Default:

<Space>

Configurable.

⸻

7.2 Command Hierarchy

Prefix Domain
<Space> f Find
<Space> n Notes
<Space> t Templates
<Space> v View
<Space> z Zen
<Space> m Markdown
<Space> g Git

Which-key popup enabled.

⸻

7.3 Core Journeys

⸻

Find or Create Note

<Space> f n

 • Type query
 • If no match → “Create ‘query’”
 • Enter → new note + template

⸻

Daily Note

<Space> n d

⸻

Inbox Capture

<Space> n i

⸻

Insert Template

<Space> t i

⸻

Follow/Create Link

On [[foo]]:

Enter

⸻

7.4 Focus & Distraction Control

⸻

Zen Mode

<Space> z z

Effects:
 • Hide sidebars
 • Center editor
 • Dim UI

⸻

Panel Toggles

Shortcut Action
<Space> v t Toggle tree
<Space> v b Toggle backlinks
<Space> v p Toggle preview
<Space> v s Toggle status

⸻

Workspace Presets

Layouts saved as:
 • Writing
 • Research
 • Review
 • Daily

⸻

1. Creation & Organization

⸻

8.1 Smart Creation
 • Title → slug
 • Directory inference (optional)
 • UUID support

Example:

My New Idea → my-new-idea.md

⸻

8.2 Auto-Placement Rules (Optional)

rules:
  daily: daily/
  inbox: inbox/

⸻

8.3 Rename Refactor

<Space> n r

 • Rename file
 • Update all links
 • Atomic

⸻

1. Visual System

⸻

9.1 Themes

Built-in:
 • Catppuccin
 • Nord
 • Gruvbox
 • Tokyo Night

Terminal-native.

⸻

9.2 Typography
 • Respects terminal font
 • Ligatures supported
 • Optional reading mode spacing

⸻

1. Onboarding

⸻

10.1 First Launch Wizard
 1. Select vault
 2. Choose theme
 3. Import notes
 4. Install default nvim config (optional)

⸻

10.2 Help System

<Space> h

Interactive cheatsheet.

⸻

1. Non-Functional Requirements

⸻

11.1 Deployment
 • Single Docker image
 • Debian base
 • <150MB
 • ARM/x86

⸻

11.2 Performance

Metric Target
Startup <2s
Search <100ms
Reindex <5s / 10k files
UI latency <50ms

⸻

11.3 Reliability
 • Atomic writes
 • Crash-safe index
 • No data loss
 • Backup hooks

⸻

11.4 Maintainability
 • Go modules only
 • Vendored deps
 • <10 core dependencies
 • No runtime scripting

⸻

1. Technical Stack

Layer Tech
Language Go
TUI Bubble Tea
Styling Lip Gloss
DB SQLite FTS5
FS Watch fsnotify
Markdown Goldmark
PTY creack/pty
Editor Neovim ≥ 0.9

⸻

1. MVP Scope (Phase 1 – 3 Months)

Must Ship

✅ Vault
✅ Neovim embed
✅ File tree
✅ Finder
✅ Wiki links
✅ Backlinks
✅ Templates
✅ Zen mode
✅ Formatter
✅ SSH launcher
✅ SQLite index

Excluded:
 • Images
 • Graph
 • Web UI

⸻

1. Phase 2 (6–9 Months)
 • Inline images
 • Graph view
 • Git UI
 • Browser terminal
 • Multi-vault
 • Query DSL

⸻

1. Success Criteria

Metric Target
Setup <10 min
Retention >80%
Crashes <0.1%
Data loss 0

⸻

1. Risks & Mitigations

Risk Mitigation
PTY bugs Mature libs
Terminal variance Capability detection
Neovim changes Version pinning
Scope creep Strict MVP

⸻

1. Open Questions
 1. Ship default Neovim config?
 2. Built-in git workflows?
 3. Encrypted vault?
 4. Multi-user locking?
 5. External formatter engine?

⸻

Strategic Summary

VimVault is:

Obsidian’s knowledge model
 • Neovim’s editing power
 • tmux’s reliability
 • Go’s maintainability

Delivered as a single, coherent, self-hosted system.
