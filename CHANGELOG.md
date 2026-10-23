# Changelog

All notable changes to this project are documented here. Versions follow semantic versioning.

## [v1.3.0] - 2026-10-23

### Removed

- The Elastic APM dependency (`go.elastic.co/apm/v2`). Consumers that do not use the agent
  themselves no longer pull it in.

### Deprecated

- `SetLabelTransactionAPM` is now a no-op kept for source compatibility. Put identifiers in
  structured log fields or OpenTelemetry span attributes instead. It will be removed in the next
  major version.

## [v1.2.0] - 2026-10-23

### Changed

- **Prometheus `url` label now uses the route template by default.** A request to
  `/alerts/42` on route `/alerts/:id` is recorded as `url="/alerts/:id"`, and requests that match
  no route (404, and 405 when `HandleMethodNotAllowed` is enabled) as `url="unmatched"`. Previously the raw path was used, which created one time series
  per distinct ID and let memory grow without bound in the application and in Prometheus. Set
  `p.ReqCntURLLabelMappingFn = middleware.RawPathURLLabel` to restore the old behavior on
  routers without path parameters.
- The access logger applies its level to the standard logrus logger once, in `Use`, instead of
  on every request. A level set by the application afterwards is no longer overridden.
- `SetLabelTransactionAPM` logs a warning without the object contents when there is no APM
  transaction (it previously logged the whole object as an error).
- Minimum Go version is 1.20 (required by gin v1.9.1).

### Fixed

- A second `NewPrometheus` with the same subsystem in one process now reuses the registered
  collectors; previously its metrics were silently never exported.
- `URLLabelFromContext` with a non-string context value no longer panics (records `unknown`).
- Push gateway: an unreachable metrics URL no longer panics the push goroutine.
- `NewAuthJWT` rejects with 401, instead of panicking, a `Bearer` header without a token and
  tokens that are not structurally valid JWTs (wrong number of segments or invalid header
  segment). Tokens accepted before keep being accepted with the same, possibly partial, claims.
- `SetLabelTransactionAPM` accepts pointers, ignores nil values and unsupported kinds instead of
  panicking, and records map values as plain strings.
- `NewPrometheus` no longer panics when an unrelated collector with the same metric name is
  already registered; it logs the error as before.

### Deprecated

- `NewAuthJWT`: it only decodes claims and never verified signatures. It will be removed in a
  future major version.

### Security

- Dependencies: gin v1.9.1, golang-jwt/jwt/v4 v4.5.2, logrus v1.9.3, golang.org/x/net v0.35.0,
  golang.org/x/text v0.22.0. The unmaintained `github.com/golang-jwt/jwt` v3 was removed.
  These are the newest versions compatible with Go 1.20; later `golang.org/x/net` and
  `golang.org/x/text` releases require Go 1.23 or newer, so applications on recent Go versions
  should keep them up to date in their own `go.mod`.
- Push gateway requests use a 10 s timeout and non-2xx responses are reported instead of pushed.

### Added

- Tests for all middlewares, CI (vet, race tests on Go 1.21.x and stable, tidy check),
  `AGENTS.md` and this changelog.

## Earlier releases

v1.0.0 to v1.1.3 (2023-04 to 2024-07): see the git tags and their commits.
