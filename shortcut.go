package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const pluginActionCommand = pluginID + ".open"

type shortcutStatus int

const (
	shortcutInstalled shortcutStatus = iota
	shortcutAdded
	shortcutConflict
)

func herdrConfigPath() (string, error) {
	if path := strings.TrimSpace(os.Getenv("HERDR_CONFIG_PATH")); path != "" {
		return path, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve Herdr config directory: %w", err)
	}
	return filepath.Join(dir, "herdr", "config.toml"), nil
}

func installShortcutCmd() error {
	for _, env := range []string{"SSH_CONNECTION", "SSH_CLIENT", "SSH_TTY"} {
		if os.Getenv(env) != "" {
			return fmt.Errorf("refusing to edit Herdr config over SSH (%s is set); add the prefix+u binding manually", env)
		}
	}
	path, err := herdrConfigPath()
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	updated, status, err := applyShortcutInstall(string(raw))
	if err != nil {
		return err
	}
	if status == shortcutInstalled {
		fmt.Printf("Shortcut prefix+u already installed in %s\n", path)
		return nil
	}
	if err := os.WriteFile(path, []byte(updated), 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	herdr := os.Getenv("HERDR_BIN_PATH")
	if herdr == "" {
		herdr = "herdr"
	}
	if out, checkErr := exec.Command(herdr, "config", "check").CombinedOutput(); checkErr != nil {
		_ = os.WriteFile(path, raw, 0o600)
		return fmt.Errorf("herdr config check failed (%v), restored previous config: %s", checkErr, compactError(out))
	}
	fmt.Printf("Added prefix+u shortcut to %s\n", path)
	if err := exec.Command(herdr, "server", "reload-config").Run(); err != nil {
		fmt.Println("Run `herdr server reload-config` to apply the binding now.")
	}
	return nil
}

// applyShortcutInstall appends the plugin keybinding to Herdr config content.
// It never overwrites an existing prefix+u binding owned by another command.
func applyShortcutInstall(content string) (string, shortcutStatus, error) {
	inCommandBlock := false
	blockKey := ""
	blockCommand := ""
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "[[") {
			inCommandBlock = trimmed == "[[keys.command]]"
			blockKey, blockCommand = "", ""
			continue
		}
		if strings.HasPrefix(trimmed, "[") {
			inCommandBlock = false
			continue
		}
		if !inCommandBlock {
			continue
		}
		if value, ok := tomlStringField(trimmed, "key"); ok {
			blockKey = value
		}
		if value, ok := tomlStringField(trimmed, "command"); ok {
			blockCommand = value
		}
		if blockKey == "prefix+u" && blockCommand != "" {
			if blockCommand == pluginActionCommand {
				return content, shortcutInstalled, nil
			}
			return "", shortcutConflict, fmt.Errorf("prefix+u is already bound to %s", blockCommand)
		}
	}
	block := "[[keys.command]]\nkey = \"prefix+u\"\ntype = \"plugin_action\"\ncommand = \"" + pluginActionCommand + "\"\ndescription = \"open CPA quota\"\n"
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return content + "\n" + block, shortcutAdded, nil
}

func tomlStringField(line, field string) (string, bool) {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return "", false
	}
	if strings.TrimSpace(parts[0]) != field {
		return "", false
	}
	value := strings.TrimSpace(parts[1])
	if len(value) < 2 || !strings.HasPrefix(value, "\"") || !strings.HasSuffix(value, "\"") {
		return "", false
	}
	return value[1 : len(value)-1], true
}
