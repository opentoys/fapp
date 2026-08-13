.PHONY: build run frontend reset dev-fe dev-api

frontend:
	cd frontend && npm install && npm run build

build: frontend
	rm -rf backend/static/dist
	cp -r frontend/dist backend/static/dist
	cd backend && go build -o ../bin/disapp ./cmd/server

run: build
	./bin/disapp

# Dev-only: blow away the local SQLite DB and uploaded files. The server
# will rebuild the schema from the model on next start. Use whenever the
# schema changes or the local state gets weird.
reset:
	rm -rf data/

# Dev workflow — run each in its own terminal:
#   make dev-fe   # Vite dev server on :5173, proxies /api → :8080
#   make dev-api  # Go server on :8080, no frontend bundling involved
# Then open http://localhost:5173 in your browser.
dev-fe:
	cd frontend && npm run dev

dev-api:
	cd backend && APP_CONFIG=../config.json go run ./cmd/server
