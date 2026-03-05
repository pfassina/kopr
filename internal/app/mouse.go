package app

import tea "github.com/charmbracelet/bubbletea"

// mouseTarget identifies which panel a mouse event landed in.
type mouseTarget int

const (
	mouseTargetNone   mouseTarget = iota
	mouseTargetTree
	mouseTargetEditor
	mouseTargetInfo
	mouseTargetStatus
)

// mouseHitResult contains the result of hit-testing a mouse event.
type mouseHitResult struct {
	target mouseTarget

	// editorCol and editorRow are 0-based coordinates relative to the
	// Neovim buffer area. Only valid when target == mouseTargetEditor
	// and editorRow >= 0.
	editorCol int
	editorRow int // -1 when click is on the editor title row

	// screenX and screenY are the original screen coordinates.
	screenX int
	screenY int
}

// hitTestMouse determines which panel a mouse event lands in and translates
// coordinates for the editor panel.
func (a *App) hitTestMouse(msg tea.MouseMsg) mouseHitResult {
	showTree, showInfo := a.panelsVisible()
	layout := ComputeLayout(a.width, a.height, showTree, showInfo, a.cfg.TreeWidth, a.cfg.InfoWidth)

	result := mouseHitResult{
		screenX: msg.X,
		screenY: msg.Y,
	}

	// Status bar occupies the last row(s)
	if msg.Y >= layout.Height {
		result.target = mouseTargetStatus
		return result
	}

	// Determine editor column boundaries
	editorStartX := 0
	if showTree {
		// Tree occupies columns [0, TreeWidth). Content is TreeWidth-1 wide
		// plus a 1-char right border, totaling TreeWidth columns.
		if msg.X < layout.TreeWidth {
			result.target = mouseTargetTree
			return result
		}
		editorStartX = layout.TreeWidth
	}

	if showInfo {
		// Info panel starts after the editor column.
		infoStartX := editorStartX + layout.EditorWidth
		if msg.X >= infoStartX {
			result.target = mouseTargetInfo
			return result
		}
	}

	// Everything else falls in the editor column
	result.target = mouseTargetEditor
	result.editorCol = msg.X - editorStartX

	// Row 0 of the editor area is the title row.
	// Neovim buffer content starts at row 1.
	if msg.Y < 1 {
		result.editorRow = -1 // title row
	} else {
		result.editorRow = msg.Y - 1
	}

	return result
}
