# AtomicURL project recovery review

Reviewed on 2026-09-15.

## The product decision

Build **a link that can be corrected after it is shared**.

The annoying problem is not making a long URL shorter. It is discovering that
a link in a resume, printed QR code, old message, social bio, or client email is
wrong or outdated after people already have it.

AtomicURL should let someone keep one memorable link, update where it goes, see
the complete destination history, and undo a mistaken update.

Suggested product line:

> AtomicURL — an undo button for links you already shared.

### The differentiator

Editing a destination alone exists in established link products, so it is not a
credible differentiator by itself. AtomicURL's focused feature should be
**versioned destinations with one-click rollback**:

- every destination change is recorded automatically;
- the owner can see what changed, when, and why;
- a bad change can be rolled back without changing the public short link;
- existing links and printed QR codes continue to work.

That gives the portfolio project a clear story and demonstrates useful product
thinking without requiring unusually deep technology.

### First audience

Start with freelancers, job seekers, and developers sharing portfolios and live
demos. They are easy to understand, easy to design a demo for, and fit the
portfolio context of this project itself.

Example: `atomicurl.dev/denzil-api` can be placed in a resume today. If the demo
moves from Render to Fly.io later, the owner changes the destination. If the new
deployment is broken, they roll back to the previous version.

### MVP experience

1. A visitor sees a focused landing page and a working demo.
2. A user creates a named link such as `/portfolio-api`.
3. The link remains active until its owner deliberately pauses or deletes it.
4. The owner changes the destination and adds a short note such as "moved demo
   to new host."
5. A history page shows all versions and offers rollback.
6. The dashboard shows total clicks and recent activity.

Do not add teams, billing, AI suggestions, geographic charts, browser
extensions, or dozens of link options to the first version. They weaken the
story.

## What exists now

This is a single-package Go application with about a dozen source files.

### Main pieces

| Area | Files | Current behavior |
| --- | --- | --- |
| Startup and routes | `main.go` | Connects to PostgreSQL, migrates tables, and starts on port 3000 |
| Accounts | `auth.go` | Register, login, logout, and cookie sessions |
| Link API | `api.go`, `link_*.go` | Creates, updates, versions, rolls back, and redirects permanent links |
| Dashboard | `dashboard.go` | Returns a user's links as JSON |
| Database | `db.go`, `models.go` | GORM models for users, legacy URLs, and permanent links |
| Short codes | `shortner.go` | Converts database IDs to and from base 62 |
| Traffic control | `ratelimiter.go` | Per-IP in-memory create-route limiter |
| Tests | `url_handler_test.go`, `ratelimiter_test.go` | Basic create, redirect, and limiter coverage |

### Data currently stored

Users have a username, email, password hash, timestamps, and associated links.
Permanent links have a title, unique code, current destination, active state,
owner, click count, timestamps, and append-only destination versions. Legacy
`Url` rows remain stored for compatibility; their old expiration column is
retained but no longer drives runtime behavior.

### Runtime configuration currently expected

The app reads `HOST`, `PORT`, `USER`, `PASSWORD`, `DBNAME`, and `SECRETKEY` from
the environment. There is no checked-in example configuration, local database
recipe, or startup validation yet.

### Repository condition

- The README was empty before this review.
- There is no frontend.
- There are no deployment files or continuous-integration checks.
- A compiled 17 MB development binary is committed under `tmp/runner-build`.
- The obsolete tracked dashboard source file has been removed.
- The latest commit says "Refactored and cleaned," but it removed the only HTML
  templates and left the project as an API-only prototype.

## Confirmed problems, in repair order

### 1. Broken rate limiting — fixed

The limiter returned a 429 response but then continued into the protected
handler. Its global map was also unsafe when requests arrived concurrently.

It now stops rejected requests, protects shared state with a mutex, and has
tests for both rejection and concurrent access. `go test`, `go test -race`, and
`go vet` pass.

### 2. Account and session safety — fixed

Registration no longer returns the stored user or password hash. Failed login
uses a generic message and the correct unauthorized status. Session errors are
handled, the app refuses to start with a short or missing `SECRETKEY`, cookies
are HTTP-only and same-site protected (and secure in production), and routes
now restrict their HTTP methods. Logout also deletes the session instead of
leaving user data in its cookie.

Dedicated tests cover secret validation, password-data leakage, and failed
login responses. A CSRF token should still be added when browser forms are
built.

### 3. Permanent link flow — implemented

The new `Link` model validates HTTP(S) destinations, creates an initial
`LinkVersion` transactionally, supports destination updates and rollback, and
redirects with atomic click counting. New links do not expire. The old `Url`
model and `/create` route remain only for non-destructive compatibility while
existing rows are migrated explicitly.

### 5. Database and configuration reliability

- `USER` and `PORT` are overly generic names and can collide with normal shell
  environment variables.
- Schema migration happens twice and one result is ignored.
- The server prints port 8000 but listens on port 3000.
- There are no server timeouts or graceful shutdown.
- Database errors are exposed directly to API clients.

### 6. API quality

- The legacy `/create` handler and `Url` model still need eventual removal after
  data migration.
- Debug prints and old planning comments remain in some legacy code.

### 7. Test gaps

The suite now covers registration, login, authorization, invalid input, missing
links, database failures, click concurrency, dashboard ownership, history, and
rollback behavior. Explicit migration tests for converting legacy `Url` rows
are still pending.

## Build sequence

Work through this in small, demonstrable slices:

1. **Make the existing API trustworthy.** Account/session responses are now
   repaired. Next, validate link input, make click counts safe, clean
   configuration, and test failure paths.
2. **Replace forced expiry with editable permanent links.** Add a custom alias,
   title, current destination, active state, and an authenticated update route.
3. **Add the signature feature.** Store a new destination revision on every
   update and implement rollback with a required note.
4. **Build one polished web interface.** Landing page, create/edit screen,
   dashboard, and history timeline. Keep it responsive and accessible.
5. **Make the demo easy to run and trust.** Example environment file, local
   PostgreSQL setup, seeded demo account, CI, deployment, and screenshots.
6. **Add useful proof, not vanity metrics.** Total clicks, last visit, and clicks
   per destination version are enough for the first release.

## Portfolio demo script

A reviewer should understand the project in under one minute:

1. Open a permanent short link and land on demo version A.
2. Change its destination in the dashboard with a note.
3. Open the exact same short link and land on version B.
4. Show the version timeline and roll back.
5. Refresh the short link and see version A again.

That is the "wow" moment. Everything we build should support it.
