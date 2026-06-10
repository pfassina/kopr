package app

import (
	"bytes"
	"fmt"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pfassina/kopr/internal/markdown"
	"github.com/pfassina/kopr/internal/panel"
)

// diagnosticsMsg carries lint results for a note back into Update.
type diagnosticsMsg struct {
	relPath string
	diags   []markdown.Diagnostic
	err     error
}

// lintCurrentNote returns a command that lints the current buffer.
// Returns nil when diagnostics are disabled or no note is open.
func (a *App) lintCurrentNote() tea.Cmd {
	if !a.cfg.DiagnosticsEnabled {
		return nil
	}
	rpc := a.editor.GetRPC()
	if rpc == nil || a.currentFile == "" {
		return nil
	}

	relPath := a.currentFile
	db := a.db
	return func() tea.Msg {
		content, err := rpc.BufferContent()
		if err != nil {
			return diagnosticsMsg{relPath: relPath, err: fmt.Errorf("nvim buffer content: %w", err)}
		}

		var b bytes.Buffer
		for i, line := range content {
			b.Write(line)
			if i < len(content)-1 {
				b.WriteByte('\n')
			}
		}

		var resolve func(string) bool
		if db != nil {
			resolve = func(target string) bool {
				basename := filepath.Base(markdown.ResolveWikiLinkTarget(target))
				resolved, err := db.FindNoteByBasename(basename)
				return err == nil && resolved != ""
			}
		}

		diags := markdown.Lint(b.Bytes(), markdown.LintOptions{ResolveLink: resolve})
		return diagnosticsMsg{relPath: relPath, diags: diags}
	}
}

// applyDiagnostics pushes lint results into the panel and Neovim.
func (a *App) applyDiagnostics(diags []markdown.Diagnostic) {
	items := make([]panel.DiagnosticItem, len(diags))
	for i, d := range diags {
		items[i] = panel.DiagnosticItem{
			Line:     d.Line,
			Col:      d.Col,
			Severity: d.Severity.String(),
			Message:  d.Message,
		}
	}
	a.diagnostics.SetItems(items)
	if a.diagnosticsVisible() {
		a.updateLayout()
	}

	a.pushDiagnosticsToEditor(diags)
}

// pushDiagnosticsToEditor mirrors diagnostics into Neovim's diagnostic API
// for inline hints (virtual text, signs, underline).
func (a *App) pushDiagnosticsToEditor(diags []markdown.Diagnostic) {
	rpc := a.editor.GetRPC()
	if rpc == nil {
		return
	}

	entries := make([]map[string]interface{}, len(diags))
	for i, d := range diags {
		severity := 1 // vim.diagnostic.severity.ERROR
		if d.Severity == markdown.SeverityWarning {
			severity = 2 // vim.diagnostic.severity.WARN
		}
		entry := map[string]interface{}{
			"lnum":     d.Line - 1,
			"col":      d.Col,
			"severity": severity,
			"message":  d.Message,
			"source":   "kopr",
		}
		if d.EndCol > d.Col {
			entry["end_col"] = d.EndCol
		}
		entries[i] = entry
	}

	if err := rpc.SetDiagnostics(entries); err != nil {
		a.status.SetError(fmt.Sprintf("diagnostics: %v", err))
		return
	}
	if err := rpc.SetDiagnosticsVisible(a.inlineDiags); err != nil {
		a.status.SetError(fmt.Sprintf("diagnostics: %v", err))
	}
}

// clearDiagnostics empties the panel and removes editor marks.
func (a *App) clearDiagnostics() {
	a.diagnostics.SetItems(nil)
	rpc := a.editor.GetRPC()
	if rpc == nil {
		return
	}
	if err := rpc.SetDiagnostics(nil); err != nil {
		a.status.SetError(fmt.Sprintf("diagnostics: %v", err))
	}
}

// diagnosticsVisible reports whether the diagnostics panel should render.
func (a *App) diagnosticsVisible() bool {
	return a.diagnostics.Visible() && !a.zenMode && !a.editor.ShowSplash()
}

// ToggleDiagnosticsPanel shows/hides the diagnostics list below the editor.
func (a *App) ToggleDiagnosticsPanel() tea.Cmd {
	if !a.cfg.DiagnosticsEnabled {
		a.status.SetError("diagnostics disabled (set diagnostics_enabled = true in config.toml)")
		return nil
	}
	if a.diagnostics.Visible() {
		a.diagnostics.Hide()
		if a.focused == focusDiagnostics {
			a.setFocus(focusEditor)
		}
		a.updateLayout()
		return nil
	}
	a.diagnostics.Show()
	a.setFocus(focusDiagnostics)
	a.updateLayout()
	return a.lintCurrentNote()
}

// ToggleInlineDiagnostics flips inline hint visibility in the editor.
func (a *App) ToggleInlineDiagnostics() {
	if !a.cfg.DiagnosticsEnabled {
		a.status.SetError("diagnostics disabled (set diagnostics_enabled = true in config.toml)")
		return
	}
	a.inlineDiags = !a.inlineDiags

	rpc := a.editor.GetRPC()
	if rpc != nil {
		if err := rpc.SetDiagnosticsVisible(a.inlineDiags); err != nil {
			a.status.SetError(fmt.Sprintf("diagnostics: %v", err))
			return
		}
	}
	if a.inlineDiags {
		a.status.SetMessage("Inline diagnostics on")
	} else {
		a.status.SetMessage("Inline diagnostics off")
	}
}
