package editor

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/creack/pty"
)

type nvimPTY struct {
	cmd    *exec.Cmd
	file   *os.File
	socket string
}

func startNvim(width, height int, socketPath, vaultPath string) (*nvimPTY, error) {
	cmd := exec.Command("nvim",
		"--listen", socketPath,
	)
	cmd.Dir = vaultPath
	cmd.Env = append(os.Environ(), NvimEnv()...)

	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{
		Rows: uint16(height),
		Cols: uint16(width),
	})
	if err != nil {
		return nil, fmt.Errorf("start nvim: %w", err)
	}

	return &nvimPTY{
		cmd:    cmd,
		file:   ptmx,
		socket: socketPath,
	}, nil
}

func (n *nvimPTY) resize(width, height int) error {
	if err := pty.Setsize(n.file, &pty.Winsize{
		Rows: uint16(height),
		Cols: uint16(width),
	}); err != nil {
		return err
	}

	// Many TUI apps (including Neovim) rely on SIGWINCH in addition to TIOCSWINSZ
	// to reliably redraw after resizes.
	if n.cmd != nil && n.cmd.Process != nil {
		if err := syscall.Kill(n.cmd.Process.Pid, syscall.SIGWINCH); err != nil {
			// If the process is already gone, there's nothing to signal.
			if err != syscall.ESRCH {
				return fmt.Errorf("sigwinch nvim: %w", err)
			}
		}
	}

	return nil
}

func (n *nvimPTY) close() error {
	if err := n.file.Close(); err != nil {
		return err
	}
	err := n.cmd.Wait()
	if rmErr := os.Remove(n.socket); rmErr != nil && !os.IsNotExist(rmErr) {
		return rmErr
	}
	return err
}
