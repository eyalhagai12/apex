-- name: UpsertBar :exec
INSERT INTO bars (time, symbol, high, open, low, close, volume)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (symbol, time) DO UPDATE SET
    high = excluded.high,
    open = excluded.open,
    low = excluded.low,
    close = excluded.close,
    volume = excluded.volume;

-- name: UpsertBars :exec
INSERT INTO bars (time, symbol, high, open, low, close, volume)
SELECT * FROM unnest(
    @time::timestamptz[],
    @symbol::text[],
    @high::float8[],
    @open::float8[],
    @low::float8[],
    @close::float8[],
    @volume::bigint[]
) AS t(time, symbol, high, open, low, close, volume)
ON CONFLICT (symbol, time) DO UPDATE SET
    high = excluded.high,
    open = excluded.open,
    low = excluded.low,
    close = excluded.close,
    volume = excluded.volume;

-- name: ListBarsBySymbol :many
SELECT time, symbol, open, high, low, close, volume
FROM bars WHERE symbol = $1 ORDER BY time ASC;

-- name: ListAggregatedBars :many
SELECT time_bucket(@bucket::interval, time)::timestamptz AS bucket,
       symbol,
       first(open, time)::float8 AS open,
       max(high)::float8 AS high,
       min(low)::float8 AS low,
       last(close, time)::float8 AS close,
       sum(volume)::bigint AS volume
FROM bars
WHERE symbol = @symbol
GROUP BY bucket, symbol
ORDER BY bucket ASC;
