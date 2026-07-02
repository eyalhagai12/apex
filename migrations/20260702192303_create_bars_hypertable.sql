-- +goose Up
CREATE TABLE bars (
    time      TIMESTAMPTZ      NOT NULL,
    symbol    TEXT             NOT NULL,
    timeframe TEXT             NOT NULL,
    open      DOUBLE PRECISION NOT NULL,
    high      DOUBLE PRECISION NOT NULL,
    low       DOUBLE PRECISION NOT NULL,
    close     DOUBLE PRECISION NOT NULL,
    volume    BIGINT           NOT NULL,
    PRIMARY KEY (symbol, timeframe, time)
);

SELECT create_hypertable('bars', 'time');

-- +goose Down
DROP TABLE IF EXISTS bars;
