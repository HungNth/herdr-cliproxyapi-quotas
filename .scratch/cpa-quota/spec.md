# CPA Quota Version Status and Tab-Local Quota View

Status: ready-for-agent

## Problem Statement

CPA Quota currently opens as a session-modal Herdr popup. While it is visible, the user cannot switch to another Herdr Tab to continue working; closing it also destroys the in-memory Quota Snapshot and requires a new launch and refresh later.

The Quota View also gives no indication whether its CPA Endpoint is running the latest stable CLIProxyAPI release. The current CPA Version is available in the `x-cpa-version` response header from the authentication-files request, while the latest stable release is available from the Management API latest-version endpoint with an optional leading `v`.

## Solution

Replace the session-modal popup with a tab-local Herdr overlay. The overlay remains alive in its originating Herdr Tab while the user visits other Herdr Tabs, preserving its Quota Snapshot, scroll position, configuration state, and Reset Countdown. Each Herdr Tab may own one live Quota View; invoking the Shortcut Binding again focuses the existing view rather than opening a duplicate.

Add an independent, read-only CPA Version check. The Quota View reports the current CPA Version and whether a newer stable release is available without delaying or failing quota retrieval. Version status refreshes when the view opens and when the user presses `R`; it never performs background polling or updates CLIProxyAPI.

## User Stories

1. As a Herdr user, I want to open CPA Quota in my current Herdr Tab, so that the quota information stays associated with the work in that tab.
2. As a Herdr user, I want the Quota View to be non-modal with respect to Herdr Tab navigation, so that I can continue working elsewhere without closing it.
3. As a Herdr user, I want to switch to another Herdr Tab while the Quota View remains alive, so that checking quota does not interrupt my workflow.
4. As a Herdr user, I want to return to the original Herdr Tab and see the same Quota View, so that I do not need to reopen it.
5. As a Herdr user, I want my last Quota Snapshot to remain visible after switching tabs, so that I can compare the same data when I return.
6. As a Herdr user, I want the Quota View scroll position to survive tab switches, so that I return to the account I was inspecting.
7. As a Herdr user, I want Reset Countdowns to continue updating while the Quota View is alive, so that displayed reset timing remains useful when I return.
8. As a Herdr user, I want repeated Shortcut Binding invocation in the same Herdr Tab to focus the existing Quota View, so that duplicate overlays do not accumulate.
9. As a Herdr user, I want another Herdr Tab to be able to own its own Quota View, so that independent work contexts are not forced to share one global view.
10. As a Herdr user, I want a stale Quota View reference to recover automatically, so that an exited or removed pane does not break the Shortcut Binding.
11. As a Herdr user, I want `q` or Escape in quota mode to close the Quota View and restore the underlying pane, so that dismissal remains quick and predictable.
12. As a configured Herdr user, I want Escape in configuration mode to cancel my edits and return to quota, so that an accidental configuration change is not saved.
13. As a first-time Herdr user, I want Escape during initial configuration to close the Quota View, so that I can leave without creating an invalid configuration.
14. As a CPA operator, I want to see the CPA Version reported by the configured CPA Endpoint, so that I know which CLIProxyAPI release is serving quota data.
15. As a CPA operator, I want the Quota View to detect a newer stable CLIProxyAPI release, so that I know when an update is available.
16. As a CPA operator, I want `7.2.154` and `v7.2.154` to compare as the same release, so that the latest-version response prefix does not produce a false update warning.
17. As a CPA operator, I want numeric version components compared by value, so that `7.2.10` is correctly newer than `7.2.9`.
18. As a CPA operator, I want the current CPA Version shown without extra warning when it matches the latest stable release, so that the normal state stays compact.
19. As a CPA operator, I want an update-available warning when the current CPA Version is older, so that the actionable state is visually prominent.
20. As a CPA operator running a development or unreleased build, I want the view to report that the current CPA Version is ahead of the latest stable release, so that it does not incorrectly request a downgrade.
21. As a CPA operator, I want an unrecognized version string displayed without an update verdict, so that useful server metadata is preserved without making an unsafe comparison.
22. As a CPA operator, I want missing current-version metadata reported as unknown, so that older or customized CPA Endpoints continue to show quota normally.
23. As a CPA operator, I want a latest-version request failure reported separately, so that GitHub, proxy, or rate-limit failures are not mistaken for quota failures.
24. As a Herdr user, I want quota content rendered as soon as it is available, so that a slow latest-version request cannot delay the primary feature.
25. As a Herdr user, I want the previous Quota Snapshot to remain visible while refreshing, so that the view does not become blank during network activity.
26. As a Herdr user, I want `R` to refresh quota and version status together, so that one explicit action updates all displayed CPA information.
27. As a Herdr user, I want older asynchronous responses ignored after a newer refresh starts, so that delayed network results cannot replace fresher state.
28. As a CPA administrator, I want every Management API request to use the configured management authentication, so that version checking follows the same access boundary as quota retrieval.
29. As a security-conscious user, I want version checking to remain read-only, so that viewing status cannot change CPA configuration, routing, accounts, quota, or installed software.
30. As a CPA administrator, I want the plugin to report update availability only, so that deployment-specific update mechanisms remain under operator control.
31. As a user on macOS, Linux, or Windows, I want the Quota View and CPA Version status to behave consistently, so that the plugin remains portable across its declared platforms.
32. As an existing CPA Quota user, I want provider grouping, account visibility, quota ordering, progress bars, local reset times, and Manual Reset Credits to remain unchanged, so that this feature does not regress current quota behavior.

