# apex

## Prerequisites

- [Go 1.26+](https://go.dev/dl/) (the toolchain directive in `go.mod` will auto-download 1.26.4 if your installed version is older)
- [Docker](https://www.docker.com/) with Compose v2

## Setup

1. Copy the example environment file and adjust values if needed:

   ```sh
   cp .env.example .env
   ```

2. Start the backing services:

   ```sh
   docker compose up -d
   ```

3. Run the app:

   ```sh
   go run ./cmd/server
   ```

## Services

| Service | URL | Default credentials |
|---|---|---|
| Postgres 17 + TimescaleDB | `localhost:5432` | db/user/pass: `apex` |
| pgAdmin | http://localhost:5050 | `admin@apex.com` / `apex` |
| Redis 8 | `localhost:6379` | password: `apex` |
| RedisInsight | http://localhost:5540 | add a connection to host `redis`, port `6379`, password `apex` |
| MinIO API | `localhost:9000` | `apex` / `apex123456` |
| MinIO Console | http://localhost:9001 | `apex` / `apex123456` |
| Loki | `localhost:3100` | — |
| Grafana | http://localhost:3000 | anonymous access (Admin role) |

In pgAdmin, add the Postgres server using host `postgres` (the Compose service name), not `localhost`.

Grafana is pre-provisioned with Loki as a datasource — open http://localhost:3000/explore to query logs (e.g. `{app="apex"}`).

Ports and credentials can be overridden via `.env` — see `.env.example` for the full list of variables.

## Migrations

Schema migrations are managed with [goose](https://github.com/pressly/goose), tracked as a Go tool dependency in `go.mod` (`go tool goose ...`) — no separate install needed, but Postgres must be running (`docker compose up -d`).

```sh
# apply all pending migrations
go tool goose -dir migrations postgres "postgres://apex:apex@localhost:5432/apex?sslmode=disable" up

# create a new migration
go tool goose -dir migrations create <name> sql

# roll back the last migration
go tool goose -dir migrations postgres "postgres://apex:apex@localhost:5432/apex?sslmode=disable" down

# check status
go tool goose -dir migrations postgres "postgres://apex:apex@localhost:5432/apex?sslmode=disable" status
```

Adjust the connection string if you've overridden `POSTGRES_USER`/`POSTGRES_PASSWORD`/`POSTGRES_DB`/`POSTGRES_PORT` in `.env`.

The `postgres` service runs the [TimescaleDB](https://www.timescale.com/) image; the `timescaledb` extension is enabled by the first migration (`migrations/20260702185630_enable_timescaledb.sql`). If you already had the plain `postgres:17-alpine` volume from before this change, recreate it so `shared_preload_libraries` picks up the extension:

```sh
docker compose down -v postgres
docker compose up -d
```

## Logging

The app logs JSON via `log/slog`, wired up in `logging.New` (`logging/logger.go`). Logs are written to stdout and to `logs/apex.log`. Promtail (in `docker-compose.yml`) tails `logs/apex.log` and ships entries to Loki — the app never talks to Loki directly.

## Stopping

```sh
docker compose down
```

Add `-v` to also remove the named volumes (Postgres/Redis/MinIO/pgAdmin/RedisInsight data).
