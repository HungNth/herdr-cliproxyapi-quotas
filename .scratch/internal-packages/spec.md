# Internal Packages and Manifest Naming Cutover

Status: ready-for-agent

## Problem Statement

The plugin currently keeps its Go application, CPA client, Quota Snapshot models, configuration handling, Herdr integration, and terminal UI in the root main package. This makes responsibilities harder to navigate and leaves unrelated implementation details accessible throughout the application.

The maintainer has also changed the plugin manifest to use the plugin ID `herdr-cliproxyapi-quotas`, executable name `cpa-quotas`, pane entrypoint `quotas`, and shortcut action/subcommand `shortcut`. The application still uses the previous plugin ID, pane entrypoint, and shortcut subcommand. Updating installation documentation alone would leave the advertised actions disconnected from the executable's actual behavior.

The README still documents the old executable and direct `install-shortcut` invocation, and its Go requirement does not match the module's Go 1.26 requirement.

## Solution

Organize the application as a small command entrypoint and four internal packages with clear responsibilities: CPA data access and models, configuration, Herdr integration, and terminal UI. Move tests alongside the behavior they protect rather than creating a second implementation or a new testing architecture.

Complete the naming cutover already initiated in the manifest across the executable, launcher, shortcut installation, tests, build instructions, and README. The documented shortcut installation command becomes `herdr plugin action invoke herdr-cliproxyapi-quotas.shortcut` on all supported platforms.

Preserve the existing Quota View behavior, CPA data boundary, and configuration protections. This is a structural refactor and naming cutover, not a product behavior redesign.

## User Stories

1. As a maintainer, I want a small command entrypoint outside the repository root, so that startup and command dispatch are easy to locate.
2. As a maintainer, I want application implementation in internal packages, so that the repository root is not a flat collection of application Go files.
3. As a maintainer, I want CPA HTTP access, Quota Snapshot models, and CPA Version comparison grouped together, so that provider behavior can be understood without navigating Herdr or UI code.
4. As a maintainer, I want configuration loading, saving, defaults, and validation grouped together, so that credential handling has one owner.
5. As a maintainer, I want pane launching, focus, toggle, registry handling, locking, and Shortcut Binding installation grouped together, so that Herdr integration has one owner.
6. As a maintainer, I want Bubble Tea state, rendering, and keyboard handling grouped together, so that changes to the Quota View stay local to its UI.
7. As a maintainer, I want a small exported package interface and no cyclic dependencies, so that moving files creates useful separation rather than exposing every helper.
8. As a maintainer, I want existing tests and their supporting fixtures located with the owning package, so that verification remains runnable after the move.
9. As a maintainer, I want the manifest and local development instructions to build the new command entrypoint, so that Herdr installation and manual builds produce the same application.
10. As a plugin user, I want the executable and manifest to agree on the plugin ID and pane entrypoint, so that opening the Quota View targets the installed plugin.
11. As a plugin user, I want one documented Herdr action command to install the shortcut on Windows, macOS, and Linux, so that I do not need platform-specific executable invocation instructions for that operation.
12. As a plugin user, I want the `shortcut` action to run shortcut installation rather than launch the Quota View, so that the manifest action does what its title promises.
13. As a plugin user, I want the installed Shortcut Binding to invoke `herdr-cliproxyapi-quotas.open`, so that it opens the renamed plugin.
14. As a plugin user, I want shortcut installation to retain its conflict protection and SSH refusal, so that restructuring does not weaken safeguards around my Herdr configuration.
15. As a plugin user, I want a configured Quota View to open as a right-hand split without taking focus from my work pane, so that I can continue working while monitoring quota.
16. As a first-use plugin user, I want the configuration view to receive focus when configuration is required, so that I can enter my CPA Endpoint and management key immediately.
17. As a plugin user, I want repeated shortcut invocation from a work pane to focus the existing Quota View and invocation from that view to close it, so that the current toggle behavior is preserved.
18. As a plugin user, I want a closed Quota View to reopen on the next invocation without leaving duplicates, so that moving code does not break registry recovery.
19. As a plugin user, I want Quota Snapshots, provider grouping, reset countdowns, and CPA Version status to behave as before, so that the refactor does not alter the information I rely on.
20. As a plugin user, I want quota and version lookups to retain their read-only Management API boundary, so that restructuring cannot introduce account or quota mutations.
21. As a maintainer, I want the README to state the actual Go requirement and new executable name, so that development instructions match the project configuration.
22. As a maintainer working on an unreleased development setup, I want a clean naming cutover without aliases, migrations, or migration instructions, so that obsolete names do not become a supported compatibility contract.
23. As a maintainer, I want the built command and actual Herdr action flow smoke-tested in addition to the existing tests, so that cross-package wiring and manifest integration are exercised rather than merely compiled.