## Implementation Decisions

- The canonical user-facing surface is a **Quota View**, not a popup. It uses Herdr's `overlay` placement because an overlay is a normal, tab-local pane, whereas popup placement is session-modal.
- The overlay opens in the Herdr Tab from which the Shortcut Binding was invoked and restores the previously visible pane when the Quota View exits.
- Each Herdr Tab owns at most one live Quota View. A small plugin-state registry associates a Herdr Tab with its current plugin pane identifier.
- Shortcut invocation validates the registered pane before focusing it. If the pane no longer exists, the stale association is removed and a new Quota View is opened.
- Per-tab registry updates are atomic and serialized sufficiently to prevent rapid repeated Shortcut Binding invocations from creating duplicate views.
- Invoking the Shortcut Binding from another Herdr Tab creates or focuses that tab's independent Quota View; it does not redirect to a view owned by another tab.
- The existing Bubble Tea process remains alive while its Herdr Tab is in the background. Snapshot, viewport, input, and countdown state remain in memory without disk persistence.
- In quota mode, `q` and Escape terminate the Quota View. In configuration mode, Escape cancels back to quota when a valid configuration already exists; during first-use configuration it terminates the view.
- The read-only Management API boundary expands to three calls: authentication files, latest stable version, and upstream read-only API forwarding.
- The current CPA Version is read from the case-insensitive `x-cpa-version` response header returned with authentication files.
- The latest stable CPA Version is read from the latest-version JSON field. The server-provided value may have a leading `v`.
- Version comparison accepts exactly numeric `major.minor.patch` values after removing one optional leading `v`. It uses standard-library parsing rather than adding a SemVer dependency.
- Unrecognized version values remain available for display but do not produce up-to-date, update-available, or ahead-of-latest conclusions.
- Quota retrieval and latest-version retrieval are independent asynchronous operations. Quota success is committed and rendered without waiting for latest-version success.
- Each open or manual refresh receives a monotonically increasing refresh generation. Responses from older generations are ignored.
- Starting a refresh preserves the previously rendered Quota Snapshot and version result while showing the appropriate refreshing or checking state.
- Version status appears on a dedicated compact line below the title. Supported states are checking latest, current, update available, ahead of latest, current version unknown, and latest check unavailable.
- Only update availability uses warning emphasis. Unknown or unavailable metadata is informational and never turns the Quota Snapshot into an unavailable state.
- Version checks run when the Quota View opens and whenever `R` is pressed. The minute-based Reset Countdown tick performs no network request.
- The plugin reports status only. It exposes no automatic update, download, installation, or deployment action.
- Existing provider semantics, account classification, quota calculations, ordering, configuration persistence, and Shortcut Binding ownership remain unchanged.
- No placement setting is added. Overlay behavior is the single supported Quota View presentation for this feature.

