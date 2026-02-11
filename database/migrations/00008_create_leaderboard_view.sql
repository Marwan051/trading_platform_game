-- +goose Up
-- +goose StatementBegin
CREATE MATERIALIZED VIEW leaderboard AS
SELECT up.user_id,
    up.username,
    up.cash_balance_cents,
    up.total_portfolio_value_cents,
    RANK() OVER (
        ORDER BY up.total_portfolio_value_cents DESC
    ) as rank,
    NOW() as last_updated
FROM user_profile up
ORDER BY up.total_portfolio_value_cents DESC;
CREATE UNIQUE INDEX idx_leaderboard_user_id ON leaderboard(user_id);
CREATE INDEX idx_leaderboard_rank ON leaderboard(rank);
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP MATERIALIZED VIEW IF EXISTS leaderboard CASCADE;
-- +goose StatementEnd