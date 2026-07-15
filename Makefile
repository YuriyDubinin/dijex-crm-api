.PHONY: run build test lint migrate-up migrate-down migrate-create docker-up docker-down docker-logs docker-rebuild

BINARY      := bin/api
PKG         := ./...
MIGRATIONS  := migrations
# Приложение само применяет миграции на старте (RunMigrations). Цели migrate-*
# нужны лишь для ручного управления. DB_DSN переопредели под своё окружение,
# например: make migrate-up DB_DSN="postgres://bms_db_user_:PASS@localhost:15432/bms_db?sslmode=disable"
# (спецсимволы пароля URL-энкодь: @→%40 !→%21 #→%23 $→%24 &→%26 ^→%5E %→%25 *→%2A)
DB_DSN      ?= postgres://bms_db_user_:pass@localhost:15432/bms_db?sslmode=disable

run:
	go run ./cmd/api

build:
	mkdir -p bin
	go build -o $(BINARY) ./cmd/api

test:
	go test -race -count=1 $(PKG)

lint:
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not installed"; exit 1; }
	golangci-lint run

migrate-up:
	migrate -path $(MIGRATIONS) -database "$(DB_DSN)" up

migrate-down:
	migrate -path $(MIGRATIONS) -database "$(DB_DSN)" down 1

migrate-create:
	@test -n "$(name)" || { echo "usage: make migrate-create name=<migration_name>"; exit 1; }
	migrate create -ext sql -dir $(MIGRATIONS) -seq $(name)

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

docker-rebuild:
	docker compose up -d --build
