# 02: User-configurable Refresh Interval

**What to build:** The configuration form gains a third field, "Refresh interval (seconds)". Blank means the 60-second default; whole numbers of at least 5 are accepted; anything else fails with the same inline validation error style as the other fields. Saving applies the new interval immediately and restarts the countdown. The interval persists in the configuration document under `refresh_interval` (seconds); absent or zero means the default, and the file stays editable by hand. The UI schedules from the configured value, so a short interval like 10 seconds polls quickly. The README documents the setting, its default, and the minimum.

**Blocked by:** 01: Auto-refresh on the default 60-second interval.

**Status:** ready-for-agent

- [ ] Configuration field defaults to 60 on absence and zero; values below 5 (including negatives) fail validation; legal values round-trip
- [ ] Form field participates in focus cycling, shows the effective value, and validates like the others (blank → default, non-numeric → clear error)
- [ ] Saving applies the interval immediately and restarts the countdown (observed at the model seam)
- [ ] Auto-refresh honors the configured interval, including sub-minute values
- [ ] Configuration round-trip tests extended: default, legal, below-minimum, non-numeric
- [ ] Real Herdr smoke: short interval set via the form produces visibly faster polling; config restored afterwards
- [ ] README documents setting, default, and minimum
- [ ] Full suite, gofmt, and cross-compile pass
