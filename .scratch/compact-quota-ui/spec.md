# Compact Responsive Quota Window Rows

Status: ready-for-agent

## Problem Statement

When CPA Quota is displayed in a narrow Herdr pane, each Quota Window is rendered as one long fixed-width line. The line requires approximately 94 terminal columns in normal mode and 86 columns even after the progress bar is shortened. Bubbles' viewport does not wrap or horizontally scroll these lines, so Herdr clips the right side.

The Reset Countdown is currently the final field and therefore the first important value lost. The user can still see the progress bar and verbose text such as `remaining`, but cannot see values such as `in 1h30m`, which are more useful for deciding when quota will become available again.

The full-cell progress bar is also visually heavy. The user wants a compact hierarchy where percentage and reset timing are read first and a thin progress indicator supports them rather than dominating the row.

## Solution

Render every Quota Window using a consistent two-row compact layout. The first row presents the Window Label, remaining percentage, Reset Countdown, and absolute reset time in priority order. The second row presents a visually thin, single-terminal-row progress bar aligned below the percentage.

Remove the redundant words `remaining` and `resets` from normal rows. Fit content against the actual available terminal width instead of fixed wide/medium/narrow breakpoints. When space is constrained, omit the absolute reset time first, then shorten or truncate the Window Label, and finally shrink the progress bar from 24 to a minimum of 6 characters. Remaining percentage and Reset Countdown must remain visible.

Use `━` for the filled segment and `─` for the empty segment. This remains one terminal row—the minimum possible terminal layout height—but appears substantially thinner than the existing full-height block characters.

## User Stories

1. As a Herdr user, I want the Reset Countdown visible in a narrow pane, so that I can decide when quota will become available again.
2. As a Herdr user, I want remaining percentage visible at every supported pane width, so that I can assess current capacity immediately.
3. As a Herdr user, I want Reset Countdown prioritized over the absolute reset timestamp, so that the most actionable timing information survives constrained layouts.
4. As a Herdr user, I want normal quota rows to omit the word `remaining`, so that fixed prose does not consume columns needed by live values.
5. As a Herdr user, I want normal quota rows to omit the word `resets`, so that reset timing fits in narrow panes.
6. As a Herdr user, I want the percentage displayed before reset timing, so that the first values describe current capacity and future availability in that order.
7. As a Herdr user, I want the full local reset timestamp visible when the pane is wide enough, so that I can correlate the Reset Countdown with an exact time.
8. As a Herdr user, I want the timestamp omitted cleanly rather than partially clipped, so that no incomplete date or time is displayed.
9. As a Herdr user, I want every Quota Window to use the same two-row structure, so that resizing does not move information between unrelated layouts.
10. As a Herdr user, I want the progress bar below the values it represents, so that the values remain the primary information.
11. As a Herdr user, I want the progress bar to look thinner than a full-cell block bar, so that it does not visually overpower the Reset Countdown.
12. As a terminal user, I want the progress bar to use one terminal row, so that it consumes the minimum layout height possible in a text terminal.
13. As a Herdr user, I want the progress bar to grow in a wide pane and shrink in a narrow pane, so that it remains useful without causing clipping.
14. As a Herdr user, I want progress bars aligned within each Provider Group, so that related Quota Windows are easy to compare.
15. As a Codex or Claude user, I want short labels such as `5-hour` and `Weekly` preserved, so that familiar Quota Window names do not change unnecessarily.
16. As an Antigravity user, I want full Model Family labels in wide panes, so that the represented model category is explicit.
17. As an Antigravity user, I want compact labels such as `Claude/GPT` and `Gemini` when the full Model Family label would hide higher-priority data, so that the row remains readable.
18. As a user of an extremely narrow pane, I want an overlong label truncated with an ellipsis, so that percentage and Reset Countdown remain complete.
19. As a Herdr user, I want responsive fitting based on actual rendered width, so that the layout adapts correctly to content and terminal font metrics rather than arbitrary breakpoint assumptions.
20. As a Herdr user, I want the filled bar segment colored by quota health, so that low capacity is visible without reading every number.
21. As a Herdr user, I want the percentage text to use the same quota-health color as its bar, so that the visual signal is consistent.
22. As a Herdr user, I want percentages rounded to whole numbers, so that compact rows remain stable and easy to scan.
23. As a Herdr user, I want missing remaining capacity shown as `—`, so that unavailable data is distinguishable from zero remaining capacity.
24. As a Herdr user, I want a Quota Window with unknown remaining capacity to show a fully dimmed thin bar, so that the visual representation does not imply a numeric value.
25. As a Herdr user, I want missing reset timing shown as `reset —`, so that an isolated dash is not ambiguous.
26. As a Herdr user, I want an elapsed Reset Countdown displayed as `ready`, so that the row communicates availability without negative durations.
27. As a Codex user, I want Manual Reset Credits shown on one compact line without a progress bar, so that a count is not presented as a percentage.
28. As a Herdr user, I want consecutive Accounts displayed without an empty line, so that the two-row Quota Windows do not make the view unnecessarily tall.
29. As a Herdr user, I want one empty line preserved between Provider Groups, so that Codex, Antigravity, and Claude remain visually distinct.
30. As a narrow-pane user, I want a compact footer containing refresh, configuration, and close keys, so that essential controls remain visible.
31. As a wide-pane user, I want the full navigation footer retained, so that all existing keys remain discoverable when space permits.
32. As a narrow-pane user, I want Update Availability preserved before the `Updated` timestamp, so that actionable version information is not clipped by passive metadata.
33. As a narrow-pane user, I want the Account name or email preserved before status badges, so that I can still identify which Account a quota belongs to.
34. As a user resizing a Herdr pane, I want the Quota View to reflow immediately without horizontal clipping, so that the layout stays usable while arranging my workspace.
35. As an existing CPA Quota user, I want provider grouping, quota calculation, ordering, Reset Countdown semantics, and Manual Reset Credit values unchanged, so that this is a presentation-only improvement.
36. As a user on macOS, Linux, or Windows, I want the thin-line progress bar to render consistently, so that the compact UI remains portable.

