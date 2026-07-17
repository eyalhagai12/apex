-- name: UpsertSubscription :exec
INSERT INTO subscriptions (symbol) VALUES ($1)
ON CONFLICT (symbol) DO NOTHING;

-- name: DeleteSubscription :exec
DELETE FROM subscriptions WHERE symbol = $1;

-- name: ListSubscriptions :many
SELECT symbol FROM subscriptions ORDER BY symbol;
