package panel

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pfassina/kopr/internal/theme"
)

// DiagnosticsGotoMsg is emitted when the user activates a diagnostic row.
type DiagnosticsGotoMsg struct {
	Line int // 1-based
	Col  int // 0-based
}

// DiagnosticsClosedMsg is emitted when the user dismisses the panel.
type DiagnosticsClosedMsg struct{}

// DiagnosticItem is a single row in the diagnostics panel.
type DiagnosticItem struct {
	Line     int    // 1-based
	Col      int    // 0-based
	Severity string // "error" or "warning"
	Message  string
}

// maxDiagnosticsHeight caps the panel height (rows incl. header).
const maxDiagnosticsHeight = 8

// Diagnostics is a quickfix-style list of lint findings for the current note.
type Diagnostics struct {
	items   []DiagnosticItem
	cursor  int
	offset  int
	width   int
	height  int
	visible bool
	focused bool
	theme   *theme.Theme
}

func NewDiagnostics() Diagnostics {
	return Diagnostics{}
}

// SetTheme sets the color theme for the panel.
func (d *Diagnostics) SetTheme(th *theme.Theme) { d.theme = th }

// SetItems replaces the diagnostic rows, clamping cursor and scroll.
func (d *Diagnostics) SetItems(items []DiagnosticItem) {
	d.items = items
	if d.cursor >= len(items) {
		d.cursor = len(items) - 1
	}
	if d.cursor < 0 {
		d.cursor = 0
	}
	if d.offset > d.cursor {
		d.offset = d.cursor
	}
}

// Items returns the current diagnostic rows.
func (d Diagnostics) Items() []DiagnosticItem { return d.items }

func (d *Diagnostics) Show()             { d.visible = true }
func (d *Diagnostics) Hide()             { d.visible = false }
func (d Diagnostics) Visible() bool      { return d.visible }
func (d *Diagnostics) SetFocused(f bool) { d.focused = f }

func (d *Diagnostics) SetSize(width, height int) {
	d.width = width
	d.height = height
	d.scrollIntoView()
}

// PreferredHeight returns the desired panel height (header + rows), capped.
func (d Diagnostics) PreferredHeight() int {
	rows := len(d.items)
	if rows == 0 {
		rows = 1 // "No diagnostics" placeholder
	}
	h := rows + 1 // +1 for the header
	if h > maxDiagnosticsHeight {
		h = maxDiagnosticsHeight
	}
	return h
}

func (d Diagnostics) viewHeight() int {
	return max(d.height-1, 1) // -1 for the header row
}

func (d *Diagnostics) scrollIntoView() {
	if d.cursor < d.offset {
		d.offset = d.cursor
	}
	if d.cursor-d.offset >= d.viewHeight() {
		d.offset = d.cursor - d.viewHeight() + 1
	}
	if d.offset < 0 {
		d.offset = 0
	}
}

func (d Diagnostics) Update(msg tea.Msg) (Diagnostics, tea.Cmd) {
	if !d.focused {
		return d, nil
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return d, nil
	}

	switch keyMsg.String() {
	case "j", "down":
		if d.cursor < len(d.items)-1 {
			d.cursor++
			d.scrollIntoView()
		}
	case "k", "up":
		if d.cursor > 0 {
			d.cursor--
			d.scrollIntoView()
		}
	case "g":
		d.cursor = 0
		d.offset = 0
	case "G":
		if len(d.items) > 0 {
			d.cursor = len(d.items) - 1
			d.scrollIntoView()
		}
	case "enter":
		if d.cursor < len(d.items) {
			item := d.items[d.cursor]
			return d, func() tea.Msg {
				return DiagnosticsGotoMsg{Line: item.Line, Col: item.Col}
			}
		}
	case "esc", "q":
		return d, func() tea.Msg {
			return DiagnosticsClosedMsg{}
		}
	}

	return d, nil
}

func (d Diagnostics) View() string {
	if !d.visible || d.width == 0 || d.height == 0 {
		return ""
	}

	th := d.theme
	var b strings.Builder

	b.WriteString(d.renderHeader())

	if len(d.items) == 0 {
		dim := lipgloss.NewStyle().Foreground(th.Dim).Padding(0, 1)
		b.WriteByte('\n')
		b.WriteString(dim.Render("No diagnostics"))
		return b.String()
	}

	end := min(d.offset+d.viewHeight(), len(d.items))
	for idx := d.offset; idx < end; idx++ {
		b.WriteByte('\n')
		b.WriteString(d.renderItem(d.items[idx], idx == d.cursor))
	}

	return b.String()
}

func (d Diagnostics) renderHeader() string {
	th := d.theme

	errors, warnings := 0, 0
	for _, item := range d.items {
		if item.Severity == "error" {
			errors++
		} else {
			warnings++
		}
	}

	title := "Diagnostics"
	switch {
	case errors > 0 && warnings > 0:
		title = fmt.Sprintf("Diagnostics — %s, %s", pluralize(errors, "error"), pluralize(warnings, "warning"))
	case errors > 0:
		title = fmt.Sprintf("Diagnostics — %s", pluralize(errors, "error"))
	case warnings > 0:
		title = fmt.Sprintf("Diagnostics — %s", pluralize(warnings, "warning"))
	}

	style := lipgloss.NewStyle().Bold(true).Padding(0, 1)
	if d.focused {
		style = style.Foreground(th.Accent).Underline(true)
	} else {
		style = style.Foreground(th.Dim)
	}
	return style.Render(title)
}

func pluralize(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}

func (d Diagnostics) renderItem(item DiagnosticItem, selected bool) string {
	th := d.theme

	glyph := "E"
	glyphColor := th.Error
	if item.Severity == "warning" {
		glyph = "W"
		glyphColor = th.Warning
	}

	pos := fmt.Sprintf("%d:%d", item.Line, item.Col+1)
	text := fmt.Sprintf(" %-7s %s", pos, item.Message)

	maxW := d.width - 3 // glyph + space + right margin
	if maxW < 4 {
		maxW = 4
	}
	if len(text) > maxW {
		text = text[:maxW-3] + "..."
	}
	if len(text) < maxW {
		text += strings.Repeat(" ", maxW-len(text))
	}

	glyphStr := lipgloss.NewStyle().Foreground(glyphColor).Bold(true).Render(" " + glyph)
	if selected && d.focused {
		return glyphStr + lipgloss.NewStyle().Foreground(th.Accent2).Bold(true).Render(text)
	}
	return glyphStr + text
}
