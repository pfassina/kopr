package panel

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/yourusername/vimvault/internal/vault"
)

// FileSelectedMsg is sent when a file is selected in the tree.
type FileSelectedMsg struct {
	Path string
}

// Tree is the file tree panel.
type Tree struct {
	vault    *vault.Vault
	entries  []vault.Entry
	cursor   int
	offset   int
	width    int
	height   int
	focused  bool
}

func NewTree(v *vault.Vault) Tree {
	return Tree{
		vault: v,
	}
}

func (t *Tree) Refresh() {
	entries, _ := t.vault.ListEntries()
	t.entries = entries
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
				if !entry.IsDir {
					return t, func() tea.Msg {
						return FileSelectedMsg{Path: entry.Path}
					}
				}
			}
		case "G":
			t.cursor = len(t.entries) - 1
			if t.cursor-t.offset >= t.height-2 {
				t.offset = t.cursor - t.height + 3
			}
		case "g":
			t.cursor = 0
			t.offset = 0
		}
	}

	return t, nil
}

func (t Tree) View() string {
	if t.width == 0 || t.height == 0 {
		return ""
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("212")).
		Padding(0, 1)

	var b strings.Builder
	b.WriteString(titleStyle.Render("Files"))
	b.WriteByte('\n')

	viewHeight := t.height - 2 // title + bottom padding
	if viewHeight < 0 {
		viewHeight = 0
	}

	for i := t.offset; i < len(t.entries) && i-t.offset < viewHeight; i++ {
		entry := t.entries[i]
		indent := strings.Repeat("  ", entry.Depth)
		icon := "  "
		if entry.IsDir {
			icon = "▸ "
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

	return b.String()
}

func (t *Tree) SetSize(width, height int) {
	t.width = width
	t.height = height
}

func (t *Tree) SetFocused(focused bool) {
	t.focused = focused
}
