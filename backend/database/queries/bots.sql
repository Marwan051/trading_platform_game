-- name: GetActiveBots :many
SELECT *
FROM bots
WHERE is_active = TRUE;
-- name: GetBotByID :one
SELECT *
FROM bots
WHERE id = $1;
-- name: UpdateBotLastTrade :exec
UPDATE bots
SET last_trade_at = NOW(),
    total_trades_count = total_trades_count + 1,
    updated_at = NOW()
WHERE id = $1;