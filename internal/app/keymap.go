package app

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pfassina/kopr/internal/editor"
	"github.com/pfassina/kopr/internal/markdown"
)

// Binding represents a leader key binding.
type Binding struct {
	Key         string
	Label       string
	Action      func(a *App) tea.Cmd
	Children    map[string]*Binding
}

// LeaderState tracks the leader key sequence.
type LeaderState struct {
	active   bool
	keys     string
	node     map[string]*Binding
	timer    *time.Timer
	showHelp bool
}

// leaderTimeoutMsg signals leader key timeout.
type leaderTimeoutMsg struct{}

func newBindings() map[string]*Binding {
	return map[string]*Binding{
		" ": {
			Key: "Space", Label: "Fuzzy finder",
			Action: func(a *App) tea.Cmd {
				a.ToggleFinder()
				return nil
			},
		},
		"f": {
			Key: "f", Label: "+find",
			Children: map[string]*Binding{
				"n": {Key: "n", Label: "Find/create note", Action: func(a *App) tea.Cmd {
					a.ToggleFinder()
					return nil
				}},
			},
		},
		"n": {
			Key: "n", Label: "+note",
			Children: map[string]*Binding{
				"n": {Key: "n", Label: "New note", Action: func(a *App) tea.Cmd {
					a.CreateBlankNote()
					return nil
				}},
				"d": {Key: "d", Label: "Daily note", Action: func(a *App) tea.Cmd {
					a.CreateDailyNote()
					return nil
				}},
				"i": {Key: "i", Label: "Inbox capture", Action: func(a *App) tea.Cmd {
					a.CreateInboxNote()
					return nil
				}},
				"r": {Key: "r", Label: "Rename note", Action: func(a *App) tea.Cmd {
					return nil // TODO
				}},
			},
		},
		"t": {
			Key: "t", Label: "+template",
			Children: map[string]*Binding{
				"i": {Key: "i", Label: "Insert template", Action: func(a *App) tea.Cmd {
					a.InsertTemplate()
					return nil
				}},
			},
		},
		"v": {
			Key: "v", Label: "+view",
			Children: map[string]*Binding{
				"t": {Key: "t", Label: "Toggle tree", Action: func(a *App) tea.Cmd {
					a.ToggleTree()
					return nil
				}},
				"b": {Key: "b", Label: "Toggle backlinks", Action: func(a *App) tea.Cmd {
					a.ToggleInfo()
					return nil
				}},
				"s": {Key: "s", Label: "Toggle status", Action: func(a *App) tea.Cmd {
					return nil // TODO
				}},
			},
		},
		"z": {
			Key: "z", Label: "+zen",
			Children: map[string]*Binding{
				"z": {Key: "z", Label: "Zen mode", Action: func(a *App) tea.Cmd {
					a.ToggleZen()
					return nil
				}},
			},
		},
		"q": {
			Key: "q", Label: "+quit",
			Children: map[string]*Binding{
				"q": {Key: "q", Label: "Quit Kopr", Action: func(a *App) tea.Cmd {
					a.Close()
					return tea.Quit
				}},
			},
		},
		"m": {
			Key: "m", Label: "+markdown",
			Children: map[string]*Binding{
				"f": {Key: "f", Label: "Format document", Action: func(a *App) tea.Cmd {
					a.FormatDocument()
					return nil
				}},
			},
		},
	}
}

func (a *App) initLeader() {
	a.bindings = newBindings()
	a.leader = LeaderState{}
}

