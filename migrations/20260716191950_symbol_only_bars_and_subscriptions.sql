-- +goose Up
DELETE FROM bars WHERE timeframe <> '1Min';
ALTER TABLE bars DROP CONSTRAINT bars_pkey;
ALTER TABLE bars DROP COLUMN timeframe;
ALTER TABLE bars ADD PRIMARY KEY (symbol, time);

DELETE FROM subscriptions s USING subscriptions s2
    WHERE s.symbol = s2.symbol AND s.ctid < s2.ctid;
ALTER TABLE subscriptions DROP CONSTRAINT subscriptions_pkey;
ALTER TABLE subscriptions DROP COLUMN timeframe;
ALTER TABLE subscriptions ADD PRIMARY KEY (symbol);

-- +goose Down
ALTER TABLE subscriptions DROP CONSTRAINT subscriptions_pkey;
ALTER TABLE subscriptions ADD COLUMN timeframe TEXT NOT NULL DEFAULT '1Min';
ALTER TABLE subscriptions ADD PRIMARY KEY (symbol, timeframe);

ALTER TABLE bars DROP CONSTRAINT bars_pkey;
ALTER TABLE bars ADD COLUMN timeframe TEXT NOT NULL DEFAULT '1Min';
ALTER TABLE bars ADD PRIMARY KEY (symbol, timeframe, time);
