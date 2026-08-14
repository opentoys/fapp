# Backend — Go API & static file server

Go 1.22+ standard library HTTP with `mux.HandleFunc("METHOD /path")` routing.
GORM + SQLite, JWT auth (golang-jwt/jwt/v5).

## Commands

```bash
# Build (from backend/)
go build -o bin/disapp ./cmd/server

# Run with config
APP_CONFIG=../config.json go run ./cmd/server
APP_CONFIG=../config.json ./bin/disapp

# Test
go test ./...

# Wipe local DB
cd .. && make reset
```

## Architecture

```
cmd/server/main.go     Entry point, wires config/DB/storage/server
internal/server/        HTTP handlers, routes, middleware, auth
internal/auth/          JWT create/parse
internal/config/        config.json loader
internal/db/            GORM SQLite setup
internal/model/         DB models (User, App, Version, Channel, etc.)
internal/password/      sha256 hash/salt
internal/web/           JSON response helpers, middleware (recoverer, logger, rate limit)
internal/storage/       File storage abstraction (local dir)
static/                 embed.FS root — frontend dist is copied here by `make build`
```

## Key conventions

- No migration code — schema auto-migrated by GORM on startup
- Super-admin is authenticated from `config.json` admin block, uid=-1 in JWT
- `internal/` packages never import `cmd/`
- Backend error messages are NOT translated; only UI strings are