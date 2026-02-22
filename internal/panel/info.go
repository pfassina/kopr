package panel

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pfassina/kopr/internal/theme"
)

// InfoGotoLineMsg is emitted when the user presses enter on an outline heading.
type InfoGotoLineMsg struct {
	Line int
}

// InfoItem represents an item in the info panel.
type InfoItem struct {
	Title string
	Path  string
	Line  int
	Level int
}

// section represents a collapsible section in the info panel.
type section struct {
	title     string
	items     []InfoItem
	collapsed bool
	emptyMsg  string
}

// flatRowKind distinguishes section headers from items in the flat list.
type flatRowKind int

const (
	rowHeader flatRowKind = iota
	rowItem
	rowSeparator // blank line between sections, not selectable
)

// flatRow is a single row in the virtual flat list.
type flatRow struct {
	kind       flatRowKind
	sectionIdx int
	itemIdx    int // only valid when kind == rowItem
}

// Info is the info panel with collapsible sections.
type Info struct {
	width    int
	height   int
	sections [3]section
	cursor   int
	offset   int
	focused  bool
	theme    *theme.Theme
}

// SetTheme sets the color theme for the info panel.
func (i *Info) SetTheme(th *theme.Theme) { i.theme = th }

func NewInfo() Info {
	return Info{
		sections: [3]section{
			{title: "Backlinks", emptyMsg: "No backlinks"},
			{title: "Outgoing Links", emptyMsg: "No outgoing links"},
			{title: "Outline", emptyMsg: "No headings"},
		},
	}
}

func (i *Info) SetBacklinks(items []InfoItem) {
	i.sections[0].items = items
	i.clampCursor()
}

func (i *Info) SetOutgoingLinks(items []InfoItem) {
	i.sections[1].items = items
	i.clampCursor()
}

func (i *Info) SetOutline(items []InfoItem) {
	i.sections[2].items = items
	i.clampCursor()
}

func (i *Info) Clear() {
	for idx := range i.sections {
		i.sections[idx].items = nil
	}
	i.cursor = 0
	i.offset = 0
}

// flatList builds the virtual flat list from sections.
func (i Info) flatList() []flatRow {
	var rows []flatRow
	for si := range i.sections {
		if si > 0 {
			rows = append(rows, flatRow{kind: rowSeparator})
		}
		rows = append(rows, flatRow{kind: rowHeader, sectionIdx: si})
		if !i.sections[si].collapsed {
			for ii := range i.sections[si].items {
				rows = append(rows, flatRow{kind: rowItem, sectionIdx: si, itemIdx: ii})
			}
		}
	}
	return rows
}

func (i *Info) clampCursor() {
	rows := i.flatList()
	if len(rows) == 0 {
		i.cursor = 0
		i.offset = 0
		return
	}
	if i.cursor >= len(rows) {
		i.cursor = len(rows) - 1
	}
	if i.cursor < 0 {
		i.cursor = 0
	}
	// Skip separators
	for i.cursor < len(rows) && rows[i.cursor].kind == rowSeparator {
		i.cursor++
	}
	if i.cursor >= len(rows) {
		i.cursor = len(rows) - 1
		for i.cursor > 0 && rows[i.cursor].kind == rowSeparator {
			i.cursor--
		}
	}
}

func (i Info) Update(msg tea.Msg) (Info, tea.Cmd) {
	if !i.focused {
		return i, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		rows := i.flatList()
		if len(rows) == 0 {
			return i, nil
		}

		viewHeight := i.viewHeight()

		switch msg.String() {
		case "j", "down":
			next := i.cursor + 1
			// Skip separators
			for next < len(rows) && rows[next].kind == rowSeparator {
				next++
			}
			if next < len(rows) {
				i.cursor = next
				if i.cursor-i.offset >= viewHeight {
					i.offset = i.cursor - viewHeight + 1
				}
			}
		case "k", "up":
			prev := i.cursor - 1
			// Skip separators
			for prev >= 0 && rows[prev].kind == rowSeparator {
				prev--
			}
			if prev >= 0 {
				i.cursor = prev
				if i.cursor < i.offset {
					i.offset = i.cursor
				}
			}
		case "enter":
			if i.cursor < len(rows) {
				row := rows[i.cursor]
				if row.kind == rowHeader {
					i.sections[row.sectionIdx].collapsed = !i.sections[row.sectionIdx].collapsed
					i.clampCursor()
				} else {
					item := i.sections[row.sectionIdx].items[row.itemIdx]
					if item.Path != "" {
						return i, func() tea.Msg {
							return FileSelectedMsg{Path: item.Path}
						}
					}
					if item.Line > 0 {
						return i, func() tea.Msg {
							return InfoGotoLineMsg{Line: item.Line}
						}
					}
				}
			}
		case "G":
			i.cursor = len(rows) - 1
			for i.cursor > 0 && rows[i.cursor].kind == rowSeparator {
				i.cursor--
			}
			if i.cursor-i.offset >= viewHeight {
				i.offset = i.cursor - viewHeight + 1
			}
		case "g":
			i.cursor = 0
			for i.cursor < len(rows)-1 && rows[i.cursor].kind == rowSeparator {
				i.cursor++
			}
			i.offset = 0
		case "tab", "}":
			i.jumpSection(rows, 1)
		case "shift+tab", "{":
			i.jumpSection(rows, -1)
		}
	}

	return i, nil
}

