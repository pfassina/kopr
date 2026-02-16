package panel

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Info is the info/backlinks panel.
type Info struct {
	width     int
	height    int
	title     string
	lines     []string
	focused   bool
}

func NewInfo() Info {
	return Info{
		title: "Info",
	}
}

func (i *Info) SetBacklinks(links []string) {
	i.title = "Backlinks"
	i.lines = links
}

func (i *Info) SetOutline(headings []string) {
	i.title = "Outline"
	i.lines = headings
}

func (i *Info) Clear() {
	i.lines = nil
}

func (i Info) View() string {
	if i.width == 0 || i.height == 0 {
		return ""
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("212")).
		Padding(0, 1)

	var b strings.Builder
	b.WriteString(titleStyle.Render(i.title))
	b.WriteByte('\n')

	viewHeight := i.height - 2
	if viewHeight < 0 {
		viewHeight = 0
	}

	if len(i.lines) == 0 {
		dim := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Padding(0, 1)
		b.WriteString(dim.Render("No items"))
		b.WriteByte('\n')
	} else {
		for j := 0; j < len(i.lines) && j < viewHeight; j++ {
			line := i.lines[j]
			if len(line) > i.width-2 {
				line = line[:i.width-5] + "..."
			}
			b.WriteString(" ")
			b.WriteString(line)
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
