package editor

import (
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/x/vt"
)

type vtScreen struct {
	term       *vt.SafeEmulator
	done       chan struct{}
	showCursor bool
}

// newVTScreen creates a VT emulator and starts a goroutine that drains
// terminal responses (DA1, DECRQM, etc.) back to the PTY. Without this,
// the emulator's internal io.Pipe blocks on Write when nvim sends queries.
func newVTScreen(width, height int, ptyFile *os.File) *vtScreen {
	term := vt.NewSafeEmulator(width, height)
	done := make(chan struct{})

	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := term.Read(buf)
			if n > 0 {
				ptyFile.Write(buf[:n])
			}
			if err != nil {
				return
			}
			select {
			case <-done:
				return
			default:
			}
		}
	}()

	return &vtScreen{term: term, done: done, showCursor: true}
}

func (v *vtScreen) write(p []byte) (int, error) {
	return v.term.Write(p)
}

func (v *vtScreen) resize(width, height int) {
	v.term.Resize(width, height)
}

func (v *vtScreen) render() string {
	rendered := v.term.Render()
	// Render() uses \r\n; Bubble Tea expects \n
	rendered = strings.ReplaceAll(rendered, "\r\n", "\n")
	if v.showCursor {
		pos := v.term.CursorPosition()
		return overlayCursor(rendered, pos.X, pos.Y)
	}
	return rendered
}

func (v *vtScreen) setShowCursor(show bool) {
	v.showCursor = show
}

func (v *vtScreen) close() error {
	close(v.done)
	return v.term.Close()
}

// overlayCursor inserts a reverse-video block at the cursor position.
func overlayCursor(s string, cx, cy int) string {
	lines := strings.Split(s, "\n")
	if cy < 0 || cy >= len(lines) {
		return s
	}
	lines[cy] = insertCursor(lines[cy], cx)
	return strings.Join(lines, "\n")
}

// insertCursor adds reverse video at visual column col, skipping ANSI escapes.
func insertCursor(line string, col int) string {
	runes := []rune(line)
	vcol := 0
	i := 0

	for i < len(runes) {
		// Skip ANSI escape sequences
		if runes[i] == 0x1b {
			i++
			if i < len(runes) && runes[i] == '[' {
				// CSI sequence: skip until final byte (0x40-0x7E)
				i++
				for i < len(runes) && runes[i] >= 0x20 && runes[i] < 0x40 {
					i++
				}
				if i < len(runes) {
					i++ // skip final byte
				}
			} else if i < len(runes) {
				i++ // simple ESC + one char
			}
			continue
		}

		if vcol == col {
			ch := string(runes[i])
			before := string(runes[:i])
			after := string(runes[i+1:])
			return before + "\x1b[7m" + ch + "\x1b[27m" + after
		}

		vcol++
		i++
	}

	// Cursor past end of line — append a reversed space
	pad := strings.Repeat(" ", col-vcol)
	return line + pad + "\x1b[7m \x1b[27m"
}

// Compile-time check that SafeEmulator implements io.Reader.
var _ io.Reader = (*vt.SafeEmulator)(nil)
