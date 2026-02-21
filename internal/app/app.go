package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/pfassina/kopr/internal/config"
	"github.com/pfassina/kopr/internal/editor"
	"github.com/pfassina/kopr/internal/index"
	"github.com/pfassina/kopr/internal/markdown"
	"github.com/pfassina/kopr/internal/panel"
	"github.com/pfassina/kopr/internal/session"
	"github.com/pfassina/kopr/internal/theme"
	"github.com/pfassina/kopr/internal/vault"
)

type focusedPanel int

const (
	focusEditor focusedPanel = iota
	focusTree
	focusInfo
	focusFinder
)

type promptAction struct {
	kind  string   // "save", "close", "create-note", "delete-note", "delete-notes", "rename-note"
	path  string   // target file path for delete/rename
	paths []string // multiple paths for multi-delete
}

type App struct {
	cfg      config.Config
	editor   editor.Editor
	program  *tea.Program
	tree     panel.Tree
	info     panel.Info
	status   panel.Status
	whichKey panel.WhichKey
	finder   panel.Finder
	prompt   panel.Prompt
	vault    *vault.Vault
	db       *index.DB
	indexer  *index.Indexer
	watcher  *index.Watcher
	store    *session.Store
	theme    theme.Theme
	width    int
	height   int
	focused  focusedPanel
	showTree bool
	showInfo bool
	zenMode  bool

	// Leader key system
	bindings map[string]*Binding
	leader   LeaderState

	// pendingPrompt tracks which action the overlay prompt is serving.
	pendingPrompt promptAction

	// currentFile caches the open file's relative path for use in View().
	// Never call RPC from View() — it can hang if the connection is dead.
	currentFile string

	// prevFile stores the previously opened note for gb (go back) navigation.
	prevFile string
}

// navigateTo opens a note and updates the navigation history.
func (a *App) navigateTo(relPath string) {
	if a.currentFile != "" && a.currentFile != relPath {
		a.prevFile = a.currentFile
	}
	fullPath := filepath.Join(a.cfg.VaultPath, relPath)
	a.openInEditor(fullPath)
	a.status.ClearError()
	a.status.SetFile(relPath)
	a.currentFile = relPath
	a.updateBacklinks(relPath)
}

func New(cfg config.Config) App {
	v := vault.New(cfg.VaultPath)
	t := panel.NewTree(v)
	t.Refresh()

	f := panel.NewFinder()
	store := session.NewStore(cfg.VaultPath)
	state, _ := store.Load()

	a := App{
		cfg:      cfg,
		editor:   editor.New(cfg.VaultPath, editor.ProfileMode(cfg.NvimMode), cfg.Colorscheme),
		tree:     t,
		info:     panel.NewInfo(),
		status:   panel.NewStatus(cfg.VaultPath),
		whichKey: panel.NewWhichKey(),
		finder:   f,
		prompt:   panel.NewPrompt(),
		vault:    v,
		store:    store,
		theme:    theme.DefaultTheme(),
		focused:  focusEditor,
		showTree: state.ShowTree,
		showInfo: state.ShowInfo,
	}
	a.initLeader()
	a.tree.SetTheme(&a.theme)
	a.info.SetTheme(&a.theme)
	a.finder.SetTheme(&a.theme)
	a.prompt.SetTheme(&a.theme)
	a.status.SetTheme(&a.theme)
	a.whichKey.SetTheme(&a.theme)
	a.editor.SetTheme(&a.theme)

	// Initialize index
	dbPath := filepath.Join(cfg.VaultPath, ".kopr", "index.db")
	ensureDir(filepath.Dir(dbPath))
	db, err := index.Open(dbPath)
	if err != nil {
		// Fail loud (but keep app usable): without an index the finder/search won't work.
		a.status.SetError(fmt.Sprintf("index open failed: %v", err))
	} else {
		a.db = db
		a.indexer = index.NewIndexer(db, cfg.VaultPath)
		a.finder.SetSearchFunc(a.searchNotes)
	}

	return a
}

func (a *App) SetProgram(p *tea.Program) {
	a.program = p
	a.editor.SetProgram(p)
}

