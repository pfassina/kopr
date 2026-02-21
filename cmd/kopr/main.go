package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pfassina/kopr/internal/app"
	"github.com/pfassina/kopr/internal/config"
	"github.com/pfassina/kopr/internal/editor"
	"github.com/pfassina/kopr/internal/ssh"
)

func main() {
	cfg := config.Default()
	configExisted, err := config.LoadFile(&cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error loading config:", err)
		os.Exit(1)
	}

	vault := flag.String("vault", cfg.VaultPath, "path to vault directory")
	serve := flag.Bool("serve", cfg.Serve, "run in SSH server mode")
	listen := flag.String("listen", cfg.Listen, "listen address for --serve (e.g. :2222)")
	colorscheme := flag.String("colorscheme", cfg.Colorscheme, "vim colorscheme name")
	nvimMode := flag.String("nvim-mode", cfg.NvimMode, "neovim config mode: managed|user")
	leaderKey := flag.String("leader-key", cfg.LeaderKey, "leader key (default: space)")
	leaderTimeout := flag.Int("leader-timeout", cfg.LeaderTimeout, "leader timeout in ms")
	resetNvimConfig := flag.Bool("reset-nvim-config", false, "reset managed Neovim config to defaults")

	flag.Parse()

	// Normalize vault path: expand ~ and make absolute so Neovim cwd + :w use stable paths.
	cfg.VaultPath = config.ExpandHome(*vault)
	if abs, err := filepath.Abs(cfg.VaultPath); err == nil {
		cfg.VaultPath = abs
	}
	cfg.Serve = *serve
	cfg.Listen = *listen
	cfg.Colorscheme = *colorscheme
	cfg.NvimMode = *nvimMode
	cfg.LeaderKey = *leaderKey
	cfg.LeaderTimeout = *leaderTimeout
	cfg.ResetNvimConfig = *resetNvimConfig

	// First-run: if no config file exists and vault wasn't explicitly provided,
	// prompt for a vault path and persist it.
	if !configExisted && !argHas("--vault") {
		res, err := config.RunSetup()
		if err != nil {
			fmt.Fprintln(os.Stderr, "setup failed:", err)
			os.Exit(1)
		}
		if res.Cancelled {
			os.Exit(0)
		}
		cfg.VaultPath = res.VaultPath
		if abs, err := filepath.Abs(cfg.VaultPath); err == nil {
			cfg.VaultPath = abs
		}
	}

	if err := os.MkdirAll(cfg.VaultPath, 0755); err != nil {
		fmt.Fprintln(os.Stderr, "error creating vault dir:", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(filepath.Join(cfg.VaultPath, ".kopr"), 0755); err != nil {
		fmt.Fprintln(os.Stderr, "error creating .kopr dir:", err)
		os.Exit(1)
	}

	if err := editor.CheckNvimVersion(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if cfg.ResetNvimConfig {
		if err := editor.ResetProfile(); err != nil {
			fmt.Fprintln(os.Stderr, "reset nvim config:", err)
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "reset Neovim config")
	}

	if err := editor.EnsureProfile(editor.ProfileMode(cfg.NvimMode)); err != nil {
		fmt.Fprintln(os.Stderr, "neovim profile:", err)
		os.Exit(1)
	}
	if err := editor.EnsureThemePlugin(cfg.ColorschemeRepo); err != nil {
		fmt.Fprintln(os.Stderr, "colorscheme plugin:", err)
		os.Exit(1)
	}

	if cfg.Serve {
		runServe(cfg)
		return
	}
	runLocal(cfg)
}

func runLocal(cfg config.Config) {
	// Ensure lipgloss/termenv uses truecolor so extracted colorscheme colors
	// render accurately instead of being approximated to the 256-color palette.
	if err := os.Setenv("COLORTERM", "truecolor"); err != nil {
		fmt.Fprintln(os.Stderr, "error setting COLORTERM:", err)
		os.Exit(1)
	}

	a := app.New(cfg)
	a.SetOutput(os.Stdout)
	p := tea.NewProgram(&a, tea.WithAltScreen(), tea.WithMouseCellMotion())
	a.SetProgram(p)
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

func runServe(cfg config.Config) {
	s, err := ssh.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	// Graceful shutdown on SIGINT/SIGTERM.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sig
		if err := s.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "error closing server: %v\n", err)
		}
	}()

	if err := s.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func argHas(name string) bool {
	for _, a := range os.Args[1:] {
		if a == name || a == "-"+name[2:] {
			return true
		}
	}
	return false
}
