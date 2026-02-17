package editor

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

//go:embed nvim_init.lua
var defaultInitLua []byte

type ProfileMode string

const (
	ProfileManaged ProfileMode = "managed"
	ProfileUser    ProfileMode = "user"
)

// ConfigDir returns the Kopr Neovim config directory.
// Respects XDG_CONFIG_HOME, defaults to ~/.config/kopr.
func ConfigDir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "kopr"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("config dir: %w", err)
	}
	return filepath.Join(home, ".config", "kopr"), nil
}

// EnsureProfile sets up the Neovim config directory for the given mode.
// In managed mode, writes init.lua if it doesn't exist.
// In user mode, logs a warning if the directory doesn't exist.
func EnsureProfile(mode ProfileMode) error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}

	if mode == ProfileUser {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "warning: %s does not exist, nvim will start with no config\n", dir)
		}
		return nil
	}

	// Managed mode: create dir and write init.lua if missing
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	initPath := filepath.Join(dir, "init.lua")
	if _, err := os.Stat(initPath); os.IsNotExist(err) {
		if err := os.WriteFile(initPath, defaultInitLua, 0644); err != nil {
			return fmt.Errorf("write init.lua: %w", err)
		}
	}

	if err := ensurePlugins(); err != nil {
		return err
	}

	return nil
}

// ResetProfile overwrites init.lua with the embedded default.
func ResetProfile() error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	initPath := filepath.Join(dir, "init.lua")
	if err := os.WriteFile(initPath, defaultInitLua, 0644); err != nil {
		return fmt.Errorf("write init.lua: %w", err)
	}

	return nil
}

// DataDir returns the Kopr Neovim data directory.
// Respects XDG_DATA_HOME, defaults to ~/.local/share/kopr.
func DataDir() (string, error) {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "kopr"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("data dir: %w", err)
	}
	return filepath.Join(home, ".local", "share", "kopr"), nil
}

// managed plugins to install: {directory name, git URL}
var managedPlugins = []struct {
	name string
	url  string
}{
	{"no-clown-fiesta.nvim", "https://github.com/aktersnurra/no-clown-fiesta.nvim.git"},
	{"render-markdown.nvim", "https://github.com/MeanderingProgrammer/render-markdown.nvim.git"},
}

// ensurePlugins clones managed plugins into the nvim pack directory if missing.
func ensurePlugins() error {
	dataDir, err := DataDir()
	if err != nil {
		return err
	}
	packDir := filepath.Join(dataDir, "site", "pack", "kopr", "start")
	if err := os.MkdirAll(packDir, 0755); err != nil {
		return fmt.Errorf("create pack dir: %w", err)
	}

	for _, p := range managedPlugins {
		dest := filepath.Join(packDir, p.name)
		if _, err := os.Stat(dest); err == nil {
			continue // already installed
		}
		cmd := exec.Command("git", "clone", "--depth", "1", p.url, dest)
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("clone %s: %w", p.name, err)
		}
	}
	return nil
}

// CheckNvimVersion verifies that nvim is installed and >= 0.9.
func CheckNvimVersion() error {
	out, err := exec.Command("nvim", "--version").Output()
	if err != nil {
		return fmt.Errorf("nvim not found: %w", err)
	}

	// First line is like "NVIM v0.10.2"
	lines := strings.SplitN(string(out), "\n", 2)
	if len(lines) == 0 {
		return fmt.Errorf("could not parse nvim version")
	}

	version := strings.TrimSpace(lines[0])
	// Strip "NVIM v" prefix
	version = strings.TrimPrefix(version, "NVIM v")

	major, minor, err := parseSemver(version)
	if err != nil {
		return fmt.Errorf("could not parse nvim version %q: %w", version, err)
	}

	if major == 0 && minor < 9 {
		return fmt.Errorf("nvim >= 0.9 required, found %d.%d", major, minor)
	}

	return nil
}

func parseSemver(s string) (int, int, error) {
	parts := strings.SplitN(s, ".", 3)
	if len(parts) < 2 {
		return 0, 0, fmt.Errorf("invalid version: %s", s)
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}
	return major, minor, nil
}

// NvimEnv returns environment variables for the managed Neovim process.
func NvimEnv() []string {
	return []string{
		"NVIM_APPNAME=kopr",
		"TERM=xterm-256color",
	}
}
