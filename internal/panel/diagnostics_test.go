package panel

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pfassina/kopr/internal/theme"
)

func newTestDiagnostics(items []DiagnosticItem) Diagnostics {
	d := NewDiagnostics()
	th := theme.DefaultTheme()
	d.SetTheme(&th)
	d.SetSize(60, 6)
	d.SetFocused(true)
	d.SetItems(items)
	d.Show()
	return d
}

func keyMsg(s string) tea.KeyMsg {
	if s == "enter" {
		return tea.KeyMsg{Type: tea.KeyEnter}
	}
	if s == "esc" {
		return tea.KeyMsg{Type: tea.KeyEsc}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestDiagnosticsNavigationBounds(t *testing.T) {
	d := newTestDiagnostics([]DiagnosticItem{
		{Line: 1, Col: 0, Severity: "error", Message: "first"},
		{Line: 5, Col: 2, Severity: "warning", Message: "second"},
	})

	// k at top stays at 0
	d, _ = d.Update(keyMsg("k"))
	if d.cursor != 0 {
		t.Errorf("cursor = %d after k at top, want 0", d.cursor)
	}

	// j moves down, then stops at last item
	d, _ = d.Update(keyMsg("j"))
	d, _ = d.Update(keyMsg("j"))
	if d.cursor != 1 {
		t.Errorf("cursor = %d after j past end, want 1", d.cursor)
	}
}

func TestDiagnosticsEnterEmitsGoto(t *testing.T) {
	d := newTestDiagnostics([]DiagnosticItem{
		{Line: 3, Col: 7, Severity: "error", Message: "boom"},
		{Line: 9, Col: 1, Severity: "warning", Message: "hmm"},
	})

	d, _ = d.Update(keyMsg("j"))
	_, cmd := d.Update(keyMsg("enter"))
	if cmd == nil {
		t.Fatal("enter returned nil cmd")
	}
	msg, ok := cmd().(DiagnosticsGotoMsg)
	if !ok {
		t.Fatalf("got %T, want DiagnosticsGotoMsg", cmd())
	}
	if msg.Line != 9 || msg.Col != 1 {
		t.Errorf("goto = %d:%d, want 9:1", msg.Line, msg.Col)
	}
}

func TestDiagnosticsEscEmitsClosed(t *testing.T) {
	d := newTestDiagnostics(nil)
	_, cmd := d.Update(keyMsg("esc"))
	if cmd == nil {
		t.Fatal("esc returned nil cmd")
	}
	if _, ok := cmd().(DiagnosticsClosedMsg); !ok {
		t.Fatalf("got %T, want DiagnosticsClosedMsg", cmd())
	}
}

func TestDiagnosticsIgnoresKeysWhenUnfocused(t *testing.T) {
	d := newTestDiagnostics([]DiagnosticItem{
		{Line: 1, Col: 0, Severity: "error", Message: "x"},
	})
	d.SetFocused(false)
	_, cmd := d.Update(keyMsg("enter"))
	if cmd != nil {
		t.Error("unfocused panel responded to enter")
	}
}

func TestDiagnosticsViewCountsAndRows(t *testing.T) {
	d := newTestDiagnostics([]DiagnosticItem{
		{Line: 2, Col: 4, Severity: "error", Message: "bad frontmatter"},
		{Line: 7, Col: 0, Severity: "warning", Message: "dup heading"},
	})
	view := d.View()
	if !strings.Contains(view, "1 error, 1 warning") {
		t.Errorf("view missing counts:\n%s", view)
	}
	if !strings.Contains(view, "bad frontmatter") || !strings.Contains(view, "dup heading") {
		t.Errorf("view missing items:\n%s", view)
	}
	if !strings.Contains(view, "2:5") {
		t.Errorf("view missing 1-based position 2:5:\n%s", view)
	}
}

func TestDiagnosticsViewEmptyState(t *testing.T) {
	d := newTestDiagnostics(nil)
	view := d.View()
	if !strings.Contains(view, "No diagnostics") {
		t.Errorf("view missing empty state:\n%s", view)
	}
}

func TestDiagnosticsViewTruncatesLongMessages(t *testing.T) {
	long := strings.Repeat("x", 200)
	d := newTestDiagnostics([]DiagnosticItem{
		{Line: 1, Col: 0, Severity: "error", Message: long},
	})
	for _, line := range strings.Split(d.View(), "\n") {
		if len(line) > 400 { // generous: includes ANSI codes
			t.Errorf("line too long (%d): %q", len(line), line)
		}
	}
	if !strings.Contains(d.View(), "...") {
		t.Errorf("long message not truncated:\n%s", d.View())
	}
}

func TestDiagnosticsPreferredHeight(t *testing.T) {
	d := newTestDiagnostics(nil)
	if got := d.PreferredHeight(); got != 2 {
		t.Errorf("empty PreferredHeight = %d, want 2", got)
	}

	items := make([]DiagnosticItem, 3)
	for i := range items {
		items[i] = DiagnosticItem{Line: i + 1, Severity: "error", Message: "m"}
	}
	d.SetItems(items)
	if got := d.PreferredHeight(); got != 4 {
		t.Errorf("3-item PreferredHeight = %d, want 4", got)
	}

	items = make([]DiagnosticItem, 20)
	for i := range items {
		items[i] = DiagnosticItem{Line: i + 1, Severity: "error", Message: "m"}
	}
	d.SetItems(items)
	if got := d.PreferredHeight(); got != maxDiagnosticsHeight {
		t.Errorf("20-item PreferredHeight = %d, want %d", got, maxDiagnosticsHeight)
	}
}

func TestDiagnosticsSetItemsClampsCursor(t *testing.T) {
	d := newTestDiagnostics([]DiagnosticItem{
		{Line: 1, Severity: "error", Message: "a"},
		{Line: 2, Severity: "error", Message: "b"},
		{Line: 3, Severity: "error", Message: "c"},
	})
	d, _ = d.Update(keyMsg("G"))
	if d.cursor != 2 {
		t.Fatalf("cursor = %d after G, want 2", d.cursor)
	}
	d.SetItems([]DiagnosticItem{{Line: 1, Severity: "error", Message: "only"}})
	if d.cursor != 0 {
		t.Errorf("cursor = %d after shrink, want 0", d.cursor)
	}
}
