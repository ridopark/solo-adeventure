.PHONY: dev dev-backend dev-web build build-backend build-web test tidy clean fmt

ROOT := $(shell pwd)

dev:
	./scripts/start.sh

dev-backend:
	cd backend && go run ./cmd/solo-adeventure-server

dev-web:
	cd apps/web && npm run dev

build: build-backend build-web

build-backend:
	cd backend && go build -o bin/solo-adeventure-server ./cmd/solo-adeventure-server

build-web:
	cd apps/web && npm run build

test:
	cd backend && go test -race ./...

tidy:
	cd backend && go mod tidy

fmt:
	cd backend && go fmt ./...

clean:
	rm -rf backend/bin apps/web/.next apps/web/out
