# CPA Quota Plugin

Status: confirmed

## Goal

Provide a read-only Herdr popup for viewing CLIProxyAPI account quotas across Codex, Antigravity, and Claude.

## Boundary

The plugin calls only `GET /v0/management/auth-files` and `POST /v0/management/api-call` on CLIProxyAPI. It never reads local token files, calls providers directly, consumes reset credits, resets quota state, or mutates account configuration or routing state.

## Behavior

- `prefix+u` opens a 90% by 80% popup.
- First use configures one CPA base URL and management key inside the popup.
- `R` refreshes, `C` configures, navigation keys scroll, and `q` or Escape closes.
- Groups are Codex, Antigravity, and Claude; empty groups are hidden.
- Disabled and unavailable accounts remain visible.
- Account identity falls back through email, account, name/label, then auth index.
- Quotas show remaining percentage, a progress bar, and local reset time.
- Accounts sort by lowest remaining quota, with failures last.

## Provider semantics

- Codex shows 5-hour, weekly, and available manual reset credit count.
- Claude includes OAuth subscription accounts only and shows 5-hour and weekly quota.
- Antigravity combines `claude-4.6-*` and `gpt-*` into `Claude & GPT models`, and `gemini-3.*` into `Gemini models`. Other models are ignored. Each family reports its lowest remaining quota.

## Platforms

The plugin supports macOS, Linux, and Windows. Development runtime verification occurs on macOS; Linux and Windows receive cross-compilation checks, with Windows runtime verification deferred to the user.
