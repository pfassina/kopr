package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pfassina/kopr/internal/config"
)

func newMouseMsg(x, y int) tea.MouseMsg {
	return tea.MouseMsg{X: x, Y: y}
}

func TestHitTestMouse(t *testing.T) {
	// 100 wide, 30 tall, tree=25, info=25.
	// With both panels visible:
	//   Tree:   cols 0-24  (25 wide including border)
	//   Editor: cols 25-76 (EditorWidth = 100 - 24 - 24 = 52)
	//   Info:   cols 77-99
	//   Status: row 29
	//   Editor title: row 0
	//   Neovim buffer: rows 1-28
	cfg := config.Config{
		TreeWidth: 25,
		InfoWidth: 25,
	}
	a := App{
		cfg:      cfg,
		width:    100,
		height:   30,
		showTree: true,
		showInfo: true,
	}

	tests := []struct {
		name      string
		x, y      int
		target    mouseTarget
		editorCol int
		editorRow int
	}{
		{
			name:   "tree click",
			x:      10,
			y:      5,
			target: mouseTargetTree,
		},
		{
			name:   "tree border click",
			x:      24,
			y:      5,
			target: mouseTargetTree,
		},
		{
			name:      "editor title row",
			x:         30,
			y:         0,
			target:    mouseTargetEditor,
			editorCol: 5,
			editorRow: -1,
		},
		{
			name:      "editor content click",
			x:         35,
			y:         5,
			target:    mouseTargetEditor,
			editorCol: 10,
			editorRow: 4,
		},
		{
			name:      "editor first content row",
			x:         25,
			y:         1,
			target:    mouseTargetEditor,
			editorCol: 0,
			editorRow: 0,
		},
		{
			name:   "info panel click",
			x:      80,
			y:      5,
			target: mouseTargetInfo,
		},
		{
			name:   "status bar click",
			x:      50,
			y:      29,
			target: mouseTargetStatus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := a.hitTestMouse(newMouseMsg(tt.x, tt.y))
			if result.target != tt.target {
				t.Errorf("target: got %d, want %d", result.target, tt.target)
			}
			if tt.target == mouseTargetEditor {
				if result.editorCol != tt.editorCol {
					t.Errorf("editorCol: got %d, want %d", result.editorCol, tt.editorCol)
				}
				if result.editorRow != tt.editorRow {
					t.Errorf("editorRow: got %d, want %d", result.editorRow, tt.editorRow)
				}
			}
		})
	}
}

func TestHitTestMouseNoSidePanels(t *testing.T) {
	cfg := config.Config{
		TreeWidth: 25,
		InfoWidth: 25,
	}
	a := App{
		cfg:    cfg,
		width:  100,
		height: 30,
	}

	result := a.hitTestMouse(newMouseMsg(50, 10))
	if result.target != mouseTargetEditor {
		t.Errorf("target: got %d, want mouseTargetEditor", result.target)
	}
	if result.editorCol != 50 {
		t.Errorf("editorCol: got %d, want 50", result.editorCol)
	}
	if result.editorRow != 9 {
		t.Errorf("editorRow: got %d, want 9", result.editorRow)
	}
}

func TestHitTestMouseTreeOnlyNoInfo(t *testing.T) {
	cfg := config.Config{
		TreeWidth: 25,
		InfoWidth: 25,
	}
	a := App{
		cfg:      cfg,
		width:    100,
		height:   30,
		showTree: true,
	}

	result := a.hitTestMouse(newMouseMsg(90, 5))
	if result.target != mouseTargetEditor {
		t.Errorf("target: got %d, want mouseTargetEditor", result.target)
	}
	if result.editorCol != 65 {
		t.Errorf("editorCol: got %d, want 65", result.editorCol)
	}
}
