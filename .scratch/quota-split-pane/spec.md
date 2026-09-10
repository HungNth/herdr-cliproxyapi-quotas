# Split-Pane Quota View

Status: ready-for-agent

## Problem Statement

CPA Quota currently opens as an overlay. Although the overlay is tab-local, it covers the entire Herdr Tab and hides every underlying pane. A user cannot see a newly created right or down split until closing the Quota View, so the current surface conflicts with the intended workflow of working in one pane while monitoring quota in another.

The desired workflow is a normal Herdr layout: an existing work pane remains live on the left, a Quota View appears beside it on the right, and the user can resize, swap, create further splits, or continue working without the quota display obscuring the tab.

## Solution

Replace overlay placement with a normal right-hand Herdr split pane. The Shortcut Binding targets the invoking pane, splitting it to the right at Herdr's native 50/50 ratio. The Quota View is a normal pane in the tab layout and does not cover sibling panes.

A configured CPA Endpoint opens the Quota View without taking focus away from the work pane. If CPA Quota requires first-use configuration, the new Quota View receives focus so the user can enter the CPA Endpoint and management key. Each Herdr Tab still owns at most one Quota View. Repeating the Shortcut Binding focuses that view from another pane, and closes it when it is already focused.

## User Stories

1. As a Herdr user, I want the Shortcut Binding to open CPA Quota beside my current pane, so that I can work and monitor quota simultaneously.
2. As a Herdr user, I want my original work pane to remain live when opening the Quota View, so that no shell, editor, or agent process is replaced.
3. As a Herdr user, I want the Quota View placed to the right of the pane where I invoked the Shortcut Binding, so that its location is predictable.
4. As a Herdr user, I want the Quota View to be a normal pane rather than an overlay, so that it does not hide the rest of the Herdr Tab.
5. As a Herdr user, I want to create a right or down split after opening CPA Quota and still see the Quota View, so that layout changes remain visible.
6. As a Herdr user, I want the initial native Herdr split to be 50/50, so that the plugin does not impose a hidden layout ratio.
7. As a Herdr user, I want to resize the Quota View pane using normal Herdr controls, so that I can choose the workspace balance myself.
8. As a Herdr user, I want to swap the Quota View with another pane using normal Herdr controls, so that I can rearrange my tab.
9. As a Herdr user, I want to create further splits around the Quota View, so that a quota monitor does not constrain future layout changes.
10. As a configured user, I want focus to remain in my work pane when I open CPA Quota, so that monitoring quota does not interrupt typing or agent work.
11. As a first-time user, I want the Quota View focused after it opens, so that I can immediately configure the CPA Endpoint and management key.
12. As a user who has already opened the Quota View in a Herdr Tab, I want another Shortcut Binding invocation from a work pane to focus the existing view, so that duplicate quota panes do not accumulate.
13. As a user focused in the Quota View, I want the Shortcut Binding to close it, so that the same key acts as a predictable toggle.
14. As a user whose Quota View was closed with `q`, Escape, or a Herdr pane command, I want the next Shortcut Binding invocation to create a fresh view, so that stale registry state does not block access.
15. As a user with several panes in one Herdr Tab, I want only one Quota View there, so that the monitor remains a singular status surface for that tab.
16. As a user with multiple Herdr Tabs, I want each tab to be able to have its own Quota View, so that separate work contexts can monitor quota independently.
17. As a user invoking the Shortcut Binding from a pane other than the current Quota View, I want the existing Quota View focused rather than duplicated, so that its live Quota Snapshot and Reset Countdown remain intact.
18. As a user invoking the Shortcut Binding from the focused Quota View, I want the pane to close and its sibling work pane to reclaim the layout, so that I can dismiss monitoring without another key.
19. As a user, I want `q` and Escape in quota mode to continue closing the Quota View, so that existing dismissal behavior remains available.
20. As a configured user, I want the compact responsive quota layout to remain usable at the native 50/50 split width, so that moving to a split pane does not regress Reset Countdown visibility.
21. As a user, I want CPA Version and Update Availability to continue appearing in the Quota View, so that changing placement does not remove version monitoring.
22. As a user, I want quota refresh and Reset Countdown behavior to remain unchanged after the placement change, so that only the Herdr surface changes.
23. As a user whose active pane cannot be split right because of Herdr layout constraints, I want an explicit error and unchanged layout, so that the plugin does not unexpectedly open below or as an overlay.
24. As a user, I do not want the plugin to silently fall back to a down split, overlay, or tab when a right split fails, so that placement remains predictable.
25. As a user, I want manually resizing or swapping the Quota View within its Herdr Tab to be respected, so that the plugin does not undo my layout decisions.
26. As a user moving the Quota View to a different Herdr Tab manually, I want the plugin not to auto-move or close panes to reconcile state, so that it does not override deliberate topology changes.
27. As a user on macOS, Linux, or Windows, I want split-pane opening and toggle behavior to work consistently, so that the plugin stays portable.

