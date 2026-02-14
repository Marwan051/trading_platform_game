-- +goose Up
-- +goose StatementBegin
CREATE TABLE traders (
    id BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    trader_type TEXT NOT NULL CHECK (trader_type IN ('USER', 'BOT')),
    auth_user_id TEXT UNIQUE,
    -- References Better Auth's user.id (for USER traders)
    owner_trader_id BIGINT DEFAULT NULL REFERENCES traders(id) ON DELETE RESTRICT,
    -- Owner for BOT traders, NULL for system bots, defaults to -1 for orphaned bots
    display_name TEXT UNIQUE NOT NULL,
    cash_balance_cents BIGINT DEFAULT 1000000 CHECK (cash_balance_cents >= 0),
    cash_hold_cents BIGINT DEFAULT 0 CHECK (cash_hold_cents >= 0),
    total_portfolio_value_cents BIGINT DEFAULT 1000000,
    is_active BOOLEAN DEFAULT TRUE,
    last_active_at TIMESTAMPTZ DEFAULT NOW(),
    last_trade_at TIMESTAMPTZ,
    total_trades_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CHECK (
        (
            trader_type = 'USER'
            AND auth_user_id IS NOT NULL
            AND owner_trader_id IS NULL
        )
        OR (
            trader_type = 'BOT'
            AND auth_user_id IS NULL
        )
    )
);
CREATE INDEX idx_traders_auth_user_id ON traders(auth_user_id)
WHERE auth_user_id IS NOT NULL;
CREATE INDEX idx_traders_owner ON traders(owner_trader_id)
WHERE owner_trader_id IS NOT NULL;
CREATE INDEX idx_traders_type ON traders(trader_type);
CREATE INDEX idx_traders_active_bots ON traders(is_active)
WHERE trader_type = 'BOT'
    AND is_active = TRUE;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS traders CASCADE;
-- +goose StatementEnd