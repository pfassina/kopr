package app

// Layout computes the dimensions for each panel.
type Layout struct {
	TreeWidth    int
	EditorWidth  int
	InfoWidth    int
	Height       int
	StatusHeight int
}

// ComputeLayout calculates panel dimensions based on total width/height
// and whether each panel is visible.
func ComputeLayout(totalWidth, totalHeight int, showTree, showInfo bool, treeWidth, infoWidth int) Layout {
	// During live resizes some terminals momentarily report 0 (or even negative)
	// dimensions; clamp to avoid propagating invalid sizes into panels.
	if totalWidth < 1 {
		totalWidth = 1
	}
	if totalHeight < 2 { // need at least 1 row for content + 1 for status
		totalHeight = 2
	}

	l := Layout{
		StatusHeight: 1,
		Height:       totalHeight - 1, // reserve 1 row for status bar
	}

	remaining := totalWidth

	if showTree {
		l.TreeWidth = treeWidth
		if l.TreeWidth > remaining/3 {
			l.TreeWidth = remaining / 3
		}
		remaining -= l.TreeWidth - 1 // -1 for border overlap
	}

	if showInfo {
		l.InfoWidth = infoWidth
		if l.InfoWidth > remaining/3 {
			l.InfoWidth = remaining / 3
		}
		remaining -= l.InfoWidth - 1 // -1 for border overlap
	}

	l.EditorWidth = remaining
	// During extreme resizes the terminal can get very narrow; never force a
	// minimum width larger than the available space.
	if l.EditorWidth < 1 {
		l.EditorWidth = 1
	}

	return l
}
