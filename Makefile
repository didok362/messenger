# Переменные подключения к БД (берем те, что ты указал в compose)
export PATH := $(HOME)/go/bin:$(PATH)
-include .env

DB_URL="postgres://didok362:qwerty67@localhost:5432/messenger_db?sslmode=disable"
MIGRATIONS_DIR=./migrations

.PHONY: migrate-up migrate-down migrate-status migrate-create env-up env-down

# Команды для миграций через goose 
env-up:
	@docker compose up -d postgres

env-down:
	@docker compose stop postgres

server:
	@go run cmd/server/main.go

tunnel:
	@cloudflared tunnel --url http://localhost:8080

deploy: 
	@docker compose up -d --build

undeploy:
	@docker compose down

logs:
	@docker compose logs -f messenger

ps:
	@docker compose ps

migrate-up:
	@docker compose run --rm migrate

migrate-status:
	@docker compose run --rm --entrypoint goose migrate -dir /migrations postgres "postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@postgres:5432/$(POSTGRES_DB)?sslmode=disable" status

migrate-create:
	@docker compose run --rm --entrypoint goose migrate -dir /migrations create $(name) sql