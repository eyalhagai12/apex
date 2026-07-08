POSTGRES_USER ?= apex
POSTGRES_PASSWORD ?= apex
POSTGRES_DB ?= apex
POSTGRES_PORT ?= 5432

DB_DSN := postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable

.PHONY: run dev templ setup teardown migrate-up migrate-down migrate-create dev templ reboot-setup

run:
	go run ./cmd/server

dev:
	go tool air

templ:
	go tool templ generate

setup:
	docker compose up -d

teardown:
	docker compose down -v

migrate-up:
	go tool goose -dir migrations postgres "$(DB_DSN)" up

migrate-down:
	go tool goose -dir migrations postgres "$(DB_DSN)" down

migrate-create:
	go tool goose -dir migrations create $(name) sql

reboot-setup: teardown setup migrate-up
