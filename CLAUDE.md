# disapp — App Distribution Platform

Vue 3 + Tailwind CSS v4 + shadcn-vue frontend, Go backend, SQLite storage.

## Project structure

```
frontend/    Vue 3 + Vite + TypeScript + Tailwind CSS v4 + shadcn-vue
backend/     Go 1.22+ std-lib HTTP, GORM + SQLite
```

## Build & test

Always stop processes after testing. The backend embeds the frontend
dist via `embed.FS`, so any frontend change requires a backend rebuild
for the integrated binary.

```bash
# Frontend type-check + build
cd frontend && npm run build

# Backend build + test (default build has NO embedded frontend)
cd backend && go build -o bin/disapp . && go test ./...

# Build with embedded frontend dist
cd backend && go build -tags dist -o bin/disapp .

# Wipe local DB (dev only)
make reset
```

## Dev workflow (hot-reload)

Two terminals, one command each:

```bash
cd backend && APP_CONFIG=../config.json go run .    # :8080
cd frontend && npm run dev                                      # :5173 → proxy /api → :8080
```

Open http://localhost:5173. Kill both when done.

## Key conventions

- No migration code — `make reset` to rebuild schema
- Super-admin is configured in `config.json`, not in the DB
- API responses are NOT translated; only UI strings are
- Icons: `lucide-vue-next`; `vue-sonner` for toasts
- Reuse `frontend/src/components/ui/*` shadcn primitives, never raw Vuetify
- Webhook templates use `{{key}}` placeholders — see `service/notify.go`
- Public home `/` is a blank placeholder; apps are reached by direct detail
  links (`/app/:name`), there is no public app-list endpoint (`/api/v1/apps`)