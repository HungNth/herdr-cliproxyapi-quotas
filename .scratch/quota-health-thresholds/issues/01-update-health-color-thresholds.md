# 01: Update quota health color thresholds

**What to build:** Update Quota Window remaining capacity health coloring in the UI so that 0%-30% is critical (red), >30%-<70% is warning (yellow), and 70%-100% is healthy (green). Floating-point comparisons must be direct and continuous without rounding artifacts. Both the percentage label and the filled segments of the thin progress bar must share the updated health styling.

**Blocked by:** None (can start immediately).

**Status:** resolved

- [x] `healthStyle` returns `errorStyle` (red) for remaining capacity <= 30.0%
- [x] `healthStyle` returns `warningStyle` (yellow) for remaining capacity > 30.0% and < 70.0%
- [x] `healthStyle` returns `goodStyle` (green) for remaining capacity >= 70.0%
- [x] Unit tests in `internal/ui/ui_test.go` test boundaries at 0.0%, 15.0%, 30.0%, 30.1%, 50.0%, 69.9%, 70.0%, 85.0%, 100.0%
- [x] Existing UI snapshot and model tests pass with the new thresholds
- [x] Full test suite (`go test ./...`) and binary build pass

## Answer

Updated `healthStyle` in `internal/ui/ui.go` to use:
- `remaining <= 30.0`: `errorStyle` (red, color 196)
- `remaining < 70.0`: `warningStyle` (yellow, color 214)
- `default` (>= 70.0%): `goodStyle` (green, color 42)

Added `TestHealthStyleThresholds` in `internal/ui/ui_test.go` asserting foreground color style at 0%, 15%, 30%, 30.1%, 50%, 69.9%, 70%, 85%, and 100%. All unit tests and binary builds pass.
