package editor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigDir_Default(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	dir, err := ConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".config", "kopr")
	if dir != want {
		t.Errorf("ConfigDir() = %q, want %q", dir, want)
	}
}

func TestConfigDir_XDG(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	dir, err := ConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(tmp, "kopr")
	if dir != want {
		t.Errorf("ConfigDir() = %q, want %q", dir, want)
	}
}

func TestEnsureProfile_Managed_WritesInitLua(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	if err := EnsureProfile(ProfileManaged); err != nil {
		t.Fatal(err)
	}

	initPath := filepath.Join(tmp, "kopr", "init.lua")
	data, err := os.ReadFile(initPath)
	if err != nil {
		t.Fatalf("init.lua not created: %v", err)
	}
	if len(data) == 0 {
		t.Error("init.lua is empty")
	}
	if string(data) != string(defaultInitLua) {
		t.Error("init.lua content doesn't match embedded default")
	}
}

func TestEnsureProfile_Managed_DoesNotOverwrite(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	dir := filepath.Join(tmp, "kopr")
	os.MkdirAll(dir, 0755)
	custom := []byte("-- my custom config\n")
	os.WriteFile(filepath.Join(dir, "init.lua"), custom, 0644)

	if err := EnsureProfile(ProfileManaged); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "init.lua"))
	if string(data) != string(custom) {
		t.Error("EnsureProfile overwrote existing init.lua")
	}
}

func TestEnsureProfile_User_NoError(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	// Directory doesn't exist — should still succeed (just warns)
	if err := EnsureProfile(ProfileUser); err != nil {
		t.Fatalf("EnsureProfile(user) should not error: %v", err)
	}
}

func TestResetProfile_Overwrites(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	dir := filepath.Join(tmp, "kopr")
	os.MkdirAll(dir, 0755)
	os.WriteFile(filepath.Join(dir, "init.lua"), []byte("-- custom"), 0644)

	if err := ResetProfile(); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "init.lua"))
	if string(data) != string(defaultInitLua) {
		t.Error("ResetProfile did not restore default init.lua")
	}
}

func TestNvimEnv(t *testing.T) {
	env := NvimEnv()

	hasAppName := false
	hasTerm := false
	for _, e := range env {
		if e == "NVIM_APPNAME=kopr" {
			hasAppName = true
		}
		if e == "TERM=xterm-256color" {
			hasTerm = true
		}
	}

	if !hasAppName {
		t.Error("NvimEnv missing NVIM_APPNAME=kopr")
	}
	if !hasTerm {
		t.Error("NvimEnv missing TERM=xterm-256color")
	}
}

func TestParseSemver(t *testing.T) {
	tests := []struct {
		input string
		major int
		minor int
		err   bool
	}{
		{"0.9.5", 0, 9, false},
		{"0.10.2", 0, 10, false},
		{"1.0.0", 1, 0, false},
		{"invalid", 0, 0, true},
	}

	for _, tt := range tests {
		major, minor, err := parseSemver(tt.input)
		if tt.err && err == nil {
			t.Errorf("parseSemver(%q) expected error", tt.input)
			continue
		}
		if !tt.err && err != nil {
			t.Errorf("parseSemver(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if major != tt.major || minor != tt.minor {
			t.Errorf("parseSemver(%q) = %d.%d, want %d.%d", tt.input, major, minor, tt.major, tt.minor)
		}
	}
}
