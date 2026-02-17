package panel

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// InfoItem represents an item in the info panel.
type InfoItem struct {
	Title string
	Path  string
}

// Info is the info/backlinks panel.
type Info struct {
	width   int
	height  int
	title   string
	items   []InfoItem
	cursor  int
	offset  int
	focused bool
}

func NewInfo() Info {
	return Info{
		title: "Info",
	}
}

func (i *Info) SetBacklinks(items []InfoItem) {
	i.title = "Backlinks"
	i.items = items
	i.cursor = 0
	i.offset = 0
}

func (i *Info) SetOutline(headings []string) {
	i.title = "Outline"
	i.items = make([]InfoItem, len(headings))
	for j, h := range headings {
		i.items[j] = InfoItem{Title: h}
	}
	i.cursor = 0
	i.offset = 0
}

func (i *Info) Clear() {
	i.items = nil
	i.cursor = 0
	i.offset = 0
}

func (i Info) Update(msg tea.Msg) (Info, tea.Cmd) {
	if !i.focused {
		return i, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		viewHeight := i.height - 2
		if viewHeight < 1 {
			viewHeight = 1
		}
		switch msg.String() {
		case "j", "down":
			if i.cursor < len(i.items)-1 {
				i.cursor++
				if i.cursor-i.offset >= viewHeight {
					i.offset++
				}
			}
		case "k", "up":
			if i.cursor > 0 {
				i.cursor--
				if i.cursor < i.offset {
					i.offset = i.cursor
				}
			}
		case "enter":
			if i.cursor < len(i.items) {
				item := i.items[i.cursor]
				if item.Path != "" {
					return i, func() tea.Msg {
						return FileSelectedMsg{Path: item.Path}
					}
				}
			}
		case "G":
			i.cursor = len(i.items) - 1
			if i.cursor-i.offset >= viewHeight {
				i.offset = i.cursor - viewHeight + 1
			}
		case "g":
			i.cursor = 0
			i.offset = 0
		}
	}

	return i, nil
}

func (i Info) View() string {
	if i.width == 0 || i.height == 0 {
		return ""
	}

	var titleStyle lipgloss.Style
	if i.focused {
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
	b.WriteString(titleStyle.Render(i.title))
	b.WriteByte('\n')

	viewHeight := i.height - 2
	if viewHeight < 0 {
		viewHeight = 0
	}

	if len(i.items) == 0 {
		dim := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Padding(0, 1)
		b.WriteString(dim.Render("No items"))
		b.WriteByte('\n')
	} else {
		for j := i.offset; j < len(i.items) && j-i.offset < viewHeight; j++ {
			line := i.items[j].Title
			if len(line) > i.width-2 {
				line = line[:i.width-5] + "..."
			}

			// Pad to width
			padded := " " + line
			if len(padded) < i.width-2 {
				padded += strings.Repeat(" ", i.width-2-len(padded))
			}

			if j == i.cursor && i.focused {
				style := lipgloss.NewStyle().
					Foreground(lipgloss.Color("212")).
					Bold(true)
				b.WriteString(style.Render(padded))
			} else {
				b.WriteString(padded)
			}
			b.WriteByte('\n')
		}
	}

	return b.String()
}

func (i *Info) SetSize(width, height int) {
	i.width = width
	i.height = height
}

func (i *Info) SetFocused(focused bool) {
	i.focused = focused
}
