.PHONY: test lint build up down migrate tidy

GO ?= go
COMPOSE ?= sudo docker compose -f deploy/docker-compose.yml

test:
	$(GO) test -race ./...

lint:
	@command -v golangci-lint >/dev/null && golangci-lint run ./... || echo "golangci-lint not installed, skip"

build:
	mkdir -p bin
	$(GO) build -o bin/cup-api ./cmd/cup-api

tidy:
	$(GO) mod tidy

up:
	$(COMPOSE) up -d --build

down:
	$(COMPOSE) down -v

migrate:
	$(GO) run ./cmd/cup-api -migrate-only