# How We Did It

- Added permanent `Link` and `LinkVersion` GORM models with user/link relationships.
- Added indexes for link codes, user ownership, and version link ownership.
- Kept the legacy `Url` model and handlers unchanged.
- Added model relationship and uniqueness tests.
- Centralized migrations and stopped ignoring migration errors.
- Verified with `gofmt`, `go test ./...`, and `git diff --check`.

Commits:

- `1c4383c feat: add permanent versioned link models`
- `1b8eb4c fix: propagate database migration errors`

Remaining: existing `Url` data still needs an explicit data migration; GORM `AutoMigrate` does not convert it automatically.
