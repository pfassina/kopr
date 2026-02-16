package index

import "testing"

func TestOpenMemory(t *testing.T) {
	db, err := OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

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
	defer db.Close()

	db.UpsertNote("daily/2024-01-01.md", "2024-01-01", "2024-01-01", "", "a", 1000, 10)
	db.UpsertNote("inbox/note.md", "Quick Note", "quick-note", "", "b", 1000, 10)

	results, err := db.SearchFiles("daily", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestBacklinks(t *testing.T) {
	db, err := OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	id1, _ := db.UpsertNote("a.md", "Note A", "a", "", "a", 1000, 10)
	db.UpsertNote("b.md", "Note B", "b", "", "b", 1000, 10)

	db.InsertLink(id1, "b.md", "", "", 5, 10)

	backlinks, err := db.GetBacklinks("b.md")
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
