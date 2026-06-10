package app

import (
	"path/filepath"
	"testing"
)

func TestIsKoprConfigFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	koprDir := filepath.Join(dir, "kopr")

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"config.toml", filepath.Join(koprDir, "config.toml"), true},
		{"config.toml unclean path", filepath.Join(koprDir, ".", "config.toml"), true},
		{"managed init.lua", filepath.Join(koprDir, "init.lua"), true},
		{"vault note", filepath.Join(dir, "notes", "note.md"), false},
		{"other toml", filepath.Join(dir, "other", "config.toml"), false},
		{"sibling file in kopr dir", filepath.Join(koprDir, "kopr_latex.sh"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isKoprConfigFile(tt.path); got != tt.want {
				t.Errorf("isKoprConfigFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
