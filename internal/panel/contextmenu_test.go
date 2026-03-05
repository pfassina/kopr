package panel

import (
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/pfassina/kopr/internal/theme"
)

func TestContextMenuDimensions(t *testing.T) {
	th := theme.DefaultTheme()
	cm := NewContextMenu()
	cm.SetTheme(&th)

	items := []ContextMenuItem{
		{Label: "Cut", Action: "cut"},
		{Label: "Copy", Action: "copy"},
		{Label: "Paste", Action: "paste"},
		{Label: "Select All", Action: "select-all"},
	}

	cm.Show(10, 10, items)

	// Verify dimensions computed in Show() match actual rendered output
	rendered := cm.View()
	renderedW := lipgloss.Width(rendered)
	renderedH := lipgloss.Height(rendered)

	w, h := cm.Dimensions()
	if w != renderedW {
		t.Errorf("width: Show() computed %d, View() rendered %d", w, renderedW)
	}
	if h != renderedH {
		t.Errorf("height: Show() computed %d, View() rendered %d", h, renderedH)
	}
}

func TestContextMenuDimensionsWithDisabledItems(t *testing.T) {
	th := theme.DefaultTheme()
	cm := NewContextMenu()
	cm.SetTheme(&th)

	items := []ContextMenuItem{
		{Label: "Cut", Action: "cut", Disabled: true},
		{Label: "Copy", Action: "copy", Disabled: true},
		{Label: "Paste", Action: "paste"},
		{Label: "Delete", Action: "delete", Disabled: true},
		{Label: "Select All", Action: "select-all"},
	}

	cm.Show(0, 0, items)

	rendered := cm.View()
	renderedW := lipgloss.Width(rendered)
	renderedH := lipgloss.Height(rendered)

	w, h := cm.Dimensions()
	if w != renderedW {
		t.Errorf("width: Show() computed %d, View() rendered %d", w, renderedW)
	}
	if h != renderedH {
		t.Errorf("height: Show() computed %d, View() rendered %d", h, renderedH)
	}
}
