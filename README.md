# AtomicURL

AtomicURL is a small link-management API for permanent short links. A user can
create one public code, change its destination later, inspect destination
history, and roll back a bad change without changing the public URL.

New links do not expire. The legacy `Url` model and `/create` route remain only
to keep existing data and clients working during migration; legacy rows are
redirectable, and their historical `ExpirationTime` column is no longer used.

## Local setup

Requirements:

- Go 1.24 or newer
- PostgreSQL for the application database

The Go module and executable are in `backend/`, following the same layout as
the Disaster-Watcher project.

Create a database, then export the preferred configuration variables:

```sh
export ATOMICURL_DB_HOST=localhost
export ATOMICURL_DB_PORT=5432
export ATOMICURL_DB_USER=postgres
export ATOMICURL_DB_PASSWORD=your-password
export ATOMICURL_DB_NAME=atomicurl
export ATOMICURL_SECRET_KEY='use-at-least-32-random-characters-here'
export ATOMICURL_APP_ENV=development
export ATOMICURL_HTTP_PORT=3000
cd backend
go run .
```

To load safe, repeatable demo data in a non-production database, enable the
opt-in seed before starting the app:

```sh
export ATOMICURL_SEED_DEMO=true
cd backend
go run .
```

Then log in with `demo` / `demo-password` and open `/dashboard`. The seed
creates three links, click counts, one inactive link, and version history. It
is refused when `ATOMICURL_APP_ENV=production`.

The older names `HOST`, `PORT`, `USER`, `PASSWORD`, `DBNAME`, `SECRETKEY`, and
`APP_ENV` are still accepted for compatibility. `ATOMICURL_HTTP_PORT` defaults
to `3000`; `PORT` is treated as the database port when using legacy names.

## CLI

The recommended way to use AtomicURL from a terminal is the Go CLI in `cli/`.
It handles authentication cookies locally and currently supports registration,
login/logout, and permanent link creation:

```sh
cd cli
go run . register \
  --username denzil \
  --email denzil@example.com \
  --password correct-horse

go run . login --username denzil --password correct-horse
go run . create \
  --destination https://example.com/demo \
  --title "Portfolio API" \
  --code portfolio-api
```

The server defaults to `http://localhost:3000`. Use `--base-url` or
`ATOMICURL_URL` for another server. Use `ATOMICURL_PASSWORD` instead of the
password flag when you do not want the password in shell history.

## HTTP API

The backend also exposes the HTTP API for other clients. Authenticated
requests use the session cookie returned by `POST /login`:

- `POST /register` — create an account
- `POST /login` and `POST /logout` — manage the session
- `POST /links` — create a permanent link
- `PATCH /links/{id}` — update a link destination
- `GET /links/{id}/history` — inspect destination history
- `POST /links/{id}/versions/{versionID}/rollback` — restore a version
- `GET /dashboard` — list the authenticated user’s links
- `GET /{code}` — redirect through a public short code

The legacy `POST /create` endpoint remains available for compatibility but is
not the versioned link API. Open `GET /dashboard` in a browser for the HTML
dashboard; API clients receive the same dashboard data as JSON.

## Development checks

```sh
cd backend
gofmt -w $(find . -name '*.go')
go test ./...
go test -race ./...
go vet ./...
```

Tests use SQLite in memory and do not make external network requests.
