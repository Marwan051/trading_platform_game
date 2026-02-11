-- name: GetUserPositions :many
SELECT p.*,
    s.company_name,
    s.current_price_cents
FROM positions p
    JOIN stocks s ON p.stock_ticker = s.ticker
WHERE p.user_id = $1;
-- name: GetPosition :one
SELECT *
FROM positions
WHERE user_id = $1
    AND stock_ticker = $2;
-- name: UpsertPosition :one
INSERT INTO positions (
        user_id,
        bot_id,
        stock_ticker,
        quantity,
        average_cost_cents,
        total_cost_cents
    )
VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (user_id, stock_ticker)
WHERE user_id IS NOT NULL DO
UPDATE
SET quantity = positions.quantity + EXCLUDED.quantity,
    total_cost_cents = positions.total_cost_cents + EXCLUDED.total_cost_cents,
    average_cost_cents = (
        positions.total_cost_cents + EXCLUDED.total_cost_cents
    ) / (positions.quantity + EXCLUDED.quantity),
    updated_at = NOW()
RETURNING *;
-- name: ReducePosition :one
UPDATE positions
SET quantity = quantity - $3,
    total_cost_cents = total_cost_cents - ($3 * average_cost_cents),
    updated_at = NOW()
WHERE user_id = $1
    AND stock_ticker = $2
RETURNING *;
-- name: DeletePosition :exec
DELETE FROM positions
WHERE user_id = $1
    AND stock_ticker = $2;