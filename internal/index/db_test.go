package index

import "testing"

func TestOpenMemory(t *testing.T) {
	db, err := OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	// Insert a note
	id, err := db.UpsertNote("test.md", "Test", "test", "", "abc123", 1000, 42)
	if err != nil {
		t.Fatal(err)
	}
	if id <= 0 {
		t.Fatalf("expected positive id, got %d", id)
	}

	// Update FTS
	err = db.UpdateFTS(id, "Test", "Hello world content", "tag1 tag2", "Heading 1")
	if err != nil {
		t.Fatal(err)
	}

	// Search
	results, err := db.Search("world", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Path != "test.md" {
		t.Errorf("path: got %q, want %q", results[0].Path, "test.md")
	}
}

func TestSearchFiles(t *testing.T) {
	db, err := OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	if _, err := db.UpsertNote("daily/2024-01-01.md", "2024-01-01", "2024-01-01", "", "a", 1000, 10); err != nil {
		t.Fatal(err)
	}
	if _, err := db.UpsertNote("inbox/note.md", "Quick Note", "quick-note", "", "b", 1000, 10); err != nil {
		t.Fatal(err)
	}

	results, err := db.SearchFiles("daily", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestFindNoteByBasename(t *testing.T) {
	db, err := OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	if _, err := db.UpsertNote("projects/My-Note.md", "My Note", "my-note", "", "a", 1000, 10); err != nil {
		t.Fatal(err)
	}
	if _, err := db.UpsertNote("daily/2024-01-01.md", "2024-01-01", "2024-01-01", "", "b", 1000, 10); err != nil {
		t.Fatal(err)
	}
	if _, err := db.UpsertNote("root-note.md", "Root Note", "root-note", "", "c", 1000, 10); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		basename string
		want     string
	}{
		{"my-note.md", "projects/My-Note.md"},
		{"MY-NOTE.MD", "projects/My-Note.md"},
		{"2024-01-01.md", "daily/2024-01-01.md"},
		{"root-note.md", "root-note.md"},
		{"nonexistent.md", ""},
	}

	for _, tt := range tests {
		got, err := db.FindNoteByBasename(tt.basename)
		if err != nil {
			t.Errorf("FindNoteByBasename(%q): %v", tt.basename, err)
			continue
		}
		if got != tt.want {
			t.Errorf("FindNoteByBasename(%q) = %q, want %q", tt.basename, got, tt.want)
		}
	}

	// Case-insensitive uniqueness should be enforced at the DB level.
	if _, err := db.UpsertNote("otherdir/my-note.md", "Dup", "dup", "", "z", 1000, 10); err == nil {
		t.Fatalf("expected duplicate basename insert to fail")
	}
}

func TestBacklinks(t *testing.T) {
	db, err := OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	id1, err := db.UpsertNote("a.md", "Note A", "a", "", "a", 1000, 10)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.UpsertNote("projects/b.md", "Note B", "b", "", "b", 1000, 10); err != nil {
		t.Fatal(err)
	}

	// Links store basenames
	if err := db.InsertLink(id1, "b.md", "", "", 5, 10); err != nil {
		t.Fatal(err)
	}

	// GetBacklinks extracts basename from the target path
	backlinks, err := db.GetBacklinks("projects/b.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(backlinks) != 1 {
		t.Fatalf("expected 1 backlink, got %d", len(backlinks))
	}
	if backlinks[0].SourcePath != "a.md" {
		t.Errorf("backlink source: got %q, want %q", backlinks[0].SourcePath, "a.md")
	}
}

func TestGetOutgoingLinks(t *testing.T) {
	db, err := OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	idA, err := db.UpsertNote("a.md", "Note A", "a", "", "a", 1000, 10)
	if err != nil {
		t.Fatal(err)
	}
	idB, err := db.UpsertNote("b.md", "Note B", "b", "", "b", 1000, 10)
	if err != nil {
		t.Fatal(err)
	}

	// Resolved link: a -> b (target_id set via direct SQL since InsertLink doesn't set it)
	if err := db.InsertLink(idA, "b.md", "", "", 3, 0); err != nil {
		t.Fatal(err)
	}
	// Manually resolve the link's target_id
	if _, err := db.conn.Exec("UPDATE links SET target_id = ? WHERE source_id = ? AND target_path = ?", idB, idA, "b.md"); err != nil {
		t.Fatal(err)
	}

	// Unresolved link: a -> nonexistent
	if err := db.InsertLink(idA, "nonexistent.md", "", "", 5, 0); err != nil {
		t.Fatal(err)
	}

	results, err := db.GetOutgoingLinks("a.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 outgoing links, got %d", len(results))
	}

	// First link: resolved
	if results[0].TargetTitle != "Note B" {
		t.Errorf("first link title: got %q, want %q", results[0].TargetTitle, "Note B")
	}
	if !results[0].Resolved {
		t.Error("first link should be resolved")
	}

	// Second link: unresolved (falls back to target_path)
	if results[1].TargetTitle != "nonexistent.md" {
		t.Errorf("second link title: got %q, want %q", results[1].TargetTitle, "nonexistent.md")
	}
	if results[1].Resolved {
		t.Error("second link should not be resolved")
	}
}

func TestGetOutgoingLinksEmpty(t *testing.T) {
	db, err := OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	if _, err := db.UpsertNote("a.md", "Note A", "a", "", "a", 1000, 10); err != nil {
		t.Fatal(err)
	}

	results, err := db.GetOutgoingLinks("a.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 outgoing links, got %d", len(results))
	}
}

func TestGetHeadingsForNote(t *testing.T) {
	db, err := OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	id, err := db.UpsertNote("a.md", "Note A", "a", "", "a", 1000, 10)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.InsertHeading(id, 1, "Introduction", 1); err != nil {
		t.Fatal(err)
	}
	if err := db.InsertHeading(id, 2, "Details", 5); err != nil {
		t.Fatal(err)
	}
	if err := db.InsertHeading(id, 3, "Sub-details", 10); err != nil {
		t.Fatal(err)
	}

	results, err := db.GetHeadingsForNote("a.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 headings, got %d", len(results))
	}

	if results[0].Text != "Introduction" || results[0].Level != 1 || results[0].Line != 1 {
		t.Errorf("heading 0: got %+v", results[0])
	}
	if results[1].Text != "Details" || results[1].Level != 2 || results[1].Line != 5 {
		t.Errorf("heading 1: got %+v", results[1])
	}
	if results[2].Text != "Sub-details" || results[2].Level != 3 || results[2].Line != 10 {
		t.Errorf("heading 2: got %+v", results[2])
	}
}

func TestGetHeadingsForNoteEmpty(t *testing.T) {
	db, err := OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	if _, err := db.UpsertNote("a.md", "Note A", "a", "", "a", 1000, 10); err != nil {
		t.Fatal(err)
	}

	results, err := db.GetHeadingsForNote("a.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 headings, got %d", len(results))
	}
}

func TestGetHeadingsForNoteNonexistent(t *testing.T) {
	db, err := OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	results, err := db.GetHeadingsForNote("nonexistent.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 headings for nonexistent note, got %d", len(results))
	}
}
