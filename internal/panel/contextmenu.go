package panel

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pfassina/kopr/internal/theme"
)

// ContextMenuItem represents a single menu entry.
type ContextMenuItem struct {
	Label    string
	Action   string // identifier (e.g., "cut", "copy", "paste")
	Disabled bool
}

// ContextMenuResultMsg is sent when a context menu item is selected.
type ContextMenuResultMsg struct {
	Action string
}

// ContextMenuClosedMsg is sent when the context menu is dismissed.
type ContextMenuClosedMsg struct{}

// ContextMenu is a right-click popup menu.
type ContextMenu struct {
	items      []ContextMenuItem
	cursor     int
	x          int // screen position (column)
	y          int // screen position (row)
	visible    bool
	theme      *theme.Theme
	width      int  // computed in Show() from item labels
	height     int  // computed in Show() from item count
	mouseMoved bool // true once the cursor changes via mouse motion (for hold-and-drag)
}

// NewContextMenu creates a new context menu.
func NewContextMenu() ContextMenu {
	return ContextMenu{}
}

// SetTheme sets the color theme.
func (c *ContextMenu) SetTheme(th *theme.Theme) { c.theme = th }

// Show displays the context menu at the given screen position.
func (c *ContextMenu) Show(x, y int, items []ContextMenuItem) {
	c.x = x
	c.y = y
	c.items = items
	c.visible = true
	c.mouseMoved = false

	// Start cursor on the first non-disabled item
	c.cursor = 0
	for i, item := range items {
		if !item.Disabled {
			c.cursor = i
			break
		}
	}

	// Pre-compute dimensions to match View() output.
	// View renders: border(1) + padding(1) + content + padding(1) + border(1)
	// Height: border(1) + items + border(1)
	maxLabelW := 0
	for _, item := range items {
		w := len(item.Label) + 2 // "> " or "  " prefix
		if w > maxLabelW {
			maxLabelW = w
		}
	}
	c.width = maxLabelW + 4  // 2 border + 2 padding
	c.height = len(items) + 2 // 2 border
}

// Hide dismisses the context menu.
func (c *ContextMenu) Hide() {
	c.visible = false
}

// Visible reports whether the context menu is showing.
func (c ContextMenu) Visible() bool {
	return c.visible
}

// Position returns the screen position for overlay rendering.
func (c ContextMenu) Position() (x, y int) {
	return c.x, c.y
}

// Dimensions returns the width and height of the menu.
func (c ContextMenu) Dimensions() (w, h int) {
	return c.width, c.height
}

// Update handles key and mouse events for the context menu.
func (c ContextMenu) Update(msg tea.Msg) (ContextMenu, tea.Cmd) {
	if !c.visible {
		return c, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			c.visible = false
			return c, func() tea.Msg { return ContextMenuClosedMsg{} }
		case "j", "down":
			c.moveCursorDown()
		case "k", "up":
			c.moveCursorUp()
		case "enter":
			if c.cursor < len(c.items) && !c.items[c.cursor].Disabled {
				action := c.items[c.cursor].Action
				c.visible = false
				return c, func() tea.Msg {
					return ContextMenuResultMsg{Action: action}
				}
			}
		}

	case tea.MouseMsg:
		// The menu is rendered with a border (1 char top/bottom) and
		// padding (1 char left/right). Item rows start at local row 1
		// (after top border) and occupy one row each.
		// msg.X and msg.Y are menu-local coordinates set by the app layer.
		itemRow := msg.Y - 1 // subtract top border

		switch msg.Action {
		case tea.MouseActionMotion:
			if itemRow >= 0 && itemRow < len(c.items) && !c.items[itemRow].Disabled {
				if c.cursor != itemRow {
					c.mouseMoved = true
				}
				c.cursor = itemRow
			}
		case tea.MouseActionPress:
			if msg.Button == tea.MouseButtonLeft && itemRow >= 0 && itemRow < len(c.items) && !c.items[itemRow].Disabled {
				action := c.items[itemRow].Action
				c.visible = false
				return c, func() tea.Msg {
					return ContextMenuResultMsg{Action: action}
				}
			}
		case tea.MouseActionRelease:
			// Right-click hold-and-drag: if the user moved the cursor via
			// mouse motion (button held) and then released, select the item.
			if c.mouseMoved && itemRow >= 0 && itemRow < len(c.items) && !c.items[itemRow].Disabled {
				action := c.items[itemRow].Action
				c.visible = false
				return c, func() tea.Msg {
					return ContextMenuResultMsg{Action: action}
				}
			}
		}
	}

	return c, nil
}

func (c *ContextMenu) moveCursorDown() {
	for i := c.cursor + 1; i < len(c.items); i++ {
		if !c.items[i].Disabled {
			c.cursor = i
			return
		}
	}
}

func (c *ContextMenu) moveCursorUp() {
	for i := c.cursor - 1; i >= 0; i-- {
		if !c.items[i].Disabled {
			c.cursor = i
			return
		}
	}
}

// View renders the context menu.
func (c ContextMenu) View() string {
	if !c.visible || len(c.items) == 0 {
		return ""
	}

	th := c.theme

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(th.Border).
		Padding(0, 1)

	var lines []string
	for i, item := range c.items {
		label := item.Label

		var style lipgloss.Style
		switch {
		case item.Disabled:
			style = lipgloss.NewStyle().Foreground(th.Dim)
			label = "  " + label
		case i == c.cursor:
			style = lipgloss.NewStyle().
				Foreground(th.Accent).
				Bold(true)
			label = "> " + label
		default:
			style = lipgloss.NewStyle().Foreground(th.Text)
			label = "  " + label
		}

		lines = append(lines, style.Render(label))
	}

	content := strings.Join(lines, "\n")
	return borderStyle.Render(content)
}