## Implementation Decisions

- The canonical Quota View surface changes from a tab-covering overlay to a normal Herdr split pane.
- The plugin entrypoint declares split placement. The launcher also explicitly requests split placement to make the runtime action unambiguous.
- A new Quota View is opened to the right of the invoking pane. The launcher receives that pane identity through the Herdr action context and passes it as the split target.
- Split direction is always right. There is no automatic down-split, overlay, or tab fallback.
- The plugin accepts Herdr's native 50/50 split ratio. It does not issue a follow-up resize command, infer geometry, or maintain a quota-pane width setting.
- A configured Quota View opens without focus. A first-use configuration view opens with focus.
- Existing per-Herdr-Tab registry semantics remain: one registry entry maps a Herdr Tab to its live Quota View pane.
- A Shortcut Binding invocation validates the registry pane. A missing pane removes the stale entry before opening a replacement split.
- If the registered Quota View exists and is not the invoking pane, the plugin focuses it. If it is the invoking pane, the plugin closes it as a toggle.
- Closing through the toggle, `q`, Escape, or Herdr removes the pane; a later Shortcut Binding self-heals the stale registry entry.
- User-driven resize, swap, and further split operations within the same Herdr Tab are not intercepted or restored by the plugin.
- User-driven moves of the Quota View across Herdr Tabs receive no special reconciliation behavior. The plugin must not close, move, or duplicate panes automatically to resolve manually changed topology.
- A failed right split returns the Herdr error through the action command. The registry must not retain a pane entry for a failed launch.
- The CPA Endpoint, Management API boundary, Quota Snapshot, Reset Countdown, CPA Version, Update Availability, compact quota renderer, and existing key behavior remain unchanged.
- No new configuration setting, layout abstraction, sizing policy, or dependency is introduced.

## Testing Decisions

- Tests assert externally observable launch decisions and Herdr topology, not implementation-specific argument arrays or private registry structure.
- Three seams are required because deterministic launcher behavior, first-use configuration state, and native terminal topology must be proven separately.
- **Launcher and registry seam:** use a fake Herdr executable to observe the open, focus, and close commands through the same command boundary used in production. Verify split placement, right direction, invoking-pane targeting, configured `--no-focus`, first-use `--focus`, per-Herdr-Tab deduplication, focus behavior, toggle-close behavior, stale pane recovery, and no registry entry after failed open.
- **Configuration-state seam:** use the existing configuration loading behavior to distinguish configured and first-use states. Verify that the launcher only changes focus policy; it must not change persisted configuration validation or credentials.
- **Actual Herdr smoke seam:** open the linked plugin from a real work pane and confirm the Quota View is a right sibling pane. Create an additional right and down split, resize and swap panes, confirm the view remains visible, verify Shortcut Binding focus/toggle behavior, and confirm a split-right failure leaves the layout unchanged with an actionable error.
- Existing registry persistence tests, shortcut behavior tests, and real Herdr smoke checks provide prior art. Extend them rather than adding a second command-launch mechanism.
- Platform-independent Go tests must pass on the development platform. Build verification must cover macOS, Linux, and Windows targets. The split topology must be smoke-tested in a real available Herdr session.

## Out of Scope

- Configurable split direction, default pane placement, split ratio, automatic resize, or a remembered pane width.
- Automatic fallback to down split, overlay, tab, zoomed pane, popup, or any alternate surface.
- Automatic reconciliation when a user moves the Quota View to another Herdr Tab.
- Multiple Quota Views per Herdr Tab or a view per source pane.
- Changing the existing CPA Endpoint, Management API calls, provider requests, quota parsing, quota calculations, provider grouping, Account ordering, Reset Countdown, CPA Version, or Update Availability.
- Changing compact responsive quota-row rendering, footer behavior, or version presentation.
- New user configuration, new dependencies, or plugin-owned layout restoration after Herdr restart.
- Replacing existing `q` or Escape close behavior.

## Further Notes

- Herdr's plugin-pane open interface supports split placement, right/down direction, target pane identity, and focus policy, but it does not support a split ratio. Native 50/50 is therefore both the smallest and the most predictable option.
- The compact responsive Quota View already renders its protected percentage and Reset Countdown at narrow widths, making additional automatic resizing unnecessary.
- There is no ADR for this change. The placement is easy to reverse, follows the documented plugin host surface, and creates no architectural lock-in.
