# CPA Quota

A Go terminal plugin that shows CLIProxyAPI quotas in a Herdr popup.

## Data boundary

The plugin only calls these CLIProxyAPI Management API endpoints:

- `GET /v0/management/auth-files`
- `POST /v0/management/api-call`

All provider quota lookups are read-only requests forwarded through `api-call`. The plugin does not read local token files, call providers directly, consume Codex reset credits, reset CPA quota state, or modify accounts.

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

The first popup asks for the CLIProxyAPI base URL and management key. Configuration is stored in Herdr's plugin config directory; manual JSON editing is not required.

## Keys

- `R` or `r`: refresh
- `C` or `c`: configure
- `j`/`k`, arrows, Page Up/Page Down, `g`/`G`: scroll
- `q` or Escape: close

## Providers

- Codex: 5-hour, weekly, and available manual reset credits
- Claude OAuth: 5-hour and weekly
- Antigravity: `Claude & GPT models` and `Gemini models`

Empty provider groups are hidden. Disabled and unavailable accounts remain visible with status badges.
