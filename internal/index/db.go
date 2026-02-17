package index

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS notes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    path TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL DEFAULT '',
    slug TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT '',
    mod_time INTEGER NOT NULL,
    size INTEGER NOT NULL DEFAULT 0,
    hash TEXT NOT NULL DEFAULT ''
);

CREATE VIRTUAL TABLE IF NOT EXISTS notes_fts USING fts5(
    title, content, tags, headings,
    content=notes, content_rowid=id,
    tokenize='porter unicode61 remove_diacritics 2'
);

CREATE TABLE IF NOT EXISTS tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS note_tags (
    note_id INTEGER REFERENCES notes(id) ON DELETE CASCADE,
    tag_id INTEGER REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (note_id, tag_id)
);

CREATE TABLE IF NOT EXISTS links (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_id INTEGER NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    target_path TEXT NOT NULL,
    target_id INTEGER REFERENCES notes(id) ON DELETE SET NULL,
    section TEXT DEFAULT '',
    alias TEXT DEFAULT '',
    line INTEGER NOT NULL,
    col INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS headings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    note_id INTEGER NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    level INTEGER NOT NULL,
    text TEXT NOT NULL,
    line INTEGER NOT NULL
);
`

// DB wraps the SQLite database connection.
type DB struct {
	conn *sql.DB
}

// Open opens or creates the database at the given path.
func Open(path string) (*DB, error) {
	conn, err := sql.Open("sqlite", path+"?_pragma=journal_mode(wal)&_pragma=foreign_keys(on)")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if _, err := conn.Exec(schema); err != nil {
		if closeErr := conn.Close(); closeErr != nil {
			return nil, fmt.Errorf("init schema: %w (close: %v)", err, closeErr)
		}
		return nil, fmt.Errorf("init schema: %w", err)
	}

	return &DB{conn: conn}, nil
}

// OpenMemory opens an in-memory database (for testing).
func OpenMemory() (*DB, error) {
	conn, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(on)")
	if err != nil {
		return nil, err
	}
	if _, err := conn.Exec(schema); err != nil {
		if closeErr := conn.Close(); closeErr != nil {
			return nil, fmt.Errorf("init schema: %w (close: %v)", err, closeErr)
		}
		return nil, fmt.Errorf("init schema: %w", err)
	}
	return &DB{conn: conn}, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}

// Conn returns the underlying sql.DB for advanced queries.
func (db *DB) Conn() *sql.DB {
	return db.conn
}

// UpsertNote inserts or updates a note and returns its ID.
func (db *DB) UpsertNote(path, title, slug, status, hash string, modTime, size int64) (int64, error) {
	res, err := db.conn.Exec(`
		INSERT INTO notes (path, title, slug, status, mod_time, size, hash)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			title = excluded.title,
			slug = excluded.slug,
			status = excluded.status,
			mod_time = excluded.mod_time,
			size = excluded.size,
			hash = excluded.hash
	`, path, title, slug, status, modTime, size, hash)
	if err != nil {
		return 0, err
	}

	// Get the ID (either inserted or existing)
	var id int64
	err = db.conn.QueryRow("SELECT id FROM notes WHERE path = ?", path).Scan(&id)
	if err != nil {
		return 0, err
	}

	_ = res
	return id, nil
}

// UpdateFTS updates the FTS index for a note.
func (db *DB) UpdateFTS(noteID int64, title, content, tags, headings string) error {
	// Delete old FTS entry
	_, err := db.conn.Exec("INSERT INTO notes_fts(notes_fts, rowid, title, content, tags, headings) VALUES('delete', ?, '', '', '', '')", noteID)
	if err != nil {
		// Ignore delete errors for new entries; the insert below will populate the row.
		_ = err
	}

	// Insert new FTS entry
	_, err = db.conn.Exec("INSERT INTO notes_fts(rowid, title, content, tags, headings) VALUES(?, ?, ?, ?, ?)",
		noteID, title, content, tags, headings)
	return err
}

// UpsertTag ensures a tag exists and returns its ID.
func (db *DB) UpsertTag(name string) (int64, error) {
	_, err := db.conn.Exec("INSERT OR IGNORE INTO tags (name) VALUES (?)", name)
	if err != nil {
		return 0, err
	}
	var id int64
	err = db.conn.QueryRow("SELECT id FROM tags WHERE name = ?", name).Scan(&id)
	return id, err
}

// LinkNoteTag associates a tag with a note.
func (db *DB) LinkNoteTag(noteID, tagID int64) error {
	_, err := db.conn.Exec("INSERT OR IGNORE INTO note_tags (note_id, tag_id) VALUES (?, ?)", noteID, tagID)
	return err
}

// ClearNoteTags removes all tag associations for a note.
func (db *DB) ClearNoteTags(noteID int64) error {
	_, err := db.conn.Exec("DELETE FROM note_tags WHERE note_id = ?", noteID)
	return err
}

// InsertLink adds a wiki link record.
func (db *DB) InsertLink(sourceID int64, targetPath, section, alias string, line, col int) error {
	_, err := db.conn.Exec(`
		INSERT INTO links (source_id, target_path, section, alias, line, col)
		VALUES (?, ?, ?, ?, ?, ?)
	`, sourceID, targetPath, section, alias, line, col)
	return err
}

// ClearNoteLinks removes all links from a note.
func (db *DB) ClearNoteLinks(noteID int64) error {
	_, err := db.conn.Exec("DELETE FROM links WHERE source_id = ?", noteID)
	return err
}

// InsertHeading adds a heading record.
func (db *DB) InsertHeading(noteID int64, level int, text string, line int) error {
	_, err := db.conn.Exec("INSERT INTO headings (note_id, level, text, line) VALUES (?, ?, ?, ?)",
		noteID, level, text, line)
	return err
}

// ClearNoteHeadings removes all headings for a note.
func (db *DB) ClearNoteHeadings(noteID int64) error {
	_, err := db.conn.Exec("DELETE FROM headings WHERE note_id = ?", noteID)
	return err
}

// GetNoteHash returns the stored hash for a note path.
func (db *DB) GetNoteHash(path string) (string, error) {
	var hash string
	err := db.conn.QueryRow("SELECT hash FROM notes WHERE path = ?", path).Scan(&hash)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return hash, err
}

// DeleteNote removes a note and all its related data.
func (db *DB) DeleteNote(path string) error {
	_, err := db.conn.Exec("DELETE FROM notes WHERE path = ?", path)
	return err
}
