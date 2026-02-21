package panel

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pfassina/kopr/internal/theme"
	"github.com/pfassina/kopr/internal/vault"
)

// ClipboardOp represents a pending clipboard operation.
type ClipboardOp int

const (
	ClipboardNone ClipboardOp = iota
	ClipboardCopy
	ClipboardCut
)

// Clipboard holds files staged for a paste operation.
type Clipboard struct {
	Op    ClipboardOp
	Paths []string
}

// FileSelectedMsg is sent when a file is selected in the tree.
type FileSelectedMsg struct {
	Path string
}

// TreeNewNoteMsg is sent when the user presses 'a' to add a note.
type TreeNewNoteMsg struct{}

// TreeDeleteNoteMsg is sent when the user presses 'd' to delete a single note.
type TreeDeleteNoteMsg struct {
	Path string
	Name string
}

// TreeDeleteNotesMsg is sent when deleting multiple selected notes.
type TreeDeleteNotesMsg struct {
	Paths []string
}

// TreeRenameNoteMsg is sent when the user presses 'r' to rename a note.
type TreeRenameNoteMsg struct {
	Path string
	Name string
}

// TreePasteMsg is sent when the user presses 'p' to paste.
type TreePasteMsg struct {
	Op      ClipboardOp
	Sources []string
	DestDir string
}

// TreeClipboardChangedMsg notifies the app that clipboard state changed.
type TreeClipboardChangedMsg struct {
	Op    ClipboardOp
	Count int
}

// Tree is the file tree panel.
type Tree struct {
	vault      *vault.Vault
	allEntries []vault.Entry
	entries    []vault.Entry
	collapsed  map[string]bool
	selected   map[string]bool
	clipboard  Clipboard
	cursor     int
	offset     int
	width      int
	height     int
	focused    bool
	showHelp   bool
	theme      *theme.Theme
}

func NewTree(v *vault.Vault) Tree {
	return Tree{
		vault:     v,
		collapsed: make(map[string]bool),
		selected:  make(map[string]bool),
	}
}

// SetTheme sets the color theme for the tree panel.
func (t *Tree) SetTheme(th *theme.Theme) { t.theme = th }

func (t *Tree) Refresh() {
	entries, _ := t.vault.ListEntries()
	t.allEntries = entries
	t.rebuildVisible()
	t.pruneStale()
}

// rebuildVisible filters allEntries based on collapsed state.
func (t *Tree) rebuildVisible() {
	t.entries = t.entries[:0]
	for _, e := range t.allEntries {
		if t.isHiddenByCollapse(e.Path) {
			continue
		}
		t.entries = append(t.entries, e)
	}
	// Clamp cursor
	if t.cursor >= len(t.entries) {
		t.cursor = len(t.entries) - 1
	}
	if t.cursor < 0 {
		t.cursor = 0
	}
}

// pruneStale removes selected/clipboard entries that no longer exist.
func (t *Tree) pruneStale() {
	exists := make(map[string]bool, len(t.allEntries))
	for _, e := range t.allEntries {
		exists[e.Path] = true
	}

	for p := range t.selected {
		if !exists[p] {
			delete(t.selected, p)
		}
	}

	if t.clipboard.Op != ClipboardNone {
		valid := t.clipboard.Paths[:0]
		for _, p := range t.clipboard.Paths {
			if exists[p] {
				valid = append(valid, p)
			}
		}
		t.clipboard.Paths = valid
		if len(valid) == 0 {
			t.clipboard = Clipboard{}
		}
	}
}

// isHiddenByCollapse checks if any ancestor directory of path is collapsed.
func (t *Tree) isHiddenByCollapse(path string) bool {
	dir := filepath.Dir(path)
	for dir != "." {
		if t.collapsed[dir] {
			return true
		}
		dir = filepath.Dir(dir)
	}
	return false
}

// collectTargets returns selected file paths, or the cursor file if none selected.
func (t *Tree) collectTargets() []string {
	if len(t.selected) > 0 {
		paths := make([]string, 0, len(t.selected))
		for p := range t.selected {
			paths = append(paths, p)
		}
		return paths
	}
	if t.cursor < len(t.entries) {
		entry := t.entries[t.cursor]
		if !entry.IsDir {
			return []string{entry.Path}
		}
	}
	return nil
}

// resolveDestDir returns the directory to paste into based on cursor position.
func (t *Tree) resolveDestDir() string {
	if t.cursor >= len(t.entries) {
		return "."
	}
	entry := t.entries[t.cursor]
	if entry.IsDir {
		return entry.Path
	}
	dir := filepath.Dir(entry.Path)
	if dir == "." {
		return ""
	}
	return dir
}

