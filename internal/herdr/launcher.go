package herdr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"cpa-quota/internal/config"
)

func OpenQuotaView() error {
	herdr := os.Getenv("HERDR_BIN_PATH")
	if herdr == "" {
		herdr = "herdr"
	}
	tabID := os.Getenv("HERDR_TAB_ID")
	path, err := registryPath()
	if err != nil {
		return err
	}
	release, err := lockRegistry(path)
	if err != nil {
		return err
	}
	defer release()
	reg, err := loadViewRegistry(path)
	if err != nil {
		return err
	}
	invokingPane := os.Getenv("HERDR_PANE_ID")
	if invokingPane != "" {
		for registeredTab, registeredPane := range reg {
			if registeredPane == invokingPane {
				if err := runHerdr(herdr, "plugin", "pane", "close", registeredPane); err != nil {
					return err
				}
				reg = clearStaleEntry(reg, registeredTab)
				return saveViewRegistry(path, reg)
			}
		}
	}

	if existing := reg.entryFor(tabID); existing != "" {
		if paneExists(herdr, existing) {
			if invokingPane != "" && invokingPane == existing {
				if err := runHerdr(herdr, "plugin", "pane", "close", existing); err != nil {
					return err
				}
				reg = clearStaleEntry(reg, tabID)
				return saveViewRegistry(path, reg)
			}
			return runHerdr(herdr, "plugin", "pane", "focus", existing)
		}
		reg = clearStaleEntry(reg, tabID)
		if err := saveViewRegistry(path, reg); err != nil {
			return err
		}
	}

	configured := hasSavedConfig()
	out, err := openNewPane(herdr, invokingPane, configured)
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

// hasSavedConfig reports whether a valid CPA Endpoint configuration exists.
func hasSavedConfig() bool {
	path, err := config.Path()
	if err != nil {
		return false
	}
	_, err = config.Load(path)
	return err == nil
}

// lockRegistry serializes registry mutations across concurrent plugin action processes.
// ponytail: one global lock file per plugin; per-tab locks only if multi-key throughput ever matters
func lockRegistry(path string) (func(), error) {
	lock, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open registry lock: %w", err)
	}
	if err := lockFile(lock); err != nil {
		lock.Close()
		return nil, fmt.Errorf("lock registry: %w", err)
	}
	return func() {
		unlockFile(lock)
		lock.Close()
	}, nil
}

func openNewPane(herdr string, targetPane string, configured bool) (string, error) {
	args := []string{
		"plugin", "pane", "open",
		"--plugin", PluginID,
		"--entrypoint", PaneEntrypoint,
		"--placement", "split",
		"--direction", "right",
	}
	if targetPane != "" {
		args = append(args, "--target-pane", targetPane)
	}
	if configured {
		args = append(args, "--no-focus")
	} else {
		args = append(args, "--focus")
	}
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
