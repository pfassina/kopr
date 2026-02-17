package editor

import (
	"fmt"
	"os"
	"os/exec"

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
	return pty.Setsize(n.file, &pty.Winsize{
		Rows: uint16(height),
		Cols: uint16(width),
	})
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
