# 02: Toggle and Dedup Quota View per Herdr Tab

**What to build:** Repeating `prefix+u` toggles the existing per-Herdr-Tab Quota View: invoked from a work pane it focuses the existing view; invoked while the view is focused it closes the pane like a toggle. A pane that no longer exists clears its registry entry and reopens as a fresh split. User-driven resize, swap, and further splits are never touched, and the toggle plus stale recovery work on a real Herdr session.

**Blocked by:** 01: Open Quota View as Right Split.

**Status:** ready-for-agent

- [ ] With the fake Herdr executable, an invocation from a non-quota pane focuses the registered Quota View pane without opening a new one
- [ ] With the fake Herdr executable, an invocation from the registered Quota View pane closes that pane
- [ ] A stale registry entry for a closed pane is removed and a fresh right split is opened on the next invocation
- [ ] Exactly one Quota View exists per Herdr Tab after repeated invocations
- [ ] On a real Herdr session, repeated Shortcut Binding invocations demonstrate focus, toggle-close, and stale recovery, and user-driven pane rearrangements survive untouched