func (a *App) Init() tea.Cmd {
	cmds := []tea.Cmd{a.editor.Init()}
	if a.indexer != nil {
		cmds = append(cmds, a.initIndex())
	}
	return tea.Batch(cmds...)
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			a.Close()
			return a, tea.Quit
		}

		// Save-as prompt takes priority when visible
		if a.prompt.Visible() {
			var cmd tea.Cmd
			a.prompt, cmd = a.prompt.Update(msg)
			return a, cmd
		}

		// Finder takes priority when visible
		if a.finder.Visible() {
			var cmd tea.Cmd
			a.finder, cmd = a.finder.Update(msg)
			return a, cmd
		}

		// When splash is showing, only leader keys work
		if a.editor.ShowSplash() && a.focused == focusEditor {
			// Escape returns from side panels to editor
			if msg.String() == "esc" {
				return a, nil
			}
			// Try leader key system
			if consumed, cmd := a.handleLeaderKey(msg.String()); consumed {
				a.updateWhichKey()
				return a, cmd
			}
			// Don't send other keys to the editor while splash is showing
			return a, nil
		}

		// Ctrl+h/l to switch panel focus
		switch msg.String() {
		case "ctrl+h":
			a.focusLeft()
			return a, nil
		case "ctrl+l":
			a.focusRight()
			return a, nil
		}

		// Escape returns from side panels to editor (unless tree help is showing)
		if msg.String() == "esc" && (a.focused == focusTree || a.focused == focusInfo) {
			if a.focused == focusTree && a.tree.ShowingHelp() {
				break // let tree handle it to dismiss help
			}
			a.setFocus(focusEditor)
			return a, nil
		}

		// Try leader key system (works from editor and side panels)
		// Skip when tree help is showing so any key dismisses help first
		if a.focused != focusTree || !a.tree.ShowingHelp() {
			if consumed, cmd := a.handleLeaderKey(msg.String()); consumed {
				a.updateWhichKey()
				return a, cmd
			}
		}

		// Side panels handle their own keys
		if a.focused == focusTree || a.focused == focusInfo {
			break
		}

	case leaderTimeoutMsg:
		a.handleLeaderTimeout()
		a.updateWhichKey()
		return a, nil

	case fatalErrorMsg:
		return a, tea.Batch(tea.Printf("fatal: %v\n", msg.err), tea.Quit)

	case tea.WindowSizeMsg:
		// Some terminals send transient 0x0 sizes during live resizes; ignore them.
		if msg.Width <= 0 || msg.Height <= 0 {
			return a, nil
		}
		a.width = msg.Width
		a.height = msg.Height
		a.finder.SetSize(msg.Width, msg.Height)

		minW, minH := a.minWindowSize()
		if a.width < minW || a.height < minH {
			return a, tea.ClearScreen
		}

		// Size prompt relative to the center/editor panel (Neovim buffer area), not the full screen.
		showTree, showInfo := a.panelsVisible()
		layout := ComputeLayout(a.width, a.height, showTree, showInfo, a.cfg.TreeWidth, a.cfg.InfoWidth)
		promptW := int(float64(layout.EditorWidth) * 0.80)
		// Clamp to a sane modal width; 80% of a wide terminal is still too wide.
		if promptW > 100 {
			promptW = 100
		}
		if promptW < 40 {
			promptW = 40
		}
		if promptW > layout.EditorWidth-2 {
			promptW = layout.EditorWidth - 2
		}
		a.prompt.SetSize(promptW, layout.Height)

		cmd := a.updateLayout()
		// Force a full terminal repaint on resize; some terminals/bubbletea render
		// paths can end up visually blank without an explicit clear.
		if cmd != nil {
			return a, tea.Batch(tea.ClearScreen, cmd)
		}
		return a, tea.ClearScreen

	case editor.ModeChangedMsg:
		a.status.SetMode(modeDisplayName(msg.Mode))
		if msg.Mode != editor.ModeNormal {
			a.cancelLeader()
			a.updateWhichKey()
		}

	case panel.FileSelectedMsg:
		a.navigateTo(msg.Path)
		a.setFocus(focusEditor)

	case panel.FinderResultMsg:
		a.handleFinderResult(msg.Path)
		a.setFocus(focusEditor)

	case panel.FinderCreateRequestMsg:
		// Keep finder visible so cancel returns the user to the same query.
		a.pendingPrompt = promptAction{kind: "finder-create", path: msg.Name}
		a.prompt.ShowConfirm(fmt.Sprintf("Create note %q?", msg.Name))
		return a, nil

	case panel.FinderClosedMsg:
		a.setFocus(focusEditor)

	case editor.FollowLinkMsg:
		a.FollowLink()
		return a, nil

	case editor.GoBackMsg:
		a.GoBack()
		return a, nil

	case editor.NoteClosedMsg:
		// If prompt is already active, upgrade the pending action to "close"
		// instead of interrupting (e.g. :wq on unnamed sends both
		// save-unnamed and close-note in quick succession)
		if a.prompt.Visible() {
			a.pendingPrompt.kind = "close"
			return a, nil
		}
		return a, a.handleNoteClose(msg.Save)

	case editor.SaveUnnamedMsg:
		a.pendingPrompt = promptAction{kind: "save"}
		a.prompt.Show("Save as", "my-note.md")
		return a, nil

	case editor.BufferWrittenMsg:
		return a, a.handleBufferWritten(msg.Path)

	case panel.TreeNewNoteMsg:
		a.pendingPrompt = promptAction{kind: "create-note"}
		a.prompt.Show("New note", "my-note.md")
		return a, nil

	case panel.TreeDeleteNoteMsg:
		a.pendingPrompt = promptAction{kind: "delete-note", path: msg.Path}
		a.prompt.ShowConfirm("Delete " + msg.Name + "?")
		return a, nil

	case panel.TreeRenameNoteMsg:
		a.pendingPrompt = promptAction{kind: "rename-note", path: msg.Path}
		a.prompt.Show("Rename", msg.Name)
		return a, nil

	case panel.TreeDeleteNotesMsg:
		names := make([]string, len(msg.Paths))
		for i, p := range msg.Paths {
			names[i] = filepath.Base(p)
		}
		label := fmt.Sprintf("Delete %d files (%s)?", len(msg.Paths), strings.Join(names, ", "))
		a.pendingPrompt = promptAction{kind: "delete-notes", paths: msg.Paths}
		a.prompt.ShowConfirm(label)
		return a, nil

	case panel.TreePasteMsg:
		return a, a.handlePaste(msg)

	case panel.TreeClipboardChangedMsg:
		a.updateClipboardStatus(msg.Op, msg.Count)
		return a, nil

	case panel.PromptResultMsg:
		return a, a.handlePromptResult(msg.Value)

	case panel.PromptCancelledMsg:
		return a, a.handlePromptCancelled()

	case noteIndexedMsg:
		if msg.err != nil {
			return a, fatalCmd(fmt.Errorf("index note: %w", msg.err))
		}
		// If the user saved the currently open note, refresh backlinks in-place.
		if msg.relPath != "" && msg.relPath == a.currentFile {
			a.updateBacklinks(a.currentFile)
		}
		return a, nil

	case editor.ColorsReadyMsg:
		if msg.Err != nil {
			a.status.SetError(msg.Err.Error())
			return a, nil
		}
		if msg.Colors != nil {
			updated := theme.FromExtracted(msg.Colors, a.theme)
			a.theme = updated
			// Re-set pointers since we replaced the struct value.
			a.tree.SetTheme(&a.theme)
			a.info.SetTheme(&a.theme)
			a.finder.SetTheme(&a.theme)
			a.prompt.SetTheme(&a.theme)
			a.status.SetTheme(&a.theme)
			a.whichKey.SetTheme(&a.theme)
			a.editor.SetTheme(&a.theme)
		}
		return a, nil

	case indexInitDoneMsg:
		if msg.err != nil {
			// Fail fast and loud: indexing is a core feature.
			return a, tea.Batch(tea.Printf("fatal: indexing failed: %v\n", msg.err), tea.Quit)
		}
		// Index is ready - start file watcher
		if a.indexer != nil {
			w, err := index.NewWatcher(a.indexer, a.cfg.VaultPath, func() {
				a.tree.Refresh()
			}, func(err error) {
				if a.program != nil {
					a.program.Send(fatalErrorMsg{err: err})
				}
			})
			if err != nil {
				return a, tea.Batch(tea.Printf("fatal: watcher init failed: %v\n", err), tea.Quit)
			}
			a.watcher = w
			go w.Start()
		}
		return a, nil
	}

	// Route key events based on focus
	var cmd tea.Cmd
	switch msg.(type) {
	case tea.KeyMsg:
		switch a.focused {
		case focusTree:
			a.tree, cmd = a.tree.Update(msg)
			return a, cmd
		case focusInfo:
			a.info, cmd = a.info.Update(msg)
			return a, cmd
		default:
			a.editor, cmd = a.editor.Update(msg)
			return a, cmd
		}
	default:
		a.editor, cmd = a.editor.Update(msg)
	}

	return a, cmd
}

