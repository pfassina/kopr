package theme

import "github.com/charmbracelet/lipgloss"

// FromExtracted maps raw Neovim highlight group colors onto a Theme.
// The colors map uses highlight group names as keys and [fg, bg] hex strings
// as values (empty string means the group didn't define that attribute).
// Any field without a corresponding extracted color keeps the base value.
func FromExtracted(colors map[string][2]string, base Theme) Theme {
	t := base

	if c := bg(colors, "Normal"); isSet(c) {
		t.Bg = lipgloss.Color(c)
	}

	if c := fg(colors, "Normal"); isSet(c) {
		t.Text = lipgloss.Color(c)
	}

	// Accent: prefer Function, fall back to Keyword
	if c := fg(colors, "Function"); isSet(c) {
		t.Accent = lipgloss.Color(c)
	} else if c := fg(colors, "Keyword"); isSet(c) {
		t.Accent = lipgloss.Color(c)
	}

	if c := fg(colors, "Comment"); isSet(c) {
		t.Subtle = lipgloss.Color(c)
	}

	// Dim: prefer NonText, fall back to LineNr
	if c := fg(colors, "NonText"); isSet(c) {
		t.Dim = lipgloss.Color(c)
	} else if c := fg(colors, "LineNr"); isSet(c) {
		t.Dim = lipgloss.Color(c)
	}

	if c := fg(colors, "WinSeparator"); isSet(c) {
		t.Border = lipgloss.Color(c)
	}

	if c := bg(colors, "StatusLine"); isSet(c) {
		t.StatusBg = lipgloss.Color(c)
	}
	if c := fg(colors, "StatusLine"); isSet(c) {
		t.StatusFg = lipgloss.Color(c)
	}

	if c := fg(colors, "DiagnosticError"); isSet(c) {
		t.Error = lipgloss.Color(c)
	}

	// Mode colors derived from the palette
	t.NormalMode = t.Accent

	if c := fg(colors, "String"); isSet(c) {
		t.InsertMode = lipgloss.Color(c)
	}

	// Visual: prefer Visual bg, fall back to WarningMsg fg
	if c := bg(colors, "Visual"); isSet(c) {
		t.VisualMode = lipgloss.Color(c)
	} else if c := fg(colors, "WarningMsg"); isSet(c) {
		t.VisualMode = lipgloss.Color(c)
	}

	t.CmdMode = t.Error

	return t
}

// isSet returns true when the color string represents an explicitly set color.
// Neovim returns fg/bg = 0 for groups that inherit from the default, which
// intToHex converts to "#000000". We treat both empty and pure-black as unset.
func isSet(c string) bool {
	return c != "" && c != "#000000"
}

func fg(colors map[string][2]string, group string) string {
	if pair, ok := colors[group]; ok {
		return pair[0]
	}
	return ""
}

func bg(colors map[string][2]string, group string) string {
	if pair, ok := colors[group]; ok {
		return pair[1]
	}
	return ""
}
