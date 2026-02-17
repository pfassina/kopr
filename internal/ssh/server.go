package ssh

import (
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	bts "github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"

	"github.com/pfassina/kopr/internal/config"
)

// Server wraps a Wish SSH server.
type Server struct {
	server *ssh.Server
	cfg    config.Config
}

// New creates a new SSH server.
func New(cfg config.Config) (*Server, error) {
	hostKeyPath := filepath.Join(cfg.VaultPath, ".kopr", "ssh_host_key")

	s, err := wish.NewServer(
		wish.WithAddress(cfg.Listen),
		wish.WithHostKeyPath(hostKeyPath),
		wish.WithMiddleware(
			logging.Middleware(),
			activeterm.Middleware(),
			bts.Middleware(NewHandler(cfg)),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create ssh server: %w", err)
	}

	return &Server{server: s, cfg: cfg}, nil
}

// ListenAndServe starts the SSH server.
func (s *Server) ListenAndServe() error {
	return s.server.ListenAndServe()
}

// Close stops the SSH server.
func (s *Server) Close() error {
	return s.server.Close()
}