## Implementation Decisions

- The change is limited to presentation and responsive layout. It does not change Management API calls, Quota Snapshot data, quota calculations, sorting, provider semantics, or configuration.
- Each Quota Window always renders as two terminal rows. The metadata row contains Window Label, remaining percentage, Reset Countdown, and optional absolute reset time. The progress row contains only indentation and the thin progress bar.
- Normal rows contain no `remaining`, `resets`, or countdown parentheses. Metadata is separated by compact spacing.
- Field priority is Window Label, percentage, Reset Countdown, progress bar, then absolute reset time. Percentage and Reset Countdown are protected from clipping.
- Responsive decisions use actual rendered terminal-cell width. The renderer must not rely solely on fixed pane-width tiers such as 45, 60, or 80 columns.
- The absolute reset timestamp uses the existing local `DD/MM HH:mm` representation and is included only when the complete field fits.
- Label width is calculated independently for each Provider Group so related percentage values and bars align without preserving the current global 22-column padding.
- Full Model Family labels are used when they fit. Compact labels are `Claude/GPT` and `Gemini`. If a compact label still does not fit, it is truncated with an ellipsis.
- The progress bar starts below the percentage column and uses between 6 and 24 character cells according to available width.
- The filled progress glyph is `━`; the empty glyph is `─`. Filled and empty segments use the existing health and dim styles respectively.
- The glyph change reduces visual thickness only. A terminal cannot allocate less than one row to a text progress bar.
- Remaining capacity and its progress bar retain the existing thresholds: green above 50%, yellow from 21% through 50%, and red at or below 20%.
- Percentage display retains integer rounding. No decimal precision is added.
- Unknown remaining capacity renders as `—` with a fully dimmed bar. It is not displayed as zero and does not receive a percent sign.
- Unknown reset timing renders as `reset —`. An elapsed Reset Countdown renders as `ready`, followed by the absolute reset time only when that full timestamp fits.
- Manual Reset Credits remain a one-row label-and-count value without a progress bar.
- Empty lines between Accounts are removed. Empty lines between Provider Groups remain.
- In constrained widths, the header preserves title and version/update information before dropping the passive `Updated` timestamp.
- In constrained widths, the Account identity is preserved before optional status badges.
- The footer has a compact narrow-pane form containing refresh, configuration, and close controls. The current full footer remains available when it fits.
- No new user configuration or rendering dependency is introduced.

## Testing Decisions

- Tests assert consumer-visible layout and information priority, not internal padding constants, helper call structure, or source text.
- Three seams are used because deterministic countdown/layout checks and real terminal glyph rendering have different requirements.
- **Quota Snapshot rendering seam:** render a representative fixed Quota Snapshot with a fixed clock at widths including approximately 30, 45, 60, and 80 columns. Verify every rendered line fits the requested width, percentage and Reset Countdown remain visible, verbose prose is absent, the full timestamp appears only when it fits, Model Family labels shorten or truncate as required, and bar length remains between 6 and 24 cells.
- The rendering seam covers normal, low-quota, critical-quota, unknown remaining capacity, unknown reset timing, elapsed reset timing, Manual Reset Credits, multiple Accounts, and multiple Provider Groups.
- Width assertions use rendered terminal-cell width rather than byte length because Unicode glyphs and ANSI styling are present.
- **Quota View model seam:** resize a populated UI model to narrow and wide dimensions and inspect the rendered View. Verify compact/full footer selection, Update Availability priority over the `Updated` timestamp, Account identity priority over badges, Account spacing, Provider Group spacing, and preservation of the previous scrolling behavior.
- **Actual Herdr smoke seam:** build and open the linked plugin in a real Herdr session, display it in both a full Herdr Tab and a narrow split pane, resize the pane, and visually confirm `━/─` rendering, color thresholds, alignment, lack of clipping, and readability in the actual terminal surface.
- Existing Reset Countdown tests provide prior art for deterministic time boundaries. Existing version-line and UI-model tests provide prior art for testing rendered user-visible state without asserting implementation plumbing.
- The actual Herdr smoke check is required because automated string-width tests cannot prove how a user's terminal font visually renders thin Unicode box-drawing characters.

## Out of Scope

- Changing any CLIProxyAPI endpoint, authentication behavior, provider request, quota parser, or Quota Snapshot field.
- Changing provider grouping, account eligibility, account ordering, quota percentage calculation, Reset Countdown calculation, or Manual Reset Credit semantics.
- Adding horizontal scrolling or relying on terminal line wrapping.
- Rendering a true sub-row-height graphical bar; the terminal progress bar remains one text row.
- Adding decimal percentage precision.
- Adding new configurable breakpoints, bar characters, colors, layout modes, or compact-mode settings.
- Adding a third metadata row for extremely narrow panes.
- Localizing labels, dates, countdowns, or key hints.
- Changing the tab-local Quota View lifecycle, registry, Shortcut Binding, configuration flow, or CPA Version fetching behavior.

## Further Notes

- The current row format places the Reset Countdown at the extreme right and spends 22 columns on a fixed label field plus approximately 21 columns on the words `remaining` and `resets`. Reordering and deleting this static prose addresses the root cause rather than merely shortening the existing progress bar.
- The minimum physical terminal height for a text progress bar is one row. Thin box-drawing characters reduce visual weight but do not reduce row allocation.
- There is no ADR for this change. The layout is easy to revise, follows existing terminal capabilities, and creates no architectural lock-in.
