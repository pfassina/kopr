package panel

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTree_GKey_EmptyEntries(t *testing.T) {
	tr := Tree{
		focused: true,
		height:  20,
		width:   30,
	}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
	result, _ := tr.Update(msg)

	if result.cursor != 0 {
		t.Errorf("cursor = %d after G on empty tree, want 0", result.cursor)
	}
}

func TestTree_Enter_EmptyEntries(t *testing.T) {
	tr := Tree{
		focused: true,
		height:  20,
		width:   30,
	}

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	result, cmd := tr.Update(msg)

	if result.cursor != 0 {
		t.Errorf("cursor = %d after enter on empty tree, want 0", result.cursor)
	}
	if cmd != nil {
		t.Error("expected nil cmd for enter on empty tree")
	}
}

func TestTree_JKey_EmptyEntries(t *testing.T) {
	tr := Tree{
		focused: true,
		height:  20,
		width:   30,
	}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	result, _ := tr.Update(msg)

	if result.cursor != 0 {
		t.Errorf("cursor = %d after j on empty tree, want 0", result.cursor)
	}
}
