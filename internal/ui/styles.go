package ui

import "github.com/charmbracelet/lipgloss"

var (
	PanelBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))

	StatusBar = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("252")).
			Padding(0, 1)

	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212"))

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	SelectedItem = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")).
			Bold(true)

	NormalItem = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	DimText = lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))
)
