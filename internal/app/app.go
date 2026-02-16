package app

import (
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/yourusername/vimvault/internal/config"
	"github.com/yourusername/vimvault/internal/editor"
	"github.com/yourusername/vimvault/internal/panel"
	"github.com/yourusername/vimvault/internal/vault"
)

type focusedPanel int

const (
	focusEditor focusedPanel = iota
	focusTree
	focusInfo
)

type App struct {
	cfg      config.Config
	editor   editor.Editor
	tree     panel.Tree
	info     panel.Info
	status   panel.Status
	vault    *vault.Vault
	width    int
	height   int
	focused  focusedPanel
	showTree bool
	showInfo bool
	zenMode  bool
}

func New(cfg config.Config) App {
	v := vault.New(cfg.VaultPath)
	t := panel.NewTree(v)
	t.Refresh()

	return App{
		cfg:      cfg,
		editor:   editor.New(cfg.VaultPath),
		tree:     t,
		info:     panel.NewInfo(),
		status:   panel.NewStatus(cfg.VaultPath),
		vault:    v,
		focused:  focusEditor,
		showTree: cfg.ShowTree,
		showInfo: cfg.ShowInfo,
	}
}

func (a *App) SetProgram(p *tea.Program) {
	a.editor.SetProgram(p)
}

func (a *App) Init() tea.Cmd {
	return a.editor.Init()
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			a.editor.Close()
			return a, tea.Quit
		}

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.updateLayout()

	case editor.ModeChangedMsg:
		a.status.SetMode(modeDisplayName(msg.Mode))

	case panel.FileSelectedMsg:
		fullPath := filepath.Join(a.cfg.VaultPath, msg.Path)
		a.editor.OpenFile(fullPath)
		a.status.SetFile(msg.Path)
		a.focused = focusEditor
		a.tree.SetFocused(false)
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

	if a.zenMode || (!a.showTree && !a.showInfo) {
		return editorView + "\n" + a.status.View()
	}

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

	main := lipgloss.JoinHorizontal(lipgloss.Top, columns...)
	return main + "\n" + a.status.View()
}

func (a *App) updateLayout() {
	layout := ComputeLayout(a.width, a.height, a.showTree && !a.zenMode, a.showInfo && !a.zenMode, a.cfg.TreeWidth, a.cfg.InfoWidth)

	a.tree.SetSize(layout.TreeWidth, layout.Height)
	a.info.SetSize(layout.InfoWidth, layout.Height)
	a.status.SetWidth(a.width)

	// Resize editor to its allocated space
	editorSize := tea.WindowSizeMsg{
		Width:  layout.EditorWidth,
		Height: layout.Height,
	}
	a.editor, _ = a.editor.Update(editorSize)
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
