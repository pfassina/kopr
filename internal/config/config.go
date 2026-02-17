package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	VaultPath      string
	Listen         string
	Serve          bool
	Theme          string
	TreeWidth      int
	InfoWidth      int
	ShowTree       bool
	ShowInfo       bool
	ShowStatus     bool
	LeaderKey      string
	LeaderTimeout  int // milliseconds
	NvimMode       string
	ResetNvimConfig bool
}

func Default() Config {
	home, _ := os.UserHomeDir()
	return Config{
		VaultPath:     filepath.Join(home, "notes"),
		Listen:        ":2222",
		Serve:         false,
		Theme:         "catppuccin",
		TreeWidth:     30,
		InfoWidth:     30,
		ShowTree:      true,
		ShowInfo:      true,
		ShowStatus:    true,
		LeaderKey:     " ",
		LeaderTimeout: 500,
		NvimMode:      "managed",
	}
}
