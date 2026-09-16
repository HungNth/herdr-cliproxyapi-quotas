# Quota Health Color Thresholds

Status: ready-for-agent

## Problem Statement

In the current Quota View, remaining capacity coloring uses thresholds that trigger warnings too late and flag healthy capacity too aggressively. Quota Windows turn yellow only when remaining capacity drops to 50% or below, and red only at 20% or below. 

From the user's perspective:
1. Having 25% or 30% remaining capacity is already a critical level where tasks may soon fail or require fallback models, but the UI still displays it in yellow warning rather than an urgent red warning.
2. Capacity between 51% and 69% is displayed as fully healthy green, giving a false impression of abundant quota when more than a third of the allowance has already been consumed.
3. Users need immediate, intuitive visual signals: critical capacity (0% to 30%) in red, moderate warning capacity (> 30% to < 70%) in yellow, and safe capacity (70% to 100%) in green.

## Solution

Update the quota health color mapping so that both the numeric percentage text and the thin progress bar reflect the refined health boundaries:

- **Critical / Low (Red)**: Remaining capacity from 0.0% through 30.0% inclusive (`remaining <= 30.0`).
- **Moderate / Warning (Yellow)**: Remaining capacity greater than 30.0% and strictly less than 70.0% (`30.0 < remaining < 70.0`).
- **Healthy / Safe (Green)**: Remaining capacity from 70.0% through 100.0% inclusive (`remaining >= 70.0`).

All existing visual hierarchy conventions are preserved:
- Unknown remaining capacity (`nil`) continues to render as `—` with a dimmed progress bar.
- Both the percentage label and the filled segments of the thin progress bar share the exact same health style.
- The color palette (ANSI colors for red, yellow, and green) and thin bar glyphs remain unchanged.

## User Stories

1. As a Herdr user, I want remaining capacity between 0% and 30% displayed in red, so that I can immediately notice critically low quota before running expensive agent workflows.
2. As a Herdr user, I want a Quota Window at exactly 30% displayed in red, so that the boundary is predictably included in the critical tier.
3. As a Herdr user, I want a Quota Window at 0% displayed in red, so that fully exhausted quota communicates maximum urgency.
4. As a Herdr user, I want remaining capacity between 31% and 69% displayed in yellow, so that I am cautioned that substantial quota has been consumed.
5. As a Herdr user, I want remaining capacity just above 30% (e.g., 30.5%) displayed in yellow rather than red, so that the threshold transition on real floating-point data is smooth and accurate.
6. As a Herdr user, I want remaining capacity at exactly 70% displayed in green, so that reaching 70% capacity is recognized as being in the safe tier.
7. As a Herdr user, I want remaining capacity above 70% displayed in green, so that I know plenty of allowance is available.
8. As a Herdr user, I want 100% capacity displayed in green, so that completely refreshed quota is clearly indicated.
9. As a Herdr user, I want the filled segment of the progress bar to use the exact same color as the percentage text, so that the visual cues are harmonized.
10. As a Herdr user, I want unknown remaining capacity to remain dimmed without color highlights, so that missing data is never mistaken for critical or healthy quota.
11. As a Herdr user, I want the threshold change applied uniformly across all Provider Groups (Codex, Claude, and Antigravity), so that quota health is consistent across different upstream providers.
12. As a Herdr user, I want the threshold change applied across both 5-hour and Weekly Quota Windows, so that short-term and long-term allowances share the same mental model.
13. As a terminal user, I want the ANSI color definitions preserved, so that my existing terminal theme contrast remains legible.
14. As a user resizing Herdr panes, I want the color styles to persist across narrow and wide viewport widths, so that resizing does not affect color semantics.
15. As a developer maintaining the codebase, I want unit tests that assert style decisions at exact boundary points (0%, 30%, 30.1%, 50%, 69.9%, 70%, 100%), so that future refactors cannot unintentionally regress color thresholds.

## Implementation Decisions

- Update the health style selector in the UI module to compare remaining capacity against the new boundary values:
  - Return critical error style when `remaining <= 30.0`.
  - Return warning style when `remaining < 70.0`.
  - Return healthy good style otherwise (when `remaining >= 70.0`).
- Comparisons are performed directly on unrounded floating-point capacity numbers so that fractional quotas near boundaries transition consistently without rounding distortion.
- Do not modify ANSI color codes, text layout, row padding, bar glyphs, or responsive width fitting.
- Do not alter Quota Snapshot fetching, CPA API interactions, or data parsing.

## Testing Decisions

- Only test user-visible behavior and style assignment, not internal function naming or styling implementation details.
- **Seam**: Quota Window health styling seam in the UI test suite.
  - Test explicit numeric values:
    - 0.0%: critical style (red)
    - 15.0%: critical style (red)
    - 30.0%: critical style (red)
    - 30.1%: warning style (yellow)
    - 50.0%: warning style (yellow)
    - 69.9%: warning style (yellow)
    - 70.0%: healthy style (green)
    - 85.0%: healthy style (green)
    - 100.0%: healthy style (green)
- Existing tests in the UI test suite verifying snapshot rendering, compact row formatting, and bar styles provide prior art for asserting rendered styles and ANSI output.

## Out of Scope

- Changing the three color definitions (ANSI color values).
- Changing layout structure, character glyphs, or label compacting rules.
- Allowing user configuration of color thresholds in config files.
- Modifying quota calculation or upstream provider querying.

## Further Notes

- This change sharpens the operational utility of CPA Quota: 30% is a natural warning threshold for agent workloads, giving developers adequate warning to switch accounts or wait for a Quota Window reset.
