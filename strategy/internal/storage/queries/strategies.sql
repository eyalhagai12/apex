-- name: UpsertStrategy :exec
INSERT INTO strategies (id, name, status, version, identifier)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (id) DO UPDATE SET
    name = excluded.name,
    status = excluded.status,
    version = excluded.version,
    identifier = excluded.identifier;

-- name: GetStrategy :one
SELECT id, name, status, version, identifier FROM strategies WHERE id = $1;

-- name: DeleteStrategy :exec
DELETE FROM strategies WHERE id = $1;

-- name: ListStrategies :many
SELECT id, name, status, version, identifier FROM strategies ORDER BY name;