// ClearClipboard resets clipboard state.
func (t *Tree) ClearClipboard() {
	t.clipboard = Clipboard{}
}

// ClearSelected resets selection state.
func (t *Tree) ClearSelected() {
	t.selected = make(map[string]bool)
}

// ClipboardInfo returns the current clipboard operation and count.
func (t *Tree) ClipboardInfo() (ClipboardOp, int) {
	return t.clipboard.Op, len(t.clipboard.Paths)
}


func (t Tree) Init() tea.Cmd {
	return nil
}

func (t Tree) Update(msg tea.Msg) (Tree, tea.Cmd) {
	if !t.focused {
		return t, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// When help is shown, any key dismisses it
		if t.showHelp {
			t.showHelp = false
			return t, nil
		}

		switch msg.String() {
		case "j", "down":
			if t.cursor < len(t.entries)-1 {
				t.cursor++
				if t.cursor-t.offset >= t.height-2 {
					t.offset++
				}
			}
		case "k", "up":
			if t.cursor > 0 {
				t.cursor--
				if t.cursor < t.offset {
					t.offset = t.cursor
				}
			}
		case "enter":
			if t.cursor < len(t.entries) {
				entry := t.entries[t.cursor]
				if entry.IsDir {
					t.collapsed[entry.Path] = !t.collapsed[entry.Path]
					t.rebuildVisible()
				} else {
					return t, func() tea.Msg {
						return FileSelectedMsg{Path: entry.Path}
					}
				}
			}
		case "G":
			if len(t.entries) == 0 {
				break
			}
			t.cursor = len(t.entries) - 1
			if t.cursor-t.offset >= t.height-2 {
				t.offset = t.cursor - t.height + 3
			}
		case "g":
			t.cursor = 0
			t.offset = 0
		case "a":
			return t, func() tea.Msg { return TreeNewNoteMsg{} }
		case "v":
			if t.cursor < len(t.entries) {
				entry := t.entries[t.cursor]
				if !entry.IsDir {
					if t.selected[entry.Path] {
						delete(t.selected, entry.Path)
					} else {
						t.selected[entry.Path] = true
					}
				}
			}
		case "V":
			t.selected = make(map[string]bool)
			t.clipboard = Clipboard{}
			return t, func() tea.Msg {
				return TreeClipboardChangedMsg{Op: ClipboardNone, Count: 0}
			}
		case "y":
			targets := t.collectTargets()
			if len(targets) > 0 {
				t.clipboard = Clipboard{Op: ClipboardCopy, Paths: targets}
				t.selected = make(map[string]bool)
				op, count := t.clipboard.Op, len(t.clipboard.Paths)
				return t, func() tea.Msg {
					return TreeClipboardChangedMsg{Op: op, Count: count}
				}
			}
		case "x":
			targets := t.collectTargets()
			if len(targets) > 0 {
				t.clipboard = Clipboard{Op: ClipboardCut, Paths: targets}
				t.selected = make(map[string]bool)
				op, count := t.clipboard.Op, len(t.clipboard.Paths)
				return t, func() tea.Msg {
					return TreeClipboardChangedMsg{Op: op, Count: count}
				}
			}
		case "p":
			if t.clipboard.Op == ClipboardNone || len(t.clipboard.Paths) == 0 {
				return t, nil
			}
			destDir := t.resolveDestDir()
			pasteMsg := TreePasteMsg{
				Op:      t.clipboard.Op,
				Sources: append([]string(nil), t.clipboard.Paths...),
				DestDir: destDir,
			}
			t.clipboard = Clipboard{}
			return t, func() tea.Msg { return pasteMsg }
		case "d":
			targets := t.collectTargets()
			if len(targets) == 1 {
				name := filepath.Base(targets[0])
				path := targets[0]
				return t, func() tea.Msg {
					return TreeDeleteNoteMsg{Path: path, Name: name}
				}
			} else if len(targets) > 1 {
				paths := append([]string(nil), targets...)
				return t, func() tea.Msg {
					return TreeDeleteNotesMsg{Paths: paths}
				}
			}
		case "r":
			if t.cursor < len(t.entries) {
				entry := t.entries[t.cursor]
				if !entry.IsDir {
					return t, func() tea.Msg {
						return TreeRenameNoteMsg{Path: entry.Path, Name: entry.Name}
					}
				}
			}
		case "?":
			t.showHelp = !t.showHelp
		}
	}

	return t, nil
}

