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
	VaultPath         *string `toml:"vault_path"`
	Colorscheme       *string `toml:"colorscheme"`
	ColorschemeRepo   *string `toml:"colorscheme_repo"`
	NvimMode          *string `toml:"nvim_mode"`
	LeaderKey         *string `toml:"leader_key"`
	LeaderTimeout     *int    `toml:"leader_timeout"`
	AutoFormatOnSave  *bool   `toml:"auto_format_on_save"`
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
	if fc.Colorscheme != nil {
		cfg.Colorscheme = *fc.Colorscheme
	}
	if fc.ColorschemeRepo != nil {
		cfg.ColorschemeRepo = *fc.ColorschemeRepo
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
	if fc.AutoFormatOnSave != nil {
		cfg.AutoFormatOnSave = *fc.AutoFormatOnSave
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

	encErr := toml.NewEncoder(f).Encode(fc)
	closeErr := f.Close()
	if encErr != nil {
		return encErr
	}
	if closeErr != nil {
		return closeErr
	}
	return nil
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
