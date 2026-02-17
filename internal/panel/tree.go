package panel

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pfassina/kopr/internal/vault"
)

// FileSelectedMsg is sent when a file is selected in the tree.
type FileSelectedMsg struct {
	Path string
}

// TreeNewNoteMsg is sent when the user presses 'a' to add a note.
type TreeNewNoteMsg struct{}

// TreeDeleteNoteMsg is sent when the user presses 'd' to delete a note.
type TreeDeleteNoteMsg struct {
	Path string
	Name string
}

// TreeRenameNoteMsg is sent when the user presses 'r' to rename a note.
type TreeRenameNoteMsg struct {
	Path string
	Name string
}

// TreeMoveNoteMsg is sent when the user presses 'm' to move a note.
type TreeMoveNoteMsg struct {
	Path string
	Name string
}

// Tree is the file tree panel.
type Tree struct {
	vault      *vault.Vault
	allEntries []vault.Entry
	entries    []vault.Entry
	collapsed  map[string]bool
	cursor     int
	offset     int
	width      int
	height     int
	focused    bool
	showHelp   bool
}

func NewTree(v *vault.Vault) Tree {
	return Tree{
		vault:     v,
		collapsed: make(map[string]bool),
	}
}

func (t *Tree) Refresh() {
	entries, _ := t.vault.ListEntries()
	t.allEntries = entries
	t.rebuildVisible()
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
		case "d":
			if t.cursor < len(t.entries) {
				entry := t.entries[t.cursor]
				if !entry.IsDir {
					return t, func() tea.Msg {
						return TreeDeleteNoteMsg{Path: entry.Path, Name: entry.Name}
					}
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
		case "m":
			if t.cursor < len(t.entries) {
				entry := t.entries[t.cursor]
				if !entry.IsDir {
					return t, func() tea.Msg {
						return TreeMoveNoteMsg{Path: entry.Path, Name: entry.Name}
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

	var titleStyle lipgloss.Style
	if t.focused {
		titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212")).
			Underline(true).
			Padding(0, 1)
	} else {
		titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("240")).
			Padding(0, 1)
	}

	var b strings.Builder

	// Title row with optional ? hint
	title := titleStyle.Render("Files")
	if t.focused && !t.showHelp {
		hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
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
		helpLines = 11 // help box height
		viewHeight -= helpLines
		if viewHeight < 0 {
			viewHeight = 0
		}
	}

	for i := t.offset; i < len(t.entries) && i-t.offset < viewHeight; i++ {
		entry := t.entries[i]
		indent := strings.Repeat("  ", entry.Depth)
		icon := "  "
		if entry.IsDir {
			if t.collapsed[entry.Path] {
				icon = "▸ "
			} else {
				icon = "▾ "
			}
		}

		line := fmt.Sprintf("%s%s%s", indent, icon, entry.Name)

		// Truncate to width
		if len(line) > t.width-2 {
			line = line[:t.width-5] + "..."
		}

		// Pad to width
		if len(line) < t.width-2 {
			line += strings.Repeat(" ", t.width-2-len(line))
		}

		if i == t.cursor && t.focused {
			style := lipgloss.NewStyle().
				Foreground(lipgloss.Color("212")).
				Bold(true)
			b.WriteString(style.Render(line))
		} else {
			b.WriteString(line)
		}
		b.WriteByte('\n')
	}

	if t.showHelp {
		b.WriteString(t.renderHelp())
	}

	return b.String()
}

func (t Tree) renderHelp() string {
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	key := lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Width(t.width - 6)

	lines := []struct{ k, v string }{
		{"j/k", "Navigate"},
		{"enter", "Open / Toggle dir"},
		{"a", "New note or dir"},
		{"d", "Delete note"},
		{"r", "Rename note"},
		{"m", "Move note"},
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
