package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadViewRegistryMissingFileReturnsEmpty(t *testing.T) {
	t.Parallel()

	reg, err := loadViewRegistry(filepath.Join(t.TempDir(), "views.json"))
	if err != nil {
		t.Fatalf("loadViewRegistry() error = %v", err)
	}
	if len(reg) != 0 {
		t.Fatalf("registry = %#v, want empty", reg)
	}
}

func TestSaveThenLoadViewRegistryRoundTrip(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "views.json")
	if err := saveViewRegistry(path, viewRegistry{"w1:t1": "w1:p2"}); err != nil {
		t.Fatalf("saveViewRegistry() error = %v", err)
	}
	reg, err := loadViewRegistry(path)
	if err != nil {
		t.Fatalf("loadViewRegistry() error = %v", err)
	}
	if reg["w1:t1"] != "w1:p2" {
		t.Fatalf("registry = %#v, want w1:t1 -> w1:p2", reg)
	}
}

func TestSaveViewRegistryIsJSON(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "views.json")
	if err := saveViewRegistry(path, nil); err != nil {
		t.Fatalf("saveViewRegistry() error = %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read registry: %v", err)
	}
	var reg viewRegistry
	if err := json.Unmarshal(raw, &reg); err != nil {
		t.Fatalf("registry is not JSON: %v", err)
	}
}

func TestFocusExistingPaneTakesPrecedence(t *testing.T) {
	t.Parallel()

	actions, err := planPopupActions(viewRegistry{"w1:t1": "w1:p2"}, "w1:t1")
	if err != nil {
		t.Fatalf("planPopupActions() error = %v", err)
	}
	want := []popupAction{popupFocus("w1:p2")}
	if len(actions) != len(want) || actions[0] != want[0] {
		t.Fatalf("actions = %#v, want %v", actions, want)
	}
}

func TestUnknownTabOpensNewView(t *testing.T) {
	t.Parallel()

	actions, err := planPopupActions(viewRegistry{"w1:t1": "w1:p2"}, "w1:t2")
	if err != nil {
		t.Fatalf("planPopupActions() error = %v", err)
	}
	want := []popupAction{popupOpen()}
	if len(actions) != len(want) || actions[0] != want[0] {
		t.Fatalf("actions = %#v, want %v", actions, want)
	}
}

func TestMissingTabIDOpensNewView(t *testing.T) {
	t.Parallel()

	actions, err := planPopupActions(viewRegistry{}, "")
	if err != nil {
		t.Fatalf("planPopupActions() error = %v", err)
	}
	want := []popupAction{popupOpen()}
	if len(actions) != len(want) || actions[0] != want[0] {
		t.Fatalf("actions = %#v, want %v", actions, want)
	}
}

func TestClearStaleEntryKeepsOtherTabs(t *testing.T) {
	t.Parallel()

	cleared := clearStaleEntry(viewRegistry{"w1:t1": "w1:p2", "w1:t3": "w1:p7"}, "w1:t1")
	if _, ok := cleared["w1:t1"]; ok {
		t.Fatalf("stale entry survived: %#v", cleared)
	}
	if cleared["w1:t3"] != "w1:p7" {
		t.Fatalf("other entry lost: %#v", cleared)
	}
}