// jumpSection moves the cursor to the next or previous section header.
func (i *Info) jumpSection(rows []flatRow, dir int) {
	if len(rows) == 0 {
		return
	}
	pos := i.cursor + dir
	for pos >= 0 && pos < len(rows) {
		if rows[pos].kind == rowHeader {
			i.cursor = pos
			i.scrollIntoView()
			return
		}
		pos += dir
	}
}

func (i *Info) scrollIntoView() {
	viewHeight := i.viewHeight()
	if i.cursor < i.offset {
		i.offset = i.cursor
	}
	if i.cursor-i.offset >= viewHeight {
		i.offset = i.cursor - viewHeight + 1
	}
}

func (i Info) viewHeight() int {
	return max(i.height-2, 1) // -1 for title row, -1 for bottom padding
}

func (i Info) View() string {
	if i.width == 0 || i.height == 0 {
		return ""
	}

	th := i.theme
	rows := i.flatList()

	var b strings.Builder

	// Panel title
	var titleStyle lipgloss.Style
	if i.focused {
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
	b.WriteString(titleStyle.Render("Info"))
	b.WriteByte('\n')

	viewHeight := i.viewHeight()
	end := min(i.offset+viewHeight, len(rows))

	for idx := i.offset; idx < end; idx++ {
		row := rows[idx]
		switch row.kind {
		case rowSeparator:
			b.WriteByte('\n')
			continue
		case rowHeader:
			b.WriteString(i.renderHeader(row.sectionIdx, idx == i.cursor))
		case rowItem:
			item := i.sections[row.sectionIdx].items[row.itemIdx]
			b.WriteString(i.renderItem(item, row.sectionIdx, idx == i.cursor))
		}
		b.WriteByte('\n')
	}

	// If all sections are empty and nothing rendered, show a dim message.
	if len(rows) == 0 {
		dim := lipgloss.NewStyle().Foreground(th.Dim).Padding(0, 1)
		b.WriteString(dim.Render("No items"))
		b.WriteByte('\n')
	}

	return b.String()
}

func (i Info) renderHeader(sectionIdx int, selected bool) string {
	th := i.theme
	sec := i.sections[sectionIdx]

	indicator := "▾"
	if sec.collapsed {
		indicator = "▸"
	}

	text := fmt.Sprintf("%s %s (%d)", indicator, sec.title, len(sec.items))

	// Pad to width
	padded := " " + text
	if len(padded) < i.width-2 {
		padded += strings.Repeat(" ", i.width-2-len(padded))
	}

	if selected && i.focused {
		return lipgloss.NewStyle().
			Foreground(th.Accent2).
			Bold(true).
			Render(padded)
	}

	style := lipgloss.NewStyle().Bold(true)
	if i.focused {
		style = style.Foreground(th.Accent)
	} else {
		style = style.Foreground(th.Dim)
	}
	return style.Render(padded)
}

func (i Info) renderItem(item InfoItem, sectionIdx int, selected bool) string {
	th := i.theme

	title := item.Title
	indent := "   "
	// Outline items get extra indentation by heading level.
	if sectionIdx == 2 && item.Level > 1 {
		indent += strings.Repeat("  ", item.Level-1)
	}

	line := indent + title
	maxW := i.width - 2
	if len(line) > maxW && maxW > 3 {
		line = line[:maxW-3] + "..."
	}

	// Pad to width
	if len(line) < maxW {
		line += strings.Repeat(" ", maxW-len(line))
	}

	if selected && i.focused {
		style := lipgloss.NewStyle().
			Foreground(th.Accent2).
			Bold(true)
		return style.Render(line)
	}
	return line
}

func (i *Info) SetSize(width, height int) {
	i.width = width
	i.height = height
}

func (i *Info) SetFocused(focused bool) {
	i.focused = focused
}
