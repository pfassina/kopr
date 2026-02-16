package panel

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Status is the status bar at the bottom.
type Status struct {
	width    int
	mode     string
	file     string
	vaultDir string
}

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

func (s Status) View() string {
	if s.width == 0 {
		return ""
	}

	modeColors := map[string]lipgloss.Color{
		"NORMAL":  lipgloss.Color("212"),
		"INSERT":  lipgloss.Color("114"),
		"VISUAL":  lipgloss.Color("216"),
		"COMMAND": lipgloss.Color("75"),
		"REPLACE": lipgloss.Color("203"),
	}

	color, ok := modeColors[s.mode]
	if !ok {
		color = lipgloss.Color("252")
	}

	modeStyle := lipgloss.NewStyle().
		Background(color).
		Foreground(lipgloss.Color("0")).
		Bold(true).
		Padding(0, 1)

	fileStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("252")).
		Padding(0, 1)

	mode := modeStyle.Render(s.mode)
	file := s.file
	if file == "" {
		file = s.vaultDir
	}
	fileSection := fileStyle.Render(file)

	left := fmt.Sprintf("%s %s", mode, fileSection)

	// Pad the rest with background
	bgStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("236"))

	padLen := s.width - lipgloss.Width(left)
	if padLen < 0 {
		padLen = 0
	}
	padding := bgStyle.Render(strings.Repeat(" ", padLen))

	return left + padding
}
