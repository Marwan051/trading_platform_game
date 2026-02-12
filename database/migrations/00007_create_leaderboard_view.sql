-- +goose Up
-- +goose StatementBegin
CREATE MATERIALIZED VIEW leaderboard AS
SELECT t.id AS trader_id,
    t.auth_user_id,
    t.display_name,
    t.cash_balance_cents,
    t.total_portfolio_value_cents,
    RANK() OVER (
        ORDER BY t.total_portfolio_value_cents DESC
    ) as rank,
    NOW() as last_updated
FROM traders t
WHERE t.trader_type = 'USER'
ORDER BY t.total_portfolio_value_cents DESC;
CREATE UNIQUE INDEX idx_leaderboard_trader_id ON leaderboard(trader_id);
CREATE INDEX idx_leaderboard_rank ON leaderboard(rank);
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP MATERIALIZED VIEW IF EXISTS leaderboard CASCADE;
-- +goose StatementEnd