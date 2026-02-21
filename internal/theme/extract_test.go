package theme

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestFromExtracted_OverridesBase(t *testing.T) {
	base := DefaultTheme()

	colors := map[string][2]string{
		"Normal":          {"#ffffff", "#1a1b26"},
		"Function":        {"#ff0000", ""},
		"Comment":         {"#888888", ""},
		"NonText":         {"#444444", ""},
		"WinSeparator":    {"#333333", ""},
		"StatusLine":      {"#aaaaaa", "#222222"},
		"DiagnosticError": {"#ff5555", ""},
		"String":          {"#00ff00", ""},
		"Visual":          {"", "#553399"},
	}

	th := FromExtracted(colors, base)

	tests := []struct {
		name string
		got  lipgloss.Color
		want string
	}{
		{"Bg", th.Bg, "#1a1b26"},
		{"Text", th.Text, "#ffffff"},
		{"Accent", th.Accent, "#ff0000"},
		{"Subtle", th.Subtle, "#888888"},
		{"Dim", th.Dim, "#444444"},
		{"Border", th.Border, "#333333"},
		{"StatusBg", th.StatusBg, "#222222"},
		{"StatusFg", th.StatusFg, "#aaaaaa"},
		{"Error", th.Error, "#ff5555"},
		{"NormalMode", th.NormalMode, "#ff0000"}, // derived from Accent
		{"InsertMode", th.InsertMode, "#00ff00"},
		{"VisualMode", th.VisualMode, "#553399"},
		{"CmdMode", th.CmdMode, "#ff5555"}, // derived from Error
	}

	for _, tt := range tests {
		if string(tt.got) != tt.want {
			t.Errorf("FromExtracted %s = %q, want %q", tt.name, string(tt.got), tt.want)
		}
	}
}

func TestFromExtracted_KeepsBaseWhenEmpty(t *testing.T) {
	base := DefaultTheme()
	th := FromExtracted(map[string][2]string{}, base)

	if th.Accent != base.Accent {
		t.Errorf("expected Accent to stay %q, got %q", string(base.Accent), string(th.Accent))
	}
	if th.Text != base.Text {
		t.Errorf("expected Text to stay %q, got %q", string(base.Text), string(th.Text))
	}
}

func TestFromExtracted_SkipsBlackAsUnset(t *testing.T) {
	base := DefaultTheme()

	// #000000 means "no explicit color" from Neovim's default colorscheme.
	// These should be ignored, preserving the base theme values.
	colors := map[string][2]string{
		"Normal":   {"#000000", ""},
		"Function": {"#000000", ""},
		"Comment":  {"#000000", ""},
	}
	th := FromExtracted(colors, base)

	if th.Text != base.Text {
		t.Errorf("expected Text to stay %q (base), got %q", string(base.Text), string(th.Text))
	}
	if th.Accent != base.Accent {
		t.Errorf("expected Accent to stay %q (base), got %q", string(base.Accent), string(th.Accent))
	}
	if th.Subtle != base.Subtle {
		t.Errorf("expected Subtle to stay %q (base), got %q", string(base.Subtle), string(th.Subtle))
	}
}

func TestFromExtracted_FallbackGroups(t *testing.T) {
	base := DefaultTheme()

	// Keyword as fallback for Function (Accent)
	colors := map[string][2]string{
		"Keyword": {"#abcdef", ""},
	}
	th := FromExtracted(colors, base)
	if string(th.Accent) != "#abcdef" {
		t.Errorf("expected Accent fallback to Keyword #abcdef, got %q", string(th.Accent))
	}

	// LineNr as fallback for NonText (Dim)
	colors2 := map[string][2]string{
		"LineNr": {"#112233", ""},
	}
	th2 := FromExtracted(colors2, base)
	if string(th2.Dim) != "#112233" {
		t.Errorf("expected Dim fallback to LineNr #112233, got %q", string(th2.Dim))
	}

	// WarningMsg fg as fallback for Visual bg (VisualMode)
	colors3 := map[string][2]string{
		"WarningMsg": {"#ffaa00", ""},
	}
	th3 := FromExtracted(colors3, base)
	if string(th3.VisualMode) != "#ffaa00" {
		t.Errorf("expected VisualMode fallback to WarningMsg #ffaa00, got %q", string(th3.VisualMode))
	}
}
