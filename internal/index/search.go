package index

import (
	"database/sql"
	"path/filepath"
)

// SearchResult represents a single search result.
type SearchResult struct {
	ID    int64
	Path  string
	Title string
	Rank  float64
}

// BacklinkResult represents a backlink to a note.
type BacklinkResult struct {
	SourcePath  string
	SourceTitle string
	Line        int
	Col         int
}

// HeadingResult represents a heading in a note.
type HeadingResult struct {
	NoteID   int64
	NotePath string
	Level    int
	Text     string
	Line     int
}

// Search performs a full-text search across notes.
func (db *DB) Search(query string, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := db.conn.Query(`
		SELECT n.id, n.path, n.title, rank
		FROM notes_fts
		JOIN notes n ON n.id = notes_fts.rowid
		WHERE notes_fts MATCH ?
		ORDER BY rank
		LIMIT ?
	`, query, limit)
	if err != nil {
		return nil, err
	}

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.ID, &r.Path, &r.Title, &r.Rank); err != nil {
			_ = rows.Close()
			return nil, err
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	return results, nil
}

// SearchFiles searches note titles/paths (for fuzzy file finding).
func (db *DB) SearchFiles(query string, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 50
	}

	pattern := "%" + query + "%"
	rows, err := db.conn.Query(`
		SELECT id, path, title, 0 as rank
		FROM notes
		WHERE path LIKE ? OR title LIKE ?
		ORDER BY path
		LIMIT ?
	`, pattern, pattern, limit)
	if err != nil {
		return nil, err
	}

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.ID, &r.Path, &r.Title, &r.Rank); err != nil {
			_ = rows.Close()
			return nil, err
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	return results, nil
}

// ListAllNotes returns all notes, sorted by path.
func (db *DB) ListAllNotes(limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 200
	}

	rows, err := db.conn.Query(`
		SELECT id, path, title, 0 as rank
		FROM notes
		ORDER BY path
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.ID, &r.Path, &r.Title, &r.Rank); err != nil {
			_ = rows.Close()
			return nil, err
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	return results, nil
}

// GetBacklinks returns all notes that link to the given path.
// Matches by basename since target_path stores basenames.
func (db *DB) GetBacklinks(targetPath string) ([]BacklinkResult, error) {
	basename := filepath.Base(targetPath)
	rows, err := db.conn.Query(`
		SELECT n.path, n.title, l.line, l.col
		FROM links l
		JOIN notes n ON n.id = l.source_id
		WHERE l.target_path = ?
		ORDER BY n.path
	`, basename)
	if err != nil {
		return nil, err
	}

	var results []BacklinkResult
	for rows.Next() {
		var r BacklinkResult
		if err := rows.Scan(&r.SourcePath, &r.SourceTitle, &r.Line, &r.Col); err != nil {
			_ = rows.Close()
			return nil, err
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	return results, nil
}

// FindNoteByBasename returns the relative path of a note matching the given basename.
// Returns empty string if no match is found.
func (db *DB) FindNoteByBasename(basename string) (string, error) {
	var path string
	err := db.conn.QueryRow(
		`SELECT path FROM notes WHERE path = ? OR path LIKE ? LIMIT 1`,
		basename, "%/"+basename,
	).Scan(&path)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return path, err
}

// GetNoteIDByPath returns the ID of a note by its path.
func (db *DB) GetNoteIDByPath(path string) (int64, error) {
	var id int64
	err := db.conn.QueryRow("SELECT id FROM notes WHERE path = ?", path).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return id, err
}

// SearchHeadings searches headings across all notes.
func (db *DB) SearchHeadings(query string, limit int) ([]HeadingResult, error) {
	if limit <= 0 {
		limit = 50
	}

	pattern := "%" + query + "%"
	rows, err := db.conn.Query(`
		SELECT h.note_id, n.path, h.level, h.text, h.line
		FROM headings h
		JOIN notes n ON n.id = h.note_id
		WHERE h.text LIKE ?
		ORDER BY n.path, h.line
		LIMIT ?
	`, pattern, limit)
	if err != nil {
		return nil, err
	}

	var results []HeadingResult
	for rows.Next() {
		var r HeadingResult
		if err := rows.Scan(&r.NoteID, &r.NotePath, &r.Level, &r.Text, &r.Line); err != nil {
			_ = rows.Close()
			return nil, err
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	return results, nil
}
