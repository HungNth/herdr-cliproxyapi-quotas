package herdr

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"cpa-quota/internal/config"
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
	} else {
		t.Setenv("FAKE_HERDR_FAIL", "")
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

	if err := OpenQuotaView(); err != nil {
		t.Fatalf("OpenQuotaView() error = %v", err)
	}
	calls := readCalls(t, log)
	if len(calls) != 1 {
		t.Fatalf("calls = %v, want exactly one open", calls)
	}
	call := calls[0]
	for _, want := range []string{
		"plugin pane open",
		"--plugin " + config.PluginID,
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

	if err := OpenQuotaView(); err != nil {
		t.Fatalf("OpenQuotaView() error = %v", err)
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

	err := OpenQuotaView()
	if err == nil {
		t.Fatal("OpenQuotaView() must report a failed open")
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

	if err := OpenQuotaView(); err != nil {
		t.Fatalf("OpenQuotaView() error = %v", err)
	}
	raw, err := os.ReadFile(registryFile(t))
	if err != nil {
		t.Fatalf("read registry: %v", err)
	}
	if !strings.Contains(string(raw), tab) {
		t.Fatalf("registry missing tab entry: %s", raw)
	}
}

func TestInvokingFromWorkPaneFocusesExistingQuotaView(t *testing.T) {
	_, tab, invoker := setupLauncher(t, true)
	reg := viewRegistry{tab: "w1:quota9"}
	if err := saveViewRegistry(registryFile(t), reg); err != nil {
		t.Fatal(err)
	}
	log := fakeHerdrEnv(t, struct {
		existingPanes string
		fail          bool
	}{existingPanes: "w1:quota9," + invoker})

	if err := OpenQuotaView(); err != nil {
		t.Fatalf("OpenQuotaView() error = %v", err)
	}
	calls := readCalls(t, log)
	foundFocus := false
	for _, call := range calls {
		if strings.Contains(call, "plugin pane focus w1:quota9") {
			foundFocus = true
		}
		if strings.Contains(call, "plugin pane open") {
			t.Fatalf("must not open a new pane when one exists: %s", call)
		}
	}
	if !foundFocus {
		t.Fatalf("calls missing focus: %v", calls)
	}
}

func TestInvokingFromFocusedQuotaViewClosesIt(t *testing.T) {
	_, tab, _ := setupLauncher(t, true)
	quotaPane := "w1:quota9"
	t.Setenv("HERDR_PANE_ID", quotaPane)
	reg := viewRegistry{tab: quotaPane}
	if err := saveViewRegistry(registryFile(t), reg); err != nil {
		t.Fatal(err)
	}
	log := fakeHerdrEnv(t, struct {
		existingPanes string
		fail          bool
	}{existingPanes: quotaPane})

	if err := OpenQuotaView(); err != nil {
		t.Fatalf("OpenQuotaView() error = %v", err)
	}
	calls := readCalls(t, log)
	foundClose := false
	for _, call := range calls {
		if strings.Contains(call, "plugin pane close "+quotaPane) {
			foundClose = true
		}
		if strings.Contains(call, "plugin pane focus") {
			t.Fatalf("must not focus self when toggle-closing: %s", call)
		}
	}
	if !foundClose {
		t.Fatalf("calls missing close toggle: %v", calls)
	}
	afterReg, err := loadViewRegistry(registryFile(t))
	if err != nil {
		t.Fatal(err)
	}
	if afterReg[tab] != "" {
		t.Fatalf("closed pane must be removed from registry: %v", afterReg)
	}
}

func TestStaleRegistryEntryReopensFreshSplit(t *testing.T) {
	_, tab, invoker := setupLauncher(t, true)
	reg := viewRegistry{tab: "w1:deadpane"}
	if err := saveViewRegistry(registryFile(t), reg); err != nil {
		t.Fatal(err)
	}
	log := fakeHerdrEnv(t, struct {
		existingPanes string
		fail          bool
	}{existingPanes: invoker})

	if err := OpenQuotaView(); err != nil {
		t.Fatalf("OpenQuotaView() error = %v", err)
	}
	calls := readCalls(t, log)
	foundOpen := false
	for _, call := range calls {
		if strings.Contains(call, "plugin pane open") {
			foundOpen = true
		}
		if strings.Contains(call, "plugin pane focus w1:deadpane") {
			t.Fatalf("must not focus stale pane: %s", call)
		}
	}
	if !foundOpen {
		t.Fatalf("calls: %v | log path: %s", calls, log)
	}
	afterReg, err := loadViewRegistry(registryFile(t))
	if err != nil {
		t.Fatal(err)
	}
	if afterReg[tab] == "w1:deadpane" || afterReg[tab] == "" {
		t.Fatalf("registry must hold newly opened pane: %v", afterReg)
	}
}

func TestInvokingFromMovedQuotaPaneTogglesClose(t *testing.T) {
	_, _, _ = setupLauncher(t, true)
	t.Setenv("HERDR_TAB_ID", "w1:destTab")
	quotaPane := "w1:quotaMoved"
	t.Setenv("HERDR_PANE_ID", quotaPane)

	reg := viewRegistry{"w1:origTab": quotaPane}
	if err := saveViewRegistry(registryFile(t), reg); err != nil {
		t.Fatal(err)
	}
	log := fakeHerdrEnv(t, struct {
		existingPanes string
		fail          bool
	}{existingPanes: quotaPane})

	if err := OpenQuotaView(); err != nil {
		t.Fatalf("OpenQuotaView() error = %v", err)
	}
	calls := readCalls(t, log)
	foundClose := false
	for _, call := range calls {
		if strings.Contains(call, "plugin pane close "+quotaPane) {
			foundClose = true
		}
	}
	if !foundClose {
		t.Fatalf("must toggle-close when invoked from quota pane even if tab moved: %v", calls)
	}
	afterReg, err := loadViewRegistry(registryFile(t))
	if err != nil {
		t.Fatal(err)
	}
	if afterReg["w1:origTab"] != "" {
		t.Fatalf("closed pane must be removed from original tab entry: %v", afterReg)
	}
}
