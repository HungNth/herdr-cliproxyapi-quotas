package ui

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rivo/uniseg"

	"cpa-quota/internal/cpa"
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
	model.snapshot = cpa.Snapshot{Groups: []cpa.ProviderQuota{{Provider: cpa.ProviderCodex, Title: "Codex", Accounts: []cpa.AccountQuota{{Provider: cpa.ProviderCodex, Name: "a@b.c"}}}}, FetchedAt: fixedTime()}
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

	current := cpa.Snapshot{FetchedAt: fixedTime()}
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
	snapshot := cpa.Snapshot{CurrentVersion: current}
	return uiModel{snapshot: snapshot, latestVersion: latest, latestErr: latestErr}
}

func staleSnapshot() cpa.Snapshot {
	stale := cpa.Snapshot{FetchedAt: fixedTime()}
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
	snapshot := cpa.Snapshot{
		FetchedAt: fixedTime(),
		Groups: []cpa.ProviderQuota{{
			Provider: cpa.ProviderCodex,
			Title:    "Codex",
			Accounts: []cpa.AccountQuota{{
				Provider: cpa.ProviderCodex,
				Name:     "a@b.c",
				Windows: []cpa.QuotaWindow{
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
	snapshot := cpa.Snapshot{
		FetchedAt: fixedTime(),
		Groups: []cpa.ProviderQuota{{
			Provider: cpa.ProviderClaude,
			Title:    "Claude",
			Accounts: []cpa.AccountQuota{{
				Provider: cpa.ProviderClaude,
				Name:     "x@y.z",
				Windows: []cpa.QuotaWindow{
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
	snapshot := cpa.Snapshot{
		FetchedAt: fixedTime(),
		Groups: []cpa.ProviderQuota{
			{
				Provider: cpa.ProviderCodex,
				Title:    "Codex",
				Accounts: []cpa.AccountQuota{{
					Provider: cpa.ProviderCodex,
					Name:     "user@example.com",
					Windows: []cpa.QuotaWindow{
						{Label: "5-hour", Remaining: floatPtr(86), ResetAt: &reset},
						{Label: "Weekly", Remaining: floatPtr(70), ResetAt: &reset},
					},
					ManualResets: intPtr(3),
				}},
			},
			{
				Provider: cpa.ProviderAntigravity,
				Title:    "Antigravity",
				Accounts: []cpa.AccountQuota{{
					Provider: cpa.ProviderAntigravity,
					Name:     "user@example.com",
					Windows: []cpa.QuotaWindow{
						{Label: "Claude 5-hour", Remaining: floatPtr(100), ResetAt: &reset},
						{Label: "Claude Weekly", Remaining: floatPtr(70), ResetAt: &reset},
						{Label: "Gemini 5-hour", Remaining: floatPtr(95), ResetAt: &reset},
						{Label: "Gemini Weekly", Remaining: floatPtr(85), ResetAt: &reset},
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
	if !strings.Contains(joined, "Claude 5h") {
		t.Fatalf("at width 45: Claude 5h compact label must be used: %q", joined)
	}
	if strings.Contains(joined, "Claude 5-hour") {
		t.Fatalf("at width 45: long label must be dropped: %q", joined)
	}
	if !strings.Contains(joined, "Gemini 5h") {
		t.Fatalf("at width 45: Gemini 5h compact label must be used: %q", joined)
	}

	wide := plainLines(t, renderFullSnapshot(80))
	joinedWide := strings.Join(wide, "\n")
	if !strings.Contains(joinedWide, "Claude 5-hour") {
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

	snapshot := cpa.Snapshot{
		FetchedAt: fixedTime(),
		Groups: []cpa.ProviderQuota{{
			Provider: cpa.ProviderCodex,
			Title:    "Codex",
			Accounts: []cpa.AccountQuota{
				{Provider: cpa.ProviderCodex, Name: "acc1@example.com", Windows: []cpa.QuotaWindow{{Label: "5-hour", Remaining: floatPtr(80)}}},
				{Provider: cpa.ProviderCodex, Name: "acc2@example.com", Windows: []cpa.QuotaWindow{{Label: "5-hour", Remaining: floatPtr(90)}}},
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

	snapshot := cpa.Snapshot{
		FetchedAt: fixedTime(),
		Groups: []cpa.ProviderQuota{
			{Provider: cpa.ProviderCodex, Title: "Codex", Accounts: []cpa.AccountQuota{{Provider: cpa.ProviderCodex, Name: "a@x.com"}}},
			{Provider: cpa.ProviderClaude, Title: "Claude", Accounts: []cpa.AccountQuota{{Provider: cpa.ProviderClaude, Name: "b@x.com"}}},
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
	model.snapshot = cpa.Snapshot{FetchedAt: fixedTime(), CurrentVersion: "7.2.155"}
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

func TestResetCountdown(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		target time.Time
		want   string
	}{
		{now.Add(-time.Minute), "ready"},
		{now.Add(30 * time.Second), "in <1m"},
		{now.Add(45 * time.Minute), "in 45m"},
		{now.Add(90 * time.Minute), "in 1h30m"},
		{now.Add(51 * time.Hour), "in 2d3h"},
	}
	for _, testCase := range cases {
		if got := resetCountdown(testCase.target, now); got != testCase.want {
			t.Fatalf("resetCountdown(%s) = %q, want %q", testCase.target, got, testCase.want)
		}
	}
}

func TestSubmitConfigSetsErrorOnInvalidInput(t *testing.T) {
	t.Parallel()

	model := newUIModel(configPathForTest(t))
	model.mode = modeConfig
	model.baseInput.SetValue("ftp://invalid")
	model.keyInput.SetValue("secret")

	nextModel, _ := model.submitConfig()
	uiNext, ok := nextModel.(uiModel)
	if !ok {
		t.Fatalf("expected uiModel, got %T", nextModel)
	}
	if uiNext.errText == "" {
		t.Fatal("expected validation error text on invalid URL")
	}
}

func autoRefreshModel(t *testing.T) uiModel {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"latest-version":"v7.2.155"}`))
	}))
	t.Cleanup(server.Close)

	model := newUIModel(configPathForTest(t))
	model.config.BaseURL = server.URL
	model.mode = modeQuota
	model.hasConfig = true
	model.refreshSeq = 5
	model.fetching = false
	model.refreshInterval = 60 * time.Second
	return model
}

// runBatch executes a batch's sub-commands and returns the messages that arrive
// promptly. Long-lived timer commands are skipped by the timeout.
func runBatch(t *testing.T, cmd tea.Cmd) []tea.Msg {
	t.Helper()
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected batched commands, got %T", msg)
	}
	var msgs []tea.Msg
	for _, sub := range batch {
		done := make(chan tea.Msg, 1)
		go func(c tea.Cmd) { done <- c() }(sub)
		select {
		case m := <-done:
			msgs = append(msgs, m)
		case <-time.After(100 * time.Millisecond):
		}
	}
	return msgs
}

func assertFetchBatch(t *testing.T, msgs []tea.Msg, wantSeq int) {
	t.Helper()
	var snapshots, versions int
	for _, msg := range msgs {
		switch m := msg.(type) {
		case snapshotMsg:
			snapshots++
			if m.seq != wantSeq {
				t.Fatalf("snapshot seq = %d, want %d", m.seq, wantSeq)
			}
		case versionMsg:
			versions++
			if m.seq != wantSeq {
				t.Fatalf("version seq = %d, want %d", m.seq, wantSeq)
			}
		}
	}
	if snapshots != 1 || versions != 1 {
		t.Fatalf("batch messages = %#v, want one snapshot and one version", msgs)
	}
}

func TestTickAfterIntervalStartsAutoRefresh(t *testing.T) {
	t.Parallel()

	model := autoRefreshModel(t)
	start := fixedTime()
	model.lastFetchAt = start
	fire := start.Add(61 * time.Second)

	updated, cmd := model.Update(tickMsg(fire))
	after := updated.(uiModel)
	if !after.fetching || !after.versionChecking {
		t.Fatal("auto refresh must set fetching and versionChecking")
	}
	if !after.lastFetchAt.Equal(fire) {
		t.Fatalf("countdown must restart at tick time: %v", after.lastFetchAt)
	}
	if after.refreshSeq != 6 {
		t.Fatalf("seq = %d, want 6", after.refreshSeq)
	}
	assertFetchBatch(t, runBatch(t, cmd), 6)
}

func TestTickBeforeIntervalDoesNotFetch(t *testing.T) {
	t.Parallel()

	model := autoRefreshModel(t)
	start := fixedTime()
	model.lastFetchAt = start

	updated, _ := model.Update(tickMsg(start.Add(30 * time.Second)))
	after := updated.(uiModel)
	if after.fetching || after.versionChecking {
		t.Fatal("must not fetch before the interval elapses")
	}
	if !after.lastFetchAt.Equal(start) || after.refreshSeq != 5 {
		t.Fatalf("state must be untouched: lastFetchAt=%v seq=%d", after.lastFetchAt, after.refreshSeq)
	}
}

func TestTickWhileFetchingIsDropped(t *testing.T) {
	t.Parallel()

	model := autoRefreshModel(t)
	overdue := fixedTime().Add(-2 * time.Minute)
	model.lastFetchAt = overdue
	model.fetching = true
	model.refreshSeq = 9

	updated, _ := model.Update(tickMsg(fixedTime()))
	after := updated.(uiModel)
	if !after.fetching || after.refreshSeq != 9 {
		t.Fatal("in-flight tick must not issue a new fetch or change fetching state")
	}
	if !after.lastFetchAt.Equal(fixedTime()) {
		t.Fatalf("dropped in-flight tick must restart countdown from tick time: got %v, want %v", after.lastFetchAt, fixedTime())
	}
}

func TestManualRefreshRestartsCountdown(t *testing.T) {
	t.Parallel()

	model := autoRefreshModel(t)
	start := time.Now().Add(-time.Minute)
	model.lastFetchAt = start

	updated, cmd := model.updateQuota(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}})
	after := updated.(uiModel)
	if !after.lastFetchAt.After(start) {
		t.Fatal("manual refresh must restart the countdown")
	}
	assertFetchBatch(t, runBatch(t, cmd), 6)
}

func TestTickNeverAutoFetchesInConfigModeOrWithoutConfig(t *testing.T) {
	t.Parallel()

	configMode := autoRefreshModel(t)
	configMode.mode = modeConfig
	configMode.lastFetchAt = fixedTime().Add(-time.Hour)
	updated, _ := configMode.Update(tickMsg(fixedTime()))
	if updated.(uiModel).fetching {
		t.Fatal("configuration mode must not auto-fetch")
	}

	noConfig := autoRefreshModel(t)
	noConfig.hasConfig = false
	noConfig.lastFetchAt = time.Time{}
	updated2, _ := noConfig.Update(tickMsg(fixedTime()))
	if updated2.(uiModel).fetching {
		t.Fatal("missing configuration must not auto-fetch")
	}
}

func TestSubmitConfigWithCustomInterval(t *testing.T) {
	t.Parallel()

	model := newUIModel(configPathForTest(t))
	model.mode = modeConfig
	model.baseInput.SetValue("http://127.0.0.1:8317")
	model.keyInput.SetValue("secret")
	model.intervalInput.SetValue("10")

	nextModel, cmd := model.submitConfig()
	uiNext, ok := nextModel.(uiModel)
	if !ok {
		t.Fatalf("expected uiModel, got %T", nextModel)
	}
	if uiNext.errText != "" {
		t.Fatalf("unexpected validation error: %s", uiNext.errText)
	}
	if cmd == nil {
		t.Fatal("expected configureCmd")
	}
}

func TestSubmitConfigWithBlankIntervalDefaults(t *testing.T) {
	t.Parallel()

	model := newUIModel(configPathForTest(t))
	model.mode = modeConfig
	model.baseInput.SetValue("http://127.0.0.1:8317")
	model.keyInput.SetValue("secret")
	model.intervalInput.SetValue("   ")

	nextModel, cmd := model.submitConfig()
	uiNext := nextModel.(uiModel)
	if uiNext.errText != "" {
		t.Fatalf("unexpected validation error on blank interval: %s", uiNext.errText)
	}
	if cmd == nil {
		t.Fatal("expected configureCmd")
	}
}

func TestSubmitConfigValidationErrorsOnInterval(t *testing.T) {
	t.Parallel()

	t.Run("non-numeric input", func(t *testing.T) {
		model := newUIModel(configPathForTest(t))
		model.mode = modeConfig
		model.baseInput.SetValue("http://127.0.0.1:8317")
		model.keyInput.SetValue("secret")
		model.intervalInput.SetValue("fast")

		nextModel, _ := model.submitConfig()
		uiNext := nextModel.(uiModel)
		if !strings.Contains(uiNext.errText, "whole number of seconds") {
			t.Fatalf("errText = %q, want whole number error", uiNext.errText)
		}
	})

	t.Run("below minimum", func(t *testing.T) {
		model := newUIModel(configPathForTest(t))
		model.mode = modeConfig
		model.baseInput.SetValue("http://127.0.0.1:8317")
		model.keyInput.SetValue("secret")
		model.intervalInput.SetValue("4")

		nextModel, _ := model.submitConfig()
		uiNext := nextModel.(uiModel)
		if !strings.Contains(uiNext.errText, "at least 5") {
			t.Fatalf("errText = %q, want at least 5 error", uiNext.errText)
		}
	})
}

func TestConfigFieldThreeWayFocusCycling(t *testing.T) {
	t.Parallel()

	model := newUIModel(configPathForTest(t))
	model.mode = modeConfig
	model.focusConfigField(0)

	// Tab forward: 0 -> 1 -> 2 -> 0
	m1, _ := model.updateConfig(tea.KeyMsg{Type: tea.KeyTab})
	if m1.(uiModel).focused != 1 {
		t.Fatalf("focused = %d, want 1", m1.(uiModel).focused)
	}
	m2, _ := m1.(uiModel).updateConfig(tea.KeyMsg{Type: tea.KeyTab})
	if m2.(uiModel).focused != 2 {
		t.Fatalf("focused = %d, want 2", m2.(uiModel).focused)
	}
	m3, _ := m2.(uiModel).updateConfig(tea.KeyMsg{Type: tea.KeyTab})
	if m3.(uiModel).focused != 0 {
		t.Fatalf("focused = %d, want 0", m3.(uiModel).focused)
	}

	// Shift+Tab backward: 0 -> 2 -> 1 -> 0
	b1, _ := model.updateConfig(tea.KeyMsg{Type: tea.KeyShiftTab})
	if b1.(uiModel).focused != 2 {
		t.Fatalf("backward focused = %d, want 2", b1.(uiModel).focused)
	}
	b2, _ := b1.(uiModel).updateConfig(tea.KeyMsg{Type: tea.KeyShiftTab})
	if b2.(uiModel).focused != 1 {
		t.Fatalf("backward focused = %d, want 1", b2.(uiModel).focused)
	}
	b3, _ := b2.(uiModel).updateConfig(tea.KeyMsg{Type: tea.KeyShiftTab})
	if b3.(uiModel).focused != 0 {
		t.Fatalf("backward focused = %d, want 0", b3.(uiModel).focused)
	}
}

func TestAutoRefreshHonorsCustomInterval(t *testing.T) {
	t.Parallel()

	model := autoRefreshModel(t)
	model.refreshInterval = 10 * time.Second
	start := fixedTime()
	model.lastFetchAt = start

	// 9s is before 10s interval: no fetch
	updated, _ := model.Update(tickMsg(start.Add(9 * time.Second)))
	if updated.(uiModel).fetching {
		t.Fatal("must not fetch at 9s for 10s interval")
	}

	// 11s is after 10s interval: auto-fetch triggered
	updated2, cmd := model.Update(tickMsg(start.Add(11 * time.Second)))
	after := updated2.(uiModel)
	if !after.fetching {
		t.Fatal("must fetch at 11s for 10s interval")
	}
	assertFetchBatch(t, runBatch(t, cmd), 6)
}

func TestHealthStyleThresholds(t *testing.T) {
	t.Parallel()

	cases := []struct {
		remaining float64
		want      lipgloss.Style
		name      string
	}{
		{remaining: 0.0, want: errorStyle, name: "0% is critical red"},
		{remaining: 15.0, want: errorStyle, name: "15% is critical red"},
		{remaining: 30.0, want: errorStyle, name: "30% boundary is critical red"},
		{remaining: 30.1, want: warningStyle, name: "30.1% is warning yellow"},
		{remaining: 50.0, want: warningStyle, name: "50% is warning yellow"},
		{remaining: 69.9, want: warningStyle, name: "69.9% is warning yellow"},
		{remaining: 70.0, want: goodStyle, name: "70% boundary is healthy green"},
		{remaining: 85.0, want: goodStyle, name: "85% is healthy green"},
		{remaining: 100.0, want: goodStyle, name: "100% is healthy green"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := healthStyle(tc.remaining)
			if got.GetForeground() != tc.want.GetForeground() {
				t.Fatalf("healthStyle(%v) foreground = %v, want %v", tc.remaining, got.GetForeground(), tc.want.GetForeground())
			}
		})
	}
}
