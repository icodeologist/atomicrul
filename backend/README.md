# AtomicURL backend

The executable and Go module live in this directory. Application code is
organized under `internal/`:

- `internal/api` — HTTP handlers, authentication, dashboard, and API tests
- `internal/config` — environment configuration and validation
- `internal/db` — PostgreSQL connection, migrations, and demo seeding
- `internal/models` — GORM models and model tests
- `internal/routes` — HTTP route registration
- `internal/utils` — reusable URL, short-code, and rate-limiting utilities

Run locally from this directory:

```sh
set -a
source .env
set +a
go run .
```

Use `.env.example` as the configuration template. Set
`ATOMICURL_SEED_DEMO=true` for the local demo account and sample links.
