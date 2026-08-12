.PHONY: build run frontend

frontend:
	cd frontend && npm install && npm run build

build: frontend
	rm -rf backend/static/dist
	cp -r frontend/dist backend/static/dist
	cd backend && go build -o ../bin/disapp ./cmd/server

run: build
	./bin/disapp
