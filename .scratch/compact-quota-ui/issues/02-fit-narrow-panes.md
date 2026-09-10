# 02: Fit Quota Windows into Narrow Panes

**What to build:** No horizontal clipping at any pane width. The renderer measures real terminal-cell width and, in order: drops the absolute reset timestamp, switches Model Family labels to `Claude/GPT`/`Gemini`, truncates labels with an ellipsis, and shrinks the progress bar from 24 toward 6 cells. Percentage and Reset Countdown are never clipped, and bars align within each Provider Group.

**Blocked by:** 01: Render Compact Two-Row Quota Windows.

**Status:** ready-for-agent

- [ ] Rendering at approximately 30, 45, 60, and 80 columns keeps every line within the requested width, verified by terminal-cell width rather than byte length
- [ ] Percentage and Reset Countdown remain visible at every tested width; the absolute timestamp appears only when the complete field fits
- [ ] Antigravity labels use `Claude/GPT` and `Gemini` when full labels would crowd out protected fields, and truncate with an ellipsis as a last resort
- [ ] Progress bars stay between 6 and 24 cells and share a common start column within each Provider Group
- [ ] No rendered line exceeds the pane width at any tested size
