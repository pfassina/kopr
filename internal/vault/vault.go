package vault

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Entry represents a file or directory in the vault.
type Entry struct {
	Name  string
	Path  string
	IsDir bool
	Depth int
}

// Vault represents a knowledge vault directory.
type Vault struct {
	Root string
}

func New(root string) *Vault {
	return &Vault{Root: root}
}

// ListEntries returns a flat list of all files/directories in the vault,
// sorted with directories first, then alphabetically.
func (v *Vault) ListEntries() ([]Entry, error) {
	var entries []Entry

	err := filepath.Walk(v.Root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors
		}

		rel, err := filepath.Rel(v.Root, path)
		if err != nil {
			return nil
		}
		if rel == "." {
			return nil
		}

		// Skip hidden files/directories and .kopr
		name := filepath.Base(path)
		if strings.HasPrefix(name, ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		depth := strings.Count(rel, string(filepath.Separator))

		entries = append(entries, Entry{
			Name:  name,
			Path:  rel,
			IsDir: info.IsDir(),
			Depth: depth,
		})
		return nil
	})

	sort.Slice(entries, func(i, j int) bool {
		return entryLess(entries[i], entries[j])
	})

	return entries, err
}

// ListNotes returns only markdown files in the vault.
func (v *Vault) ListNotes() ([]Entry, error) {
	all, err := v.ListEntries()
	if err != nil {
		return nil, err
	}

	var notes []Entry
	for _, e := range all {
		if !e.IsDir && strings.HasSuffix(e.Name, ".md") {
			notes = append(notes, e)
		}
	}
	return notes, nil
}

// entryLess implements hierarchical tree ordering: within each directory,
// subdirectories come before files, both sorted alphabetically.
func entryLess(a, b Entry) bool {
	// If one is a parent directory of the other, parent comes first
	if a.IsDir && strings.HasPrefix(b.Path, a.Path+string(filepath.Separator)) {
		return true
	}
	if b.IsDir && strings.HasPrefix(a.Path, b.Path+string(filepath.Separator)) {
		return false
	}

	// Compare by directory segments to group siblings
	aParts := strings.Split(a.Path, string(filepath.Separator))
	bParts := strings.Split(b.Path, string(filepath.Separator))

	minLen := len(aParts)
	if len(bParts) < minLen {
		minLen = len(bParts)
	}

	for i := 0; i < minLen; i++ {
		if aParts[i] == bParts[i] {
			continue
		}

		// At this level, determine if either entry is a file at this depth
		aIsFileHere := !a.IsDir && i == len(aParts)-1
		bIsFileHere := !b.IsDir && i == len(bParts)-1

		// Dirs before files at the same level
		if aIsFileHere != bIsFileHere {
			return !aIsFileHere
		}

		return strings.ToLower(aParts[i]) < strings.ToLower(bParts[i])
	}

	// Shorter path (directory) comes before longer path (its contents)
	return len(aParts) < len(bParts)
}
