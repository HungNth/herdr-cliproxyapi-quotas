# 01: Render Compact Two-Row Quota Windows

**What to build:** Every Quota Window in the Quota View renders as two rows: a metadata row with Window Label, remaining percentage, Reset Countdown, and absolute reset time; a thin progress bar row beneath it using `━/─`. Normal rows no longer contain `remaining`, `resets`, or countdown parentheses. Manual Reset Credits remain a single label-and-count line without a bar.

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

- [ ] Rendering a fixed Quota Snapshot with a fixed clock shows `label  NN%  in 1h30m  DD/MM HH:mm` followed by a thin bar row for normal Quota Windows
- [ ] Normal rows contain no `remaining`, `resets`, or parentheses; percentage and Reset Countdown use existing green/yellow/red thresholds with integer rounding on both text and bar
- [ ] Unknown remaining capacity renders as `—` with a fully dimmed bar; unknown reset timing renders as `reset —`; elapsed countdown renders as `ready`
- [ ] Manual Reset Credits render as one `Manual resets  N` row without a progress bar
- [ ] At a wide width, no rendered line exceeds the available terminal-cell width, and smoke rendering on the real surface reads as thin lines, not blocks