// handleLeaderKey processes a key during leader mode.
// Returns true if the key was consumed by the leader system.
func (a *App) handleLeaderKey(key string) (consumed bool, cmd tea.Cmd) {
	// Only intercept Space in normal mode when not in leader mode
	if !a.leader.active {
		if key != " " {
			return false, nil
		}
		// Only check Neovim mode when editor is focused
		if a.focused == focusEditor && a.editor.Mode() != editor.ModeNormal {
			return false, nil
		}
		a.leader.active = true
		a.leader.keys = ""
		a.leader.node = a.bindings
		a.leader.showHelp = false
		// Start timeout for which-key popup
		return true, tea.Tick(time.Duration(a.cfg.LeaderTimeout)*time.Millisecond, func(time.Time) tea.Msg {
			return leaderTimeoutMsg{}
		})
	}

	// We're in leader mode - accumulate the key
	a.leader.keys += key

	if binding, ok := a.leader.node[key]; ok {
		if binding.Children != nil {
			// This is a group - wait for next key
			a.leader.node = binding.Children
			a.leader.showHelp = false
			return true, tea.Tick(time.Duration(a.cfg.LeaderTimeout)*time.Millisecond, func(time.Time) tea.Msg {
				return leaderTimeoutMsg{}
			})
		}
		// Leaf binding - execute
		a.leader.active = false
		a.leader.showHelp = false
		if binding.Action != nil {
			return true, binding.Action(a)
		}
		return true, nil
	}

	// No match - cancel leader mode
	a.leader.active = false
	a.leader.showHelp = false
	return true, nil
}

func (a *App) handleLeaderTimeout() {
	if a.leader.active {
		a.leader.showHelp = true
	}
}

func (a *App) cancelLeader() {
	a.leader.active = false
	a.leader.showHelp = false
}

func (a *App) ToggleFinder() {
	if a.finder.Visible() {
		a.finder.Hide()
		a.focused = focusEditor
	} else {
		a.finder.Show()
		a.focused = focusFinder
	}
}

func (a *App) CreateBlankNote() {
	rpc := a.editor.GetRPC()
	if rpc == nil {
		return
	}
	rpc.NewBuffer()
	a.editor.SetShowSplash(false)
	a.currentFile = ""
	a.status.SetFile("")
	a.updateLayout()
}

func (a *App) CreateDailyNote() {
	path, err := a.vault.CreateDailyNote()
	if err != nil {
		return
	}
	a.openInEditor(path)
	rel, _ := filepath.Rel(a.cfg.VaultPath, path)
	a.status.SetFile(rel)
	a.currentFile = rel
	a.tree.Refresh()
}

func (a *App) CreateInboxNote() {
	path, err := a.vault.CreateInboxNote()
	if err != nil {
		return
	}
	a.openInEditor(path)
	rel, _ := filepath.Rel(a.cfg.VaultPath, path)
	a.status.SetFile(rel)
	a.currentFile = rel
	a.tree.Refresh()
}

func (a *App) InsertTemplate() {
	templates, err := a.vault.LoadTemplates()
	if err != nil || len(templates) == 0 {
		return
	}
	// For now, use the first template. A template picker UI can be added later.
	if len(templates) > 0 {
		path, err := a.vault.CreateFromTemplate(templates[0], "New Note")
		if err != nil {
			return
		}
		a.openInEditor(path)
		rel, _ := filepath.Rel(a.cfg.VaultPath, path)
		a.status.SetFile(rel)
		a.currentFile = rel
		a.tree.Refresh()
	}
}

func (a *App) FormatDocument() {
	rpc := a.editor.GetRPC()
	if rpc == nil {
		return
	}

	// Get current buffer content
	content, err := rpc.BufferContent()
	if err != nil {
		return
	}

	// Join lines
	var buf bytes.Buffer
	for i, line := range content {
		buf.Write(line)
		if i < len(content)-1 {
			buf.WriteByte('\n')
		}
	}

	// Format
	formatted := markdown.Format(buf.Bytes())

	// Write back via RPC - use Neovim's command to replace buffer
	lines := strings.Split(string(formatted), "\n")
	// Remove trailing empty line added by Format()
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	// Build a Lua command to set buffer lines
	luaLines := make([]string, len(lines))
	for i, l := range lines {
		// Escape for Lua string
		l = strings.ReplaceAll(l, "\\", "\\\\")
		l = strings.ReplaceAll(l, "'", "\\'")
		luaLines[i] = "'" + l + "'"
	}

	lua := fmt.Sprintf("vim.api.nvim_buf_set_lines(0, 0, -1, false, {%s})", strings.Join(luaLines, ","))
	rpc.ExecLua(lua, nil)
}
