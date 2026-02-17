package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pfassina/kopr/internal/config"
	"github.com/pfassina/kopr/internal/editor"
	"github.com/pfassina/kopr/internal/index"
	"github.com/pfassina/kopr/internal/panel"
	"github.com/pfassina/kopr/internal/session"
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
	kind string // "save", "close", "create-note", "delete-note", "rename-note", "move-note"
	path string // target file path for delete/rename
}

type App struct {
	cfg      config.Config
	editor   editor.Editor
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
	theme    Theme
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
		editor:   editor.New(cfg.VaultPath, editor.ProfileMode(cfg.NvimMode)),
		tree:     t,
		info:     panel.NewInfo(),
		status:   panel.NewStatus(cfg.VaultPath),
		whichKey: panel.NewWhichKey(),
		finder:   f,
		prompt:   panel.NewPrompt(),
		vault:    v,
		store:    store,
		theme:    GetTheme(cfg.Theme),
		focused:  focusEditor,
		showTree: state.ShowTree,
		showInfo: state.ShowInfo,
	}
	a.initLeader()

	// Initialize index
	dbPath := filepath.Join(cfg.VaultPath, ".kopr", "index.db")
	ensureDir(filepath.Dir(dbPath))
	db, err := index.Open(dbPath)
	if err == nil {
		a.db = db
		a.indexer = index.NewIndexer(db, cfg.VaultPath)
		a.finder.SetSearchFunc(a.searchNotes)
	}

	return a
}

func (a *App) SetProgram(p *tea.Program) {
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
		if !(a.focused == focusTree && a.tree.ShowingHelp()) {
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

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.finder.SetSize(msg.Width, msg.Height)
		a.prompt.SetSize(msg.Width, msg.Height)
		cmd := a.updateLayout()
		if cmd != nil {
			return a, cmd
		}

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

	case panel.FinderCreateMsg:
		a.createNoteFromFinder(msg.Name)
		a.setFocus(focusEditor)

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

	case panel.TreeNewNoteMsg:
		a.pendingPrompt = promptAction{kind: "create-note"}
		a.prompt.Show("New note", "my-note.md")
		return a, nil

	case panel.TreeDeleteNoteMsg:
		a.pendingPrompt = promptAction{kind: "delete-note", path: msg.Path}
		a.prompt.Show("Delete "+msg.Name+"?", "type yes to confirm")
		return a, nil

	case panel.TreeRenameNoteMsg:
		a.pendingPrompt = promptAction{kind: "rename-note", path: msg.Path}
		a.prompt.Show("Rename", msg.Name)
		return a, nil

	case panel.TreeMoveNoteMsg:
		a.pendingPrompt = promptAction{kind: "move-note", path: msg.Path}
		dir := filepath.Dir(msg.Path)
		if dir == "." {
			dir = ""
		}
		a.prompt.Show("Move to", dir)
		return a, nil

	case panel.PromptResultMsg:
		return a, a.handlePromptResult(msg.Value)

	case panel.PromptCancelledMsg:
		return a, a.handlePromptCancelled()

	case indexInitDoneMsg:
		// Index is ready - start file watcher
		if a.indexer != nil {
			w, err := index.NewWatcher(a.indexer, a.cfg.VaultPath, func() {
				a.tree.Refresh()
			})
			if err == nil {
				a.watcher = w
				go w.Start()
			}
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

	showTree, showInfo := a.panelsVisible()
	layout := ComputeLayout(a.width, a.height, showTree, showInfo, a.cfg.TreeWidth, a.cfg.InfoWidth)

	// Editor title row
	editorTitle := a.editorTitle()

	editorView := editorTitle + "\n" + a.editor.View()

	var main string

	if !showTree && !showInfo {
		main = editorView
	} else {
		var columns []string

		if showTree {
			borderStyle := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder(), false, true, false, false).
				BorderForeground(lipgloss.Color("240")).
				Width(layout.TreeWidth - 1).
				Height(layout.Height)
			columns = append(columns, borderStyle.Render(a.tree.View()))
		}

		editorStyle := lipgloss.NewStyle().
			Width(layout.EditorWidth).
			Height(layout.Height)
		columns = append(columns, editorStyle.Render(editorView))

		if showInfo {
			borderStyle := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder(), false, false, false, true).
				BorderForeground(lipgloss.Color("240")).
				Width(layout.InfoWidth - 1).
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
			Theme:     a.theme.Name,
		}
		a.store.Save(state)
	}

	a.editor.Close()
	if a.watcher != nil {
		a.watcher.Stop()
	}
	if a.db != nil {
		a.db.Close()
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
		rpc.ExecCommand("w")
	}

	a.showSplash()
	return nil
}