func (a *App) View() string {
	if a.width == 0 || a.height == 0 {
		return "Loading..."
	}

	minW, minH := a.minWindowSize()
	if a.width < minW || a.height < minH {
		msg := fmt.Sprintf("Window too small (%dx%d)\nMinimum supported: %dx%d", a.width, a.height, minW, minH)
		// Use the terminal's default background so the placeholder matches whatever
		// theme the user is running.
		style := lipgloss.NewStyle().
			Foreground(a.theme.Text).
			Padding(1, 2)
		box := style.Render(msg)

		fillLines := a.height
		if fillLines < 1 {
			fillLines = 1
		}
		base := strings.Repeat("\n", fillLines)
		return overlayCenter(base, box, a.width, a.height)
	}

	showTree, showInfo := a.panelsVisible()
	layout := ComputeLayout(a.width, a.height, showTree, showInfo, a.cfg.TreeWidth, a.cfg.InfoWidth)

	// Editor title row
	editorTitle := a.editorTitle()

	editorView := editorTitle + "\n" + a.editor.View()

	var main string

	if !showTree && !showInfo {
		main = lipgloss.NewStyle().
			Width(layout.EditorWidth).
			Height(layout.Height).
			Render(editorView)
	} else {
		var columns []string

		if showTree {
			tw := layout.TreeWidth - 1
			if tw < 0 {
				tw = 0
			}
			borderStyle := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder(), false, true, false, false).
				BorderForeground(a.theme.Border).
				Width(tw).
				Height(layout.Height)
			columns = append(columns, borderStyle.Render(a.tree.View()))
		}

		editorStyle := lipgloss.NewStyle().
			Width(layout.EditorWidth).
			Height(layout.Height)
		columns = append(columns, editorStyle.Render(editorView))

		if showInfo {
			iw := layout.InfoWidth - 1
			if iw < 0 {
				iw = 0
			}
			borderStyle := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder(), false, false, false, true).
				BorderForeground(a.theme.Border).
				Width(iw).
				Height(layout.Height)
			columns = append(columns, borderStyle.Render(a.info.View()))
		}

		main = lipgloss.JoinHorizontal(lipgloss.Top, columns...)
	}

	result := main + "\n" + a.status.View()

	// Overlay which-key popup
	if a.leader.showHelp {
		wkView := a.whichKey.View()
		if wkView != "" {
			result = overlayCenter(result, wkView, a.width, a.height)
		}
	}

	// Overlay finder
	if a.finder.Visible() {
		finderView := a.finder.View()
		if finderView != "" {
			result = overlayCenter(result, finderView, a.width, a.height)
		}
	}

	// Overlay save-as prompt
	if a.prompt.Visible() {
		promptView := a.prompt.View()
		if promptView != "" {
			result = overlayCenter(result, promptView, a.width, a.height)
		}
	}

	return result
}

