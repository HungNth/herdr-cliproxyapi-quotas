# CPA Quota

A Go terminal plugin that shows CLIProxyAPI quotas and CPA version status in a tab-local Herdr Quota View.

## Data boundary

The plugin only calls these CLIProxyAPI Management API endpoints:

- `GET /v0/management/auth-files`
- `GET /v0/management/latest-version`
- `POST /v0/management/api-call`

All provider quota lookups are read-only requests forwarded through `api-call`. The plugin does not read local token files, call providers directly, consume Codex reset credits, reset CPA quota state, modify accounts, or update CLIProxyAPI.

## Install

Go 1.23 or newer is required because Herdr builds the plugin during installation.

For local development:

- MacOS/Linux:

```bash
go build -o bin/cpa-quota .
herdr plugin link .
```

- Windows:

```powershell
go build -o bin/cpa-quota.exe .
herdr plugin link .
```

The manifest also declares an `install-shortcut` action, so `herdr plugin action invoke herdr-cliproxyapi-quota-plugin.install-shortcut` runs the same setup as the command below.

Automatic keybinding setup: run once from a local (non-SSH) session:

- MacOS/Linux:

```bash
bin/cpa-quota install-shortcut
```

- Windows

```bash
.\bin\cpa-quota.exe install-shortcut
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
command = "herdr-cliproxyapi-quota-plugin.open"
description = "open CPA quota"
```

Reload Herdr configuration:

```bash
herdr server reload-config
```

The first view asks for the CLIProxyAPI base URL and management key. Configuration is stored in Herdr's plugin config directory; manual JSON editing is not required.

## Quota View behavior

- `prefix+u` opens the Quota View as an overlay in the current Herdr Tab. You can switch to another Herdr Tab and keep working; the view, its snapshot, scroll position, and reset countdowns stay alive. Press the shortcut again to focus it, or switch back to that tab.
- Each Herdr Tab owns at most one Quota View. A Herdr Tab opened from another tab gets its own view. If a view was closed or removed, the next invocation opens a new one automatically.
- The version line under the title compares the running CLIProxyAPI release (the `x-cpa-version` response header) against the latest stable release (`/v0/management/latest-version`): current, checking latest, update available, ahead of latest, version unknown, or latest check unavailable. A version check failure never affects quota data.
- Because Herdr overlays attach to the active pane at open time, open the view from the tab you are currently in; the shortcut runs while Herdr is waiting for the action, so this is the normal case.

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
