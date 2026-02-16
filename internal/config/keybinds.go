package config

// Keybind represents a key binding configuration.
type Keybind struct {
	Sequence string
	Action   string
}

// DefaultKeybinds returns the default leader key bindings.
func DefaultKeybinds() []Keybind {
	return []Keybind{
		{Sequence: "Space Space", Action: "finder"},
		{Sequence: "Space f n", Action: "find_note"},
		{Sequence: "Space n d", Action: "daily_note"},
		{Sequence: "Space n i", Action: "inbox_note"},
		{Sequence: "Space n r", Action: "rename_note"},
		{Sequence: "Space t i", Action: "insert_template"},
		{Sequence: "Space v t", Action: "toggle_tree"},
		{Sequence: "Space v b", Action: "toggle_backlinks"},
		{Sequence: "Space v s", Action: "toggle_status"},
		{Sequence: "Space z z", Action: "zen_mode"},
		{Sequence: "Space m f", Action: "format_document"},
	}
}
