package panel

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/pfassina/kopr/internal/theme"
)

// Status is the status bar at the bottom.
type Status struct {
	width     int
	mode      string
	file      string
	vaultDir  string
	clipboard string
	errMsg    string
	theme     *theme.Theme
}

// SetTheme sets the color theme for the status bar.
func (s *Status) SetTheme(th *theme.Theme) { s.theme = th }

func NewStatus(vaultDir string) Status {
	return Status{
		vaultDir: vaultDir,
		mode:     "NORMAL",
	}
}

func (s *Status) SetMode(mode string) {
	s.mode = mode
}

func (s *Status) SetFile(file string) {
	s.file = file
}

func (s *Status) SetWidth(width int) {
	s.width = width
}

func (s *Status) SetClipboard(label string) {
	s.clipboard = label
}

func (s *Status) SetError(msg string) {
	s.errMsg = msg
}

func (s *Status) ClearError() {
	s.errMsg = ""
}

func (s Status) View() string {
	if s.width == 0 {
		return ""
	}

	th := s.theme

	bgStyle := lipgloss.NewStyle().
		Background(th.StatusBg)

	modeColors := map[string]lipgloss.Color{
		"NORMAL":  th.NormalMode,
		"INSERT":  th.InsertMode,
		"VISUAL":  th.VisualMode,
		"COMMAND": th.CmdMode,
		"REPLACE": th.Error,
	}

	color, ok := modeColors[s.mode]
	if !ok {
		color = th.Text
	}

	modeStyle := lipgloss.NewStyle().
		Background(color).
		Foreground(lipgloss.Color("0")).
		Bold(true).
		Padding(0, 1)

	fileStyle := lipgloss.NewStyle().
		Background(th.StatusBg).
		Foreground(th.StatusFg).
		Padding(0, 1)

	mode := modeStyle.Render(s.mode)

	var fileSection string
	if s.errMsg != "" {
		errStyle := lipgloss.NewStyle().
			Background(th.StatusBg).
			Foreground(th.Error).
			Padding(0, 1)
		fileSection = errStyle.Render(s.errMsg)
	} else {
		file := s.file
		if file == "" {
			file = s.vaultDir
		}
		fileSection = fileStyle.Render(file)
	}

	left := fmt.Sprintf("%s %s", mode, fileSection)

	right := ""
	if s.clipboard != "" {
		clipStyle := lipgloss.NewStyle().
			Background(th.StatusBg).
			Foreground(th.StatusFg).
			Padding(0, 1)
		right = clipStyle.Render(s.clipboard)
	}

	padLen := s.width - lipgloss.Width(left) - lipgloss.Width(right)
	if padLen < 0 {
		padLen = 0
	}
	padding := bgStyle.Render(strings.Repeat(" ", padLen))

	return left + padding + right
}
