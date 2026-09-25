# Переменные подключения к БД (берем те, что ты указал в compose)
export PATH := $(HOME)/go/bin:$(PATH)

DB_URL="postgres://didok362:qwerty67@localhost:5432/messenger_db?sslmode=disable"
MIGRATIONS_DIR=./migrations

.PHONY: migrate-up migrate-down migrate-status migrate-create env-up env-down

# Команды для миграций через goose
migrate-create:
	goose -dir $(MIGRATIONS_DIR) create $(name) sql

migrate-up:
	goose -dir $(MIGRATIONS_DIR) postgres $(DB_URL) up

migrate-down:
	goose -dir $(MIGRATIONS_DIR) postgres $(DB_URL) down

migrate-status:
	goose -dir $(MIGRATIONS_DIR) postgres $(DB_URL) status

env-up:
	docker compose up -d

env-down:
	docker compose down
