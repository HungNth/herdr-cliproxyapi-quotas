package main

import (
	"github.com/rivo/uniseg"
	"path/filepath"
	"regexp"
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

func renderWindows(width int) string {
	reset := fixedTime().Add(90 * time.Minute)
	snapshot := Snapshot{
		FetchedAt: fixedTime(),
		Groups: []ProviderQuota{{
			Provider: ProviderCodex,
			Title:    "Codex",
			Accounts: []AccountQuota{{
				Provider: ProviderCodex,
				Name:     "a@b.c",
				Windows: []QuotaWindow{
					{Label: "5-hour", Remaining: floatPtr(86), ResetAt: &reset},
					{Label: "Weekly", Remaining: floatPtr(70), ResetAt: &reset},
				},
				ManualResets: intPtr(3),
			}},
		}},
	}
	return renderSnapshot(snapshot, width, fixedTime())
}

func floatPtr(v float64) *float64 { return &v }
func intPtr(v int) *int           { return &v }

func plainLines(t *testing.T, rendered string) []string {
	t.Helper()
	stripped := regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`).ReplaceAllString(rendered, "")
	return strings.Split(stripped, "\n")
}

func TestCompactRowsShowMetadataThenThinBar(t *testing.T) {
	t.Parallel()

	lines := plainLines(t, renderWindows(80))
	joined := strings.Join(lines, "\n")
	if strings.Contains(joined, "remaining") || strings.Contains(joined, "· resets") {
		t.Fatalf("verbose prose must go: %q", joined)
	}
	if strings.Contains(joined, "(") {
		t.Fatalf("countdown parentheses must go: %q", joined)
	}
	if !strings.Contains(joined, "86%") || !strings.Contains(joined, "in 1h30m") {
		t.Fatalf("metadata row missing pct/countdown: %q", joined)
	}
	if !strings.Contains(joined, "10/09 20:30") {
		t.Fatalf("absolute reset time missing: %q", joined)
	}
	thick := strings.Contains(joined, "█")
	if thick {
		t.Fatal("block glyphs must be replaced by thin lines")
	}
	if !strings.Contains(joined, "━") || !strings.Contains(joined, "─") {
		t.Fatalf("thin bar glyphs missing: %q", joined)
	}
	if !strings.Contains(joined, "Manual resets  3") {
		t.Fatalf("manual resets row wrong: %q", joined)
	}
}

func TestCompactBarRowSitsBelowPercentage(t *testing.T) {
	t.Parallel()

	lines := plainLines(t, renderWindows(80))
	var metaIndex, barIndex = -1, -1
	for i, line := range lines {
		if strings.Contains(line, "5-hour") && strings.Contains(line, "86%") {
			metaIndex = i
		}
		if strings.Contains(line, "━") && metaIndex >= 0 && barIndex < 0 {
			barIndex = i
		}
	}
	if metaIndex < 0 || barIndex != metaIndex+1 {
		t.Fatalf("bar row must directly follow metadata: meta=%d bar=%d", metaIndex, barIndex)
	}
	barLine := lines[barIndex]
	if !strings.HasPrefix(barLine, " ") {
		t.Fatalf("bar row must be indented: %q", barLine)
	}
	if len(barLine)-len(strings.TrimLeft(barLine, " ")) != len(lines[metaIndex])-len(strings.TrimLeft(lines[metaIndex], " "))+len("5-hour")+2 {
		t.Fatalf("bar must start under percentage column: %q vs %q", barLine, lines[metaIndex])
	}
}

func TestCompactUnknownAndReadyStates(t *testing.T) {
	t.Parallel()

	past := fixedTime().Add(-time.Hour)
	snapshot := Snapshot{
		FetchedAt: fixedTime(),
		Groups: []ProviderQuota{{
			Provider: ProviderClaude,
			Title:    "Claude",
			Accounts: []AccountQuota{{
				Provider: ProviderClaude,
				Name:     "x@y.z",
				Windows: []QuotaWindow{
					{Label: "5-hour", Remaining: nil, ResetAt: nil},
					{Label: "Weekly", Remaining: floatPtr(40), ResetAt: &past},
				},
			}},
		}},
	}
	joined := strings.Join(plainLines(t, renderSnapshot(snapshot, 80, fixedTime())), "\n")
	if !strings.Contains(joined, "5-hour  —  reset —") {
		t.Fatalf("unknown state wrong: %q", joined)
	}
	if !strings.Contains(joined, "Weekly   40%  ready") {
		t.Fatalf("ready state wrong: %q", joined)
	}
}

func renderFullSnapshot(width int) string {
	reset := fixedTime().Add(90 * time.Minute)
	snapshot := Snapshot{
		FetchedAt: fixedTime(),
		Groups: []ProviderQuota{
			{
				Provider: ProviderCodex,
				Title:    "Codex",
				Accounts: []AccountQuota{{
					Provider: ProviderCodex,
					Name:     "user@example.com",
					Windows: []QuotaWindow{
						{Label: "5-hour", Remaining: floatPtr(86), ResetAt: &reset},
						{Label: "Weekly", Remaining: floatPtr(70), ResetAt: &reset},
					},
					ManualResets: intPtr(3),
				}},
			},
			{
				Provider: ProviderAntigravity,
				Title:    "Antigravity",
				Accounts: []AccountQuota{{
					Provider: ProviderAntigravity,
					Name:     "user@example.com",
					Windows: []QuotaWindow{
						{Label: "Claude & GPT models", Remaining: floatPtr(100), ResetAt: &reset},
						{Label: "Gemini models", Remaining: floatPtr(95), ResetAt: &reset},
					},
				}},
			},
		},
	}
	return renderSnapshot(snapshot, width, fixedTime())
}

func lineWidth(s string) int {
	clean := regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`).ReplaceAllString(s, "")
	return uniseg.StringWidth(clean)
}

