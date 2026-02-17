package panel

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PromptResultMsg is sent when the prompt is confirmed.
type PromptResultMsg struct {
	Value string
}

// PromptCancelledMsg is sent when the prompt is dismissed.
type PromptCancelledMsg struct{}

// Prompt is a centered overlay text input dialog.
type Prompt struct {
	input   textinput.Model
	title   string
	width   int
	height  int
	visible bool
}

func NewPrompt() Prompt {
	ti := textinput.New()
	ti.CharLimit = 256
	ti.Width = 40
	ti.Focus()

	return Prompt{input: ti}
}

func (p *Prompt) Show(title, placeholder string) {
	p.visible = true
	p.title = title
	p.input.Placeholder = placeholder
	p.input.SetValue("")
	p.input.Focus()
}

func (p *Prompt) Hide() {
	p.visible = false
	p.input.Blur()
}

func (p Prompt) Visible() bool {
	return p.visible
}

func (p Prompt) Update(msg tea.Msg) (Prompt, tea.Cmd) {
	if !p.visible {
		return p, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			value := strings.TrimSpace(p.input.Value())
			p.visible = false
			if value == "" {
				return p, func() tea.Msg { return PromptCancelledMsg{} }
			}
			return p, func() tea.Msg { return PromptResultMsg{Value: value} }

		case "esc", "ctrl+c":
			p.visible = false
			return p, func() tea.Msg { return PromptCancelledMsg{} }
		}
	}

	var cmd tea.Cmd
	p.input, cmd = p.input.Update(msg)
	return p, cmd
}

func (p Prompt) View() string {
	if !p.visible {
		return ""
	}

	width := p.width
	if width == 0 {
		width = 60
	}
	innerWidth := width - 6

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("212")).
		Padding(0, 1).
		Width(innerWidth)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("212"))

	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	var lines []string
	lines = append(lines, titleStyle.Render(p.title))
	lines = append(lines, p.input.View())
	lines = append(lines, "")
	lines = append(lines, dimStyle.Render("Enter to confirm, Esc to cancel"))

	content := strings.Join(lines, "\n")
	return borderStyle.Render(content)
}

func (p *Prompt) SetSize(width, height int) {
	p.width = width
	p.height = height
	p.input.Width = width/2 - 8
}
