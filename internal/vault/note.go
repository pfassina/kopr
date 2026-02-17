package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Note represents a note in the vault.
type Note struct {
	Path    string
	Title   string
	Content string
}

// CreateNote creates a new note file with the given content.
func (v *Vault) CreateNote(relPath, content string) (string, error) {
	absPath := filepath.Join(v.Root, relPath)

	// Ensure parent directory exists
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create directory: %w", err)
	}

	// Don't overwrite existing files
	if _, err := os.Stat(absPath); err == nil {
		return absPath, nil
	}

	if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("write note: %w", err)
	}

	return absPath, nil
}

// CreateDailyNote creates a daily note with today's date.
func (v *Vault) CreateDailyNote() (string, error) {
	now := time.Now()
	date := now.Format("2006-01-02")
	relPath := filepath.Join("daily", date+".md")

	content := fmt.Sprintf(`---
title: %s
date: %s
tags: [daily]
---

# %s

`, date, date, date)

	return v.CreateNote(relPath, content)
}

// DeleteNote removes a note file from the vault.
func (v *Vault) DeleteNote(relPath string) error {
	absPath := filepath.Join(v.Root, relPath)
	return os.Remove(absPath)
}

// RenameNote renames a note file within the vault.
func (v *Vault) RenameNote(oldRel, newRel string) error {
	oldAbs := filepath.Join(v.Root, oldRel)
	newAbs := filepath.Join(v.Root, newRel)

	// Ensure target directory exists
	if err := os.MkdirAll(filepath.Dir(newAbs), 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// Don't overwrite existing files
	if _, err := os.Stat(newAbs); err == nil {
		return fmt.Errorf("%s already exists", newRel)
	}

	return os.Rename(oldAbs, newAbs)
}

// CreateDir creates a directory inside the vault.
func (v *Vault) CreateDir(relPath string) error {
	absPath := filepath.Join(v.Root, relPath)
	return os.MkdirAll(absPath, 0755)
}

// MoveNote moves a note to a new directory, keeping the same filename.
func (v *Vault) MoveNote(oldRel, newDir string) error {
	newRel := filepath.Join(newDir, filepath.Base(oldRel))
	return v.RenameNote(oldRel, newRel)
}

// CreateInboxNote creates a quick inbox note.
func (v *Vault) CreateInboxNote() (string, error) {
	now := time.Now()
	timestamp := now.Format("2006-01-02-150405")
	relPath := filepath.Join("inbox", timestamp+".md")

	content := fmt.Sprintf(`---
title: Inbox %s
date: %s
tags: [inbox]
status: inbox
---

`, timestamp, now.Format("2006-01-02"))

	return v.CreateNote(relPath, content)
}
