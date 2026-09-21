# AtomicURL Maintainer Review

## Product decision

AtomicURL is an undoable permanent short-link service: a user shares one code,
changes its destination when needed, sees the destination history, and rolls
back without changing the public URL.

## Completed implementation

- Added `Link` and `LinkVersion` models with unique codes, ownership indexes,
  version indexes, and GORM relationships.
- Added validated HTTP(S) destinations and transactional `POST /links`
  creation with an initial version.
- Added transactional destination updates, append-only history, and rollback.
- Added ownership-scoped authentication helpers and safe API errors.
- Updated `GET /{code}` to use permanent links and atomic click increments.
- Preserved a narrow legacy `Url` redirect fallback for existing published
  rows.
- Replaced the dashboard with owned permanent-link responses and version counts.
- Removed runtime expiry behavior, the remake route, and the obsolete tracked
  dashboard artifact.
- Added startup configuration validation, one migration path, database cleanup,
  HTTP timeouts, and graceful shutdown.
- Added unit, handler, concurrency, race, and end-to-end lifecycle coverage.

## Current API surface

| Method | Path | Purpose |
| --- | --- | --- |
| `POST` | `/register` | Create an account |
| `POST` | `/login` | Start a cookie session |
| `POST` | `/links` | Create a permanent versioned link |
| `PATCH` | `/links/{id}` | Update a destination and create a version |
| `GET` | `/links/{id}/history` | Read owned destination history |
| `POST` | `/links/{id}/versions/{versionID}/rollback` | Append a rollback version |
| `GET` | `/dashboard` | Read owned link summaries |
| `GET` | `/{code}` | Public redirect |
| `POST` | `/create` | Legacy compatibility route |

## Remaining risks and limitations

1. `AutoMigrate` creates the new schema but does not convert old `Url` rows into
   `Link` and `LinkVersion` records. A reviewed, resumable data migration is
   still required before removing the legacy model and `/create` route.
2. The legacy `Url.ExpirationTime` column remains in the database for
   compatibility, but runtime code no longer reads or writes it.
3. Configuration still accepts generic legacy environment names. Deployments
   should use the `ATOMICURL_*` names documented in the README.
4. The application still lacks CSRF protection for browser-based state-changing
   requests, TLS termination, rate limiting on every new write route, and
   production observability.
5. PostgreSQL deployment, backup/restore, and migration rehearsal are not
   automated in this repository.

## Issues to open

### Issue: Production database connections disable TLS

**Severity:** High

`ConnectToDatabaseWithConfig` builds the PostgreSQL DSN with
`sslmode=disable` (`db.go`). This means database credentials and application
data are sent without transport encryption whenever the database is remote.
The current configuration has no way to select an SSL mode, CA certificate, or
client certificate, so deploying the service outside a trusted local network
creates an avoidable credential and data-exposure risk.

**Acceptance criteria:**

- Production configuration defaults to TLS (at least `sslmode=require`).
- The SSL mode and certificate options are configurable through documented
  `ATOMICURL_*` settings.
- Local development can still use an explicit, documented non-TLS setting.
- Connection-string construction and startup validation have tests covering
  both production and local-development configurations.

### Completed: Refactor the flat root package into a maintainable folder structure

**Severity:** Medium

**Status:** Completed. The application now follows a `backend/` module layout
with focused `internal/api`, `internal/config`, `internal/db`,
`internal/models`, `internal/routes`, and `internal/utils` packages.

Before this refactor, every production file and test lived in the repository
root and used the single `main` package. The root mixed server startup,
database access, authentication, HTTP handlers, legacy `/create` behavior,
the versioned link API, models, and their tests.

**Acceptance criteria:**

- Keep the executable entry point in `backend/main.go` and route registration
  in `backend/internal/routes`.
- Keep API handlers, configuration, database, models, and utilities in their
  focused `backend/internal` packages.
- Keep legacy compatibility code isolated in the API package's `legacy.go`.
- Keep package-specific tests beside the package they exercise.
- Preserve the existing API behavior during future structural changes.

## Recommended next milestone

Build and rehearse the explicit legacy-data migration in a copy of production
data. It should validate malformed or duplicate legacy codes, preserve all
published codes where possible, create initial versions, record skipped rows,
and support rollback or restart. Only after that migration is verified should
the legacy `Url` model and `/create` compatibility route be removed.

## Verification

The maintainer checks are:

```sh
cd backend
gofmt -w $(find . -name '*.go')
go test ./...
go test -race ./...
go vet ./...
```

The test suite uses SQLite and does not require external network calls.
