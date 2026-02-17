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
	if l.EditorWidth < 10 {
		l.EditorWidth = 10
	}

	return l
}