func (t Tree) View() string {
	if t.width == 0 || t.height == 0 {
		return ""
	}

	th := t.theme

	var titleStyle lipgloss.Style
	if t.focused {
		titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(th.Accent).
			Underline(true).
			Padding(0, 1)
	} else {
		titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(th.Dim).
			Padding(0, 1)
	}

	var b strings.Builder

	// Title row with optional ? hint
	title := titleStyle.Render("Files")
	if t.focused && !t.showHelp {
		hintStyle := lipgloss.NewStyle().Foreground(th.Dim)
		hint := hintStyle.Render("?")
		titleWidth := lipgloss.Width(title)
		hintWidth := lipgloss.Width(hint)
		gap := t.width - 2 - titleWidth - hintWidth
		if gap > 0 {
			b.WriteString(title)
			b.WriteString(strings.Repeat(" ", gap))
			b.WriteString(hint)
		} else {
			b.WriteString(title)
		}
	} else {
		b.WriteString(title)
	}
	b.WriteByte('\n')

	viewHeight := t.height - 2 // title + bottom padding
	if viewHeight < 0 {
		viewHeight = 0
	}

	// Reserve space for help if showing
	helpLines := 0
	if t.showHelp {
		helpLines = 14 // help box height
		viewHeight -= helpLines
		if viewHeight < 0 {
			viewHeight = 0
		}
	}

	markerSelected := lipgloss.NewStyle().Foreground(th.Accent).Render("\u258e")
	markerYanked := lipgloss.NewStyle().Foreground(th.Accent).Render("\u258e")
	markerCut := lipgloss.NewStyle().Foreground(th.Dim).Render("\u258e")

	for i := t.offset; i < len(t.entries) && i-t.offset < viewHeight; i++ {
		entry := t.entries[i]

		// Marker column
		marker := " "
		if !entry.IsDir {
			if t.selected[entry.Path] {
				marker = markerSelected
			} else if t.isInClipboard(entry.Path) {
				if t.clipboard.Op == ClipboardCopy {
					marker = markerYanked
				} else {
					marker = markerCut
				}
			}
		}

		indent := strings.Repeat("  ", entry.Depth)
		icon := "  "
		if entry.IsDir {
			if t.collapsed[entry.Path] {
				icon = "\u25b8 "
			} else {
				icon = "\u25be "
			}
		}

		line := fmt.Sprintf("%s%s%s", indent, icon, entry.Name)

		// Truncate to width (account for marker column)
		maxLineWidth := t.width - 3
		if len(line) > maxLineWidth {
			line = line[:maxLineWidth-3] + "..."
		}

		// Pad to width
		if len(line) < maxLineWidth {
			line += strings.Repeat(" ", maxLineWidth-len(line))
		}

		if i == t.cursor && t.focused {
			style := lipgloss.NewStyle().
				Foreground(th.Accent).
				Bold(true)
			b.WriteString(marker + style.Render(line))
		} else {
			b.WriteString(marker + line)
		}
		b.WriteByte('\n')
	}

	if t.showHelp {
		b.WriteString(t.renderHelp())
	}

	return b.String()
}

// isInClipboard checks if a path is in the clipboard.
func (t Tree) isInClipboard(path string) bool {
	for _, p := range t.clipboard.Paths {
		if p == path {
			return true
		}
	}
	return false
}

func (t Tree) renderHelp() string {
	th := t.theme
	dim := lipgloss.NewStyle().Foreground(th.Dim)
	key := lipgloss.NewStyle().Foreground(th.Accent).Bold(true)
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(th.Border).
		Padding(0, 1).
		Width(t.width - 6)

	lines := []struct{ k, v string }{
		{"j/k", "Navigate"},
		{"enter", "Open / Toggle dir"},
		{"a", "New note or dir"},
		{"v", "Toggle select"},
		{"V", "Clear selections"},
		{"y", "Yank (copy)"},
		{"x", "Cut (move)"},
		{"p", "Paste"},
		{"d", "Delete"},
		{"r", "Rename note"},
		{"g/G", "Top / Bottom"},
		{"?", "Toggle help"},
	}

	var sb strings.Builder
	for _, l := range lines {
		sb.WriteString(fmt.Sprintf("  %s  %s\n", key.Render(fmt.Sprintf("%-5s", l.k)), dim.Render(l.v)))
	}

	return border.Render(strings.TrimRight(sb.String(), "\n"))
}

func (t *Tree) SetSize(width, height int) {
	t.width = width
	t.height = height
}

func (t *Tree) SetFocused(focused bool) {
	t.focused = focused
}

func (t Tree) ShowingHelp() bool {
	return t.showHelp
}
