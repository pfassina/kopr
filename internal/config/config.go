package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	VaultPath       string
	Listen          string
	Serve           bool
	Colorscheme     string // vim colorscheme name passed to :colorscheme
	ColorschemeRepo string // GitHub owner/repo to git-clone (optional)
	TreeWidth       int
	InfoWidth       int
	ShowTree        bool
	ShowInfo        bool
	ShowStatus      bool
	LeaderKey       string
	LeaderTimeout   int // milliseconds
	NvimMode        string
	ResetNvimConfig bool

	// AutoFormatOnSave enables Kopr's deterministic Markdown formatter after save.
	AutoFormatOnSave bool
}

func Default() Config {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	return Config{
		VaultPath:     filepath.Join(home, "notes"),
		Listen:        ":2222",
		Serve:         false,
		Colorscheme:     "no-clown-fiesta",
		ColorschemeRepo: "aktersnurra/no-clown-fiesta.nvim",
		TreeWidth:       30,
		InfoWidth:     30,
		ShowTree:      true,
		ShowInfo:      true,
		ShowStatus:    true,
		LeaderKey:     " ",
		LeaderTimeout:    500,
		NvimMode:         "managed",
		AutoFormatOnSave: true,
	}
}
