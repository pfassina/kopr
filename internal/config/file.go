package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// fileConfig mirrors Config with pointer fields so we can distinguish
// "not set" from zero values when merging TOML.
type fileConfig struct {
	VaultPath     *string `toml:"vault_path"`
	Theme         *string `toml:"theme"`
	NvimMode      *string `toml:"nvim_mode"`
	LeaderKey     *string `toml:"leader_key"`
	LeaderTimeout *int    `toml:"leader_timeout"`
}

// ConfigDir returns the kopr config directory, respecting XDG_CONFIG_HOME.
func ConfigDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "kopr")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "kopr")
}

// ConfigPath returns the full path to config.toml.
func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.toml")
}

// LoadFile reads config.toml and merges non-nil fields into cfg.
// Returns true if the file existed, false otherwise.
func LoadFile(cfg *Config) (bool, error) {
	path := ConfigPath()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	var fc fileConfig
	if err := toml.Unmarshal(data, &fc); err != nil {
		return true, err
	}

	if fc.VaultPath != nil {
		cfg.VaultPath = ExpandHome(*fc.VaultPath)
	}
	if fc.Theme != nil {
		cfg.Theme = *fc.Theme
	}
	if fc.NvimMode != nil {
		cfg.NvimMode = *fc.NvimMode
	}
	if fc.LeaderKey != nil {
		cfg.LeaderKey = *fc.LeaderKey
	}
	if fc.LeaderTimeout != nil {
		cfg.LeaderTimeout = *fc.LeaderTimeout
	}

	return true, nil
}

// SaveFile writes a minimal config.toml with the given vault path.
func SaveFile(vaultPath string) error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Store with ~ for readability if under home dir.
	home, _ := os.UserHomeDir()
	display := vaultPath
	if home != "" && strings.HasPrefix(vaultPath, home+string(os.PathSeparator)) {
		display = "~" + vaultPath[len(home):]
	}

	fc := fileConfig{VaultPath: &display}
	f, err := os.Create(filepath.Join(dir, "config.toml"))
	if err != nil {
		return err
	}
	defer f.Close()

	return toml.NewEncoder(f).Encode(fc)
}

// ExpandHome replaces a leading ~ with the user's home directory.
func ExpandHome(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}
	home, _ := os.UserHomeDir()
	if path == "~" {
		return home
	}
	return filepath.Join(home, path[2:])
}
