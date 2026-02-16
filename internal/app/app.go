package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/yourusername/vimvault/internal/config"
	"github.com/yourusername/vimvault/internal/editor"
)

type focusedPanel int

const (
	focusEditor focusedPanel = iota
	focusTree
	focusInfo
)

type App struct {
	cfg     config.Config
	editor  editor.Editor
	width   int
	height  int
	focused focusedPanel
	ready   bool
}

func New(cfg config.Config) App {
	return App{
		cfg:     cfg,
		editor:  editor.New(cfg.VaultPath),
		focused: focusEditor,
	}
}

func (a App) Init() tea.Cmd {
	return a.editor.Init()
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			a.editor.Close()
			return a, tea.Quit
		}
	}

	// Forward all messages to the editor
	var cmd tea.Cmd
	a.editor, cmd = a.editor.Update(msg)

	return a, cmd
}

func (a App) View() string {
	if !a.ready {
		// Check if we got a size yet via the editor
		return a.editor.View()
	}
	return a.editor.View()
}
