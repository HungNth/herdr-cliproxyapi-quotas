package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var fakeHerdrPath string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "cpa-quota-fakeherdr")
	if err != nil {
		os.Exit(1)
	}
	bin := filepath.Join(dir, "fakeherdr-bin.exe")
	build := exec.Command("go", "build", "-o", bin, "./testdata/fakeherdr")
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		os.Stderr.WriteString("build fake herdr: " + err.Error() + "\n")
		os.Exit(1)
	}
	fakeHerdrPath = bin
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

func fakeHerdrEnv(t *testing.T, opts struct {
	existingPanes string
	fail          bool
}) string {
	t.Helper()
	log := filepath.Join(t.TempDir(), "calls.log")
	t.Setenv("HERDR_BIN_PATH", fakeHerdrPath)
	t.Setenv("FAKE_HERDR_LOG", log)
	t.Setenv("FAKE_HERDR_EXISTING_PANES", opts.existingPanes)
	if opts.fail {
		t.Setenv("FAKE_HERDR_FAIL", "1")
	}
	t.Cleanup(func() { os.Remove(log) })
	return log
}

func readCalls(t *testing.T, log string) []string {
	t.Helper()
	raw, err := os.ReadFile(log)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("read fake herdr log: %v", err)
	}
	var calls []string
	for _, line := range strings.Split(string(raw), "\n") {
		if line != "" {
			calls = append(calls, line)
		}
	}
	return calls
}

func setupLauncher(t *testing.T, withConfig bool) (log string, tabID string, paneID string) {
	t.Helper()
	configDir := t.TempDir()
	stateDir := t.TempDir()
	if withConfig {
		if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{"base_url":"http://127.0.0.1:8317","management_key":"k"}`), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("HERDR_PLUGIN_CONFIG_DIR", configDir)
	t.Setenv("HERDR_PLUGIN_STATE_DIR", stateDir)
	t.Setenv("HERDR_TAB_ID", "w1:t1")
	t.Setenv("HERDR_PANE_ID", "w1:p1")
	log = fakeHerdrEnv(t, struct {
		existingPanes string
		fail          bool
	}{existingPanes: ""})
	return log, "w1:t1", "w1:p1"
}

func registryFile(t *testing.T) string {
	t.Helper()
	dir := os.Getenv("HERDR_PLUGIN_STATE_DIR")
	if dir == "" {
		t.Fatal("state dir not set")
	}
	return filepath.Join(dir, "views.json")
}

func TestConfiguredOpenSplitsRightBesideInvoker(t *testing.T) {
	log, _, _ := setupLauncher(t, true)

	if err := openQuotaView(); err != nil {
		t.Fatalf("openQuotaView() error = %v", err)
	}
	calls := readCalls(t, log)
	if len(calls) != 1 {
		t.Fatalf("calls = %v, want exactly one open", calls)
	}
	call := calls[0]
	for _, want := range []string{
		"plugin pane open",
		"--plugin " + pluginID,
		"--entrypoint " + paneEntrypoint,
		"--placement split",
		"--direction right",
		"--target-pane w1:p1",
		"--no-focus",
	} {
		if !strings.Contains(call, want) {
			t.Fatalf("open call missing %q: %s", want, call)
		}
	}
}

func TestFirstUseOpenFocusesQuotaView(t *testing.T) {
	log, _, _ := setupLauncher(t, false)

	if err := openQuotaView(); err != nil {
		t.Fatalf("openQuotaView() error = %v", err)
	}
	calls := readCalls(t, log)
	if len(calls) != 1 {
		t.Fatalf("calls = %v, want exactly one open", calls)
	}
	if !strings.Contains(calls[0], "--focus") {
		t.Fatalf("first-use open must focus the configuration view: %s", calls[0])
	}
	if strings.Contains(calls[0], "--no-focus") {
		t.Fatalf("first-use open must not pass --no-focus: %s", calls[0])
	}
}

func TestFailedOpenKeepsRegistryClean(t *testing.T) {
	log, tab, _ := setupLauncher(t, true)
	t.Setenv("FAKE_HERDR_FAIL", "1")

	err := openQuotaView()
	if err == nil {
		t.Fatal("openQuotaView() must report a failed open")
	}
	if calls := readCalls(t, log); len(calls) != 1 {
		t.Fatalf("calls = %v, want the failed open", calls)
	}
	if raw, readErr := os.ReadFile(registryFile(t)); readErr == nil && strings.Contains(string(raw), tab) {
		t.Fatalf("failed open must not record registry: %s", raw)
	}
}

func TestQuotaPaneRecordedInRegistryAfterOpen(t *testing.T) {
	log, tab, _ := setupLauncher(t, true)
	_ = log

	if err := openQuotaView(); err != nil {
		t.Fatalf("openQuotaView() error = %v", err)
	}
	raw, err := os.ReadFile(registryFile(t))
	if err != nil {
		t.Fatalf("read registry: %v", err)
	}
	if !strings.Contains(string(raw), tab) {
		t.Fatalf("registry missing tab entry: %s", raw)
	}
}
