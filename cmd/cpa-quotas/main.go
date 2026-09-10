package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"cpa-quota/internal/config"
	"cpa-quota/internal/herdr"
	"cpa-quota/internal/ui"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "open" {
		if err := herdr.OpenQuotaView(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "shortcut" {
		if err := herdr.InstallShortcutCmd(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	path, err := config.Path()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	program := tea.NewProgram(ui.NewModel(path), tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
