package app

import "github.com/charmbracelet/lipgloss"

// Theme defines a color palette.
type Theme struct {
	Name       string
	Accent     lipgloss.Color
	Subtle     lipgloss.Color
	Text       lipgloss.Color
	Dim        lipgloss.Color
	Border     lipgloss.Color
	StatusBg   lipgloss.Color
	StatusFg   lipgloss.Color
	NormalMode lipgloss.Color
	InsertMode lipgloss.Color
	VisualMode lipgloss.Color
	CmdMode    lipgloss.Color
}

var themes = map[string]Theme{
	"catppuccin": {
		Name:       "catppuccin",
		Accent:     lipgloss.Color("#cba6f7"),
		Subtle:     lipgloss.Color("#6c7086"),
		Text:       lipgloss.Color("#cdd6f4"),
		Dim:        lipgloss.Color("#585b70"),
		Border:     lipgloss.Color("#45475a"),
		StatusBg:   lipgloss.Color("#313244"),
		StatusFg:   lipgloss.Color("#cdd6f4"),
		NormalMode: lipgloss.Color("#89b4fa"),
		InsertMode: lipgloss.Color("#a6e3a1"),
		VisualMode: lipgloss.Color("#f9e2af"),
		CmdMode:    lipgloss.Color("#f38ba8"),
	},
	"nord": {
		Name:       "nord",
		Accent:     lipgloss.Color("#88c0d0"),
		Subtle:     lipgloss.Color("#4c566a"),
		Text:       lipgloss.Color("#eceff4"),
		Dim:        lipgloss.Color("#434c5e"),
		Border:     lipgloss.Color("#3b4252"),
		StatusBg:   lipgloss.Color("#3b4252"),
		StatusFg:   lipgloss.Color("#eceff4"),
		NormalMode: lipgloss.Color("#81a1c1"),
		InsertMode: lipgloss.Color("#a3be8c"),
		VisualMode: lipgloss.Color("#ebcb8b"),
		CmdMode:    lipgloss.Color("#bf616a"),
	},
	"gruvbox": {
		Name:       "gruvbox",
		Accent:     lipgloss.Color("#d79921"),
		Subtle:     lipgloss.Color("#665c54"),
		Text:       lipgloss.Color("#ebdbb2"),
		Dim:        lipgloss.Color("#504945"),
		Border:     lipgloss.Color("#3c3836"),
		StatusBg:   lipgloss.Color("#3c3836"),
		StatusFg:   lipgloss.Color("#ebdbb2"),
		NormalMode: lipgloss.Color("#83a598"),
		InsertMode: lipgloss.Color("#b8bb26"),
		VisualMode: lipgloss.Color("#fabd2f"),
		CmdMode:    lipgloss.Color("#fb4934"),
	},
	"tokyo-night": {
		Name:       "tokyo-night",
		Accent:     lipgloss.Color("#7aa2f7"),
		Subtle:     lipgloss.Color("#565f89"),
		Text:       lipgloss.Color("#c0caf5"),
		Dim:        lipgloss.Color("#414868"),
		Border:     lipgloss.Color("#292e42"),
		StatusBg:   lipgloss.Color("#1f2335"),
		StatusFg:   lipgloss.Color("#c0caf5"),
		NormalMode: lipgloss.Color("#7aa2f7"),
		InsertMode: lipgloss.Color("#9ece6a"),
		VisualMode: lipgloss.Color("#e0af68"),
		CmdMode:    lipgloss.Color("#f7768e"),
	},
}

// GetTheme returns a theme by name, defaulting to catppuccin.
func GetTheme(name string) Theme {
	if t, ok := themes[name]; ok {
		return t
	}
	return themes["catppuccin"]
}
