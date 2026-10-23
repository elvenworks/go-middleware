# go-middleware

A collection of middlewares for the [Gin](https://github.com/gin-gonic/gin) web framework used
in Elvenworks applications. Requires Go 1.20 or newer.

## Installation

```sh
go get github.com/elvenworks/go-middleware
```

```go
import middleware "github.com/elvenworks/go-middleware"
```

## Prometheus

```go
p := middleware.NewPrometheus("gin")
p.Use(router) // registers the middleware and GET /metrics
```

Exposes `gin_requests_total{code,method,handler,host,url}`,
`gin_request_duration_seconds{code,method,url}`, `gin_request_size_bytes` and
`gin_response_size_bytes`.

The `url` label is the **route template** (`/alerts/:id`), and `unmatched` for requests that
match no route. This keeps the number of time series bounded. Do not switch to raw paths on
routers with path parameters: every distinct path becomes a series kept in memory forever.

```go
// Legacy behavior (v1.1.x), only for routers without path parameters.
p.ReqCntURLLabelMappingFn = middleware.RawPathURLLabel
```

## Logger

```go
skipPaths := []string{"/docs", "/healthz", "/metrics"}
middleware.NewLogger(skipPaths, logrus.InfoLevel).Use(router)
```

Logs one entry per request with method, path, status, latency and client IP. Responses with
status above 299 are logged as errors. The level is applied to the standard logrus logger when
`Use` is called.

## APM labels (deprecated)

`SetLabelTransactionAPM` does nothing since v1.3.0, when the library dropped its Elastic APM
dependency. It only exists so v1 callers keep compiling and will be removed in the next major
version. Put identifiers in structured log fields or OpenTelemetry span attributes instead.

## Auth JWT (deprecated)

`NewAuthJWT()` decodes Bearer token claims into the `consumer` context key **without verifying
the signature**. Only use it behind a component that already authenticated the request. It will
be removed in a future major version.

## Development

See [AGENTS.md](AGENTS.md) for rules, commands and the release process, and
[CHANGELOG.md](CHANGELOG.md) for changes.
