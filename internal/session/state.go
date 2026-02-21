package session

// State represents persisted session state.
type State struct {
	ActiveFile string   `json:"active_file,omitempty"`
	OpenFiles  []string `json:"open_files,omitempty"`
	ShowTree   bool     `json:"show_tree"`
	ShowInfo   bool     `json:"show_info"`
	TreeWidth  int      `json:"tree_width,omitempty"`
	InfoWidth  int      `json:"info_width,omitempty"`
}

// Default returns the default session state.
func Default() State {
	return State{
		ShowTree:  true,
		ShowInfo:  true,
		TreeWidth: 30,
		InfoWidth: 30,
	}
}
