# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```sh
# Run the server
go run ./cmd/server

# Build
go build ./...

# Run tests
go test ./...

# Run a single test
go test ./path/to/package -run TestName

# Apply pending migrations (Postgres must be running)
go tool goose -dir migrations postgres "postgres://apex:apex@localhost:5432/apex?sslmode=disable" up

# Create a new migration
go tool goose -dir migrations create <name> sql

# Roll back last migration
go tool goose -dir migrations postgres "postgres://apex:apex@localhost:5432/apex?sslmode=disable" down

# Start all backing services
docker compose up -d

# Stop backing services
docker compose down

# Build the dashboard CSS (Tailwind, standalone CLI - downloads tools/tailwindcss.exe on first run)
make css

# Rebuild CSS on every markup change while developing (run in a second terminal alongside `make dev`)
make css-watch
```

## Architecture

Apex is a market data ingestion service. It connects to Alpaca's IEX feed, backfills historical bars, and subscribes to real-time bar updates, persisting everything to a TimescaleDB `bars` hypertable.

**Package layout:**

- `cmd/server/` — entrypoint: loads `.env`, wires the DB and marketdata module, runs backfill + subscribe then blocks until signal
- `internal/domain/` — shared types (`Bar`, `Stream`); no external dependencies
- `internal/httputil/` — generic `Wrap[Req, Res]` helper that decodes JSON bodies and encodes responses; `WriteJSON`/`WriteError` utilities; `LogRoutes` middleware logs every request (method, path, status, duration)
- `internal/logging/` — placeholder for `log/slog` setup (currently empty)
- `internal/web/` — templ + htmx dashboard: subscribe form, SSE live bar streaming, TradingView chart panels; mounted at `/`, `/web/*`, `/static/*`
- `marketdata/` — public module API (`Module.Subscribe`, `Module.Unsubscribe`, `Module.Backfill`); depends on `providers.Provider` interface and `MarketDataStorage` interface — both are swappable
- `marketdata/providers/` — `Provider` interface + `AlpacaProvider` implementation (real-time via `stream.StocksClient`, historical via `marketdata.Client`, both IEX feed)
- `marketdata/internal/storage/` — `MarketDataRepository` backed by `database/sql`; upserts bars on conflict
- `migrations/` — goose SQL migrations; goose is a Go tool dependency (`go tool goose ...`), no separate install needed

**Data flow:**
1. On startup: `marketdata.New` → `AlpacaProvider.Connect` (WebSocket to Alpaca)
2. `Module.Backfill` fetches historical bars via REST and upserts each into `bars`
3. `Module.Subscribe` registers a callback on the WebSocket stream; each incoming bar is upserted via `StoreBar`

**HTTP layer:** the only mounted routes today are the `internal/web` htmx dashboard and `/metrics`. A JSON REST API for `marketdata.Module` (subscribe/unsubscribe/backfill) was removed for now and will be reintroduced later; `httputil.Wrap` is a generic adapter — handlers declare typed `(w, r, Req) → (Res, status, error)` signatures and `Wrap` handles decode/encode/error — kept in place for that. Every request is logged by `httputil.LogRoutes`, wrapping the whole mux in `cmd/server/main.go`.

**Observability:** JSON logs via `log/slog` written to stdout and `logs/apex.log`. Promtail tails the log file and ships to Loki. Grafana is pre-provisioned at `localhost:3000` with Loki as a datasource (query with `{app="apex"}`).

**Timeframe strings** passed to the provider use Alpaca's format (`"1Min"`, `"5Min"`) — `parseTimeFrame` maps these to SDK constants; anything unrecognized falls back to 1-Day bars.

**Styling:** the dashboard uses Tailwind CSS v4 via the standalone CLI (no Node/npm) — utility classes live directly in `.templ` markup, no hand-maintained component CSS. Source is `internal/web/tailwind/input.css` (imports Tailwind, defines semantic color tokens like `--color-primary`/`--color-surface` in an `@theme` block, including a dark-mode override via `prefers-color-scheme`). Output is the gitignored, `go:embed`-ed `internal/web/static/css/app.css`, built by `make css` (one-shot) or `make css-watch` (rebuilds on markup changes; run alongside `make dev` since air itself doesn't watch CSS). `dev` in the Makefile runs `css` once before starting air so a fresh checkout never embeds a stale/missing CSS file; `.air.toml` also runs a one-shot Tailwind build as a `pre_cmd` on every rebuild as a safety net. `dashboard.js` reads some of the `@theme` color variables directly via `getComputedStyle` to theme the lightweight-charts instance, so those variable names must stay in sync between `input.css` and `dashboard.js`.
