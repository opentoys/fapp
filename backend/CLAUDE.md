# Backend — Go API & static file server

Go 1.22+ standard library HTTP with `mux.HandleFunc("METHOD /path")` routing.
GORM + SQLite, JWT auth (golang-jwt/jwt/v5).

## Commands

```bash
# Build (from backend/). Default: no embedded frontend dist.
go build -o bin/disapp .

# Build with the frontend dist embedded (requires frontend/dist at static/dist)
go build -tags dist -o bin/disapp .

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
internal/resources/   config loader · store/{db,model} · storage/{local,cos,oci}
pkg/web/              JSON response helpers + middleware (recoverer, logger, rate limit)
pkg/token/            JWT create/parse
pkg/pwd/              sha256 hash/salt
static/               Optional embed: static/dist is baked in only with `-tags dist`
```

## Key conventions

- No migration code — schema auto-migrated by GORM on startup
- Super-admin is authenticated from `config.json` admin block, uid=-1 in JWT
- `internal/` packages never import the root `main` package
- Backend error messages are NOT translated; only UI strings are
- Public `absURL`/`requestBase` build links from the request `Host`
  (fallback `X-Forwarded-Host`) — reverse proxies must pass the real host
- No public app-list endpoint (`/api/v1/apps`); public reach is by
  `/api/v1/apps/{name-or-id}` only
- Notification bots live under `/api/v1/admin/subscriptions` — template
  params in `service/notify.go` (`NotifyParams`), events in
  `resources/store/model` (`Event*`); test pushes hit the `/test` endpoints