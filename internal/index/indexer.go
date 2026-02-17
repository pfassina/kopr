package index

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pfassina/kopr/internal/markdown"
)

// Indexer manages the note indexing pipeline.
type Indexer struct {
	db        *DB
	parser    *markdown.Parser
	vaultRoot string
}

func NewIndexer(db *DB, vaultRoot string) *Indexer {
	return &Indexer{
		db:        db,
		parser:    markdown.NewParser(),
		vaultRoot: vaultRoot,
	}
}

// IndexAll performs a full index of all markdown files in the vault.
func (idx *Indexer) IndexAll() error {
	// Clear links and hashes so all files get fully re-indexed.
	// Links are derived data rebuilt from source on each IndexFile call.
	if _, err := idx.db.Conn().Exec("DELETE FROM links"); err != nil {
		return fmt.Errorf("clear links: %w", err)
	}
	if _, err := idx.db.Conn().Exec("UPDATE notes SET hash = ''"); err != nil {
		return fmt.Errorf("clear hashes: %w", err)
	}

	return filepath.Walk(idx.vaultRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Skip hidden directories
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}

		if info.IsDir() || !strings.HasSuffix(info.Name(), ".md") {
			return nil
		}

		return idx.IndexFile(path)
	})
}

// IndexFile indexes a single markdown file.
func (idx *Indexer) IndexFile(absPath string) error {
	content, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", absPath, err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("stat %s: %w", absPath, err)
	}

	relPath, err := filepath.Rel(idx.vaultRoot, absPath)
	if err != nil {
		relPath = absPath
	}

	// Check if file has changed
	hash := fmt.Sprintf("%x", sha256.Sum256(content))
	existingHash, _ := idx.db.GetNoteHash(relPath)
	if hash == existingHash {
		return nil // unchanged
	}

	// Parse the markdown
	parsed := idx.parser.Parse(content)

	// Extract metadata
	title := titleFromPath(relPath)
	status := ""
	var tags []string

	if parsed.Frontmatter != nil {
		if parsed.Frontmatter.Title != "" {
			title = parsed.Frontmatter.Title
		}
		status = parsed.Frontmatter.Status
		tags = parsed.Frontmatter.Tags
	}

	slug := slugify(title)

	// Upsert the note
	noteID, err := idx.db.UpsertNote(relPath, title, slug, status, hash, info.ModTime().Unix(), info.Size())
	if err != nil {
		return fmt.Errorf("upsert note: %w", err)
	}

	// Update FTS
	headingTexts := make([]string, len(parsed.Headings))
	for i, h := range parsed.Headings {
		headingTexts[i] = h.Text
	}
	tagStr := strings.Join(tags, " ")
	headingStr := strings.Join(headingTexts, " ")

	if err := idx.db.UpdateFTS(noteID, title, parsed.PlainContent(), tagStr, headingStr); err != nil {
		return fmt.Errorf("update FTS: %w", err)
	}

	// Update tags
	if err := idx.db.ClearNoteTags(noteID); err != nil {
		return fmt.Errorf("clear note tags: %w", err)
	}
	for _, tag := range tags {
		tagID, err := idx.db.UpsertTag(tag)
		if err != nil {
			return fmt.Errorf("upsert tag %q: %w", tag, err)
		}
		if err := idx.db.LinkNoteTag(noteID, tagID); err != nil {
			return fmt.Errorf("link note tag %q: %w", tag, err)
		}
	}

	// Update headings
	if err := idx.db.ClearNoteHeadings(noteID); err != nil {
		return fmt.Errorf("clear note headings: %w", err)
	}
	for _, h := range parsed.Headings {
		if err := idx.db.InsertHeading(noteID, h.Level, h.Text, h.Line); err != nil {
			return fmt.Errorf("insert heading %q: %w", h.Text, err)
		}
	}

	// Update links (store basenames for name-based resolution)
	if err := idx.db.ClearNoteLinks(noteID); err != nil {
		return fmt.Errorf("clear note links: %w", err)
	}
	for _, link := range parsed.WikiLinks {
		targetPath := markdown.ResolveWikiLinkTarget(link.Target)
		targetPath = filepath.Base(targetPath) // store only basename
		if err := idx.db.InsertLink(noteID, targetPath, link.Section, link.Alias, link.Line, link.Col); err != nil {
			return fmt.Errorf("insert link to %q: %w", targetPath, err)
		}
	}

	// Resolve link target IDs
	if err := idx.resolveLinks(noteID); err != nil {
		return fmt.Errorf("resolve links: %w", err)
	}

	return nil
}

// RemoveFile removes a file from the index.
func (idx *Indexer) RemoveFile(absPath string) error {
	relPath, err := filepath.Rel(idx.vaultRoot, absPath)
	if err != nil {
		relPath = absPath
	}
	return idx.db.DeleteNote(relPath)
}

func titleFromPath(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	// Convert hyphens/underscores to spaces
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.ReplaceAll(name, "_", " ")
	return name
}

// resolveLinks attempts to set target_id for links whose target_path (basename) matches a known note.
func (idx *Indexer) resolveLinks(sourceID int64) error {
	_, err := idx.db.Conn().Exec(`
		UPDATE links SET target_id = (
			SELECT id FROM notes WHERE path = links.target_path OR path LIKE '%/' || links.target_path
		) WHERE source_id = ? AND target_id IS NULL
	`, sourceID)
	return err
}

func slugify(title string) string {
	s := strings.ToLower(title)
	s = strings.ReplaceAll(s, " ", "-")
	// Remove non-alphanumeric characters except hyphens
	var buf strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			buf.WriteRune(r)
		}
	}
	return buf.String()
}
