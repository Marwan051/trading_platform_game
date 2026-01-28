-- name: GetUserByID :one
SELECT *
FROM user_profile
WHERE user_id = $1;
-- name: GetUserByUsername :one
SELECT *
FROM user_profile
WHERE username = $1;
-- name: CreateUser :one
INSERT INTO user_profile (user_id, username)
VALUES ($1, $2)
RETURNING *;
-- name: UpdateUserCashBalance :one
UPDATE user_profile
SET cash_balance_cents = $2,
    updated_at = NOW()
WHERE user_id = $1
RETURNING *;
-- name: GetLeaderboard :many
SELECT *
FROM leaderboard
ORDER BY rank
LIMIT $1;