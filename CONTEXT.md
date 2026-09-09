# CPA Quota

CPA Quota presents the current subscription capacity of provider accounts routed through CLIProxyAPI.

## Language

**CPA Endpoint**:
A CLIProxyAPI Management API instance from which account and quota data is read.
_Avoid_: Proxy URL, provider URL

**Account**:
An upstream provider identity represented by one CPA credential.
_Avoid_: Auth file, profile, token

**Provider Group**:
A collection of accounts sharing the same upstream quota semantics: Codex, Antigravity, or Claude.
_Avoid_: Section, channel

**Quota Window**:
A time-bounded usage allowance with remaining capacity and a reset time.
_Avoid_: Limit bucket, rate window

**Reset Countdown**:
The elapsed time remaining until a Quota Window reaches its scheduled reset time.
_Avoid_: ETA, duration remaining

**Model Family**:
An Antigravity quota category that combines models with equivalent user-facing capacity semantics.
_Avoid_: Individual model, model list

**Manual Reset Credit**:
An OpenAI-granted, one-use credit that restores Codex quota without changing the quota window's scheduled reset time.
_Avoid_: CPA quota reset, reset token

**Quota Snapshot**:
The account and quota state produced by one complete refresh operation.
_Avoid_: Usage report, cache

**Shortcut Binding**:
A user-owned Herdr keybinding that opens CPA Quota through its plugin action.
_Avoid_: Plugin shortcut, auto-keybinding