func (a *App) Close() {
	// Save session state
	if a.store != nil {
		state := session.State{
			ShowTree:  a.showTree,
			ShowInfo:  a.showInfo,
			TreeWidth: a.cfg.TreeWidth,
			InfoWidth: a.cfg.InfoWidth,
		}
		if err := a.store.Save(state); err != nil {
			fmt.Fprintln(os.Stderr, "fatal: save session state:", err)
		}
	}

	a.editor.Close()
	if a.watcher != nil {
		if err := a.watcher.Stop(); err != nil {
			fmt.Fprintln(os.Stderr, "fatal: stop watcher:", err)
		}
	}
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			fmt.Fprintln(os.Stderr, "fatal: close db:", err)
		}
	}
}

// handleNoteClose processes a quit/close command from neovim.
func (a *App) handleNoteClose(save bool) tea.Cmd {
	rpc := a.editor.GetRPC()
	if rpc == nil {
		return nil
	}

	if save {
		// Check if buffer is named using cached path (never call RPC here
		// since neovim might be in an error state from QuitPre)
		if a.currentFile == "" {
			// Unnamed buffer — ask for title, then close to splash
			a.pendingPrompt = promptAction{kind: "close"}
			a.prompt.Show("Save as", "my-note.md")
			return nil
		}
		// Named buffer — save, then go to splash
		if err := rpc.ExecCommand("w"); err != nil {
			return tea.Batch(tea.Printf("fatal: nvim write failed: %v\n", err), tea.Quit)
		}
	}

	a.showSplash()
	return nil
}

func (a *App) handleBufferWritten(path string) tea.Cmd {
	// Always re-index on save so backlinks/search stay fresh.
	cmds := []tea.Cmd{}
	if a.indexer != nil && strings.HasSuffix(strings.ToLower(path), ".md") {
		cmds = append(cmds, a.indexFile(path))
	}

	// Optional: format on save (scoped to the active buffer).
	if !a.cfg.AutoFormatOnSave {
		if len(cmds) == 0 {
			return nil
		}
		return tea.Batch(cmds...)
	}
	if !strings.HasSuffix(strings.ToLower(path), ".md") {
		if len(cmds) == 0 {
			return nil
		}
		return tea.Batch(cmds...)
	}

	rpc := a.editor.GetRPC()
	if rpc == nil {
		if len(cmds) == 0 {
			return nil
		}
		return tea.Batch(cmds...)
	}

	// Only format the currently active file.
	cur, err := rpc.CurrentFile()
	if err != nil {
		return fatalCmd(fmt.Errorf("nvim current file: %w", err))
	}
	if cur != path {
		if len(cmds) == 0 {
			return nil
		}
		return tea.Batch(cmds...)
	}

	formatCmd := func() tea.Msg {
		// Capture cursor so we can keep the user's position.
		line, col, err := rpc.CursorPosition()
		if err != nil {
			return fatalErrorMsg{err: fmt.Errorf("nvim cursor position: %w", err)}
		}

		content, err := rpc.BufferContent()
		if err != nil {
			return fatalErrorMsg{err: fmt.Errorf("nvim buffer content: %w", err)}
		}

		var b strings.Builder
		for i, ln := range content {
			b.Write(ln)
			if i < len(content)-1 {
				b.WriteByte('\n')
			}
		}

		formatted := markdown.Format([]byte(b.String()))
		if string(formatted) == b.String()+"\n" || string(formatted) == b.String() {
			return nil
		}

		// Apply formatted text back into the buffer.
		text := strings.TrimRight(string(formatted), "\n")
		lines := []string{}
		if text != "" {
			lines = strings.Split(text, "\n")
		}
		if err := rpc.SetBufferLines(lines); err != nil {
			return fatalErrorMsg{err: fmt.Errorf("nvim set buffer lines: %w", err)}
		}

		// Restore cursor (best-effort; clamp line to buffer length).
		if line < 1 {
			line = 1
		}
		if len(lines) > 0 && line > len(lines) {
			line = len(lines)
		}
		_ = rpc.SetCursorPosition(line, col)

		// Write without triggering autocommands to avoid infinite loops.
		if err := rpc.ExecCommand("noautocmd write"); err != nil {
			return fatalErrorMsg{err: fmt.Errorf("nvim write formatted buffer: %w", err)}
		}
		return nil
	}

	cmds = append(cmds, formatCmd)
	return tea.Batch(cmds...)
}

