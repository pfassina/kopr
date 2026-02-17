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
	theme := flag.String("theme", cfg.Theme, "UI theme (catppuccin, nord, gruvbox, tokyo-night)")
	nvimMode := flag.String("nvim-mode", cfg.NvimMode, "neovim config mode: managed|user")
	leaderKey := flag.String("leader-key", cfg.LeaderKey, "leader key (default: space)")
	leaderTimeout := flag.Int("leader-timeout", cfg.LeaderTimeout, "leader timeout in ms")
	resetNvimConfig := flag.Bool("reset-nvim-config", false, "reset managed Neovim config to defaults")

	flag.Parse()

	cfg.VaultPath = *vault
	cfg.Serve = *serve
	cfg.Listen = *listen
	cfg.Theme = *theme
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
	}

	if err := os.MkdirAll(cfg.VaultPath, 0755); err != nil {
		fmt.Fprintln(os.Stderr, "error creating vault dir:", err)
		os.Exit(1)
	}
	_ = os.MkdirAll(filepath.Join(cfg.VaultPath, ".kopr"), 0755)

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

	if cfg.Serve {
		runServe(cfg)
		return
	}
	runLocal(cfg)
}

func runLocal(cfg config.Config) {
	a := app.New(cfg)
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
		_ = s.Close()
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
