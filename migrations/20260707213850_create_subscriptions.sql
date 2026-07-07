-- +goose Up
CREATE TABLE subscriptions (
    symbol     TEXT        NOT NULL,
    timeframe  TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (symbol, timeframe)
);

-- +goose Down
DROP TABLE IF EXISTS subscriptions;
