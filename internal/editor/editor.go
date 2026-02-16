package editor

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// Messages
type vtOutputMsg []byte

type vtClosedMsg struct{ err error }

type editorStartedMsg struct{}

type editorErrorMsg struct{ err error }

// Editor is a Bubble Tea model that embeds Neovim in a PTY
// and renders it via a VT emulator.
type Editor struct {
	width     int
	height    int
	vaultPath string
	nvim      *nvimPTY
	screen    *vtScreen
	started   bool
	err       error
}

func New(vaultPath string) Editor {
	return Editor{
		vaultPath: vaultPath,
	}
}

func (e Editor) Init() tea.Cmd {
	return nil
}

// Start begins the Neovim process. Call this after the first WindowSizeMsg.
func (e *Editor) start() tea.Cmd {
	return func() tea.Msg {
		socketPath := fmt.Sprintf("/tmp/vimvault-%d.sock", os.Getpid())
		os.Remove(socketPath) // clean up any stale socket

		nvim, err := startNvim(e.width, e.height, socketPath, e.vaultPath)
		if err != nil {
			return editorErrorMsg{err}
		}
		e.nvim = nvim
		e.screen = newVTScreen(e.width, e.height)
		return editorStartedMsg{}
	}
}

// waitForOutput reads from the PTY and returns the output as a message.
func (e *Editor) waitForOutput() tea.Msg {
	buf := make([]byte, 32*1024)
	n, err := e.nvim.file.Read(buf)
	if err != nil {
		return vtClosedMsg{err}
	}
	return vtOutputMsg(buf[:n])
}

func (e Editor) Update(msg tea.Msg) (Editor, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		e.width = msg.Width
		e.height = msg.Height
		if !e.started {
			e.started = true
			return e, e.start()
		}
		if e.nvim != nil {
			e.nvim.resize(e.width, e.height)
			e.screen.resize(e.width, e.height)
		}
		return e, nil

	case editorStartedMsg:
		return e, e.waitForOutput

	case editorErrorMsg:
		e.err = msg.err
		return e, nil

	case vtOutputMsg:
		if e.screen != nil {
			e.screen.write([]byte(msg))
		}
		return e, e.waitForOutput

	case vtClosedMsg:
		return e, tea.Quit

	case tea.KeyMsg:
		if e.nvim == nil {
			return e, nil
		}
		raw := keyMsgToBytes(msg)
		if raw != nil {
			e.nvim.file.Write(raw)
		}
		return e, nil
	}

	return e, nil
}

func (e Editor) View() string {
	if e.err != nil {
		return fmt.Sprintf("Editor error: %v", e.err)
	}
	if e.screen == nil {
		return "Starting Neovim..."
	}
	return e.screen.render()
}

func (e *Editor) Close() {
	if e.screen != nil {
		e.screen.close()
	}
	if e.nvim != nil {
		e.nvim.close()
	}
}
