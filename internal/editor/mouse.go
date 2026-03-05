package editor

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// EditorMouseMsg is a mouse event with coordinates already translated
// to the Neovim buffer area (0-based).
type EditorMouseMsg struct {
	tea.MouseMsg
	Col int // 0-based column relative to Neovim
	Row int // 0-based row relative to Neovim
}

// mouseMsgToBytes converts a Bubble Tea mouse message to SGR 1006 escape
// sequences suitable for writing to a Neovim PTY.
// col and row are 0-based coordinates relative to the Neovim buffer area.
// Returns nil if the event cannot be translated.
func mouseMsgToBytes(msg tea.MouseMsg, col, row int, lastButton tea.MouseButton) []byte {
	var button int
	switch msg.Button {
	case tea.MouseButtonLeft:
		button = 0
	case tea.MouseButtonMiddle:
		button = 1
	case tea.MouseButtonRight:
		button = 2
	case tea.MouseButtonWheelUp:
		button = 64
	case tea.MouseButtonWheelDown:
		button = 65
	case tea.MouseButtonWheelLeft:
		button = 66
	case tea.MouseButtonWheelRight:
		button = 67
	case tea.MouseButtonNone:
		// On release, Bubble Tea reports MouseButtonNone.
		// Use the last pressed button to encode the release correctly.
		switch lastButton {
		case tea.MouseButtonLeft:
			button = 0
		case tea.MouseButtonMiddle:
			button = 1
		case tea.MouseButtonRight:
			button = 2
		default:
			button = 0
		}
	default:
		return nil
	}

	// Add motion flag for drag events
	if msg.Action == tea.MouseActionMotion {
		button |= 32
	}

	// Modifier flags
	if msg.Shift {
		button |= 4
	}
	if msg.Alt {
		button |= 8
	}
	if msg.Ctrl {
		button |= 16
	}

	// SGR uses 1-based coordinates
	sgrCol := col + 1
	sgrRow := row + 1

	// Suffix: 'M' for press/motion, 'm' for release
	suffix := 'M'
	if msg.Action == tea.MouseActionRelease {
		suffix = 'm'
	}

	return fmt.Appendf(nil, "\x1b[<%d;%d;%d%c", button, sgrCol, sgrRow, suffix)
}
