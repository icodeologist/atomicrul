# AtomicURL

> One link you can safely change after you have shared it.

AtomicURL is being rebuilt as a small link-management product for people who
regularly share work that moves: portfolio projects, resumes, demos, event
pages, menus, and documents.

The core promise is simple:

1. Share one permanent short link or QR code.
2. Change its destination later without sending a new link.
3. See every previous destination and roll back a bad change.

This makes AtomicURL more than another URL shortener. It is an undo button for
links that have already left your hands.

## Project status

This repository currently contains an early Go API. It supports accounts,
cookie-based login, permanent versioned links, authenticated destination
updates, history, rollback, redirects, click counts, and a JSON dashboard. It
does not have a frontend or a production setup yet.

New `Link` records do not expire. Legacy `Url` rows remain in the database and
are still redirectable during migration; their historical `ExpirationTime`
column is retained but is no longer read or written by the runtime.

See [PROJECT_REVIEW.md](PROJECT_REVIEW.md) for the full codebase inventory,
known issues, product decision, and build order.

## Technology

- Go HTTP server using Gorilla Mux
- PostgreSQL with GORM
- Cookie sessions and bcrypt password hashing
- SQLite-backed automated tests

## Current API

| Method | Path | Purpose |
| --- | --- | --- |
| `POST` | `/register` | Create an account |
| `POST` | `/login` | Log in and receive a session cookie |
| `POST` | `/links` | Create a permanent versioned link (login required) |
| `PATCH` | `/links/{id}` | Update a destination and create a version (login required) |
| `GET` | `/links/{id}/history` | View destination history (login required) |
| `POST` | `/links/{id}/versions/{versionID}/rollback` | Roll back to a previous destination (login required) |
| `POST` | `/create` | Legacy short-link creation route kept for compatibility |
| `GET` | `/dashboard` | List the current user's links |
| `GET` | `/{code}` | Redirect to a link's destination |

The API is under active reconstruction. The route shapes and response formats
will change as the product is cleaned up.

## Tests

The project uses a temporary SQLite database for handler tests:

```sh
go test ./...
go test -race ./...
go vet ./...
```

Local PostgreSQL setup instructions will be added when configuration cleanup is
complete.
