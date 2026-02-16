package app

import (
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/yourusername/vimvault/internal/panel"
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
	fullPath := filepath.Join(a.cfg.VaultPath, path)
	a.editor.OpenFile(fullPath)
	a.status.SetFile(path)
	a.focused = focusEditor
	return nil
}
