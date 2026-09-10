# Auto Refresh Interval for the Quota View

Status: ready-for-agent

## Problem Statement

The Quota View shows a Quota Snapshot that is fetched once when the view opens and only again when the user presses the manual refresh key. A user who keeps the Quota View open beside their work pane watches stale numbers: provider Quota Windows drain and reset silently, and the displayed CPA Version status ages, until they remember to press refresh themselves. There is no way to have the view keep itself current.

## Solution

The Quota View automatically re-fetches the Quota Snapshot and CPA Version status on a user-configurable Refresh Interval, defaulting to 60 seconds. The interval is a CPA Endpoint configuration setting, editable in the same configuration form as the base URL and management key, and the countdown restarts whenever the user refreshes manually, so a manual refresh is never immediately followed by an automatic one.

## User Stories

1. As a plugin user, I want the Quota View to re-fetch the Quota Snapshot automatically every Refresh Interval, so that the numbers I monitor stay current without my attention.
2. As a plugin user, I want the automatic refresh to also re-check the CPA Version status, so that update availability never goes stale while I watch.
3. As a plugin user, I want the Refresh Interval to default to 60 seconds, so that I get sensible auto-refresh behavior without configuring anything.
4. As a plugin user, I want to set my own Refresh Interval in the configuration form, so that I can tune how aggressively the view polls my CPA Endpoint.
5. As a plugin user, I want the interval expressed in whole seconds, so that I can choose fast polling for short-lived debugging or slow polling to stay unobtrusive.
6. As a plugin user, I want a blank interval field to mean the 60-second default, so that clearing the field never breaks my configuration.
7. As a plugin user, I want an interval below the minimum rejected with a clear validation error, so that a typo cannot hammer my CPA Endpoint with requests.
8. As a plugin user, I want no upper limit on the interval, so that I may poll as rarely as I like.
9. As a plugin user, I want an invalid interval to surface in the configuration form exactly like other validation errors, so that I learn about mistakes in one consistent place.
10. As a plugin user, I want the new interval to take effect as soon as I save the configuration, so that I do not have to reopen the Quota View to apply it.
11. As a plugin user, I want pressing the manual refresh key to restart the Refresh Interval countdown, so that my manual action is never immediately duplicated by an automatic refresh.
12. As a plugin user, I want every completed automatic refresh to restart the countdown, so that polls stay evenly spaced no matter how long a fetch takes.
13. As a plugin user, I want an automatic refresh to be skipped rather than queued if the previous fetch is still running, so that slow responses never stack up requests against my CPA Endpoint.
14. As a plugin user, I want the countdown restarted in full after a skipped tick, so that polling remains calm under load.
15. As a plugin user, I want automatic refreshes to carry a fresh generation marker, so that late responses from an abandoned fetch can never overwrite newer data.
16. As a plugin user, I want automatic refresh to behave identically to the manual refresh key apart from its trigger, so that I never wonder which path produced the data I see.
17. As a plugin user, I want no visible countdown of the next automatic refresh, so that the view stays compact as it is today.
18. As a plugin user, I want automatic refresh confined to the quota view with a saved configuration, so that the configuration form is never interrupted mid-edit by background fetching.
19. As a plugin user, I want the Refresh Interval honored only while the Quota View process is alive, so that closing the view stops all polling immediately.
20. As a plugin user, I want my existing configuration file to keep working unchanged, so that adding this feature requires no action from me.
21. As a maintainer, I want the interval owned by the configuration package alongside the other CPA Endpoint settings, so that validation and defaults live in one place.
22. As a maintainer, I want the scheduling logic owned by the terminal UI model, so that refresh behavior stays next to the fetch commands and generation counter it coordinates with.
23. As a maintainer, I want the automatic refresh to reuse the same fetch commands as the manual refresh key, so that there is exactly one fetch path to maintain.
24. As a maintainer, I want the timer behavior testable without real sleeping, so that the test suite stays fast and deterministic.
25. As a maintainer, I want the feature verified against a real Herdr session, so that scheduling survives contact with the actual host integration.
26. As a maintainer, I want the README to document the new setting and its default, so that users can discover it without reading source.

## Implementation Decisions

