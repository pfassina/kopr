package editor

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Messages
type vtOutputMsg struct {
	data []byte
	pty  *nvimPTY
}

type vtClosedMsg struct{ err error }

type editorStartedMsg struct {
	nvim   *nvimPTY
	screen *vtScreen
	socket string
}

type rpcConnectedMsg struct {
	rpc *RPC
}

type editorErrorMsg struct{ err error }

type ModeChangedMsg struct {
	Mode NvimMode
}

// NoteClosedMsg is sent when neovim quit/close commands are intercepted.
type NoteClosedMsg struct {
	Save bool
}

// SaveUnnamedMsg is sent when :w is used on an unnamed buffer.
type SaveUnnamedMsg struct{}

// BufferWrittenMsg is sent when Neovim writes a buffer to disk.
// Path is the absolute path of the written file.
type BufferWrittenMsg struct {
	Path string
}

// FollowLinkMsg is sent when the user presses gf on a wiki link.
type FollowLinkMsg struct{}

// GoBackMsg is sent when the user presses gb to go back to the previous note.
type GoBackMsg struct{}

// Editor is a Bubble Tea model that embeds Neovim in a PTY
// and renders it via a VT emulator, with RPC for programmatic control.
type Editor struct {
	width       int
	height      int
	vaultPath   string
	socketPath  string
	profileMode ProfileMode
	nvim        *nvimPTY
	rpc         *RPC
	screen      *vtScreen
	started     bool
	mode        NvimMode
	err         error
	program     *tea.Program
	focused     bool
	showSplash  bool
}

func New(vaultPath string, profileMode ProfileMode) Editor {
	return Editor{
		vaultPath:   vaultPath,
		profileMode: profileMode,
		mode:        ModeNormal,
		focused:     true,
		showSplash:  true,
	}
}

func (e *Editor) SetProgram(p *tea.Program) {
	e.program = p
}

func (e Editor) Init() tea.Cmd {
	return nil
}

// start spawns Neovim and returns resources via message.
func (e Editor) start() tea.Cmd {
	width, height, vaultPath, profileMode := e.width, e.height, e.vaultPath, e.profileMode
	return func() tea.Msg {
		if err := EnsureProfile(profileMode); err != nil {
			return editorErrorMsg{fmt.Errorf("nvim profile: %w", err)}
		}

		socketPath := fmt.Sprintf("/tmp/kopr-%d.sock", os.Getpid())
		if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
			return editorErrorMsg{fmt.Errorf("remove socket %s: %w", socketPath, err)}
		}

		nvim, err := startNvim(width, height, socketPath, vaultPath)
		if err != nil {
			return editorErrorMsg{err}
		}
		screen := newVTScreen(width, height, nvim.file)
		return editorStartedMsg{nvim: nvim, screen: screen, socket: socketPath}
	}
}

// connectRPC dials the socket and returns the client via message.
func (e Editor) connectRPC(program *tea.Program) tea.Cmd {
	socketPath := e.socketPath
	return func() tea.Msg {
		rpc, err := ConnectRPC(socketPath, func(mode NvimMode) {
			if program != nil {
				program.Send(ModeChangedMsg{Mode: mode})
			}
		})
		if err != nil {
			return editorErrorMsg{err}
		}
		return rpcConnectedMsg{rpc: rpc}
	}
}

// waitForOutput reads from the PTY and returns the output as a message.
func waitForOutput(nvim *nvimPTY) tea.Cmd {
	return func() tea.Msg {
		buf := make([]byte, 32*1024)
		n, err := nvim.file.Read(buf)
		if err != nil {
			return vtClosedMsg{err}
		}
		return vtOutputMsg{data: buf[:n], pty: nvim}
	}
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
			if err := e.nvim.resize(e.width, e.height); err != nil {
				e.err = err
				return e, tea.Quit
			}
			if e.screen != nil {
				e.screen.resize(e.width, e.height)
			}
		}
		return e, nil

	case editorStartedMsg:
		e.nvim = msg.nvim
		e.screen = msg.screen
		e.socketPath = msg.socket
		return e, tea.Batch(waitForOutput(e.nvim), e.connectRPC(e.program))

	case rpcConnectedMsg:
		e.rpc = msg.rpc
		if e.program != nil {
			if err := e.rpc.SetupQuitSaveIntercept(e.program); err != nil {
				e.err = err
				return e, tea.Quit
			}
			if err := e.rpc.SetupSaveNotify(e.program); err != nil {
				e.err = err
				return e, tea.Quit
			}
			if err := e.rpc.SetupLinkNavigation(e.program); err != nil {
				e.err = err
				return e, tea.Quit
			}
		}
		// Ensure left gutter aligns buffer text with panel titles
		if err := e.rpc.ExecCommand("set foldcolumn=1"); err != nil {
			e.err = err
			return e, tea.Quit
		}
		// Load splash buffer so neovim starts in a clean state
		if err := e.rpc.LoadSplashBuffer(); err != nil {
			e.err = err
			return e, tea.Quit
		}
		return e, nil

	case editorErrorMsg:
		e.err = msg.err
		return e, nil

	case ModeChangedMsg:
		e.mode = msg.Mode
		return e, nil

	case vtOutputMsg:
		if e.screen != nil {
			if _, err := e.screen.write(msg.data); err != nil {
				e.err = err
				return e, tea.Quit
			}
		}
		return e, waitForOutput(e.nvim)

	case vtClosedMsg:
		return e, tea.Quit

	case tea.KeyMsg:
		if e.nvim == nil || e.showSplash {
			return e, nil
		}
		raw := keyMsgToBytes(msg)
		if raw != nil {
			if _, err := e.nvim.file.Write(raw); err != nil {
				e.err = err
				return e, tea.Quit
			}
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
	if e.showSplash {
		return e.renderSplash()
	}
	return e.screen.render()
}

