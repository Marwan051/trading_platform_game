-- +goose Up
-- +goose StatementBegin
CREATE TABLE market_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    total_users INTEGER DEFAULT 0,
    total_bots INTEGER DEFAULT 0,
    active_users_24h INTEGER DEFAULT 0,
    active_bots_24h INTEGER DEFAULT 0,
    total_trades_24h BIGINT DEFAULT 0,
    total_volume_24h_cents BIGINT DEFAULT 0,
    top_gainer_stock_ticker UUID REFERENCES stocks(id),
    top_loser_stock_ticker UUID REFERENCES stocks(id),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
INSERT INTO market_stats (id)
VALUES (gen_random_uuid());
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS market_stats CASCADE;
-- +goose StatementEnd