package editor

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pfassina/kopr/internal/theme"
)

// nextImageID is a process-global counter for Kitty image IDs.
var nextImageID uint32 = 100

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

// YankMsg is sent when text is yanked in Neovim (via TextYankPost autocmd).
type YankMsg struct {
	Text string
}

// ColorsReadyMsg is sent after the colorscheme is applied and colors are extracted.
// If Err is set, the colorscheme failed to load and Colors will be nil.
type ColorsReadyMsg struct {
	Colors map[string][2]string
	Err    error
}

// Editor is a Bubble Tea model that embeds Neovim in a PTY
// and renders it via a VT emulator, with RPC for programmatic control.
type Editor struct {
	width       int
	height      int
	vaultPath   string
	socketPath  string
	profileMode ProfileMode
	colorscheme        string
	renderMath         bool
	inlineImages       bool
	treesitterParsers  string
	theme       *theme.Theme
	nvim        *nvimPTY
	rpc         *RPC
	screen      *vtScreen
	started     bool
	mode        NvimMode
	err         error
	program     *tea.Program
	focused        bool
	showSplash     bool
	lastMouseButton tea.MouseButton

	// Image rendering state
	imageCache      *ImageCache
	imagePlacements []ImagePlacement
	uploadedImages  map[uint32]bool   // kitty image IDs that have been transmitted
	imagePathToID   map[string]uint32 // abs path → kitty image ID
	kittySupported  bool
}

// SetTheme sets the color theme for the editor splash screen.
func (e *Editor) SetTheme(th *theme.Theme) { e.theme = th }

