package main

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestVersionStatusLineUpdateAvailable(t *testing.T) {
	t.Parallel()

	line := versionStatusLine(snapshotFor("7.2.154", "v7.2.155", ""), false)
	if !strings.Contains(line, "7.2.154") || !strings.Contains(line, "7.2.155") || !strings.Contains(line, "available") {
		t.Fatalf("version line = %q", line)
	}
}

func TestVersionStatusLineUpToDate(t *testing.T) {
	t.Parallel()

	line := versionStatusLine(snapshotFor("7.2.155", "v7.2.155", ""), false)
	if !strings.Contains(line, "7.2.155") || strings.Contains(line, "available") || strings.Contains(line, "ahead") {
		t.Fatalf("version line = %q", line)
	}
}

func TestVersionStatusLineChecking(t *testing.T) {
	t.Parallel()

	line := versionStatusLine(snapshotFor("7.2.154", "", ""), true)
	if !strings.Contains(line, "checking") {
		t.Fatalf("version line = %q", line)
	}
}

func TestVersionStatusLineUnknownCurrent(t *testing.T) {
	t.Parallel()

	line := versionStatusLine(snapshotFor("", "v7.2.155", ""), false)
	if !strings.Contains(line, "unknown") {
		t.Fatalf("version line = %q", line)
	}
}

func TestVersionStatusLineLatestFailure(t *testing.T) {
	t.Parallel()

	line := versionStatusLine(snapshotFor("7.2.154", "", "HTTP 502"), false)
	if !strings.Contains(line, "7.2.154") || !strings.Contains(line, "unavailable") {
		t.Fatalf("version line = %q", line)
	}
}

func TestVersionStatusLineAheadOfLatest(t *testing.T) {
	t.Parallel()

	line := versionStatusLine(snapshotFor("7.2.156", "v7.2.155", ""), false)
	if !strings.Contains(line, "ahead") || strings.Contains(line, "available") {
		t.Fatalf("version line = %q", line)
	}
}

func TestVersionStatusLineMalformedShowsRaw(t *testing.T) {
	t.Parallel()

	line := versionStatusLine(snapshotFor("dev-build", "v7.2.155", ""), false)
	if !strings.Contains(line, "dev-build") || strings.Contains(line, "available") {
		t.Fatalf("version line = %q", line)
	}
}

func TestViewShowsVersionStatus(t *testing.T) {
	t.Parallel()

	model := newUIModel(configPathForTest(t))
	model.mode = modeQuota
	model.hasConfig = true
	model.versionChecking = false
	model.latestVersion = "v7.2.155"
	model.snapshot = Snapshot{Groups: []ProviderQuota{{Provider: ProviderCodex, Title: "Codex", Accounts: []AccountQuota{{Provider: ProviderCodex, Name: "a@b.c"}}}}, FetchedAt: fixedTime()}
	model.snapshot.CurrentVersion = "7.2.154"

	view := model.View()
	if !strings.Contains(view, "7.2.154") || !strings.Contains(view, "7.2.155") {
		t.Fatalf("view missing version status: %q", view)
	}
}

func TestUpdateRejectsStaleVersionMessage(t *testing.T) {
	t.Parallel()

	model := newUIModel(configPathForTest(t))
	model.mode = modeQuota
	model.hasConfig = true
	model.refreshSeq = 2
	model.versionChecking = true

	updated, _ := model.Update(versionMsg{seq: 1, latest: "v1.0.0"})
	after := updated.(uiModel)
	if !after.versionChecking {
		t.Fatal("stale version message must not resolve the newer pending check")
	}
	if after.latestVersion == "v1.0.0" {
		t.Fatalf("stale version result applied: %q", after.latestVersion)
	}

	updated, _ = model.Update(versionMsg{seq: 2, latest: "v7.2.155"})
	after = updated.(uiModel)
	if after.latestVersion != "v7.2.155" {
		t.Fatalf("current version result dropped: %q", after.latestVersion)
	}
}

func TestUpdateRejectsStaleSnapshotMessage(t *testing.T) {
	t.Parallel()

	model := newUIModel(configPathForTest(t))
	model.hasConfig = true
	model.refreshSeq = 3
	model.fetching = true

	updated, _ := model.Update(snapshotMsg{seq: 2, snapshot: staleSnapshot()})
	after := updated.(uiModel)
	if !after.fetching {
		t.Fatal("stale snapshot message must not resolve the newer pending fetch")
	}
	if after.snapshot.CurrentVersion == "old-header" {
		t.Fatalf("stale snapshot applied: %q", after.snapshot.CurrentVersion)
	}

	current := Snapshot{FetchedAt: fixedTime()}
	current.CurrentVersion = "7.2.154"
	updated, _ = model.Update(snapshotMsg{seq: 3, snapshot: current})
	after = updated.(uiModel)
	if after.fetching {
		t.Fatal("current snapshot message must resolve the fetch")
	}
	if after.snapshot.CurrentVersion != "7.2.154" {
		t.Fatalf("current snapshot dropped: %q", after.snapshot.CurrentVersion)
	}
}

func configPathForTest(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "config.json")
}

func fixedTime() time.Time {
	return time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
}

func snapshotFor(current, latest, latestErr string) uiModel {
	snapshot := Snapshot{CurrentVersion: current}
	return uiModel{snapshot: snapshot, latestVersion: latest, latestErr: latestErr}
}

func staleSnapshot() Snapshot {
	stale := Snapshot{FetchedAt: fixedTime()}
	stale.CurrentVersion = "old-header"
	return stale
}

func TestRefreshKeyUsesOneGeneration(t *testing.T) {
	t.Parallel()

	model := newUIModel(configPathForTest(t))
	model.mode = modeQuota
	model.hasConfig = true
	model.refreshSeq = 5
	model.fetching = false

	updated, cmd := model.updateQuota(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}})
	after := updated.(uiModel)
	if cmd == nil {
		t.Fatal("R must issue refresh commands")
	}
	if !after.fetching || !after.versionChecking {
		t.Fatalf("R must set fetching and versionChecking: %v/%v", after.fetching, after.versionChecking)
	}

	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("R must batch snapshot and version commands, got %T", msg)
	}
	for _, sub := range batch {
		switch m := sub().(type) {
		case snapshotMsg:
			if m.seq != after.refreshSeq {
				t.Fatalf("snapshot seq = %d, want %d", m.seq, after.refreshSeq)
			}
		case versionMsg:
			if m.seq != after.refreshSeq {
				t.Fatalf("version seq = %d, want %d", m.seq, after.refreshSeq)
			}
		}
	}
}
