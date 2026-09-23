# AtomicURL CLI

This is the first CLI layer for AtomicURL. It is intentionally a separate Go
module and uses only the Go standard library. The CLI talks to the existing
HTTP API; it does not connect directly to PostgreSQL.

## Build and run

From this directory:

```sh
go build -o atomicurl .
./atomicurl help
```

The server URL defaults to `http://localhost:3000`. Set `ATOMICURL_URL` or
pass `--base-url`. Login sessions are stored in
`$XDG_CONFIG_HOME/atomicurl/session.json` (or the platform's standard config
directory) with restrictive file permissions. Use `--config` to select a
different path.

## First workflow

```sh
./atomicurl register \
  --username denzil \
  --email denzil@example.com \
  --password correct-horse

./atomicurl login --username denzil --password correct-horse

./atomicurl create \
  --destination https://example.com/demo \
  --title "Demo link" \
  --code demo
```

`create` prints the public short URL. The password can also be supplied with
`ATOMICURL_PASSWORD`; command-line passwords may be visible in shell history
or process listings, so an interactive prompt can be added in a later pass.

Run tests with:

```sh
go test ./...
```
