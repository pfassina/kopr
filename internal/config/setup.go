package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SetupResult is returned by RunSetup.
type SetupResult struct {
	VaultPath string
	Cancelled bool
}

type setupModel struct {
	input textinput.Model
	err   string
	done  bool
	quit  bool
}

func newSetupModel() setupModel {
	ti := textinput.New()
	ti.Placeholder = "~/notes"
	ti.CharLimit = 256
	ti.Width = 50
	ti.Focus()

	return setupModel{input: ti}
}

func (m setupModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m setupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			path := m.input.Value()
			if path == "" {
				path = "~/notes"
			}
			expanded := ExpandHome(path)

			if err := validateVaultPath(expanded); err != nil {
				m.err = err.Error()
				return m, nil
			}

			m.input.SetValue(path)
			m.done = true
			return m, tea.Quit

		case "esc", "ctrl+c":
			m.quit = true
			return m, tea.Quit
		}
	}

	m.err = ""
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m setupModel) View() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("212")).
		Render("Welcome to Kopr")

	var s string
	s += "\n " + title + "\n\n"
	s += " Enter your vault path:\n\n"
	s += "   " + m.input.View() + "\n\n"

	if m.err != "" {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		s += " " + errStyle.Render(m.err) + "\n\n"
	}

	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	s += " " + dim.Render("Press Enter to confirm, Esc to cancel") + "\n"

	return s
}

// validateVaultPath checks that a path is usable as a vault directory.
func validateVaultPath(path string) error {
	info, err := os.Stat(path)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("%s exists but is not a directory", path)
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}

	// Check that the parent directory exists or can be created.
	parent := filepath.Dir(path)
	pinfo, err := os.Stat(parent)
	if err != nil {
		return fmt.Errorf("parent directory %s does not exist", parent)
	}
	if !pinfo.IsDir() {
		return fmt.Errorf("%s is not a directory", parent)
	}
	return nil
}

// RunSetup runs the first-run TUI prompt and returns the chosen vault path.
func RunSetup() (SetupResult, error) {
	m := newSetupModel()
	p := tea.NewProgram(m)
	final, err := p.Run()
	if err != nil {
		return SetupResult{}, err
	}

	fm, ok := final.(setupModel)
	if !ok {
		return SetupResult{}, fmt.Errorf("unexpected model type from setup wizard")
	}
	if fm.quit {
		return SetupResult{Cancelled: true}, nil
	}

	path := fm.input.Value()
	if path == "" {
		path = "~/notes"
	}
	expanded := ExpandHome(path)

	if err := SaveFile(expanded); err != nil {
		return SetupResult{}, fmt.Errorf("saving config: %w", err)
	}

	return SetupResult{VaultPath: expanded}, nil
}
