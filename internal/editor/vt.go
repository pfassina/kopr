package editor

import (
	"strings"

	"github.com/charmbracelet/x/vt"
)

type vtScreen struct {
	term *vt.SafeEmulator
}

func newVTScreen(width, height int) *vtScreen {
	return &vtScreen{
		term: vt.NewSafeEmulator(width, height),
	}
}

func (v *vtScreen) write(p []byte) (int, error) {
	return v.term.Write(p)
}

func (v *vtScreen) resize(width, height int) {
	v.term.Resize(width, height)
}

func (v *vtScreen) render() string {
	return v.term.Render()
}

func (v *vtScreen) renderWithCursor() string {
	rendered := v.term.Render()
	pos := v.term.CursorPosition()

	lines := strings.Split(rendered, "\n")
	if pos.Y < 0 || pos.Y >= len(lines) {
		return rendered
	}

	// Add cursor position escape sequence at the end
	// so the terminal cursor appears at the right place
	return rendered + cursorPosition(pos.X, pos.Y)
}

// cursorPosition returns an ANSI sequence to move cursor to x,y within
// the rendered content. This is relative to where Bubble Tea places the view.
func cursorPosition(x, y int) string {
	// We can't use absolute positioning since Bubble Tea manages the screen.
	// Instead, we'll rely on the terminal's own cursor rendering.
	// For now, return empty - cursor will be handled by adding reverse video
	// to the character at the cursor position.
	return ""
}

func (v *vtScreen) close() error {
	return v.term.Close()
}