// showSplash transitions the editor to the splash screen.
func (a *App) showSplash() {
	rpc := a.editor.GetRPC()
	if rpc != nil {
		if err := rpc.LoadSplashBuffer(); err != nil {
			if a.program != nil {
				a.program.Send(fatalErrorMsg{err: err})
			}
			return
		}
	}
	a.editor.SetShowSplash(true)
	a.status.SetFile("")
	a.currentFile = ""
	a.info.Clear()
	a.setFocus(focusEditor)
	a.updateLayout()
}

// openInEditor opens a file and recalculates layout since splash is dismissed.
func (a *App) openInEditor(path string) {
	if err := a.editor.OpenFile(path); err != nil {
		if a.program != nil {
			a.program.Send(fatalErrorMsg{err: err})
		}
		return
	}
	a.updateLayout()
}

// handlePromptCancelled handles Esc/empty input on the overlay prompt.
func (a *App) handlePromptCancelled() tea.Cmd {
	action := a.pendingPrompt
	a.pendingPrompt = promptAction{}

	if action.kind == "close" {
		a.showSplash()
	}
	return nil
}

// handlePromptResult handles a confirmed value from the overlay prompt.
func (a *App) handlePromptResult(value string) tea.Cmd {
	action := a.pendingPrompt
	if action.kind == "" {
		return nil
	}

	// By default the prompt stays open after Enter. We only clear/hide it on success.
	switch action.kind {
	case "save", "close":
		if cmd, ok := a.handleSaveAsPrompt(value, action.kind == "close"); ok {
			a.prompt.Hide()
			a.pendingPrompt = promptAction{}
			return cmd
		}
		return nil
	case "create-note":
		if cmd, ok := a.handleCreateNotePrompt(value); ok {
			a.prompt.Hide()
			a.pendingPrompt = promptAction{}
			return cmd
		}
		return nil
	case "rename-note":
		if cmd, ok := a.handleRenameNotePrompt(value, action.path); ok {
			a.prompt.Hide()
			a.pendingPrompt = promptAction{}
			return cmd
		}
		return nil
	case "delete-note":
		// Confirm prompts don't need validation; keep prior behavior.
		a.pendingPrompt = promptAction{}
		a.prompt.Hide()
		return a.handleDeleteNote(value, action.path)
	case "delete-notes":
		a.pendingPrompt = promptAction{}
		a.prompt.Hide()
		return a.handleDeleteNotes(value, action.paths)
	case "finder-create":
		// Confirm-only prompt: create note on "yes", otherwise do nothing.
		a.pendingPrompt = promptAction{}
		a.prompt.Hide()
		if strings.ToLower(strings.TrimSpace(value)) != "yes" {
			return nil
		}
		a.createNoteFromFinder(action.path)
		a.finder.Hide()
		a.setFocus(focusEditor)
		return nil
	}
	return nil
}

// handleSaveAsPrompt validates and performs save-as from the overlay prompt.
// Returns ok=false when the value is rejected and the prompt should remain visible.
func (a *App) handleSaveAsPrompt(value string, closeAfter bool) (cmd tea.Cmd, ok bool) {
	relPath := value
	if !strings.HasSuffix(relPath, ".md") {
		relPath += ".md"
	}

	if msg := a.checkUniqueBasename(relPath); msg != "" {
		a.prompt.SetError(msg)
		return nil, false
	}

	rpc := a.editor.GetRPC()
	if rpc == nil {
		a.prompt.SetError("editor RPC unavailable")
		return nil, false
	}

	// Get current buffer content before setting name
	content, err := rpc.BufferContent()
	if err != nil {
		a.prompt.SetError(err.Error())
		return nil, false
	}

	// Build content string
	var buf strings.Builder
	for i, line := range content {
		buf.Write(line)
		if i < len(content)-1 {
			buf.WriteByte('\n')
		}
	}

	// Create the file on disk via vault
	fullPath, err := a.vault.CreateNote(relPath, buf.String())
	if err != nil {
		a.prompt.SetError(err.Error())
		return nil, false
	}

	// Make the buffer modifiable, set name, and write
	if err := rpc.ExecCommand("setlocal modifiable"); err != nil {
		return fatalCmd(err), true
	}
	if err := rpc.SetBufferName(fullPath); err != nil {
		return fatalCmd(err), true
	}
	// Remove BufWriteCmd interference by setting buftype back to normal
	if err := rpc.ExecCommand("setlocal buftype="); err != nil {
		return fatalCmd(err), true
	}
	if err := rpc.WriteBuffer(); err != nil {
		return fatalCmd(err), true
	}

	a.status.SetFile(relPath)
	a.currentFile = relPath
	a.tree.Refresh()
	a.updateBacklinks(relPath)

	if closeAfter {
		a.showSplash()
	}

	return nil, true
}

