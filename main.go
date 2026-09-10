package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "open" {
		if err := openPopup(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "install-shortcut" {
		if err := installShortcutCmd(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	path, err := configPath()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	program := tea.NewProgram(newUIModel(path), tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func openPopup() error {
	herdr := os.Getenv("HERDR_BIN_PATH")
	if herdr == "" {
		herdr = "herdr"
	}
	tabID := os.Getenv("HERDR_TAB_ID")
	path, err := registryPath()
	if err != nil {
		return err
	}
	reg, err := loadViewRegistry(path)
	if err != nil {
		return err
	}
	if pane := reg.entryFor(tabID); pane != "" {
		if paneExists(herdr, pane) {
			return runHerdr(herdr, "plugin", "pane", "focus", pane)
		}
		reg = clearStaleEntry(reg, tabID)
		if err := saveViewRegistry(path, reg); err != nil {
			return err
		}
	}
	out, err := openNewPane(herdr)
	if err != nil {
		return err
	}
	if tabID != "" {
		if pane := paneIDFromOpen(out); pane != "" {
			reg[tabID] = pane
			if err := saveViewRegistry(path, reg); err != nil {
				return err
			}
		}
	}
	return nil
}
func openNewPane(herdr string) (string, error) {
	args := []string{"plugin", "pane", "open", "--plugin", pluginID, "--entrypoint", paneEntrypoint}
	cmd := exec.Command(herdr, args...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	return stdout.String(), err
}

func paneIDFromOpen(out string) string {
	var payload struct {
		Result struct {
			PluginPane struct {
				Pane struct {
					PaneID string `json:"pane_id"`
				} `json:"pane"`
			} `json:"plugin_pane"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		return ""
	}
	return payload.Result.PluginPane.Pane.PaneID
}

func runHerdr(herdr string, args ...string) error {
	cmd := exec.Command(herdr, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func paneExists(herdr string, paneID string) bool {
	out, err := exec.Command(herdr, "pane", "get", paneID).Output()
	return err == nil && strings.Contains(string(out), `"pane_id":"`+paneID+`"`)
}