func TestNarrowPaneFittingNoLineOverflows(t *testing.T) {
	t.Parallel()

	for _, width := range []int{30, 40, 45, 60, 75, 80, 100} {
		rendered := renderFullSnapshot(width)
		lines := plainLines(t, rendered)
		for _, line := range lines {
			if w := lineWidth(line); w > width {
				t.Fatalf("at width %d: line overflowed with width %d: %q", width, w, line)
			}
		}
	}
}

func TestNarrowPanePreservesPercentageAndCountdown(t *testing.T) {
	t.Parallel()

	for _, width := range []int{40, 45, 60, 80} {
		rendered := renderFullSnapshot(width)
		lines := plainLines(t, rendered)
		joined := strings.Join(lines, "\n")
		if !strings.Contains(joined, "86%") || !strings.Contains(joined, "in 1h30m") {
			t.Fatalf("at width %d: percentage or countdown dropped: %q", width, joined)
		}
		if !strings.Contains(joined, "100%") {
			t.Fatalf("at width %d: antigravity percentage dropped: %q", width, joined)
		}
	}
}

func TestTimestampDroppedWhenWidthConstrained(t *testing.T) {
	t.Parallel()

	narrow := plainLines(t, renderFullSnapshot(45))
	for _, line := range narrow {
		if strings.Contains(line, "5-hour") && strings.Contains(line, "20:30") {
			t.Fatalf("at width 45: timestamp must be dropped: %q", line)
		}
	}

	wide := plainLines(t, renderFullSnapshot(80))
	foundTimestamp := false
	for _, line := range wide {
		if strings.Contains(line, "5-hour") && strings.Contains(line, "20:30") {
			foundTimestamp = true
		}
	}
	if !foundTimestamp {
		t.Fatal("at width 80: timestamp must appear")
	}
}

func TestCompactLabelsUsedUnderPressure(t *testing.T) {
	t.Parallel()

	narrow := plainLines(t, renderFullSnapshot(45))
	joined := strings.Join(narrow, "\n")
	if !strings.Contains(joined, "Claude/GPT") {
		t.Fatalf("at width 45: Claude/GPT compact label must be used: %q", joined)
	}
	if strings.Contains(joined, "Claude & GPT models") {
		t.Fatalf("at width 45: long label must be dropped: %q", joined)
	}
	if !strings.Contains(joined, "Gemini") {
		t.Fatalf("at width 45: Gemini compact label must be used: %q", joined)
	}

	wide := plainLines(t, renderFullSnapshot(80))
	joinedWide := strings.Join(wide, "\n")
	if !strings.Contains(joinedWide, "Claude & GPT models") {
		t.Fatalf("at width 80: full label must appear: %q", joinedWide)
	}
}