- The Refresh Interval is a new configuration field owned by the configuration package: an integer number of seconds stored in the existing configuration document under the key `refresh_interval`.
- Default is 60 seconds. The default applies when the field is absent, zero, or left blank in the form; zero in the document is treated as unset, not as disabled.
- Validation runs with the existing configuration validation: the value must be a whole number of seconds of at least 5; there is no upper bound. Non-numeric or below-minimum values fail with a clear error string, surfaced by the form like other validation errors.
- The terminal UI form gains a third input for the interval in seconds, with the default as its placeholder, participating in the existing focus cycling and submit flow. Saving applies the new interval immediately and restarts the countdown.
- The UI model owns scheduling: it records when the last fetch batch started and, on each wake-up, starts a new fetch batch only when the configured interval has fully elapsed.
- An automatic refresh issues exactly what the manual refresh key issues: the snapshot and version fetch commands batched together under a fresh generation counter value. There is one fetch path; the trigger differs, the commands do not.
- The countdown restarts at the start of every fetch batch, whether triggered automatically, by the manual key, or by saving a new configuration.
- A wake-up that arrives while a fetch is in flight is dropped, not queued; the schedule simply re-arms. The in-flight batch's own start already restarted the countdown, so no request stacking can occur.
- Scheduling stays armed only in the quota view with a saved configuration. The configuration form never auto-fetches. Closing the Quota View ends the process and with it the timer.
- Wake-up cadence: the view keeps its existing one-minute re-render cadence for Reset Countdown text, and the auto-refresh check runs on wake-ups, so an interval elapsing triggers within one wake-up. Sub-minute intervals remain honored because the timer is scheduled from the remaining time, not from a fixed cadence alone.
- No visible next-refresh countdown is added. No pausing, no jitter, no backoff, no per-provider intervals, and no new dependencies.
- The README's Quota View behavior section documents the setting, its default, the minimum, and the manual-refresh restart behavior.

## Testing Decisions

- The maintainer confirmed three seams: configuration round-trip tests, UI model update tests with injected time, and a real Herdr smoke test. No new framework, no permanent CLI integration suite, no tests over `tea.Tick` itself.
- A good test observes external behavior: the configuration produced by save/load/validate, and the commands and state a message elicits from the model. Tests never assert tick internals, timer goroutines, or private field names.
- Configuration package: extend the existing round-trip test pattern to cover the new field — default on absence and zero, acceptance of legal values, rejection below the minimum, and rejection of non-numeric input.
- UI model: reuse the established pattern of driving the model with messages and inspecting the returned commands and state (prior art: `TestRefreshKeyUsesOneGeneration`, which feeds a key press and unpacks the batched snapshot and version messages). New tests feed wake-up messages carrying explicit times against a model with a known interval and last-fetch marker, asserting: fetch issued with a fresh generation when elapsed, no fetch before elapsed, fetch dropped while in flight, countdown restart on manual refresh, and no fetch in configuration mode or without configuration. Time arrives inside the wake-up message, so every branch is deterministic.
- Real Herdr smoke: build and link, open the Quota View with a short interval in an isolated plugin config, observe the view re-fetching without a key press (pane output showing an updated fetch), then press the manual refresh key and confirm the next automatic poll comes a full interval later. Report the exact scenario and platform; clean up only artifacts created for the check.
- Run affected package tests during work and the full suite at the end; format changed Go code and cross-compile Windows, macOS, and Linux as usual.

## Out of Scope

- Any visible countdown, progress indicator, or status line for the next automatic refresh.
- Pausing, resuming, or muting auto-refresh while the view stays open.
- Jitter, exponential backoff, retry scheduling, or failure-specific intervals.
- Per-provider or per-Quota-Window refresh rates.
- Background refresh after the Quota View is closed, or pushing updates into other Herdr Tabs.
- Migration or rewriting of existing configuration files; absent keys simply take defaults.
- Changes to the fetch commands, generation counter semantics, provider parsing, rendering, or keybindings beyond what scheduling requires.
- New dependencies or frameworks for scheduling or testing.

## Further Notes

- All design decisions were settled in a grilling round where the maintainer accepted every recommendation: scope (snapshot plus version), bounds (minimum 5 seconds, no maximum), the third form field, drop-while-in-flight, and no countdown display.
- The glossary gained the term Refresh Interval as part of this design; existing terms (Quota Snapshot, CPA Version, CPA Endpoint, Quota View, Reset Countdown) keep their meanings.
- Before this feature the view had no automatic re-fetch at all: the periodic wake-up only re-rendered Reset Countdown text. Auto-refresh is additive, not a change to an existing poll.
- The plugin runs one process per Quota View pane, so "timer stops when the view closes" is inherent to the host model, not something this feature builds.
