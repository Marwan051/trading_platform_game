-- name: GetStockByID :one
SELECT *
FROM stocks
WHERE id = $1;
-- name: GetStockByTicker :one
SELECT *
FROM stocks
WHERE ticker = $1;
-- name: ListActiveStocks :many
SELECT *
FROM stocks
WHERE is_active = TRUE
ORDER BY ticker;
-- name: UpdateStockPrice :one
UPDATE stocks
SET current_price_cents = $2,
    previous_close_cents = current_price_cents,
    updated_at = NOW()
WHERE id = $1
RETURNING *;