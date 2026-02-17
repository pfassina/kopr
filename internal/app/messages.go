package app

import tea "github.com/charmbracelet/bubbletea"

// fatalErrorMsg is sent to the Bubble Tea program when a background subsystem
// encounters an unrecoverable error. The app should quit and show the error.
type fatalErrorMsg struct{ err error }

func fatalCmd(err error) tea.Cmd {
	return tea.Batch(tea.Printf("fatal: %v\n", err), tea.Quit)
}
