package herdr

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"cpa-quota/internal/config"
)

const pluginActionCommand = config.PluginID + ".open"

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

func InstallShortcutCmd() error {
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
	hasPrefixU := false
	hasOtherCommand := false
	hasExactCommand := false

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") {
			inCommandBlock = strings.HasPrefix(trimmed, "[[keys.command]]")
			hasPrefixU = false
			hasOtherCommand = false
			hasExactCommand = false
			continue
		}
		if !inCommandBlock {
			continue
		}
		if matchStringField(trimmed, "key", "prefix+u") {
			hasPrefixU = true
		}
		if strings.HasPrefix(trimmed, "command") {
			if matchStringField(trimmed, "command", pluginActionCommand) {
				hasExactCommand = true
			} else {
				hasOtherCommand = true
			}
		}
		if hasPrefixU && hasExactCommand {
			return content, shortcutInstalled, nil
		}
		if hasPrefixU && hasOtherCommand {
			return content, shortcutConflict, fmt.Errorf("prefix+u already bound to another command in Herdr config")
		}
	}

	block := strings.Join([]string{
		"[[keys.command]]",
		`key = "prefix+u"`,
		`type = "plugin_action"`,
		fmt.Sprintf(`command = %q`, pluginActionCommand),
		`description = "open CPA quotas"`,
	}, "\n") + "\n"

	if len(content) > 0 && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return content + "\n" + block, shortcutAdded, nil
}

func matchStringField(line, field, expected string) bool {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) != field {
		return false
	}
	value := strings.TrimSpace(parts[1])
	return strings.Trim(value, `"'`) == expected
}

func compactError(raw []byte) string {
	message := strings.TrimSpace(string(raw))
	if message == "" {
		return "unknown error"
	}
	message = strings.ReplaceAll(message, "\r\n", " ")
	message = strings.ReplaceAll(message, "\n", " ")
	if len(message) > 160 {
		return message[:157] + "..."
	}
	return message
}
