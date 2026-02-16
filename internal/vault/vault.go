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

		rel, _ := filepath.Rel(v.Root, path)
		if rel == "." {
			return nil
		}

		// Skip hidden files/directories and .vimvault
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
		// Directories before files at same depth
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return entries[i].Path < entries[j].Path
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
