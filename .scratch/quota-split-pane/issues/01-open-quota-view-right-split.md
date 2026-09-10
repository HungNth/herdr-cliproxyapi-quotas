# 01: Open Quota View as Right Split

**What to build:** Pressing `prefix+u` for the first time opens the Quota View as a right-hand Herdr split (50/50) beside the invoking pane, targeted through the Herdr action context. A configured CPA Endpoint keeps focus in the work pane; first-use configuration receives focus. Split-right failure reports the Herdr error and leaves layout unchanged with no fallback, and the registry only records a pane that successfully opened.

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

- [ ] With a fake Herdr executable, a configured state issues open with split placement, right direction, target pane from the action context, and no focus
- [ ] With no persisted configuration, the same open command requests focus for the new Quota View
- [ ] The plugin manifest declares split placement instead of overlay
- [ ] A failed open returns the Herdr error to the action caller and leaves the registry unchanged
- [ ] On a real Herdr session, the Quota View appears as a right sibling pane of the invoking work pane, which stays live
