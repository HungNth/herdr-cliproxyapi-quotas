package config_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"cpa-quota/internal/config"
)

func TestSaveConfigRoundTripsWithPrivatePermissions(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "nested", "config.json")
	want := config.Config{BaseURL: "http://127.0.0.1:8317/", ManagementKey: " secret "}
	if err := config.Save(path, want); err != nil {
		t.Fatalf("saveConfig() error = %v", err)
	}
	got, err := config.Load(path)
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}
	if got.BaseURL != "http://127.0.0.1:8317" || got.ManagementKey != "secret" {
		t.Fatalf("loaded config = %#v", got)
	}
	if runtime.GOOS != "windows" {
		stat, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat config: %v", err)
		}
		if stat.Mode().Perm() != 0o600 {
			t.Fatalf("config permissions = %o, want 600", stat.Mode().Perm())
		}
	}
}