## Implementation Decisions

- Use a `cmd` command entrypoint named `cpa-quotas` and four packages under Go's `internal` mechanism: `cpa`, `config`, `herdr`, and `ui`. Application Go implementation and its package tests no longer live in the root main package.
- Keep the command entrypoint responsible for argument dispatch, application composition, and process-level error reporting. It must not retain the CPA client, pane registry, shortcut editing, or rendering implementation.
- The `cpa` package owns Management API access, provider query behavior, Quota Snapshot and related quota models, and CPA Version handling. It must not depend on the UI or Herdr integration packages.
- The `config` package owns CPA Endpoint configuration defaults, validation, loading, and saving. Preserve credential handling, permission behavior, and environment-selected configuration storage.
- The `herdr` package owns plugin action integration, opening/focusing/closing Quota Views, per-Herdr-Tab registry behavior, platform-specific locking, and Shortcut Binding installation. Keep plugin identity and pane identity consistent wherever they affect action routing or plugin-specific storage.
- The `ui` package owns Bubble Tea model state, terminal rendering, input handling, refresh orchestration, and configuration interaction. Preserve its existing user-visible behavior.
- Maintain acyclic package dependencies. Export only the types and operations needed by actual callers. Do not add general-purpose `utils` or `common` packages, pass-through layers, speculative interfaces, factories, or new dependencies merely to accommodate the move.
- Keep configuration persistence concerns out of CPA provider behavior. Place shared types with the package that owns their meaning instead of copying them across packages.
- Move existing tests with the owning behavior and relocate the fake Herdr executable support with the Herdr tests. Update relative fixture and build references so tests remain independent of the repository root being their working directory.
- Treat the maintainer's current manifest values as the naming contract: plugin ID `herdr-cliproxyapi-quotas`, executable `cpa-quotas` with the normal Windows executable suffix, pane entrypoint `quotas`, open action/subcommand `open`, and shortcut action/subcommand `shortcut`.
- Recognize `shortcut` in executable dispatch and have it perform the existing shortcut setup. Remove support and active documentation for the obsolete `install-shortcut` subcommand; do not retain an alias.
- Use `herdr-cliproxyapi-quotas.open` as the Shortcut Binding action command. Preserve refusal to overwrite a conflicting binding, refusal to edit configuration over SSH, configuration checking, and the existing reload behavior.
- Update platform-specific manifest build targets to compile the relocated `cpa-quotas` command rather than the root package. Preserve the maintainer's manifest metadata and the existing split placement unless a change is required by the agreed entrypoint relocation.
- Update README build instructions to target the relocated command and produce the renamed executable. Replace direct shortcut executable instructions with the single Herdr action invocation shared by all supported platforms. State Go 1.26 as the required version, consistent with the current module.
- Keep the existing Go module identity and dependency versions; renaming the executable and plugin does not require a module identity change or dependency upgrade.
- Preserve current configuration and registry formats and environment overrides. New plugin-specific default storage references use the new plugin identity; do not search, import, or migrate old plugin storage.
- Preserve right-hand split placement, native split sizing, invoking-pane targeting, configured versus first-use focus policy, per-Herdr-Tab deduplication, toggle-close, stale recovery, and existing handling of manually moved views. Do not add new topology reconciliation behavior.
- Preserve the existing CPA Management API request boundary and quota/version semantics. Do not change provider parsing, UI layout, refresh policy, or user keybindings as part of package relocation.

