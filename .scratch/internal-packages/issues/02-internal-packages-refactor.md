# 02: Refactor into Command Entrypoint and Internal Packages

**What to build:** Move the root application Go files and tests into `cmd/cpa-quotas/main.go` and four dedicated internal packages (`internal/cpa`, `internal/config`, `internal/herdr`, `internal/ui`). Update manifest build targets and README build instructions to target `./cmd/cpa-quotas`. Keep package interfaces tight and acyclic, relocate tests and test fixtures alongside their owning packages, ensure no application Go files remain in root, and verify end-to-end functionality with existing tests, cross-compilation, and a real Herdr smoke test.

**Blocked by:** 01: Synchronize Manifest, Shortcut, and Documentation Naming.

**Status:** ready-for-agent

- [ ] Entrypoint lives at `cmd/cpa-quotas/main.go` with only argument dispatch and program execution
- [ ] Management API access, quota models, and version checks live in `internal/cpa`
- [ ] Configuration defaults, validation, and storage live in `internal/config`
- [ ] Pane launching, focus policy, toggle/dedup registry, locking, and shortcut installer live in `internal/herdr`
- [ ] Bubble Tea model, rendering, and keyboard interactions live in `internal/ui`
- [ ] Manifest `[[build]]` commands target `./cmd/cpa-quotas`
- [ ] README build instructions target `./cmd/cpa-quotas`
- [ ] No application Go source or test files remain in the repository root
- [ ] Full test suite passes across all internal packages and code cross-compiles cleanly
- [ ] Real Herdr smoke test verifies built binary and actions function identically
