-- name: CreateTrade :one
INSERT INTO trades (
        stock_ticker,
        buyer_order_id,
        seller_order_id,
        buyer_user_id,
        buyer_bot_id,
        seller_user_id,
        seller_bot_id,
        quantity,
        price_cents,
        total_value_cents
    )
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;
-- name: GetRecentTradesForStock :many
SELECT *
FROM trades
WHERE stock_ticker = $1
ORDER BY executed_at DESC
LIMIT $2;