package theme

import "github.com/charmbracelet/lipgloss"

// Theme defines a color palette used by all TUI panels.
// Panels hold a *Theme pointer so in-place mutations (e.g. after extracting
// colors from Neovim) are visible on the next View() call.
type Theme struct {
	Bg         lipgloss.Color
	Accent     lipgloss.Color
	Subtle     lipgloss.Color
	Text       lipgloss.Color
	Dim        lipgloss.Color
	Border     lipgloss.Color
	StatusBg   lipgloss.Color
	StatusFg   lipgloss.Color
	Error      lipgloss.Color
	NormalMode lipgloss.Color
	InsertMode lipgloss.Color
	VisualMode lipgloss.Color
	CmdMode    lipgloss.Color
}

// DefaultTheme returns the default color palette (catppuccin-inspired).
func DefaultTheme() Theme {
	return Theme{
		Bg:         lipgloss.Color("#1e1e2e"),
		Accent:     lipgloss.Color("#cba6f7"),
		Subtle:     lipgloss.Color("#6c7086"),
		Text:       lipgloss.Color("#cdd6f4"),
		Dim:        lipgloss.Color("#585b70"),
		Border:     lipgloss.Color("#45475a"),
		StatusBg:   lipgloss.Color("#313244"),
		StatusFg:   lipgloss.Color("#cdd6f4"),
		Error:      lipgloss.Color("#f38ba8"),
		NormalMode: lipgloss.Color("#89b4fa"),
		InsertMode: lipgloss.Color("#a6e3a1"),
		VisualMode: lipgloss.Color("#f9e2af"),
		CmdMode:    lipgloss.Color("#f38ba8"),
	}
}