func TestBarLengthStaysBetweenLimits(t *testing.T) {
	t.Parallel()

	for _, width := range []int{35, 45, 60, 80, 120} {
		rendered := renderFullSnapshot(width)
		lines := plainLines(t, rendered)
		for _, line := range lines {
			if strings.Contains(line, "━") || (strings.Contains(line, "─") && strings.HasPrefix(line, " ")) {
				trimmed := strings.TrimSpace(line)
				barLen := uniseg.StringWidth(trimmed)
				if barLen < 6 || barLen > 24 {
					t.Fatalf("at width %d: bar length %d outside [6, 24]: %q", width, barLen, line)
				}
			}
		}
	}
}

func TestConsecutiveAccountsHaveNoEmptyLine(t *testing.T) {
	t.Parallel()

	snapshot := Snapshot{
		FetchedAt: fixedTime(),
		Groups: []ProviderQuota{{
			Provider: ProviderCodex,
			Title:    "Codex",
			Accounts: []AccountQuota{
				{Provider: ProviderCodex, Name: "acc1@example.com", Windows: []QuotaWindow{{Label: "5-hour", Remaining: floatPtr(80)}}},
				{Provider: ProviderCodex, Name: "acc2@example.com", Windows: []QuotaWindow{{Label: "5-hour", Remaining: floatPtr(90)}}},
			},
		}},
	}
	rendered := renderSnapshot(snapshot, 80, fixedTime())
	lines := plainLines(t, rendered)
	for i := 0; i < len(lines)-1; i++ {
		if strings.Contains(lines[i], "acc1") {
			for j := i + 1; j < len(lines); j++ {
				if strings.Contains(lines[j], "acc2") {
					between := lines[i+1 : j]
					for _, b := range between {
						if strings.TrimSpace(b) == "" {
							t.Fatalf("empty line found between consecutive accounts: %q", between)
						}
					}
					break
				}
			}
		}
	}
}

func TestProviderGroupsPreserveEmptyLine(t *testing.T) {
	t.Parallel()

	snapshot := Snapshot{
		FetchedAt: fixedTime(),
		Groups: []ProviderQuota{
			{Provider: ProviderCodex, Title: "Codex", Accounts: []AccountQuota{{Provider: ProviderCodex, Name: "a@x.com"}}},
			{Provider: ProviderClaude, Title: "Claude", Accounts: []AccountQuota{{Provider: ProviderClaude, Name: "b@x.com"}}},
		},
	}
	rendered := renderSnapshot(snapshot, 80, fixedTime())
	if !strings.Contains(rendered, "\n\n") {
		t.Fatalf("provider groups must be separated by an empty line: %q", rendered)
	}
}

func TestFooterCompactAtNarrowWidths(t *testing.T) {
	t.Parallel()

	model := newUIModel(configPathForTest(t))
	model.mode = modeQuota
	model.hasConfig = true

	model.width = 45
	narrowView := model.View()
	if !strings.Contains(narrowView, "R refresh · C config · q close") {
		t.Fatalf("narrow view must have compact footer: %q", narrowView)
	}
	if strings.Contains(narrowView, "g/G top/bottom") {
		t.Fatalf("narrow footer must omit full navigation hints: %q", narrowView)
	}

	model.width = 80
	wideView := model.View()
	if !strings.Contains(wideView, "R refresh  C configure  j/k scroll  g/G top/bottom  q/Esc close") {
		t.Fatalf("wide view must have full footer: %q", wideView)
	}
}

func TestHeaderDropsPassiveUpdatedTimestampInNarrowWidth(t *testing.T) {
	t.Parallel()

	model := newUIModel(configPathForTest(t))
	model.mode = modeQuota
	model.hasConfig = true
	model.snapshot = Snapshot{FetchedAt: fixedTime(), CurrentVersion: "7.2.155"}
	model.latestVersion = "v7.2.156"

	model.width = 45
	narrow := model.View()
	if strings.Contains(narrow, "Updated") {
		t.Fatalf("narrow header must drop Updated timestamp: %q", narrow)
	}
	if !strings.Contains(narrow, "7.2.155 → v7.2.156") {
		t.Fatalf("narrow header must preserve update availability: %q", narrow)
	}

	model.width = 80
	wide := model.View()
	if !strings.Contains(wide, "Updated") {
		t.Fatalf("wide header must include Updated timestamp: %q", wide)
	}
}
