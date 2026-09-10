package herdr

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestApplyShortcutInstall(t *testing.T) {
	t.Parallel()
	binding := "[[keys.command]]\nkey = \"prefix+u\"\ntype = \"plugin_action\"\ncommand = \"" + pluginActionCommand + "\"\ndescription = \"open CPA quota\"\n"

	t.Run("adds binding when missing", func(t *testing.T) {
		updated, status, err := applyShortcutInstall("[keys]\nprefix = \"ctrl+b\"\n")
		if err != nil || status != shortcutAdded {
			t.Fatalf("status=%v err=%v", status, err)
		}
		if !strings.Contains(updated, "command = \""+pluginActionCommand+"\"") {
			t.Fatalf("binding missing: %q", updated)
		}
	})

	t.Run("idempotent for identical binding", func(t *testing.T) {
		content := "[keys]\nprefix = \"ctrl+b\"\n\n" + binding
		updated, status, err := applyShortcutInstall(content)
		if err != nil || status != shortcutInstalled || updated != content {
			t.Fatalf("status=%v err=%v changed=%v", status, err, updated != content)
		}
	})

	t.Run("refuses conflicting binding", func(t *testing.T) {
		content := "[[keys.command]]\nkey = \"prefix+u\"\ntype = \"shell\"\ncommand = \"lazygit\"\n"
		if _, status, err := applyShortcutInstall(content); err == nil || status != shortcutConflict {
			t.Fatalf("status=%v err=%v", status, err)
		}
	})
}

func TestInstallShortcutCmdSSHRefusal(t *testing.T) {
	t.Setenv("SSH_CONNECTION", "1 2 3 4")
	if err := InstallShortcutCmd(); err == nil || !strings.Contains(err.Error(), "SSH") {
		t.Fatalf("expected SSH refusal, got %v", err)
	}
}

func TestInstallShortcutCmdEndToEnd(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("HERDR_CONFIG_PATH", configFile)
	t.Setenv("HERDR_BIN_PATH", stubHerdrPath(t))
	if err := os.WriteFile(configFile, []byte("[keys]\nprefix = \"ctrl+b\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := InstallShortcutCmd(); err != nil {
		t.Fatalf("installShortcutCmd() error = %v", err)
	}
	raw, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "command = \""+pluginActionCommand+"\"") {
		t.Fatalf("binding not written: %s", raw)
	}
	if err := InstallShortcutCmd(); err != nil {
		t.Fatalf("second install should be idempotent, got %v", err)
	}
}

// stubHerdrPath returns a platform-appropriate no-op executable for herdr CLI stubbing.
func stubHerdrPath(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("SystemRoot"), "System32", "cmd.exe")
	}
	return "/usr/bin/true"
}
