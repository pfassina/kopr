package ssh

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"
	bts "github.com/charmbracelet/wish/bubbletea"

	"github.com/pfassina/kopr/internal/app"
	"github.com/pfassina/kopr/internal/config"
)

// NewHandler returns a Bubble Tea handler for SSH sessions.
func NewHandler(cfg config.Config) bts.Handler {
	return func(sess ssh.Session) (tea.Model, []tea.ProgramOption) {
		a := app.New(cfg)
		a.SetOutput(sess)

		opts := []tea.ProgramOption{
			tea.WithAltScreen(),
			tea.WithMouseCellMotion(),
		}
		opts = append(opts, bts.MakeOptions(sess)...)

		return &a, opts
	}
}
