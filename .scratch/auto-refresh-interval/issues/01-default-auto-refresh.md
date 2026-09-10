# 01: Auto-refresh on the default 60-second interval

**What to build:** An open Quota View with a saved configuration re-fetches the Quota Snapshot and CPA Version status automatically every 60 seconds, using the same fetch batch and fresh generation marker as the manual refresh key. The countdown restarts at every fetch start (automatic, manual `R`, or configuration save), and a wake-up that lands while a fetch is in flight is dropped rather than queued. Scheduling is armed only in the quota view with a saved configuration; the configuration form never auto-fetches; closing the view stops polling. The README documents the behavior in one line.

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

- [ ] Tick after the interval elapses issues a batched snapshot + version fetch with a fresh generation and restarts the countdown
- [ ] Tick before the interval elapses changes nothing about fetch state
- [ ] Tick while a fetch is in flight is dropped (no stacked request, no state change)
- [ ] Manual `R` restarts the countdown
- [ ] No auto-fetch in configuration mode or without a saved configuration
- [ ] Existing UI tests still pass; new behavior tests use explicit times via the model update seam
- [ ] Real Herdr smoke: view re-fetches without a keypress (observable pane output change)
- [ ] Full suite, gofmt, and cross-compile (windows/linux/darwin) pass
