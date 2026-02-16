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

type rpcConnectedMsg struct{}

type editorErrorMsg struct{ err error }

type ModeChangedMsg struct {
	Mode NvimMode
}

// Editor is a Bubble Tea model that embeds Neovim in a PTY
// and renders it via a VT emulator, with RPC for programmatic control.
type Editor struct {
	width      int
	height     int
	vaultPath  string
	socketPath string
	nvim       *nvimPTY
	rpc        *RPC
	screen     *vtScreen
	started    bool
	mode       NvimMode
	err        error
	program    *tea.Program // set externally for RPC event delivery
}

func New(vaultPath string) Editor {
	return Editor{
		vaultPath: vaultPath,
		mode:      ModeNormal,
	}
}

func (e *Editor) SetProgram(p *tea.Program) {
	e.program = p
}

func (e Editor) Init() tea.Cmd {
	return nil
}

// start begins the Neovim process.
func (e *Editor) start() tea.Cmd {
	return func() tea.Msg {
		e.socketPath = fmt.Sprintf("/tmp/vimvault-%d.sock", os.Getpid())
		os.Remove(e.socketPath)

		nvim, err := startNvim(e.width, e.height, e.socketPath, e.vaultPath)
		if err != nil {
			return editorErrorMsg{err}
		}
		e.nvim = nvim
		e.screen = newVTScreen(e.width, e.height)
		return editorStartedMsg{}
	}
}

// connectRPC establishes the RPC connection to Neovim.
func (e *Editor) connectRPC() tea.Cmd {
	return func() tea.Msg {
		rpc, err := ConnectRPC(e.socketPath, func(mode NvimMode) {
			if e.program != nil {
				e.program.Send(ModeChangedMsg{Mode: mode})
			}
		})
		if err != nil {
			return editorErrorMsg{err}
		}
		e.rpc = rpc
		return rpcConnectedMsg{}
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
		// Start both PTY reading and RPC connection in parallel
		return e, tea.Batch(e.waitForOutput, e.connectRPC())

	case rpcConnectedMsg:
		return e, nil

	case editorErrorMsg:
		e.err = msg.err
		return e, nil

	case ModeChangedMsg:
		e.mode = msg.Mode
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

// Mode returns the current Neovim mode.
func (e Editor) Mode() NvimMode {
	return e.mode
}

// RPC returns the RPC connection for programmatic Neovim control.
func (e Editor) GetRPC() *RPC {
	return e.rpc
}

// OpenFile opens a file in the editor via RPC.
func (e *Editor) OpenFile(path string) error {
	if e.rpc == nil {
		return fmt.Errorf("RPC not connected")
	}
	return e.rpc.OpenFile(path)
}

func (e *Editor) Close() {
	if e.rpc != nil {
		e.rpc.Close()
	}
	if e.screen != nil {
		e.screen.close()
	}
	if e.nvim != nil {
		e.nvim.close()
	}
}
