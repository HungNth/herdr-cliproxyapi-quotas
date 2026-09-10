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

func TestRefreshIntervalDefaultsAndValidation(t *testing.T) {
	t.Parallel()

	t.Run("absent field defaults to 60", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.json")
		if err := os.WriteFile(path, []byte(`{"base_url":"http://127.0.0.1:8317","management_key":"k"}`), 0o600); err != nil {
			t.Fatal(err)
		}
		got, err := config.Load(path)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if got.RefreshInterval != 60 {
			t.Fatalf("RefreshInterval = %d, want 60", got.RefreshInterval)
		}
	})

	t.Run("zero field defaults to 60", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.json")
		if err := os.WriteFile(path, []byte(`{"base_url":"http://127.0.0.1:8317","management_key":"k","refresh_interval":0}`), 0o600); err != nil {
			t.Fatal(err)
		}
		got, err := config.Load(path)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if got.RefreshInterval != 60 {
			t.Fatalf("RefreshInterval = %d, want 60", got.RefreshInterval)
		}
	})

	t.Run("legal custom interval round trips", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.json")
		cfg := config.Config{BaseURL: "http://127.0.0.1:8317", ManagementKey: "k", RefreshInterval: 10}
		if err := config.Save(path, cfg); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
		got, err := config.Load(path)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if got.RefreshInterval != 10 {
			t.Fatalf("RefreshInterval = %d, want 10", got.RefreshInterval)
		}
	})

	t.Run("below minimum rejected", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.json")
		cfg := config.Config{BaseURL: "http://127.0.0.1:8317", ManagementKey: "k", RefreshInterval: 4}
		if err := config.Save(path, cfg); err == nil {
			t.Fatal("expected error for interval below 5 seconds")
		}
	})

	t.Run("non numeric in JSON rejected", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.json")
		if err := os.WriteFile(path, []byte(`{"base_url":"http://127.0.0.1:8317","management_key":"k","refresh_interval":"ten"}`), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := config.Load(path); err == nil {
			t.Fatal("expected parse error for non-numeric interval")
		}
	})
}
