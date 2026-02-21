package theme

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestDefaultTheme(t *testing.T) {
	th := DefaultTheme()

	// Verify all fields are populated (non-empty).
	fields := []struct {
		name  string
		color lipgloss.Color
	}{
		{"Accent", th.Accent},
		{"Subtle", th.Subtle},
		{"Text", th.Text},
		{"Dim", th.Dim},
		{"Border", th.Border},
		{"StatusBg", th.StatusBg},
		{"StatusFg", th.StatusFg},
		{"Error", th.Error},
		{"NormalMode", th.NormalMode},
		{"InsertMode", th.InsertMode},
		{"VisualMode", th.VisualMode},
		{"CmdMode", th.CmdMode},
	}

	for _, f := range fields {
		if string(f.color) == "" {
			t.Errorf("DefaultTheme().%s is empty", f.name)
		}
	}
}
