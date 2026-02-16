package app

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/yourusername/vimvault/internal/editor"
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
		// Only intercept when Neovim is in normal mode
		if a.editor.Mode() != editor.ModeNormal {
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

// Stub methods for actions not yet implemented
func (a *App) ToggleFinder()    {}
func (a *App) CreateDailyNote() {}
func (a *App) CreateInboxNote() {}
func (a *App) InsertTemplate()  {}
func (a *App) FormatDocument()  {}