## Testing Decisions

- The user approved **existing behavior tests plus smoke tests**, not a new permanent CLI integration suite.
- Reuse the existing seams. A good test checks an observable result, state transition, precedence rule, trust-boundary rejection, or real error path. Do not add tests that assert directory layouts, source text, private helper names, import wiring, or forwarded values solely to prove that files moved.
- Retain meaningful existing behavior coverage and adjust it for package ownership and the new naming contract. Tests that only pin obsolete wording or incidental implementation details should not be rewritten into new implementation-pinning tests.
- For `cpa`, reuse the HTTP test-server seam used by the existing Quota Snapshot client tests. Preserve checks for allowed Management API requests, authentication, provider results, quota interpretation, and CPA Version behavior without reaching live provider accounts in automated tests.
- For `herdr`, reuse the existing fake Herdr executable and temporary registry/configuration setup. Preserve coverage for right-split targeting, focus policy, existing-view focus, toggle-close, stale recovery, shortcut conflict protection, and relevant error behavior. Isolate environment variables so inherited developer state cannot alter test outcomes.
- For `config` and `ui`, keep the existing configuration, UI state/rendering, and version-related behavior tests with their owning packages. Moving a behavior between packages must not silently drop its existing consumer-facing coverage.
- Use the highest cross-package verification seam for the naming cutover: build the actual relocated command, then exercise `open` and `shortcut` dispatch in a throwaway smoke setup. Verify that the new identities are used and that `shortcut` installs the intended Shortcut Binding rather than starting the UI. Do not retain the throwaway script as a new permanent test framework or suite.
- Exercise the actual Herdr plugin action flow after building/linking the new manifest: invoke the documented shortcut action, verify its resulting binding, and exercise opening, focusing, and toggle-closing the Quota View. Observe pane state or terminal output as evidence; command construction alone does not establish actual host integration.
- Cover both configured and first-use focus behavior using isolated test configuration rather than deleting or changing unrelated user credentials.
- Run the affected package tests during implementation and the full Go test suite after integration. Format changed Go code and compile for Windows, macOS, and Linux. Cross-compilation proves build compatibility, not runtime behavior on platforms that were not exercised.
- Report the exact smoke scenarios and platform exercised. Preserve user-owned configuration and panes; clean up only verification artifacts created for this work.

## Out of Scope

- Backward-compatible plugin IDs, pane entrypoints, executable names, subcommand aliases, re-exports, or compatibility shims.
- Automatic migration of existing plugin configuration, registry files, installed plugin links, or Shortcut Bindings.
- README migration instructions, manual cleanup instructions, or troubleshooting steps for the old development installation. The maintainer explicitly does not want them.
- Changing the Go module identity, upgrading dependencies or Go, adding frameworks, or introducing public reusable packages.
- New provider support, new Management API endpoints, quota mutations, version updates, refresh scheduling changes, or UI redesign.
- New handling of cross-tab pane moves, automatic layout repair, or changes to the established split/toggle behavior.
- Broad unrelated code cleanup, historical specification rewrites, new glossary terms, or an architectural decision record for this reversible file organization change.
- Creating implementation tickets or implementing the refactor during specification publication.

## Further Notes

- The maintainer approved the four-package structure and complete code/tests/manifest/README naming synchronization during the design interview.
- The maintainer reports having already removed the old keybindings and unlinked the plugin. Treat that as the starting development state; do not reintroduce migration work or ask for the same cleanup again.
- The manifest changes are maintainer-authored and uncommitted. Preserve them and build on their chosen names; do not restore the previous manifest as part of the refactor.
- The command and package names above express the agreed ownership and entrypoint structure. Exact exported signatures should be the smallest ones that preserve the existing behavior without import cycles; this specification does not prescribe a new abstraction layer.
- No glossary addition is needed: CPA Endpoint, Quota Snapshot, CPA Version, Herdr Tab, Quota View, and Shortcut Binding retain their existing meanings. The package move is reversible and does not warrant a new ADR.
