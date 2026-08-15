# Backend — Go API & static file server

Go 1.22+ standard library HTTP with `mux.HandleFunc("METHOD /path")` routing.
GORM + SQLite, JWT auth (golang-jwt/jwt/v5).

## Commands

```bash
# Build (from backend/)
go build -o bin/disapp .

# Run with config
APP_CONFIG=../config.json go run .
APP_CONFIG=../config.json ./bin/disapp

# Test
go test ./...

# Wipe local DB
cd .. && make reset
```

## Architecture

```
main.go               Entry point, wires config/DB/storage/service
internal/controller/  Thin HTTP handlers: parse request → call service → write JSON
internal/service/     Business logic: DB queries, storage reads/writes, validation
internal/router/      Routes (mux.HandleFunc) + SPA static handler
internal/resources/   config loader · store/{db,model} · storage/{local,cos}
pkg/web/              JSON response helpers + middleware (recoverer, logger, rate limit)
pkg/token/            JWT create/parse
pkg/pwd/              sha256 hash/salt
static/               embed.FS root — frontend dist is copied here by `make build`
```

## Key conventions

- No migration code — schema auto-migrated by GORM on startup
- Super-admin is authenticated from `config.json` admin block, uid=-1 in JWT
- `internal/` packages never import the root `main` package
- Backend error messages are NOT translated; only UI strings are