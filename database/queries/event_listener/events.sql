-- name: OrderPlacedEvent :exec
INSERT INTO orders
VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);
-- name : OrderCancelledEvent :exec
UPDATE orders
SET status = "CANCELLED"
    AND remaining_quantity = 0
WHERE order_id = $1;