// showSplash transitions the editor to the splash screen.
func (a *App) showSplash() {
	rpc := a.editor.GetRPC()
	if rpc != nil {
		rpc.LoadSplashBuffer()
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
	a.editor.OpenFile(path)
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
	a.pendingPrompt = promptAction{}

	switch action.kind {
	case "save", "close":
		return a.handleSaveAs(value, action.kind == "close")
	case "create-note":
		return a.handleCreateNote(value)
	case "delete-note":
		return a.handleDeleteNote(value, action.path)
	case "rename-note":
		return a.handleRenameNote(value, action.path)
	case "move-note":
		return a.handleMoveNote(value, action.path)
	}
	return nil
}

// handleSaveAs saves an unnamed buffer with the given name.
func (a *App) handleSaveAs(value string, closeAfter bool) tea.Cmd {
	relPath := value
	if !strings.HasSuffix(relPath, ".md") {
		relPath += ".md"
	}

	rpc := a.editor.GetRPC()
	if rpc == nil {
		return nil
	}

	// Get current buffer content before setting name
	content, err := rpc.BufferContent()
	if err != nil {
		return nil
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
		return nil
	}

	// Make the buffer modifiable, set name, and write
	rpc.ExecCommand("setlocal modifiable")
	rpc.SetBufferName(fullPath)
	// Remove BufWriteCmd interference by setting buftype back to normal
	rpc.ExecCommand("setlocal buftype=")
	rpc.WriteBuffer()

	a.status.SetFile(relPath)
	a.currentFile = relPath
	a.tree.Refresh()
	a.updateBacklinks(relPath)

	if closeAfter {
		a.showSplash()
	}

	return nil
}

// handleCreateNote creates a new empty note or directory from the tree panel.
func (a *App) handleCreateNote(name string) tea.Cmd {
	// Directory creation: name ends with /
	if strings.HasSuffix(name, "/") {
		dirPath := strings.TrimSuffix(name, "/")
		a.vault.CreateDir(dirPath)
		a.tree.Refresh()
		return nil
	}

	relPath := name
	if !strings.HasSuffix(relPath, ".md") {
		relPath += ".md"
	}

	content := fmt.Sprintf("---\ntitle: %s\n---\n\n", strings.TrimSuffix(name, ".md"))
	fullPath, err := a.vault.CreateNote(relPath, content)
	if err != nil {
		return nil
	}

	a.openInEditor(fullPath)
	a.status.SetFile(relPath)
	a.currentFile = relPath
	a.tree.Refresh()
	a.setFocus(focusEditor)
	return nil
}

// handleMoveNote moves a note to a new directory.
func (a *App) handleMoveNote(newDir, oldPath string) tea.Cmd {
	if err := a.vault.MoveNote(oldPath, newDir); err != nil {
		return nil
	}

	newRel := filepath.Join(newDir, filepath.Base(oldPath))

	// If the moved file is currently open, update the editor
	if a.currentFile == oldPath {
		fullPath := filepath.Join(a.cfg.VaultPath, newRel)
		rpc := a.editor.GetRPC()
		if rpc != nil {
			rpc.SetBufferName(fullPath)
			rpc.WriteBuffer()
		}
		a.status.SetFile(newRel)
		a.currentFile = newRel
	}

	a.tree.Refresh()
	return nil
}

// handleDeleteNote deletes a note after confirmation.
func (a *App) handleDeleteNote(confirmation, relPath string) tea.Cmd {
	if strings.ToLower(strings.TrimSpace(confirmation)) != "yes" {
		return nil
	}

	// If the deleted file is currently open, go to splash
	if a.currentFile == relPath {
		a.showSplash()
	}

	a.vault.DeleteNote(relPath)
	a.tree.Refresh()
	return nil
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

	if err := a.vault.RenameNote(oldPath, newRel); err != nil {
		return nil
	}

	// If the renamed file is currently open, update the editor
	if a.currentFile == oldPath {
		fullPath := filepath.Join(a.cfg.VaultPath, newRel)
		rpc := a.editor.GetRPC()
		if rpc != nil {
			rpc.SetBufferName(fullPath)
			rpc.WriteBuffer()
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

func (a *App) updateLayout() tea.Cmd {
	showTree, showInfo := a.panelsVisible()
	layout := ComputeLayout(a.width, a.height, showTree, showInfo, a.cfg.TreeWidth, a.cfg.InfoWidth)

	a.tree.SetSize(layout.TreeWidth, layout.Height)
	a.info.SetSize(layout.InfoWidth, layout.Height)
	a.status.SetWidth(a.width)
	a.whichKey.SetWidth(a.width / 2)

	editorSize := tea.WindowSizeMsg{
		Width:  layout.EditorWidth,
		Height: layout.Height - 1, // -1 for editor title row
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
			Foreground(lipgloss.Color("212")).
			Underline(true).
			Padding(0, 1)
	} else {
		style = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("240")).
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

	for i, overlayLine := range overlayLines {
		row := startRow + i
		if row >= len(baseLines) {
			break
		}
		baseLine := baseLines[row]
		baseRunes := []rune(baseLine)

		for len(baseRunes) < startCol+len([]rune(overlayLine)) {
			baseRunes = append(baseRunes, ' ')
		}

		overlayRunes := []rune(overlayLine)
		for j, r := range overlayRunes {
			if startCol+j < len(baseRunes) {
				baseRunes[startCol+j] = r
			}
		}

		baseLines[row] = string(baseRunes)
	}

	return strings.Join(baseLines, "\n")
}

func ensureDir(path string) {
	os.MkdirAll(path, 0755)
}
