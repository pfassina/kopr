package config

import (
	"reflect"
	"testing"
)

func TestRestartRequired(t *testing.T) {
	base := Default()

	tests := []struct {
		name   string
		mutate func(*Config)
		want   []string
	}{
		{
			name:   "no changes",
			mutate: func(c *Config) {},
			want:   nil,
		},
		{
			name:   "hot-reloadable changes only",
			mutate: func(c *Config) { c.Colorscheme = "gruvbox"; c.LeaderTimeout = 1000; c.AutoFormatOnSave = false },
			want:   nil,
		},
		{
			name:   "vault path changed",
			mutate: func(c *Config) { c.VaultPath = "/elsewhere" },
			want:   []string{"vault_path"},
		},
		{
			name:   "nvim mode changed",
			mutate: func(c *Config) { c.NvimMode = "user" },
			want:   []string{"nvim_mode"},
		},
		{
			name:   "render math changed",
			mutate: func(c *Config) { c.RenderMath = !c.RenderMath },
			want:   []string{"render_math"},
		},
		{
			name:   "treesitter parsers changed",
			mutate: func(c *Config) { c.TreesitterParsers = "/parsers" },
			want:   []string{"treesitter_parsers"},
		},
		{
			name:   "multiple restart-required changes",
			mutate: func(c *Config) { c.VaultPath = "/elsewhere"; c.NvimMode = "user" },
			want:   []string{"vault_path", "nvim_mode"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next := base
			tt.mutate(&next)
			got := RestartRequired(base, next)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RestartRequired() = %v, want %v", got, tt.want)
			}
		})
	}
}