// handleCreateNotePrompt validates and creates a new note from the overlay prompt.
// Returns ok=false when the value is rejected and the prompt should remain visible.
func (a *App) handleCreateNotePrompt(name string) (cmd tea.Cmd, ok bool) {
	// Directory creation: name ends with /
	if strings.HasSuffix(name, "/") {
		a.prompt.SetError("cannot create a directory here")
		return nil, false
	}

	relPath := name
	if !strings.HasSuffix(relPath, ".md") {
		relPath += ".md"
	}

	if msg := a.checkUniqueBasename(relPath); msg != "" {
		a.prompt.SetError(msg)
		return nil, false
	}

	content := fmt.Sprintf("---\ntitle: %s\n---\n\n", strings.TrimSuffix(name, ".md"))
	fullPath, err := a.vault.CreateNote(relPath, content)
	if err != nil {
		a.prompt.SetError(err.Error())
		return nil, false
	}

	a.openInEditor(fullPath)
	a.status.SetFile(relPath)
	a.currentFile = relPath
	a.tree.Refresh()
	a.setFocus(focusEditor)
	return nil, true
}

// handlePaste performs copy or move for files in the clipboard.
func (a *App) handlePaste(msg panel.TreePasteMsg) tea.Cmd {
	// Copy is disallowed because it would violate the vault-wide basename uniqueness invariant.
	if msg.Op == panel.ClipboardCopy {
		a.status.SetError("copy not allowed: vault requires unique basenames")
		return nil
	}

	for _, src := range msg.Sources {
		newRel := filepath.Join(msg.DestDir, filepath.Base(src))
		if m := a.checkUniqueBasenameExcept(newRel, src); m != "" {
			a.status.SetError(m)
			return nil
		}

		err := a.vault.MoveNote(src, msg.DestDir)
		if err != nil {
			a.status.SetError(err.Error())
			return nil
		}

		if a.currentFile == src {
			fullPath := filepath.Join(a.cfg.VaultPath, newRel)
			rpc := a.editor.GetRPC()
			if rpc != nil {
				if err := rpc.SetBufferName(fullPath); err != nil {
					return fatalCmd(err)
				}
				if err := rpc.WriteBuffer(); err != nil {
					return fatalCmd(err)
				}
			}
			a.status.SetFile(newRel)
			a.currentFile = newRel
		}
	}

	a.tree.ClearClipboard()
	a.tree.ClearSelected()
	a.updateClipboardStatus(panel.ClipboardNone, 0)
	a.tree.Refresh()
	return nil
}

// updateClipboardStatus updates the status bar clipboard indicator.
func (a *App) updateClipboardStatus(op panel.ClipboardOp, count int) {
	switch {
	case op == panel.ClipboardCopy && count > 0:
		a.status.SetClipboard(fmt.Sprintf("%d yanked", count))
	case op == panel.ClipboardCut && count > 0:
		a.status.SetClipboard(fmt.Sprintf("%d cut", count))
	default:
		a.status.SetClipboard("")
	}
}

// handleDeleteNote deletes a note after confirmation.
func (a *App) handleDeleteNote(confirmation, relPath string) tea.Cmd {
	if strings.ToLower(strings.TrimSpace(confirmation)) != "yes" {
		return nil
	}

	if a.currentFile == relPath {
		a.showSplash()
	}

	if err := a.vault.DeleteNote(relPath); err != nil {
		return fatalCmd(err)
	}
	a.tree.ClearSelected()
	a.tree.Refresh()
	return nil
}

// handleDeleteNotes deletes multiple notes after confirmation.
func (a *App) handleDeleteNotes(confirmation string, paths []string) tea.Cmd {
	if strings.ToLower(strings.TrimSpace(confirmation)) != "yes" {
		return nil
	}

	for _, p := range paths {
		if a.currentFile == p {
			a.showSplash()
		}
		if err := a.vault.DeleteNote(p); err != nil {
			return fatalCmd(err)
		}
	}

	a.tree.ClearSelected()
	a.tree.Refresh()
	return nil
}

