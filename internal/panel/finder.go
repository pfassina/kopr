package panel

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pfassina/kopr/internal/theme"
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

// FinderCreateRequestMsg is sent when the user requests to create a new note
// from the current finder query (typically when there are no results).
//
// The app is expected to show a confirmation prompt before actually creating
// anything.
type FinderCreateRequestMsg struct {
	Name string
}

// FinderClosedMsg is sent when the finder is dismissed.
type FinderClosedMsg struct{}

// SearchFunc is called to get results for a query.
type SearchFunc func(query string) []FinderItem

// PreviewFunc returns the content of a note for preview.
type PreviewFunc func(path string) string

// Finder is a fuzzy finder overlay with a preview pane.
type Finder struct {
	input         textinput.Model
	items         []FinderItem
	cursor        int
	width         int
	height        int
	visible       bool
	searchFn      SearchFunc
	previewFn     PreviewFunc
	preview       string
	previewScroll int
	theme         *theme.Theme
}

// SetTheme sets the color theme for the finder panel.
func (f *Finder) SetTheme(th *theme.Theme) { f.theme = th }

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

func (f *Finder) SetPreviewFunc(fn PreviewFunc) {
	f.previewFn = fn
}

func (f *Finder) Show() {
	f.visible = true
	f.input.SetValue("")
	f.cursor = 0
	f.previewScroll = 0
	f.input.Focus()
	if f.searchFn != nil {
		f.items = f.searchFn("")
	}
	f.updatePreview()
}

func (f *Finder) Hide() {
	f.visible = false
	f.input.Blur()
}

func (f Finder) Visible() bool {
	return f.visible
}

func (f *Finder) updatePreview() {
	if f.previewFn != nil && f.cursor < len(f.items) {
		f.preview = f.previewFn(f.items[f.cursor].Path)
	} else {
		f.preview = ""
	}
	f.previewScroll = 0
}

// previewHeight returns the number of visible lines in the preview pane.
func (f Finder) previewHeight() int {
	return max(f.overlayHeight()-4, 3) // border + title + input + blank line
}

// overlayHeight returns the total height of the overlay box.
func (f Finder) overlayHeight() int {
	return max(f.height*3/4, 12)
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
			// No results — request note creation (the app will confirm).
			query := strings.TrimSpace(f.input.Value())
			if query != "" {
				return f, func() tea.Msg {
					return FinderCreateRequestMsg{Name: query}
				}
			}
			return f, nil

		case "up", "ctrl+p", "ctrl+k":
			if f.cursor > 0 {
				f.cursor--
				f.updatePreview()
			}
			return f, nil

		case "down", "ctrl+n", "ctrl+j":
			if f.cursor < len(f.items)-1 {
				f.cursor++
				f.updatePreview()
			}
			return f, nil

		case "ctrl+d":
			f.scrollPreview(f.previewHeight() / 2)
			return f, nil

		case "ctrl+u":
			f.scrollPreview(-f.previewHeight() / 2)
			return f, nil
		}

	case tea.MouseMsg:
		if msg.Button == tea.MouseButtonWheelUp {
			f.scrollPreview(-3)
			return f, nil
		}
		if msg.Button == tea.MouseButtonWheelDown {
			f.scrollPreview(3)
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
		f.updatePreview()
	}

	return f, cmd
}

func (f *Finder) scrollPreview(delta int) {
	lines := strings.Split(f.preview, "\n")
	maxScroll := max(len(lines)-f.previewHeight(), 0)
	f.previewScroll = max(min(f.previewScroll+delta, maxScroll), 0)
}

func (f Finder) View() string {
	if !f.visible {
		return ""
	}

	th := f.theme

	overlayWidth := min(max(f.width*4/5, 60), f.width-4)
	overlayH := f.overlayHeight()

	// Inner width accounts for outer border (2) + padding (2)
	innerWidth := overlayWidth - 4
	// Split: left column ~35%, right column ~65%
	leftWidth := max(innerWidth*35/100, 20)
	rightWidth := innerWidth - leftWidth - 1 // -1 for the separator

	contentHeight := overlayH - 4 // border top/bottom + title + input + blank

	// --- Left column: search input + results ---
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(th.Accent)

	f.input.Width = leftWidth - 2

	var leftLines []string
	leftLines = append(leftLines, titleStyle.Render("Find Note"))
	leftLines = append(leftLines, f.input.View())
	leftLines = append(leftLines, "")

	maxResults := min(max(contentHeight-3, 3), len(f.items)) // title + input + blank

	if len(f.items) == 0 {
		dim := lipgloss.NewStyle().Foreground(th.Dim)
		leftLines = append(leftLines, dim.Render("No results"))

		query := strings.TrimSpace(f.input.Value())
		if query != "" {
			leftLines = append(leftLines, "")
			leftLines = append(leftLines, dim.Render(fmt.Sprintf("Enter: create %q", query)))
			leftLines = append(leftLines, dim.Render("Esc: cancel"))
		}
	} else {
		for i := range maxResults {
			item := f.items[i]
			prefix := "  "
			style := lipgloss.NewStyle().Foreground(th.Text)

			if i == f.cursor {
				prefix = "> "
				style = lipgloss.NewStyle().Foreground(th.Accent).Bold(true)
			}

			title := item.Title
			if title == "" {
				title = item.Path
			}

			line := fmt.Sprintf("%s%s", prefix, title)

			// Truncate to left column width
			if lipgloss.Width(line) > leftWidth {
				line = line[:leftWidth-3] + "..."
			}

			leftLines = append(leftLines, style.Render(line))
		}

		if len(f.items) > maxResults {
			dim := lipgloss.NewStyle().Foreground(th.Dim)
			leftLines = append(leftLines, dim.Render(fmt.Sprintf("  +%d more", len(f.items)-maxResults)))
		}
	}

	// Pad left column to full height
	for len(leftLines) < contentHeight {
		leftLines = append(leftLines, "")
	}

	leftCol := lipgloss.NewStyle().
		Width(leftWidth).
		Render(strings.Join(leftLines, "\n"))

	// --- Right column: preview ---
	dim := lipgloss.NewStyle().Foreground(th.Dim)

	var rightLines []string
	if f.preview == "" {
		rightLines = append(rightLines, dim.Render("No preview"))
	} else {
		allLines := strings.Split(f.preview, "\n")
		end := min(f.previewScroll+contentHeight, len(allLines))
		start := min(f.previewScroll, len(allLines))
		visible := allLines[start:end]
		for _, l := range visible {
			// Truncate long lines
			if lipgloss.Width(l) > rightWidth {
				l = l[:rightWidth-1] + "…"
			}
			rightLines = append(rightLines, dim.Render(l))
		}
	}

	// Pad right column to full height
	for len(rightLines) < contentHeight {
		rightLines = append(rightLines, "")
	}

	rightCol := lipgloss.NewStyle().
		Width(rightWidth).
		BorderLeft(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(th.Border).
		PaddingLeft(1).
		Render(strings.Join(rightLines, "\n"))

	content := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, rightCol)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(th.Accent).
		Padding(0, 1).
		Width(innerWidth)

	return borderStyle.Render(content)
}

func (f *Finder) SetSize(width, height int) {
	f.width = width
	f.height = height
}
