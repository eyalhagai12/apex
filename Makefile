POSTGRES_USER ?= apex
POSTGRES_PASSWORD ?= apex
POSTGRES_DB ?= apex
POSTGRES_PORT ?= 5432

DB_DSN := postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable

TAILWIND_VERSION ?= 4.3.3
TAILWIND_BIN := tools/tailwindcss.exe

.PHONY: run dev templ setup teardown migrate-up migrate-down migrate-create dev templ reboot-setup tailwindcss-install css css-watch sqlc

run:
	go run ./cmd/server

dev: css
	go tool air

templ:
	go tool templ generate

sqlc:
	go tool sqlc generate

tailwindcss-install:
	@if not exist $(TAILWIND_BIN) powershell -Command "New-Item -ItemType Directory -Force tools | Out-Null; Invoke-WebRequest -Uri https://github.com/tailwindlabs/tailwindcss/releases/download/v$(TAILWIND_VERSION)/tailwindcss-windows-x64.exe -OutFile $(TAILWIND_BIN)"

css: tailwindcss-install
	$(TAILWIND_BIN) -i internal/web/tailwind/input.css -o internal/web/static/css/app.css --minify

css-watch: tailwindcss-install
	$(TAILWIND_BIN) -i internal/web/tailwind/input.css -o internal/web/static/css/app.css --watch

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
