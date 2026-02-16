package markdown

import (
	"path/filepath"
	"strings"
)

// ResolveWikiLinkTarget resolves a wiki link target to a file path.
// It handles:
//   - "note" -> "note.md"
//   - "folder/note" -> "folder/note.md"
//   - "note.md" -> "note.md" (already has extension)
func ResolveWikiLinkTarget(target string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return ""
	}

	// If already has .md extension, return as-is
	if strings.HasSuffix(target, ".md") {
		return target
	}

	return target + ".md"
}

// NoteNameFromPath extracts the note name from a file path.
// "folder/my-note.md" -> "my-note"
func NoteNameFromPath(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}
