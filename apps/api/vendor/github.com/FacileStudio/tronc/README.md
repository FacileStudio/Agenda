# tronc

The Go app chassis for the [Facile Studio](https://facile.studio) suite — the HTTP plumbing
every Facile API needs and none of them should be re-deciding.

Twelve Go services once carried their own copy of this code under `apps/api/internal/`. The
copies had drifted: five variants of the error envelope, four of the request logger, five of
the CORS middleware. `tronc` is the single version, tagged and versioned; ten repos run on it
today.

## What it does

- Assembles the standard chi middleware chain — request ID, panic recovery, real IP, CORS,
  request logging — in one call
- Carries the suite error envelope, `{"error":{"code","message"}}`, and its one code-to-status map
- Decodes JSON and gzipped JSON request bodies under explicit size caps, rejecting unknown fields
- Logs one structured record per request with credential-bearing query parameters redacted
- Serves `/health` and `/ready` at both the root and `/api`, plus an argv healthcheck hook for
  distroless containers
- Serves a built SPA from the same binary, with a history fallback that refuses to answer writes
- Reads the configuration every Facile API shares, plus typed helpers for the rest
- Turns a route registry into OpenAPI 3.1 behind a Scalar reference at `/docs`, and reports
  routes the registry forgot
- Applies ordered database migrations from inside the binary, via the separate
  [`migrate`](migrate/) module — so the dependency lands only on apps that have a database
- Gives tests a real PostgreSQL, one schema per test binary, via the separate
  [`testdb`](testdb/) module

## Stack

| Layer | Tech |
|---|---|
| Runtime | Go 1.24, `github.com/go-chi/chi/v5` v5.2.3, and nothing else — deliberately |

## Install

```sh
go get github.com/FacileStudio/tronc
```

```go
package main

import (
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"github.com/FacileStudio/tronc/env"
	"github.com/FacileStudio/tronc/health"
	"github.com/FacileStudio/tronc/healthcheck"
	"github.com/FacileStudio/tronc/httpx"
	"github.com/FacileStudio/tronc/logger"
	"github.com/FacileStudio/tronc/middleware"
)

func main() {
	if healthcheck.Handle(os.Args) {
		return
	}

	cfg, err := env.LoadCore()
	if err != nil {
		slog.Error("config", slog.Any("error", err))
		os.Exit(1)
	}

	log := logger.New(logger.Config{Level: cfg.LogLevel})

	router := httpx.NewRouter(httpx.Config{
		Logger: log,
		CORS: middleware.CORSConfig{
			AllowedOrigins:   cfg.CORSAllowedOrigins,
			AllowCredentials: true,
		},
	})
	health.Mount(router, health.DB(db))

	// ... your routes

	_ = http.ListenAndServe(":"+strconv.Itoa(cfg.Port), router)
}
```

Consumers conventionally import the configuration package as `troncenv` and embed
`troncenv.Core` in their own config struct.

## Configuration

`env.LoadCore` reads what every app shares. `DATABASE_URL` is required and a missing value
returns an error, which the caller above turns into `os.Exit(1)`.

| Variable | What it does |
|---|---|
| `DATABASE_URL` | Postgres connection string; required by `LoadCore` |
| `PORT` | HTTP listen port, default `8080` |
| `LOG_LEVEL` | `debug` / `info` / `warn` / `error`, default `info` |
| `APP_ENV` | `development` / `staging` / `production`; never gates security behavior |
| `CORS_ALLOWED_ORIGINS` | Comma-separated allowed origins; unset denies every cross-origin caller |

Full reference, including the six fallback names for the origin list and the traps in adopting
`LoadCore`: [docs/configuration.md](docs/configuration.md).

## Structure

```
errors/       the error envelope and its code-to-status map
httpjson/     bounded, strict JSON decoding; WriteJSON and WriteError
logger/       slog JSON on stdout, with a Wrap seam for log shipping
middleware/   RequestLogger, CORS, Recoverer
httpx/        NewRouter and Chain — the standard middleware stack
health/       /health and /ready, mounted at the root and under /api
healthcheck/  an argv hook so a distroless container can probe itself
spa/          serves a built single-page client from the same binary
env/          the shared configuration plus typed variable helpers
apiref/       the route registry, its OpenAPI 3.1 rendering, and the Scalar page
migrate/      SEPARATE MODULE — goose migrations and a `migrate` subcommand
testdb/       SEPARATE MODULE — a real Postgres for tests, one schema per binary
docs/         architecture, configuration, development, API reference
```

`migrate/` and `testdb/` each carry their own `go.mod`, tags and dependencies. Nothing above
them changes: importing `tronc` still pulls chi and nothing else. See
[migrate/README.md](migrate/) and [testdb/README.md](testdb/).

## Documentation

| Doc | What's in it |
|---|---|
| [Architecture](docs/architecture.md) | Middleware order, request lifecycle, the defects this replaces |
| [Configuration](docs/configuration.md) | Every environment variable the code reads, and its traps |
| [Development](docs/development.md) | Local setup, the quality gate, CI, versioning |
| [API](docs/api.md) | Every exported symbol, package by package |

---

Part of the [Facile Suite](https://facile.studio) — self-hosted tools for creative studios
and freelancers. One login, zero cloud dependency.
