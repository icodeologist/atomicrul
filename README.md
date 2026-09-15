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
cookie-based login, short-link creation, redirects, click counts, temporary
link expiry, and a JSON dashboard. It does not have a frontend or a production
setup yet, and several parts of the API need hardening before new product work.

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
| `POST` | `/create` | Create a short link (login required) |
| `GET` | `/dashboard` | List the current user's links |
| `GET` | `/remake_links` | Extend expired links (temporary legacy behavior) |
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
