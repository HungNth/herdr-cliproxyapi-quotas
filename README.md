# CPA Quota

A Go terminal plugin that shows CLIProxyAPI quotas and CPA version status in a tab-local Herdr Quota View.

## Data boundary

The plugin only calls these CLIProxyAPI Management API endpoints:

- `GET /v0/management/auth-files`
- `GET /v0/management/latest-version`
- `POST /v0/management/api-call`

All provider quota lookups are read-only requests forwarded through `api-call`. The plugin does not read local token files, call providers directly, consume Codex reset credits, reset CPA quota state, modify accounts, or update CLIProxyAPI.

## Install

Go 1.26 or newer is required because Herdr builds the plugin during installation.

For local development:

- MacOS/Linux:

```bash
go build -o bin/cpa-quotas ./cmd/cpa-quotas
herdr plugin link .
```

- Windows:

```powershell
go build -o bin/cpa-quotas.exe ./cmd/cpa-quotas
herdr plugin link .
```

Automatic keybinding setup: run once from a local (non-SSH) session:

```bash
herdr plugin action invoke herdr-cliproxyapi-quotas.shortcut
```

This appends `prefix+u` to your Herdr `config.toml` unless that key is already bound to a different command. It refuses to run over SSH and never overwrites an existing binding. Remote clients with local keybindings must add the binding manually.

Herdr reads config from:

```plaintext
Linux and macOS: ~/.config/herdr/config.toml
Windows:          %APPDATA%\herdr\config.toml
```

```toml
[[keys.command]]
key = "prefix+u"
type = "plugin_action"
command = "herdr-cliproxyapi-quotas.open"
description = "open CPA quota"
```

Reload Herdr configuration:

```bash
herdr server reload-config
```

The first view asks for the CLIProxyAPI base URL and management key. Configuration is stored in Herdr's plugin config directory; manual JSON editing is not required.

## Quota View behavior

- `prefix+u` opens the Quota View as a right-hand split (50/50) beside your active pane. Your work pane remains live so you can type, edit, and create further right or down splits without the quota view covering them.
- Focus policy: a configured Quota View opens without stealing focus (`--no-focus`), allowing you to continue typing. On first use (when credentials need to be entered), it receives focus automatically.
- Toggle & dedup: each Herdr Tab owns at most one Quota View. Pressing `prefix+u` from a work pane focuses the existing Quota View; pressing `prefix+u` from inside the focused Quota View closes it as a toggle.
- Resizing and manual rearrangements: the Quota View is a normal Herdr pane; you can drag split dividers, swap panes, or create additional splits around it. If closed manually or via `q`/Escape, the next `prefix+u` reopens a fresh split automatically.

## Keys

- `R` or `r`: refresh quotas and version status
- `C` or `c`: configure
- `j`/`k`, arrows, Page Up/Page Down, `g`/`G`: scroll
- `q` or Escape: close the view

## Providers

- Codex: 5-hour, weekly, and available manual reset credits
- Claude OAuth: 5-hour and weekly
- Antigravity: `Claude & GPT models` and `Gemini models`

Empty provider groups are hidden. Disabled and unavailable accounts remain visible with status badges.
