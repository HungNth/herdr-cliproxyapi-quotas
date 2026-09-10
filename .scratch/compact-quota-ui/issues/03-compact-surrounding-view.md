# 03: Compact the Surrounding Quota View

**What to build:** The chrome around the Quota Windows compacts responsively: Accounts render without an intervening empty line while Provider Groups keep one; the footer switches between full and compact forms; the header preserves title and Update Availability before the passive `Updated` timestamp; Account identity is preserved before optional status badges; resizing reflows the whole view immediately.

**Blocked by:** 02: Fit Quota Windows into Narrow Panes.

**Status:** ready-for-agent

- [ ] Consecutive Accounts render without an empty line between them; exactly one empty line separates Provider Groups
- [ ] A narrow model view shows the compact footer (`R refresh · C config · q close`); a wide view keeps the full footer
- [ ] In narrow widths the `Updated` timestamp is dropped before Update Availability or the title; Account identity is kept before status badges
- [ ] Resizing a populated model reflows the entire view immediately at the new width without stale rows
- [ ] Full suite passes on the development platform and smoke rendering on the real surface confirms spacing and footer behavior in wide and narrow panes
