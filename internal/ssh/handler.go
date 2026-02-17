package ssh

import (
	"github.com/charmbracelet/ssh"
	tea "github.com/charmbracelet/bubbletea"
	bts "github.com/charmbracelet/wish/bubbletea"

	"github.com/pfassina/kopr/internal/app"
	"github.com/pfassina/kopr/internal/config"
)

// NewHandler returns a Bubble Tea handler for SSH sessions.
func NewHandler(cfg config.Config) bts.Handler {
	return func(sess ssh.Session) (tea.Model, []tea.ProgramOption) {
		a := app.New(cfg)

		opts := []tea.ProgramOption{
			tea.WithAltScreen(),
			tea.WithMouseCellMotion(),
		}
		opts = append(opts, bts.MakeOptions(sess)...)

		return &a, opts
	}
}
