.PHONY: build run frontend reset

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
