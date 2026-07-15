-- +goose Up
CREATE TABLE strategies (
    id         UUID        NOT NULL PRIMARY KEY,
    name       TEXT        NOT NULL,
    status     TEXT        NOT NULL,
    version    BIGINT      NOT NULL DEFAULT 0,
    identifier TEXT        NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS strategies;