func New(vaultPath string, profileMode ProfileMode, colorscheme string, renderMath bool, inlineImages bool, treesitterParsers string) Editor {
	return Editor{
		vaultPath:         vaultPath,
		profileMode:       profileMode,
		colorscheme:       colorscheme,
		renderMath:        renderMath,
		inlineImages:      inlineImages,
		treesitterParsers: treesitterParsers,
		mode:              ModeNormal,
		focused:           true,
		showSplash:        true,
		imageCache:        NewImageCache(),
		uploadedImages:    make(map[uint32]bool),
		imagePathToID:     make(map[string]uint32),
		kittySupported:    SupportsKittyGraphics(),
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
	width, height, vaultPath, profileMode, tsParsers := e.width, e.height, e.vaultPath, e.profileMode, e.treesitterParsers
	return func() tea.Msg {
		if err := EnsureProfile(profileMode); err != nil {
			return editorErrorMsg{fmt.Errorf("nvim profile: %w", err)}
		}

		socketPath := fmt.Sprintf("/tmp/kopr-%d.sock", os.Getpid())
		if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
			return editorErrorMsg{fmt.Errorf("remove socket %s: %w", socketPath, err)}
		}

		nvim, err := startNvim(width, height, socketPath, vaultPath, tsParsers)
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
		// Ignore transient invalid sizes (some terminals report 0x0 during drag-resize).
		if msg.Width <= 0 || msg.Height <= 0 {
			debugf("WindowSizeMsg ignored: %dx%d", msg.Width, msg.Height)
			return e, nil
		}
		// Note: we still resize the embedded Neovim PTY even at small sizes; the app
		// will render a "window too small" placeholder at the app layer.
		debugf("WindowSizeMsg: %dx%d started=%v splash=%v rpc=%v", msg.Width, msg.Height, e.started, e.showSplash, e.rpc != nil)
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
			// Resize / re-init the VT emulator. We've seen cases where simply resizing
			// the emulator can result in a permanently blank render after some terminal
			// resize sequences; recreating the emulator is cheap and robust.
			if e.screen != nil {
				if err := e.screen.close(); err != nil {
					e.err = err
					return e, tea.Quit
				}
			}
			e.screen = newVTScreen(e.width, e.height, e.nvim.file)

			// Defensive: after some resize sequences terminals can end up with a blank
			// frame until Neovim repaints. Force a redraw when dimensions change.
			if e.rpc != nil && !e.showSplash {
				debugf("rpc redraw! start")
				if err := e.rpc.ExecCommand("redraw!"); err != nil {
					e.err = err
					return e, tea.Quit
				}
				debugf("rpc redraw! ok")
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
			if err := e.rpc.SetupYankClipboard(e.program); err != nil {
				e.err = err
				return e, tea.Quit
			}
		}
		// Enable mouse support so Neovim processes mouse events from the PTY
		if err := e.rpc.ExecCommand("set mouse=a"); err != nil {
			e.err = err
			return e, tea.Quit
		}
		// Ensure left gutter aligns buffer text with panel titles
		if err := e.rpc.ExecCommand("set foldcolumn=1"); err != nil {
			e.err = err
			return e, tea.Quit
		}
		// Configure math rendering
		if err := e.rpc.SetupMathRendering(e.renderMath); err != nil {
			e.err = err
			return e, tea.Quit
		}
		// Configure image rendering
		if err := e.rpc.SetupImageRendering(e.inlineImages, e.program, e.kittySupported); err != nil {
			e.err = err
			return e, tea.Quit
		}
		// Apply configured colorscheme and extract colors for TUI
		colorCmd := e.applyColorscheme()
		// Load splash buffer so neovim starts in a clean state
		if err := e.rpc.LoadSplashBuffer(); err != nil {
			e.err = err
			return e, tea.Quit
		}
		return e, colorCmd

	case editorErrorMsg:
		e.err = msg.err
		return e, nil

	case ModeChangedMsg:
		e.mode = msg.Mode
		return e, nil

	case ImagePositionsMsg:
		e.imagePlacements = msg.Placements
		// Load any images we haven't seen yet and send heights to Neovim
		if e.inlineImages && e.rpc != nil {
			cmd := e.processImagePlacements()
			return e, cmd
		}
		return e, nil

	case vtOutputMsg:
		debugf("vtOutputMsg: %d bytes screen=%v", len(msg.data), e.screen != nil)
		if e.screen != nil {
			if _, err := e.screen.write(msg.data); err != nil {
				e.err = err
				return e, tea.Quit
			}
		}
		return e, waitForOutput(e.nvim)

	case vtClosedMsg:
		debugf("vtClosedMsg: %v", msg.err)
		return e, tea.Quit

	case EditorMouseMsg:
		if e.nvim == nil || e.showSplash {
			return e, nil
		}
		// Track pressed button so we can encode releases correctly
		if msg.Action == tea.MouseActionPress {
			e.lastMouseButton = msg.Button
		}
		raw := mouseMsgToBytes(msg.MouseMsg, msg.Col, msg.Row, e.lastMouseButton)
		if raw != nil {
			if _, err := e.nvim.file.Write(raw); err != nil {
				e.err = err
				return e, tea.Quit
			}
		}
		if msg.Action == tea.MouseActionRelease {
			e.lastMouseButton = tea.MouseButtonNone
		}
		return e, nil

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
	rendered := e.screen.render()
	if e.inlineImages && e.kittySupported && len(e.imagePlacements) > 0 {
		rendered = e.overlayImages(rendered)
	}
	return rendered
}

func (e Editor) renderSplash() string {
	th := e.theme
	dim := lipgloss.NewStyle().Foreground(th.Dim)
	accent := lipgloss.NewStyle().Foreground(th.Accent).Bold(true)
	keyStyle := lipgloss.NewStyle().Foreground(th.Text)

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

// applyColorscheme applies the configured colorscheme via RPC and returns a
// command that extracts colors and sends ColorsReadyMsg to the app.
func (e Editor) applyColorscheme() tea.Cmd {
	if e.rpc == nil || e.colorscheme == "" {
		return nil
	}
	rpc := e.rpc
	cs := e.colorscheme
	return func() tea.Msg {
		if err := rpc.ApplyColorscheme(cs); err != nil {
			debugf("apply colorscheme %q failed: %v", cs, err)
			return ColorsReadyMsg{Err: fmt.Errorf("colorscheme %q: %w", cs, err)}
		}
		// Extract colors before clearing backgrounds so we capture the
		// colorscheme's intended palette for TUI elements.
		colors, err := rpc.ExtractColors()
		if err != nil {
			debugf("extract colors failed: %v", err)
			colors = nil
		}
		// Clear explicit backgrounds so Neovim uses the terminal default,
		// preserving terminal transparency.
		rpc.ClearHighlightBgs()
		return ColorsReadyMsg{Colors: colors}
	}
}

// InlineImages returns whether inline image rendering is enabled.
func (e Editor) InlineImages() bool {
	return e.inlineImages
}

// SetInlineImages updates the inline images toggle state.
func (e *Editor) SetInlineImages(enabled bool) {
	e.inlineImages = enabled
	if !enabled {
		e.imagePlacements = nil
		// Delete all uploaded images
		// (Kitty delete sequences will be garbage-collected by the terminal)
	}
}

// processImagePlacements loads images that haven't been cached yet and sends
// their heights back to Neovim for virtual line reservation.
func (e Editor) processImagePlacements() tea.Cmd {
	heights := make(map[string]int)
	// Approximate cell dimensions: 8px wide, 16px tall (common for monospace)
	const cellW, cellH = 8, 16

	for i := range e.imagePlacements {
		p := &e.imagePlacements[i]
		cached := e.imageCache.Get(p.Path)
		if cached == nil {
			// Determine max dimensions: use editor width minus some margin
			maxCols := e.width - 4
			if maxCols < 10 {
				maxCols = 10
			}
			maxRows := e.height / 2
			if maxRows < 5 {
				maxRows = 5
			}
			img, err := LoadImage(p.Path, maxCols, maxRows, cellW, cellH)
			if err != nil {
				continue
			}
			e.imageCache.Put(p.Path, img)
			cached = img
		}

		p.Cols = cached.WidthCells
		p.Rows = cached.HeightCells
		heights[p.Path] = cached.HeightCells

		// Assign a Kitty image ID if not already assigned
		if _, ok := e.imagePathToID[p.Path]; !ok {
			nextImageID++
			e.imagePathToID[p.Path] = nextImageID
		}
	}

	if len(heights) == 0 {
		return nil
	}

	rpc := e.rpc
	return func() tea.Msg {
		_ = rpc.NotifyImageHeights(heights) //nolint:errcheck // best-effort height notification
		return nil
	}
}

// overlayImages injects Kitty graphics protocol escape sequences into the
// rendered editor output to display images at their screen positions.
func (e Editor) overlayImages(rendered string) string {
	if len(e.imagePlacements) == 0 {
		return rendered
	}

	lines := strings.Split(rendered, "\n")
	var prefix strings.Builder

	for _, p := range e.imagePlacements {
		cached := e.imageCache.Get(p.Path)
		if cached == nil || p.ScreenRow < 0 || p.ScreenRow >= len(lines) {
			continue
		}

		id, ok := e.imagePathToID[p.Path]
		if !ok {
			continue
		}

		if !e.uploadedImages[id] {
			// Transmit the image data (includes placement)
			prefix.WriteString(KittyTransmit(id, cached))
			e.uploadedImages[id] = true
		} else {
			// Just place the already-uploaded image
			prefix.WriteString(KittyPlace(id, cached.WidthCells, cached.HeightCells))
		}

		// Move cursor to the image row and place
		// Use CSI cursor position: \x1b[row;colH (1-based)
		row := p.ScreenRow + 1 // convert to 1-based
		fmt.Fprintf(&prefix, "\x1b[%d;1H", row)
		prefix.WriteString(KittyPlace(id, cached.WidthCells, cached.HeightCells))
	}

	if prefix.Len() == 0 {
		return rendered
	}

	// Save cursor, emit image sequences, restore cursor, then the normal content
	return "\x1b7" + prefix.String() + "\x1b8" + rendered
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
