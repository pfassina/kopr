package app

import (
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/yourusername/vimvault/internal/config"
	"github.com/yourusername/vimvault/internal/editor"
	"github.com/yourusername/vimvault/internal/index"
	"github.com/yourusername/vimvault/internal/panel"
	"github.com/yourusername/vimvault/internal/vault"
)

type focusedPanel int

const (
	focusEditor focusedPanel = iota
	focusTree
	focusInfo
	focusFinder
)

type App struct {
	cfg      config.Config
	editor   editor.Editor
	tree     panel.Tree
	info     panel.Info
	status   panel.Status
	whichKey panel.WhichKey
	finder   panel.Finder
	vault    *vault.Vault
	db       *index.DB
	indexer  *index.Indexer
	watcher  *index.Watcher
	width    int
	height   int
	focused  focusedPanel
	showTree bool
	showInfo bool
	zenMode  bool

	// Leader key system
	bindings map[string]*Binding
	leader   LeaderState
}

func New(cfg config.Config) App {
	v := vault.New(cfg.VaultPath)
	t := panel.NewTree(v)
	t.Refresh()

	f := panel.NewFinder()

	a := App{
		cfg:      cfg,
		editor:   editor.New(cfg.VaultPath),
		tree:     t,
		info:     panel.NewInfo(),
		status:   panel.NewStatus(cfg.VaultPath),
		whichKey: panel.NewWhichKey(),
		finder:   f,
		vault:    v,
		focused:  focusEditor,
		showTree: cfg.ShowTree,
		showInfo: cfg.ShowInfo,
	}
	a.initLeader()

	// Initialize index
	dbPath := filepath.Join(cfg.VaultPath, ".vimvault", "index.db")
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

		// Finder takes priority when visible
		if a.finder.Visible() {
			var cmd tea.Cmd
			a.finder, cmd = a.finder.Update(msg)
			return a, cmd
		}

		// Try leader key system
		if consumed, cmd := a.handleLeaderKey(msg.String()); consumed {
			a.updateWhichKey()
			return a, cmd
		}

	case leaderTimeoutMsg:
		a.handleLeaderTimeout()
		a.updateWhichKey()
		return a, nil

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.finder.SetSize(msg.Width, msg.Height)
		a.updateLayout()

	case editor.ModeChangedMsg:
		a.status.SetMode(modeDisplayName(msg.Mode))
		if msg.Mode != editor.ModeNormal {
			a.cancelLeader()
			a.updateWhichKey()
		}

	case panel.FileSelectedMsg:
		fullPath := filepath.Join(a.cfg.VaultPath, msg.Path)
		a.editor.OpenFile(fullPath)
		a.status.SetFile(msg.Path)
		a.focused = focusEditor
		a.tree.SetFocused(false)

	case panel.FinderResultMsg:
		a.handleFinderResult(msg.Path)
		a.focused = focusEditor

	case panel.FinderClosedMsg:
		a.focused = focusEditor

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

	layout := ComputeLayout(a.width, a.height, a.showTree && !a.zenMode, a.showInfo && !a.zenMode, a.cfg.TreeWidth, a.cfg.InfoWidth)

	editorView := a.editor.View()

	var main string

	if a.zenMode || (!a.showTree && !a.showInfo) {
		main = editorView
	} else {
		var columns []string

		if a.showTree {
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

		if a.showInfo {
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

	return result
}

func (a *App) Close() {
	a.editor.Close()
	if a.watcher != nil {
		a.watcher.Stop()
	}
	if a.db != nil {
		a.db.Close()
	}
}

func (a *App) updateLayout() {
	layout := ComputeLayout(a.width, a.height, a.showTree && !a.zenMode, a.showInfo && !a.zenMode, a.cfg.TreeWidth, a.cfg.InfoWidth)

	a.tree.SetSize(layout.TreeWidth, layout.Height)
	a.info.SetSize(layout.InfoWidth, layout.Height)
	a.status.SetWidth(a.width)
	a.whichKey.SetWidth(a.width / 2)

	editorSize := tea.WindowSizeMsg{
		Width:  layout.EditorWidth,
		Height: layout.Height,
	}
	a.editor, _ = a.editor.Update(editorSize)
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

func (a *App) ToggleTree() {
	a.showTree = !a.showTree
	a.updateLayout()
}

func (a *App) ToggleInfo() {
	a.showInfo = !a.showInfo
	a.updateLayout()
}

func (a *App) ToggleZen() {
	a.zenMode = !a.zenMode
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