// handleRenameNotePrompt validates and renames from the overlay prompt.
// Returns ok=false when the value is rejected and the prompt should remain visible.
func (a *App) handleRenameNotePrompt(newName, oldPath string) (cmd tea.Cmd, ok bool) {
	newRel := newName
	if !strings.HasSuffix(newRel, ".md") {
		newRel += ".md"
	}

	// Keep the same directory
	dir := filepath.Dir(oldPath)
	if dir != "." {
		newRel = filepath.Join(dir, newRel)
	}

	if msg := a.checkUniqueBasename(newRel); msg != "" {
		a.prompt.SetError(msg)
		return nil, false
	}

	// Reuse the existing implementation (it already handles link rewriting and editor updates).
	cmd = a.handleRenameNote(newName, oldPath)
	// If the underlying rename failed, it currently returns nil without surfacing an error.
	// Detect obvious failure by checking filesystem state.
	if _, err := os.Stat(filepath.Join(a.cfg.VaultPath, newRel)); err != nil {
		a.prompt.SetError("rename failed")
		return nil, false
	}
	return cmd, true
}

// handleRenameNote renames a note to the given name.
func (a *App) handleRenameNote(newName, oldPath string) tea.Cmd {
	newRel := newName
	if !strings.HasSuffix(newRel, ".md") {
		newRel += ".md"
	}

	// Keep the same directory
	dir := filepath.Dir(oldPath)
	if dir != "." {
		newRel = filepath.Join(dir, newRel)
	}

	if msg := a.checkUniqueBasename(newRel); msg != "" {
		a.status.SetError(msg)
		return nil
	}

	// Capture old basename for link rewriting before rename
	oldBasename := strings.TrimSuffix(filepath.Base(oldPath), ".md")
	newBasename := strings.TrimSuffix(filepath.Base(newRel), ".md")

	// Get backlinks before rename (while DB still has old data)
	var backlinkPaths []string
	if oldBasename != newBasename && a.db != nil {
		backlinks, err := a.db.GetBacklinks(oldPath)
		if err == nil {
			for _, bl := range backlinks {
				backlinkPaths = append(backlinkPaths, bl.SourcePath)
			}
		}
	}

	if err := a.vault.RenameNote(oldPath, newRel); err != nil {
		return nil
	}

	// Rewrite wiki links in all notes that linked to the old name
	if oldBasename != newBasename {
		for _, srcPath := range backlinkPaths {
			absPath := filepath.Join(a.cfg.VaultPath, srcPath)
			if _, err := vault.RewriteLinksInNote(absPath, oldBasename, newBasename); err != nil {
				return fatalCmd(err)
			}
		}
	}

	// If the renamed file is currently open, update the editor
	if a.currentFile == oldPath {
		fullPath := filepath.Join(a.cfg.VaultPath, newRel)
		rpc := a.editor.GetRPC()
		if rpc != nil {
			if err := rpc.SetBufferName(fullPath); err != nil {
				return fatalCmd(err)
			}
			if err := rpc.WriteBuffer(); err != nil {
				return fatalCmd(err)
			}
		}
		a.status.SetFile(newRel)
		a.currentFile = newRel
	}

	a.tree.Refresh()
	return nil
}

func (a *App) panelsVisible() (bool, bool) {
	splash := a.editor.ShowSplash()
	return a.showTree && !a.zenMode && !splash, a.showInfo && !a.zenMode && !splash
}

func (a *App) minWindowSize() (minW, minH int) {
	// UX-driven minimum supported terminal size. Below this we stop rendering the
	// full UI and show a placeholder message.
	//
	// 80x24 is the classic baseline and still common on low-res displays.
	return 60, 24
}

func (a *App) updateLayout() tea.Cmd {
	showTree, showInfo := a.panelsVisible()
	layout := ComputeLayout(a.width, a.height, showTree, showInfo, a.cfg.TreeWidth, a.cfg.InfoWidth)

	a.tree.SetSize(layout.TreeWidth, layout.Height)
	a.info.SetSize(layout.InfoWidth, layout.Height)
	a.status.SetWidth(a.width)
	a.whichKey.SetWidth(a.width / 2)

	editorHeight := layout.Height - 1 // -1 for editor title row
	if editorHeight < 1 {
		editorHeight = 1
	}
	editorSize := tea.WindowSizeMsg{
		Width:  layout.EditorWidth,
		Height: editorHeight,
	}
	var cmd tea.Cmd
	a.editor, cmd = a.editor.Update(editorSize)
	return cmd
}

func (a *App) updateWhichKey() {
	if !a.leader.showHelp || a.leader.node == nil {
		a.whichKey.Clear()
		return
	}

	var entries []panel.WhichKeyEntry
	for _, b := range a.leader.node {
		entries = append(entries, panel.WhichKeyEntry{
			Key:   b.Key,
			Label: b.Label,
		})
	}
	a.whichKey.SetEntries(a.leader.keys, entries)
}

