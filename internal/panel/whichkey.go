package panel

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/pfassina/kopr/internal/theme"
)

// WhichKeyEntry represents a single key binding for display.
type WhichKeyEntry struct {
	Key   string
	Label string
}

// WhichKey renders a which-key style popup showing available bindings.
type WhichKey struct {
	entries []WhichKeyEntry
	prefix  string
	width   int
	theme   *theme.Theme
}

// SetTheme sets the color theme for the which-key popup.
func (w *WhichKey) SetTheme(th *theme.Theme) { w.theme = th }

func NewWhichKey() WhichKey {
	return WhichKey{}
}

func (w *WhichKey) SetEntries(prefix string, entries []WhichKeyEntry) {
	w.prefix = prefix
	w.entries = entries
	sort.Slice(w.entries, func(i, j int) bool {
		return w.entries[i].Key < w.entries[j].Key
	})
}

func (w *WhichKey) SetWidth(width int) {
	w.width = width
}

func (w *WhichKey) Clear() {
	w.entries = nil
	w.prefix = ""
}

func (w WhichKey) View() string {
	if len(w.entries) == 0 {
		return ""
	}

	th := w.theme

	width := w.width
	if width == 0 {
		width = 60
	}

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(th.Accent).
		Padding(0, 1).
		Width(width - 4)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(th.Accent)

	keyStyle := lipgloss.NewStyle().
		Foreground(th.InsertMode).
		Bold(true)

	labelStyle := lipgloss.NewStyle().
		Foreground(th.Text)

	var lines []string
	if w.prefix != "" {
		lines = append(lines, titleStyle.Render(fmt.Sprintf("Leader > %s", w.prefix)))
	} else {
		lines = append(lines, titleStyle.Render("Leader"))
	}

	// Render entries in columns
	colWidth := (width - 4) / 2
	if colWidth < 20 {
		colWidth = width - 4
	}

	for i := 0; i < len(w.entries); i += 2 {
		left := fmt.Sprintf("%s %s",
			keyStyle.Render(w.entries[i].Key),
			labelStyle.Render(w.entries[i].Label),
		)

		if i+1 < len(w.entries) && colWidth < width-4 {
			right := fmt.Sprintf("%s %s",
				keyStyle.Render(w.entries[i+1].Key),
				labelStyle.Render(w.entries[i+1].Label),
			)
			// Pad left column
			leftPad := colWidth - lipgloss.Width(left)
			if leftPad < 1 {
				leftPad = 1
			}
			lines = append(lines, left+strings.Repeat(" ", leftPad)+right)
		} else {
			lines = append(lines, left)
		}
	}

	content := strings.Join(lines, "\n")
	return borderStyle.Render(content)
}
