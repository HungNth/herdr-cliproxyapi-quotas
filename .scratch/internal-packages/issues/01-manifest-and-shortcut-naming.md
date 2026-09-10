# 01: Synchronize Manifest, Shortcut, and Documentation Naming

**What to build:** Align the plugin codebase with the updated manifest naming contract (`herdr-cliproxyapi-quotas`, `quotas` pane, `cpa-quotas` binary, and `shortcut` action). Replace the obsolete `install-shortcut` subcommand with `shortcut`, install the `herdr-cliproxyapi-quotas.open` keybinding, update build targets in the manifest and README to produce `cpa-quotas`, update the README shortcut installation instructions to use `herdr plugin action invoke herdr-cliproxyapi-quotas.shortcut`, and state the required Go 1.26 runtime without aliases or migration instructions.

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

- [ ] Plugin identifier in code, pane launching, registry paths, and config paths matches `herdr-cliproxyapi-quotas`
- [ ] Pane entrypoint in launcher matches `quotas`
- [ ] Subcommand dispatch recognizes `shortcut` and removes `install-shortcut`
- [ ] Keybinding written to Herdr config is `herdr-cliproxyapi-quotas.open`
- [ ] README documents Go 1.26 requirement and `herdr plugin action invoke herdr-cliproxyapi-quotas.shortcut`
- [ ] Existing launcher and shortcut unit tests pass with updated names
- [ ] Real Herdr smoke test verifies plugin link, action invoke `shortcut`, and split pane open/toggle