func (a *App) editorTitle() string {
	title := "Kopr"
	if !a.editor.ShowSplash() && a.currentFile != "" {
		title = filepath.Base(a.currentFile)
	}

	var style lipgloss.Style
	if a.focused == focusEditor {
		style = lipgloss.NewStyle().
			Bold(true).
			Foreground(a.theme.Accent).
			Underline(true).
			Padding(0, 1)
	} else {
		style = lipgloss.NewStyle().
			Bold(true).
			Foreground(a.theme.Dim).
			Padding(0, 1)
	}

	return style.Render(title)
}

func (a *App) setFocus(target focusedPanel) {
	a.tree.SetFocused(target == focusTree)
	a.info.SetFocused(target == focusInfo)
	a.editor.SetFocused(target == focusEditor)
	a.focused = target
}

func (a *App) focusLeft() {
	switch a.focused {
	case focusEditor:
		if a.showTree && !a.zenMode {
			a.setFocus(focusTree)
		}
	case focusInfo:
		a.setFocus(focusEditor)
	}
}

func (a *App) focusRight() {
	switch a.focused {
	case focusEditor:
		if a.showInfo && !a.zenMode {
			a.setFocus(focusInfo)
		}
	case focusTree:
		a.setFocus(focusEditor)
	}
}

func (a *App) ToggleTree() {
	a.showTree = !a.showTree
	if !a.showTree && a.focused == focusTree {
		a.setFocus(focusEditor)
	}
	a.updateLayout()
}

func (a *App) ToggleInfo() {
	a.showInfo = !a.showInfo
	if !a.showInfo && a.focused == focusInfo {
		a.setFocus(focusEditor)
	}
	a.updateLayout()
}

func (a *App) ToggleZen() {
	a.zenMode = !a.zenMode
	if a.zenMode && (a.focused == focusTree || a.focused == focusInfo) {
		a.setFocus(focusEditor)
	}
	a.updateLayout()
}

func modeDisplayName(mode editor.NvimMode) string {
	names := map[editor.NvimMode]string{
		editor.ModeNormal:  "NORMAL",
		editor.ModeInsert:  "INSERT",
		editor.ModeVisual:  "VISUAL",
		editor.ModeVisLine: "V-LINE",
		editor.ModeVisBlk:  "V-BLOCK",
		editor.ModeCommand: "COMMAND",
		editor.ModeReplace: "REPLACE",
		editor.ModeTermnl:  "TERMINAL",
	}
	if n, ok := names[mode]; ok {
		return n
	}
	return strings.ToUpper(string(mode))
}

func overlayCenter(base, overlay string, width, height int) string {
	baseLines := strings.Split(base, "\n")
	overlayLines := strings.Split(overlay, "\n")

	overlayWidth := 0
	for _, line := range overlayLines {
		w := lipgloss.Width(line)
		if w > overlayWidth {
			overlayWidth = w
		}
	}

	startRow := (height - len(overlayLines)) / 2
	startCol := (width - overlayWidth) / 2
	if startRow < 0 {
		startRow = 0
	}
	if startCol < 0 {
		startCol = 0
	}

	padToCol := func(s string, col int) string {
		// Pad with spaces based on *visible* width (handles ANSI strings safely).
		for lipgloss.Width(s) < col {
			s += " "
		}
		return s
	}

	for i, overlayLine := range overlayLines {
		row := startRow + i
		if row >= len(baseLines) {
			break
		}

		baseLine := baseLines[row]
		baseLine = padToCol(baseLine, startCol)

		// Overlay by columns without breaking ANSI sequences.
		// Keep the left part of the base line, replace the middle with overlay,
		// and keep the right tail of the base line.
		left := ansi.Cut(baseLine, 0, startCol)
		right := ansi.Cut(baseLine, startCol+overlayWidth, width)

		line := left + overlayLine + right
		// Ensure line doesn't overflow terminal width.
		baseLines[row] = ansi.Truncate(line, width, "")
	}

	return strings.Join(baseLines, "\n")
}

// checkUniqueBasename returns an error message if a different note with the same
// basename already exists in the vault. Returns "" if the name is available.
func (a *App) checkUniqueBasename(relPath string) string {
	return a.checkUniqueBasenameExcept(relPath, "")
}

// checkUniqueBasenameExcept is like checkUniqueBasename, but allows an existing
// note at exceptPath to have the same basename (used for moves).
func (a *App) checkUniqueBasenameExcept(relPath, exceptPath string) string {
	if a.db == nil {
		return ""
	}
	basename := filepath.Base(relPath)
	existing, err := a.db.FindNoteByBasename(basename)
	if err != nil || existing == "" || existing == relPath || (exceptPath != "" && existing == exceptPath) {
		return ""
	}
	return fmt.Sprintf("%q already exists at %s", basename, existing)
}

func ensureDir(path string) {
	if err := os.MkdirAll(path, 0755); err != nil {
		// Called during startup; there is no Bubble Tea program to report to yet.
		// Crash loudly rather than continuing in a corrupted state.
		panic(err)
	}
}
