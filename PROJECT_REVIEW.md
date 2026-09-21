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

## Recommended next milestone

Build and rehearse the explicit legacy-data migration in a copy of production
data. It should validate malformed or duplicate legacy codes, preserve all
published codes where possible, create initial versions, record skipped rows,
and support rollback or restart. Only after that migration is verified should
the legacy `Url` model and `/create` compatibility route be removed.

## Verification

The maintainer checks are:

```sh
gofmt -w *.go
go test ./...
go test -race ./...
go vet ./...
```

The test suite uses SQLite and does not require external network calls.