## Testing Decisions

- Tests assert observable contracts and state transitions rather than source text, private field copies, command construction details, or manifest wording.
- Three seams are required because the feature crosses the CPA Management API, the Bubble Tea state machine, and the Herdr host surface. Combining them into one large harness would be more brittle and would still be unable to replace an actual Herdr runtime check.
- **Management API integration seam:** extend the established in-process HTTP server pattern used by existing client tests. Verify management authentication, request methods, allowed endpoint boundary, current-version header capture, latest-version JSON decoding, optional leading `v`, non-success responses, malformed payloads, missing headers, and quota success when latest-version retrieval fails.
- **UI and refresh-state seam:** drive the Bubble Tea model with completed quota and version messages and assert consumer-visible rendering and transitions. Cover current, update available, ahead of latest, unknown, latest-check unavailable, quota-first rendering, preservation of prior content during refresh, and rejection of stale refresh generations.
- Version comparison tests cover numeric ordering boundaries such as equal versions, patch ordering where lexical comparison would fail, optional leading `v`, current ahead of latest, and malformed values producing no verdict.
- **Actual Herdr smoke seam:** launch the linked plugin in a real Herdr session, open the Quota View in one Herdr Tab, switch to another tab, interact there, return to the original tab, and confirm the same live view remains.
- The Herdr smoke check also invokes the Shortcut Binding repeatedly in one tab to confirm focus without duplication, opens an independent view from another tab, closes with `q` or Escape, confirms restoration of the underlying pane, and checks configuration-mode Escape behavior.
- Existing management-client tests provide prior art for authenticated HTTP fixtures and read-only endpoint assertions. Existing shortcut tests provide prior art for deterministic behavior tests without launching a shell or mutating the user's real configuration.
- No permanent test is added solely to assert that the placement string changed. Tab locality and non-modal navigation are verified through the actual Herdr surface because that is the observable contract.
- Platform-independent Go tests must pass on the development platform. Build verification must cover macOS, Linux, and Windows targets; the tab-navigation behavior must be smoke-tested on an available real Herdr runtime.

## Out of Scope

- Automatically updating, downloading, installing, restarting, or redeploying CLIProxyAPI.
- Providing an update button, update command, release download link, package-manager instruction, or deployment-specific guidance.
- Polling latest-version on a timer or as part of the Reset Countdown tick.
- Persisting CPA Version status or latest-version responses to disk.
- Adding a SemVer dependency or supporting prerelease/build-metadata ordering.
- Treating malformed or missing version values as quota failures.
- Keeping a single global Quota View across multiple Herdr Tabs.
- Allowing multiple Quota Views in the same Herdr Tab.
- Adding user configuration for popup, overlay, split, tab, dimensions, refresh interval, or version-check behavior.
- Plugin-owned restoration or automatic reopening of Quota Views after a Herdr restart beyond Herdr's normal pane/session behavior.
- Changing provider grouping, account eligibility, quota parsing, Manual Reset Credits, account ordering, configuration format, or Shortcut Binding assignment.
- Reading local CLIProxyAPI token files, calling upstream providers directly, consuming reset credits, or mutating CPA Endpoint configuration and routing.

## Further Notes

- Herdr popup placement is a singleton session-modal resource. Overlay placement is a normal Herdr pane and is therefore the smallest host-native mechanism that satisfies tab-local persistence and cross-tab navigation.
- The latest-version Management API endpoint performs its own GitHub Releases lookup and can fail independently due to network, proxy, GitHub, or rate-limit conditions. That failure must remain subordinate to quota retrieval.
- The official latest-version response includes a leading `v`, while the current response header may not; normalization is required before numeric comparison and consistent display.
- No ADR is required. The placement and comparison choices are easy to reverse, follow documented host/API behavior, and do not create architectural lock-in.
