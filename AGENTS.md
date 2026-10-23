# AGENTS.md

Guidance for people and coding agents working on `github.com/elvenworks/go-middleware`.

## What this is

A small, single-package (`package middleware`) library of Gin middlewares. Changes affect every
consumer, so stability matters more than features.

| File | Purpose |
|---|---|
| `prometheus.go` | HTTP metrics (`<subsystem>_requests_total`, `_request_duration_seconds`, request/response size) and the `/metrics` endpoint; optional push gateway. |
| `logger.go` | Access log with logrus; responses with status above 299 are logged as errors. |
| `apm_labels.go` | **Deprecated.** `SetLabelTransactionAPM` is a no-op kept for v1 source compatibility. |
| `auth_jwt.go` | **Deprecated.** Decodes Bearer token claims without verifying them. |

## Commands

```sh
go vet ./...
go test -race -count=1 -cover ./...
go mod tidy -go=1.20   # never let tidy raise the go directive
```

CI (`.github/workflows/ci.yml`) runs vet and tests on Go 1.21.x and the current stable release,
and fails if `go.mod`/`go.sum` are not tidy.

## Rules

1. **Compatibility first.** Keep the exported API source-compatible within v1. Removing or
   changing an exported identifier requires a new major version (`/v2` module path).
2. **Keep the `go 1.20` floor** so the module stays installable on older toolchains with a plain
   `go get`. Before adding or bumping a dependency, check its `go` directive:
   `go list -m -json <module>@<version> | jq .GoVersion`. `go mod tidy` without `-go=1.20` may
   silently raise the floor through test-only dependencies.
3. **Bounded metric labels.** A label value must come from a small, fixed set (route template,
   method, status code). Never use raw paths, IDs, user input or headers as label values: every
   distinct value is a new time series kept in memory by the process and by Prometheus.
   Identifiers belong in logs and traces.
4. **No panics in request paths.** Middlewares must degrade (skip, log, reply 4xx) instead of
   panicking on unexpected input.
5. **Do not log request or object contents** that may carry customer data.
6. **Global state.** Metrics register in the default Prometheus registry and the logger uses the
   standard logrus logger. Tests must use a unique metrics subsystem per test and restore any
   global logger state they change.
7. **Tests.** Every new function gets a test; every bug fix gets a regression test. Tests use
   `httptest` and in-process fakes only, no network.
8. **Public repository.** Do not commit internal hostnames, service names, infrastructure details,
   incident reports or credentials, in code, comments, docs or commit messages.

## Releasing

Versions are git tags following semver. There is no release automation.

1. Merge to `main` with CI green and `CHANGELOG.md` updated.
2. Tag and push: `git tag vX.Y.Z && git push origin vX.Y.Z`.
3. Optionally create a GitHub Release from the tag with the changelog entry.
4. Consumers upgrade with `go get github.com/elvenworks/go-middleware@vX.Y.Z && go mod tidy`.

Use a minor version when default behavior changes, even if the Go API does not.
