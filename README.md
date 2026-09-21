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
go run .
```

The older names `HOST`, `PORT`, `USER`, `PASSWORD`, `DBNAME`, `SECRETKEY`, and
`APP_ENV` are still accepted for compatibility. `ATOMICURL_HTTP_PORT` defaults
to `3000`; `PORT` is treated as the database port when using legacy names.

## API

Register and log in to receive the session cookie:

```sh
curl -X POST http://localhost:3000/register \
  -d 'username=denzil&email=denzil@example.com&password=correct-horse'

curl -c cookies.txt -X POST http://localhost:3000/login \
  -d 'username=denzil&password=correct-horse'
```

Create a permanent link:

```sh
curl -b cookies.txt -X POST http://localhost:3000/links \
  -H 'Content-Type: application/json' \
  -d '{"title":"Portfolio API","destination":"https://example.com/demo","code":"portfolio-api"}'
```

Update its destination, inspect history, and roll back a version:

```sh
curl -b cookies.txt -X PATCH http://localhost:3000/links/1 \
  -H 'Content-Type: application/json' \
  -d '{"destination":"https://example.com/new-demo","note":"Moved the demo"}'

curl -b cookies.txt http://localhost:3000/links/1/history

curl -b cookies.txt -X POST \
  http://localhost:3000/links/1/versions/1/rollback \
  -H 'Content-Type: application/json' \
  -d '{"note":"Restored the working deployment"}'
```

The public URL remains `GET /{code}` throughout the workflow. The authenticated
`GET /dashboard` endpoint returns owned links, current destinations, click
counts, timestamps, and version counts. The legacy `POST /create` endpoint is
still available for compatibility but is not the versioned link API.

## Development checks

```sh
gofmt -w *.go
go test ./...
go test -race ./...
go vet ./...
```

Tests use SQLite in memory and do not make external network requests.
