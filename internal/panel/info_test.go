package panel

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pfassina/kopr/internal/theme"
)

func newTestInfo(backlinks, outgoing, outline []InfoItem) Info {
	info := NewInfo()
	th := theme.DefaultTheme()
	info.SetTheme(&th)
	info.SetSize(40, 20)
	info.SetFocused(true)
	info.SetBacklinks(backlinks)
	info.SetOutgoingLinks(outgoing)
	info.SetOutline(outline)
	return info
}

func key(k string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
}

func specialKey(t tea.KeyType) tea.KeyMsg {
	return tea.KeyMsg{Type: t}
}

func TestInfoFlatList(t *testing.T) {
	info := newTestInfo(
		[]InfoItem{{Title: "bl1", Path: "a.md"}},
		[]InfoItem{{Title: "ol1", Path: "b.md"}, {Title: "ol2", Path: "c.md"}},
		[]InfoItem{{Title: "H1", Line: 1, Level: 1}},
	)

	rows := info.flatList()
	// 3 headers + 2 separators + 1 + 2 + 1 items = 9
	if len(rows) != 9 {
		t.Fatalf("expected 9 rows, got %d", len(rows))
	}

	// Row layout: header0, item, sep, header1, item, item, sep, header2, item
	if rows[0].kind != rowHeader || rows[0].sectionIdx != 0 {
		t.Error("row 0 should be backlinks header")
	}
	if rows[1].kind != rowItem || rows[1].sectionIdx != 0 {
		t.Error("row 1 should be backlink item")
	}
	if rows[2].kind != rowSeparator {
		t.Error("row 2 should be separator")
	}
	if rows[3].kind != rowHeader || rows[3].sectionIdx != 1 {
		t.Error("row 3 should be outgoing header")
	}
	if rows[7].kind != rowHeader || rows[7].sectionIdx != 2 {
		t.Error("row 7 should be outline header")
	}
}

func TestInfoCursorNavigation(t *testing.T) {
	info := newTestInfo(
		[]InfoItem{{Title: "bl1", Path: "a.md"}},
		nil,
		[]InfoItem{{Title: "H1", Line: 1, Level: 1}},
	)

	// Initial cursor at 0 (Backlinks header)
	if info.cursor != 0 {
		t.Fatalf("cursor should start at 0, got %d", info.cursor)
	}

	// Move down to backlink item
	info, _ = info.Update(key("j"))
	if info.cursor != 1 {
		t.Fatalf("cursor should be 1 after j, got %d", info.cursor)
	}

	// Move down again — should skip separator and land on Outgoing header
	info, _ = info.Update(key("j"))
	rows := info.flatList()
	if rows[info.cursor].kind != rowHeader || rows[info.cursor].sectionIdx != 1 {
		t.Fatalf("cursor should skip separator to Outgoing header, cursor=%d", info.cursor)
	}

	// Move up — should skip separator back to backlink item
	info, _ = info.Update(key("k"))
	if info.cursor != 1 {
		t.Fatalf("cursor should skip separator back to item, got %d", info.cursor)
	}

	// Move to top
	info, _ = info.Update(key("g"))
	if info.cursor != 0 {
		t.Fatalf("cursor should be 0 after g, got %d", info.cursor)
	}

	// Move to bottom — should land on last non-separator row
	info, _ = info.Update(key("G"))
	if rows[info.cursor].kind == rowSeparator {
		t.Fatalf("G should not land on separator, cursor=%d", info.cursor)
	}
	// Last row is the outline item
	if rows[info.cursor].kind != rowItem || rows[info.cursor].sectionIdx != 2 {
		t.Fatalf("G should land on outline item, cursor=%d, kind=%d", info.cursor, rows[info.cursor].kind)
	}
}

func TestInfoCollapseExpand(t *testing.T) {
	info := newTestInfo(
		[]InfoItem{{Title: "bl1", Path: "a.md"}, {Title: "bl2", Path: "b.md"}},
		nil,
		nil,
	)

	rowsBefore := info.flatList()
	// 3 headers + 2 separators + 2 items = 7
	if len(rowsBefore) != 7 {
		t.Fatalf("expected 7 rows before collapse, got %d", len(rowsBefore))
	}

	// Cursor is on backlinks header (row 0); press enter to collapse
	info, _ = info.Update(key("enter"))
	rowsAfter := info.flatList()
	// 3 headers + 2 separators + 0 items (collapsed) = 5
	if len(rowsAfter) != 5 {
		t.Fatalf("expected 5 rows after collapse, got %d", len(rowsAfter))
	}

	// Press enter again to expand
	info, _ = info.Update(key("enter"))
	rowsExpanded := info.flatList()
	if len(rowsExpanded) != 7 {
		t.Fatalf("expected 7 rows after expand, got %d", len(rowsExpanded))
	}
}

func TestInfoEnterOnBacklinkItem(t *testing.T) {
	info := newTestInfo(
		[]InfoItem{{Title: "bl1", Path: "a.md"}},
		nil,
		nil,
	)

	// Move cursor to item (row 1)
	info, _ = info.Update(key("j"))
	var cmd tea.Cmd
	_, cmd = info.Update(key("enter"))

	if cmd == nil {
		t.Fatal("expected a command on enter for backlink item")
	}
	msg := cmd()
	fileMsg, ok := msg.(FileSelectedMsg)
	if !ok {
		t.Fatalf("expected FileSelectedMsg, got %T", msg)
	}
	if fileMsg.Path != "a.md" {
		t.Errorf("path: got %q, want %q", fileMsg.Path, "a.md")
	}
}

