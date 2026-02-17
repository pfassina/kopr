package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandHome(t *testing.T) {
	home, _ := os.UserHomeDir()
	tests := []struct {
		input string
		want  string
	}{
		{"~/notes", filepath.Join(home, "notes")},
		{"~", home},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ExpandHome(tt.input)
			if got != tt.want {
				t.Errorf("ExpandHome(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestLoadFile_Missing(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	cfg := Default()
	exists, err := LoadFile(&cfg)
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("LoadFile should return false for missing file")
	}
}

func TestLoadFile_Partial(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	dir := filepath.Join(tmp, "kopr")
	os.MkdirAll(dir, 0755)
	os.WriteFile(filepath.Join(dir, "config.toml"), []byte(`theme = "dracula"`+"\n"), 0644)

	cfg := Default()
	exists, err := LoadFile(&cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("LoadFile should return true for existing file")
	}
	if cfg.Theme != "dracula" {
		t.Errorf("Theme = %q, want %q", cfg.Theme, "dracula")
	}
	// VaultPath should remain the default since it wasn't in the file.
	home, _ := os.UserHomeDir()
	if cfg.VaultPath != filepath.Join(home, "notes") {
		t.Errorf("VaultPath changed unexpectedly: %q", cfg.VaultPath)
	}
}

func TestLoadFile_Full(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	dir := filepath.Join(tmp, "kopr")
	os.MkdirAll(dir, 0755)
	content := `vault_path = "~/docs"
theme = "nord"
nvim_mode = "user"
leader_key = ","
leader_timeout = 300
`
	os.WriteFile(filepath.Join(dir, "config.toml"), []byte(content), 0644)

	cfg := Default()
	exists, err := LoadFile(&cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("LoadFile should return true")
	}

	home, _ := os.UserHomeDir()
	wantPath := filepath.Join(home, "docs")
	if cfg.VaultPath != wantPath {
		t.Errorf("VaultPath = %q, want %q", cfg.VaultPath, wantPath)
	}
	if cfg.Theme != "nord" {
		t.Errorf("Theme = %q, want %q", cfg.Theme, "nord")
	}
	if cfg.NvimMode != "user" {
		t.Errorf("NvimMode = %q, want %q", cfg.NvimMode, "user")
	}
	if cfg.LeaderKey != "," {
		t.Errorf("LeaderKey = %q, want %q", cfg.LeaderKey, ",")
	}
	if cfg.LeaderTimeout != 300 {
		t.Errorf("LeaderTimeout = %d, want %d", cfg.LeaderTimeout, 300)
	}
}

func TestSaveFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	home, _ := os.UserHomeDir()
	vaultPath := filepath.Join(home, "my-vault")

	if err := SaveFile(vaultPath); err != nil {
		t.Fatal(err)
	}

	// Verify the file was created and can be loaded back.
	cfg := Default()
	exists, err := LoadFile(&cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("config file should exist after SaveFile")
	}
	if cfg.VaultPath != vaultPath {
		t.Errorf("VaultPath = %q, want %q", cfg.VaultPath, vaultPath)
	}
}

func TestConfigDir_XDG(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	want := filepath.Join(tmp, "kopr")
	if got := ConfigDir(); got != want {
		t.Errorf("ConfigDir() = %q, want %q", got, want)
	}
}

func TestConfigDir_Default(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".config", "kopr")
	if got := ConfigDir(); got != want {
		t.Errorf("ConfigDir() = %q, want %q", got, want)
	}
}
