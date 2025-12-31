package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hluaguo/commity/internal/ai"
	"github.com/hluaguo/commity/internal/config"
	"github.com/hluaguo/commity/internal/git"
	"github.com/hluaguo/commity/internal/tui"
)

var version = "0.1.0"

func main() {
	configPath := flag.String("config", "", "config file path")
	showVersion := flag.Bool("version", false, "show version")
	flag.Parse()

	if *showVersion {
		fmt.Printf("commity v%s\n", version)
		os.Exit(0)
	}

	if err := run(*configPath); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(configPath string) error {
	// Check if first run
	isFirstRun := !config.Exists()

	// Load config (uses defaults if first run)
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize git repository
	repo, err := git.New()
	if err != nil {
		return err
	}

	// Initialize AI client (may be nil if first run with no API key)
	var aiClient *ai.Client
	if !isFirstRun {
		aiClient, err = ai.New(&cfg.AI)
		if err != nil {
			return err
		}
	}

	// Initialize TUI model
	model, err := tui.New(cfg, repo, aiClient, isFirstRun)
	if err != nil {
		return err
	}

	// Run TUI
	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}
