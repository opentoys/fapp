.PHONY: build run frontend

frontend:
	cd frontend && npm install && npm run build

build: frontend
	rm -rf backend/dist && cp -r frontend/dist backend/dist
	cd backend && go build -o ../bin/disapp ./cmd/server

run: build
	./bin/disapp
