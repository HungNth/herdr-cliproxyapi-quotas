package herdr

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	PluginID       = "herdr-cliproxyapi-quotas"
	PaneEntrypoint = "quotas"
)

// viewRegistry maps a Herdr Tab to the plugin pane ID of its live Quota View.
type viewRegistry map[string]string

func (r viewRegistry) entryFor(tabID string) string {
	if tabID == "" {
		return ""
	}
	return r[tabID]
}

func registryPath() (string, error) {
	if dir := strings.TrimSpace(os.Getenv("HERDR_PLUGIN_STATE_DIR")); dir != "" {
		return filepath.Join(dir, "views.json"), nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve state directory: %w", err)
	}
	return filepath.Join(dir, "herdr", "plugins", PluginID, "state", "views.json"), nil
}

func loadViewRegistry(path string) (viewRegistry, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return viewRegistry{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read view registry: %w", err)
	}
	reg := viewRegistry{}
	if err := json.Unmarshal(raw, &reg); err != nil {
		return viewRegistry{}, fmt.Errorf("parse view registry: %w", err)
	}
	return reg, nil
}

func saveViewRegistry(path string, reg viewRegistry) error {
	raw, err := json.Marshal(reg)
	if err != nil {
		return fmt.Errorf("encode view registry: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return fmt.Errorf("write view registry: %w", err)
	}
	return nil
}

func clearStaleEntry(reg viewRegistry, tabID string) viewRegistry {
	// ponytail: race window if two shortcut invocations run concurrently; acceptable for a single-user keybinding
	cleared := viewRegistry{}
	for key, value := range reg {
		cleared[key] = value
	}
	delete(cleared, tabID)
	return cleared
}