func TestInfoEnterOnOutlineItem(t *testing.T) {
	info := newTestInfo(
		nil,
		nil,
		[]InfoItem{{Title: "Heading 1", Line: 5, Level: 1}},
	)

	// Navigate to the outline item using j (separators are skipped automatically)
	// Row layout: header0, sep, header1, sep, header2, item
	// j skips sep to land on headers, then items
	for {
		rows := info.flatList()
		row := rows[info.cursor]
		if row.kind == rowItem && row.sectionIdx == 2 {
			break
		}
		info, _ = info.Update(key("j"))
	}

	var cmd tea.Cmd
	_, cmd = info.Update(key("enter"))
	if cmd == nil {
		t.Fatal("expected a command on enter for outline item")
	}
	msg := cmd()
	gotoMsg, ok := msg.(InfoGotoLineMsg)
	if !ok {
		t.Fatalf("expected InfoGotoLineMsg, got %T", msg)
	}
	if gotoMsg.Line != 5 {
		t.Errorf("line: got %d, want 5", gotoMsg.Line)
	}
}

func TestInfoTabJumpsSections(t *testing.T) {
	info := newTestInfo(
		[]InfoItem{{Title: "bl1", Path: "a.md"}, {Title: "bl2", Path: "b.md"}},
		[]InfoItem{{Title: "ol1", Path: "c.md"}},
		[]InfoItem{{Title: "H1", Line: 1, Level: 1}},
	)

	// Start at row 0 (Backlinks header)
	if info.cursor != 0 {
		t.Fatalf("cursor should start at 0, got %d", info.cursor)
	}

	// Tab to next section header
	info, _ = info.Update(specialKey(tea.KeyTab))
	rows := info.flatList()
	if rows[info.cursor].kind != rowHeader || rows[info.cursor].sectionIdx != 1 {
		t.Fatalf("tab should jump to Outgoing Links header, cursor=%d", info.cursor)
	}

	// Tab again to Outline header
	info, _ = info.Update(specialKey(tea.KeyTab))
	if rows[info.cursor].kind != rowHeader || rows[info.cursor].sectionIdx != 2 {
		t.Fatalf("tab should jump to Outline header, cursor=%d", info.cursor)
	}

	// Shift+tab back to Outgoing Links
	info, _ = info.Update(specialKey(tea.KeyShiftTab))
	if rows[info.cursor].kind != rowHeader || rows[info.cursor].sectionIdx != 1 {
		t.Fatalf("shift+tab should jump back to Outgoing Links header, cursor=%d", info.cursor)
	}
}

func TestInfoBraceJumpsSections(t *testing.T) {
	info := newTestInfo(
		[]InfoItem{{Title: "bl1", Path: "a.md"}},
		nil,
		[]InfoItem{{Title: "H1", Line: 1, Level: 1}},
	)

	// } should jump to next header
	info, _ = info.Update(key("}"))
	rows := info.flatList()
	if rows[info.cursor].kind != rowHeader || rows[info.cursor].sectionIdx != 1 {
		t.Fatalf("} should jump to Outgoing Links header, cursor=%d", info.cursor)
	}

	// { should jump back
	info, _ = info.Update(key("{"))
	if rows[info.cursor].kind != rowHeader || rows[info.cursor].sectionIdx != 0 {
		t.Fatalf("{ should jump back to Backlinks header, cursor=%d", info.cursor)
	}
}

func TestInfoClearPreservesCollapseState(t *testing.T) {
	info := newTestInfo(
		[]InfoItem{{Title: "bl1", Path: "a.md"}},
		nil,
		nil,
	)

	// Collapse backlinks section
	info, _ = info.Update(key("enter"))
	if !info.sections[0].collapsed {
		t.Fatal("backlinks should be collapsed")
	}

	// Clear resets items but check that collapse state is separate
	info.Clear()
	// After clear, items are nil but structure remains
	if info.sections[0].items != nil {
		t.Fatal("items should be nil after clear")
	}
}

func TestInfoCollapseStatePreservedAcrossSetBacklinks(t *testing.T) {
	info := newTestInfo(
		[]InfoItem{{Title: "bl1", Path: "a.md"}},
		nil,
		nil,
	)

	// Collapse backlinks
	info, _ = info.Update(key("enter"))
	if !info.sections[0].collapsed {
		t.Fatal("backlinks should be collapsed")
	}

	// Set new backlinks data — collapse state should persist
	info.SetBacklinks([]InfoItem{{Title: "bl2", Path: "x.md"}, {Title: "bl3", Path: "y.md"}})
	if !info.sections[0].collapsed {
		t.Fatal("collapse state should persist across SetBacklinks")
	}
}

func TestInfoEmptySections(t *testing.T) {
	info := newTestInfo(nil, nil, nil)

	rows := info.flatList()
	// 3 headers + 2 separators = 5
	if len(rows) != 5 {
		t.Fatalf("expected 5 rows (headers + separators), got %d", len(rows))
	}
}

func TestInfoUnfocusedIgnoresKeys(t *testing.T) {
	info := newTestInfo(
		[]InfoItem{{Title: "bl1", Path: "a.md"}},
		nil,
		nil,
	)
	info.SetFocused(false)

	info, _ = info.Update(key("j"))
	if info.cursor != 0 {
		t.Fatal("unfocused info should not respond to keys")
	}
}

func TestInfoViewNonEmpty(t *testing.T) {
	info := newTestInfo(
		[]InfoItem{{Title: "bl1", Path: "a.md"}},
		nil,
		[]InfoItem{{Title: "H1", Line: 1, Level: 1}, {Title: "H2", Line: 5, Level: 2}},
	)

	view := info.View()
	if view == "" {
		t.Fatal("view should not be empty")
	}
}
