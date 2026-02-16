package app

import (
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/yourusername/vimvault/internal/config"
)

type focusedPanel int

const (
	focusEditor focusedPanel = iota
	focusTree
	focusInfo
)

type App struct {
	cfg     config.Config
	width   int
	height  int
	focused focusedPanel
	ready   bool
}

func New(cfg config.Config) App {
	return App{
		cfg:     cfg,
		focused: focusEditor,
	}
}

func (a App) Init() tea.Cmd {
	return nil
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return a, tea.Quit
		}

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.ready = true
	}

	return a, nil
}

func (a App) View() string {
	if !a.ready {
		return "Loading..."
	}

	placeholder := lipgloss.Place(
		a.width, a.height,
		lipgloss.Center, lipgloss.Center,
		lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")).
			Bold(true).
			Render("VimVault")+"\n"+
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("245")).
				Render("Terminal-first knowledge management"),
	)

	return placeholder
}
