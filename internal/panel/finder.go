package panel

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FinderItem represents an item in the finder results.
type FinderItem struct {
	Title string
	Path  string
	Extra string // e.g., heading text, tag
}

// FinderResultMsg is sent when a finder item is selected.
type FinderResultMsg struct {
	Path string
}

// FinderCreateMsg is sent when the user wants to create a new note.
type FinderCreateMsg struct {
	Name string
}

// FinderClosedMsg is sent when the finder is dismissed.
type FinderClosedMsg struct{}

// SearchFunc is called to get results for a query.
type SearchFunc func(query string) []FinderItem

// Finder is a fuzzy finder overlay.
type Finder struct {
	input    textinput.Model
	items    []FinderItem
	cursor   int
	width    int
	height   int
	visible  bool
	searchFn SearchFunc
}

func NewFinder() Finder {
	ti := textinput.New()
	ti.Placeholder = "Search notes..."
	ti.CharLimit = 256
	ti.Width = 50
	ti.Focus()

	return Finder{
		input: ti,
	}
}

func (f *Finder) SetSearchFunc(fn SearchFunc) {
	f.searchFn = fn
}

func (f *Finder) Show() {
	f.visible = true
	f.input.SetValue("")
	f.cursor = 0
	f.input.Focus()
	if f.searchFn != nil {
		f.items = f.searchFn("")
	}
}

func (f *Finder) Hide() {
	f.visible = false
	f.input.Blur()
}

func (f Finder) Visible() bool {
	return f.visible
}

func (f Finder) Update(msg tea.Msg) (Finder, tea.Cmd) {
	if !f.visible {
		return f, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			f.visible = false
			return f, func() tea.Msg { return FinderClosedMsg{} }

		case "enter":
			if f.cursor < len(f.items) {
				item := f.items[f.cursor]
				f.visible = false
				return f, func() tea.Msg {
					return FinderResultMsg{Path: item.Path}
				}
			}
			// No results — create a new note with the query as name
			query := strings.TrimSpace(f.input.Value())
			if query != "" {
				f.visible = false
				return f, func() tea.Msg {
					return FinderCreateMsg{Name: query}
				}
			}
			return f, nil

		case "up", "ctrl+p", "ctrl+k":
			if f.cursor > 0 {
				f.cursor--
			}
			return f, nil

		case "down", "ctrl+n", "ctrl+j":
			if f.cursor < len(f.items)-1 {
				f.cursor++
			}
			return f, nil
		}
	}

	var cmd tea.Cmd
	prevValue := f.input.Value()
	f.input, cmd = f.input.Update(msg)

	// Re-search on input change
	if f.input.Value() != prevValue && f.searchFn != nil {
		f.items = f.searchFn(f.input.Value())
		f.cursor = 0
	}

	return f, cmd
}

func (f Finder) View() string {
	if !f.visible {
		return ""
	}

	width := f.width
	if width == 0 {
		width = 60
	}
	innerWidth := width - 6

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("212")).
		Padding(0, 1).
		Width(innerWidth)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("212"))

	var lines []string
	lines = append(lines, titleStyle.Render("Find Note"))
	lines = append(lines, f.input.View())
	lines = append(lines, "")

	maxResults := f.height/2 - 4
	if maxResults < 5 {
		maxResults = 5
	}
	if maxResults > len(f.items) {
		maxResults = len(f.items)
	}

	if len(f.items) == 0 {
		dim := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		lines = append(lines, dim.Render("No results"))
	} else {
		for i := 0; i < maxResults; i++ {
			item := f.items[i]
			prefix := "  "
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

			if i == f.cursor {
				prefix = "> "
				style = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
			}

			title := item.Title
			if title == "" {
				title = item.Path
			}

			line := fmt.Sprintf("%s%s", prefix, title)
			if item.Extra != "" {
				dim := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
				line += " " + dim.Render(item.Extra)
			}

			// Truncate
			if lipgloss.Width(line) > innerWidth {
				line = line[:innerWidth-3] + "..."
			}

			lines = append(lines, style.Render(line))
		}

		if len(f.items) > maxResults {
			dim := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
			lines = append(lines, dim.Render(fmt.Sprintf("  ... and %d more", len(f.items)-maxResults)))
		}
	}

	content := strings.Join(lines, "\n")
	return borderStyle.Render(content)
}

func (f *Finder) SetSize(width, height int) {
	f.width = width
	f.height = height
	f.input.Width = width/2 - 8
}