func (e Editor) renderSplash() string {
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	accent := lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	var b strings.Builder

	// Shortcuts
	shortcuts := []struct{ key, desc string }{
		{"Space Space", "Find note"},
		{"Space n n", "New note"},
		{"Space n d", "Daily note"},
		{"Ctrl+h/l", "Navigate panels"},
		{"Space q q", "Quit"},
	}

	// Find the widest key for right-alignment
	maxKeyWidth := 0
	for _, s := range shortcuts {
		if len(s.key) > maxKeyWidth {
			maxKeyWidth = len(s.key)
		}
	}

	// Block width: right-aligned keys + gap + left-aligned descriptions
	gap := "  "
	maxDescWidth := 0
	for _, s := range shortcuts {
		if len(s.desc) > maxDescWidth {
			maxDescWidth = len(s.desc)
		}
	}
	blockWidth := maxKeyWidth + len(gap) + maxDescWidth

	// Vertical padding to center
	totalLines := 2 + len(shortcuts) // title + blank + shortcuts
	padTop := (e.height - totalLines) / 2
	if padTop < 1 {
		padTop = 1
	}
	for i := 0; i < padTop; i++ {
		b.WriteByte('\n')
	}

	// Title
	title := accent.Render("Kopr")
	titlePad := (e.width - lipgloss.Width(title)) / 2
	if titlePad < 0 {
		titlePad = 0
	}
	b.WriteString(strings.Repeat(" ", titlePad) + title + "\n\n")

	// Render shortcuts: keys right-aligned, descriptions left-aligned
	blockPad := (e.width - blockWidth) / 2
	if blockPad < 0 {
		blockPad = 0
	}

	for _, s := range shortcuts {
		keyPad := maxKeyWidth - len(s.key)
		line := strings.Repeat(" ", blockPad) +
			strings.Repeat(" ", keyPad) +
			keyStyle.Render(s.key) +
			gap +
			dim.Render(s.desc)
		b.WriteString(line + "\n")
	}

	// Fill remaining lines
	lines := strings.Count(b.String(), "\n")
	for i := lines; i < e.height; i++ {
		b.WriteByte('\n')
	}

	return b.String()
}

func (e Editor) Mode() NvimMode {
	return e.mode
}

func (e Editor) GetRPC() *RPC {
	return e.rpc
}

func (e Editor) ShowSplash() bool {
	return e.showSplash
}

func (e *Editor) SetShowSplash(show bool) {
	e.showSplash = show
}

func (e *Editor) SetFocused(focused bool) {
	e.focused = focused
	if e.screen != nil {
		e.screen.setShowCursor(focused)
	}
}

func (e *Editor) OpenFile(path string) error {
	if e.rpc == nil {
		return fmt.Errorf("RPC not connected")
	}
	e.showSplash = false
	return e.rpc.OpenFile(path)
}

func (e *Editor) Close() {
	if e.rpc != nil {
		e.rpc.Quit()
		if err := e.rpc.Close(); err != nil {
			fmt.Fprintln(os.Stderr, "fatal: close rpc:", err)
		}
	}
	if e.screen != nil {
		if err := e.screen.close(); err != nil {
			fmt.Fprintln(os.Stderr, "fatal: close vt screen:", err)
		}
	}
	if e.nvim != nil {
		if err := e.nvim.close(); err != nil {
			fmt.Fprintln(os.Stderr, "fatal: close nvim:", err)
		}
	}
}
