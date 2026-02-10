-- name: CreateOrder :one
INSERT INTO orders (
        user_id,
        bot_id,
        stock_id,
        order_type,
        side,
        quantity,
        remaining_quantity,
        limit_price_cents,
        time_in_force,
        expires_at
    )
VALUES ($1, $2, $3, $4, $5, $6, $6, $7, $8, $9)
RETURNING *;
-- name: GetOrderByID :one
SELECT *
FROM orders
WHERE id = $1;
-- name: GetPendingOrdersForStock :many
SELECT *
FROM orders
WHERE stock_id = $1
    AND status IN ('PENDING', 'PARTIAL')
ORDER BY created_at;
-- name: GetOrderBookBuys :many
SELECT limit_price_cents,
    SUM(remaining_quantity) as quantity
FROM orders
WHERE stock_id = $1
    AND side = 'BUY'
    AND status IN ('PENDING', 'PARTIAL')
    AND order_type = 'LIMIT'
GROUP BY limit_price_cents
ORDER BY limit_price_cents DESC
LIMIT $2;
-- name: GetOrderBookSells :many
SELECT limit_price_cents,
    SUM(remaining_quantity) as quantity
FROM orders
WHERE stock_id = $1
    AND side = 'SELL'
    AND status IN ('PENDING', 'PARTIAL')
    AND order_type = 'LIMIT'
GROUP BY limit_price_cents
ORDER BY limit_price_cents ASC
LIMIT $2;
-- name: UpdateOrderFill :one
UPDATE orders
SET filled_quantity = $2,
    remaining_quantity = $3,
    status = $4,
    filled_at = $5,
    updated_at = NOW()
WHERE id = $1
RETURNING *;
-- name: CancelOrder :one
UPDATE orders
SET status = 'CANCELLED',
    cancelled_at = NOW(),
    updated_at = NOW()
WHERE id = $1
RETURNING *;
-- name: ExpireOrders :exec
UPDATE orders
SET status = 'EXPIRED',
    updated_at = NOW()
WHERE status IN ('PENDING', 'PARTIAL')
    AND expires_at IS NOT NULL
    AND expires_at < NOW();