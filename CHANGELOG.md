# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] - 2026-10-20

### Breaking Changes
- Migrated all management endpoints from `/v0/management/...` to `/v8/management/...`. Requires CLIProxyAPI version 8 or newer; CLIProxyAPI v0.x endpoints are no longer queried.

### Changed
- Updated CLIProxyAPI Management API endpoints to the v8 contract:
  - Account credentials: `GET /v8/management/credentials` (previously `GET /v0/management/auth-files`).
  - Upstream request execution: `POST /v8/management/requests/api-call` (previously `POST /v0/management/api-call`).
  - Version checking: `GET /v8/management/server/latest-version` (previously `GET /v0/management/latest-version`).
- Updated Antigravity upstream request `User-Agent` header to `antigravity/hub/2.23.0 darwin/arm64` (previously `antigravity/cli/1.0.13 (aidev_client; os_type=darwin; arch=arm64)`).

### Fixed
- Fixed missing Gemini quota windows (`— reset —`) for Antigravity accounts where the upstream server previously omitted the `Gemini Models` family group when queried with the older client User-Agent.

## [0.1.4] - 2026-10-20

### Changed
- Preserved timestamps and labels dynamically when Herdr panes are resized.
