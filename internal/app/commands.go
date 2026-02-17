package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pfassina/kopr/internal/panel"
)

// indexInitDoneMsg signals indexing is complete.
type indexInitDoneMsg struct{}

// initIndex starts the indexer in a goroutine.
func (a *App) initIndex() tea.Cmd {
	return func() tea.Msg {
		if a.indexer != nil {
			a.indexer.IndexAll()
		}
		return indexInitDoneMsg{}
	}
}

// searchNotes returns finder items for a query.
func (a *App) searchNotes(query string) []panel.FinderItem {
	if a.db == nil {
		return nil
	}

	if query == "" {
		results, err := a.db.ListAllNotes(50)
		if err != nil {
			return nil
		}
		items := make([]panel.FinderItem, len(results))
		for i, r := range results {
			items[i] = panel.FinderItem{
				Title: r.Title,
				Path:  r.Path,
			}
		}
		return items
	}

	// Try FTS search first
	results, err := a.db.Search(query, 50)
	if err != nil || len(results) == 0 {
		// Fallback to file search
		results, err = a.db.SearchFiles(query, 50)
		if err != nil {
			return nil
		}
	}

	items := make([]panel.FinderItem, len(results))
	for i, r := range results {
		items[i] = panel.FinderItem{
			Title: r.Title,
			Path:  r.Path,
		}
	}
	return items
}

// handleFinderResult handles a file selection from the finder.
func (a *App) handleFinderResult(path string) tea.Cmd {
	a.navigateTo(path)
	a.focused = focusEditor
	return nil
}

// createNoteFromFinder creates a new note from a finder query string.
func (a *App) createNoteFromFinder(name string) {
	// Sanitize: add .md extension if missing
	relPath := name
	if !strings.HasSuffix(relPath, ".md") {
		relPath += ".md"
	}

	if msg := a.checkUniqueBasename(relPath); msg != "" {
		a.status.SetError(msg)
		return
	}

	content := fmt.Sprintf("---\ntitle: %s\n---\n\n", name)
	fullPath, err := a.vault.CreateNote(relPath, content)
	if err != nil {
		return
	}

	a.editor.OpenFile(fullPath)
	a.status.SetFile(relPath)
	a.currentFile = relPath
	a.tree.Refresh()
}

// updateBacklinks refreshes the backlinks panel for the given note path.
func (a *App) updateBacklinks(relPath string) {
	if a.db == nil {
		return
	}

	backlinks, err := a.db.GetBacklinks(relPath)
	if err != nil || len(backlinks) == 0 {
		a.info.SetBacklinks(nil)
		return
	}

	items := make([]panel.InfoItem, len(backlinks))
	for i, bl := range backlinks {
		title := bl.SourceTitle
		if title == "" {
			title = bl.SourcePath
		}
		items[i] = panel.InfoItem{Title: title, Path: bl.SourcePath}
	}
	a.info.SetBacklinks(items)
}
