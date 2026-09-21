# How We Did It

- Added permanent `Link` and `LinkVersion` GORM models with user/link relationships.
- Added indexes for link codes, user ownership, and version link ownership.
- Kept the legacy `Url` model and handlers unchanged.
- Added model relationship and uniqueness tests.
- Centralized migrations and stopped ignoring migration errors.
- Added a reusable URL validator for HTTP(S) destinations with normalization, host checks, credential rejection, and a 2,048-character limit.
- Added table-driven URL validation tests; link creation is not wired to the helper yet.
- Added authenticated `POST /links` for permanent link creation with transactional initial versions, custom/generated codes, duplicate handling, and stable JSON responses.
- Kept legacy `POST /create` unchanged for compatibility.
- Added endpoint tests for authentication, validation, duplicates, generated codes, initial versions, and transaction rollback.
- Updated `GET /{code}` to redirect active permanent links by `Link.Code`, return `404`/`410` for missing or inactive links, and increment clicks atomically.
- Kept a narrow legacy `Url` fallback so existing published short codes continue working; only that fallback retains expiration behavior until data migration.
- Added redirect tests for active, missing, inactive, invalid, legacy, click, and concurrent-click behavior.
- Added authenticated `PATCH /links/{id}` for ownership-scoped destination updates with required notes, transactional version history, and unchanged public codes.
- Same-destination updates are rejected with `400` and do not create a duplicate version.
- Added update tests for authorization, ownership isolation, validation, rollback, history preservation, and code stability.
- Added authenticated `GET /links/{id}/history` with ownership checks, newest-first versions, current-version marking, and empty-list responses.
- Added history tests for authorization, ownership isolation, ordering, active-version identification, missing links, and empty histories.
- Added authenticated `POST /links/{id}/versions/{versionID}/rollback` with ownership and same-link version checks, append-only history, and transactional destination updates.
- Added rollback tests for first-version restoration, new-version creation, preserved history, invalid combinations, authorization, notes, and public redirects.
- Added reusable authentication, ownership, and API-error helpers; new link handlers now share safe session/user-ID and ownership logic.
- Added helper tests for missing/unauthenticated sessions, malformed user IDs, ownership isolation, missing links, and stable error JSON.
- Replaced the legacy dashboard response with owned permanent links, current destinations, active state, click counts, timestamps, short URLs, and version counts sorted by recent updates.
- Added dashboard tests for ownership filtering, inactive links, version counts, sorting, unauthenticated access, and empty results.
- Removed runtime expiry assignment/checks, the `/remake_links` route and implementation, and the obsolete tracked dashboard artifact.
- Kept `Url.ExpirationTime` only as a legacy schema field; existing `Url` rows are preserved and remain redirectable through the compatibility fallback.
- Updated README and project review documentation for permanent-link behavior and the non-destructive migration transition.
- Added validated `ATOMICURL_*` configuration with legacy environment fallbacks, a default HTTP port, and startup checks for required values and valid ports.
- Centralized database connection/migration startup so migration runs once, added cleanup, and added HTTP read/write/idle/header timeouts with graceful shutdown.
- Verified with `gofmt`, `git diff --check`, `go test ./...`, `go test -race ./...`, and `go vet ./...`.
- Added the defining end-to-end lifecycle test: create destination A, redirect, update to B, redirect, inspect history, roll back to A, redirect again, and verify immutable code plus click counts.
- Reorganized the application into a `backend` Go module matching the Disaster-Watcher layout, with focused `internal/api`, `internal/config`, `internal/db`, `internal/models`, `internal/routes`, and `internal/utils` packages.

## Current limitations

- Existing `Url` rows still need an explicit data migration into `Link` and `LinkVersion` records.
- The legacy `Url` model and `/create` route remain temporarily for compatibility.
- PostgreSQL startup, real migration rehearsal, and graceful shutdown still need manual environment verification.
- CSRF protection, production observability, and broader write-route rate limiting are not yet implemented.

Commits:

- `1c4383c feat: add permanent versioned link models`
- `1b8eb4c fix: propagate database migration errors`

Remaining: existing `Url` data still needs an explicit data migration; GORM `AutoMigrate` does not convert it automatically.
