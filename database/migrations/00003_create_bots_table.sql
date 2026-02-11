-- +goose Up
-- +goose StatementBegin
CREATE TABLE bots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id TEXT,
    -- References Better Auth user.id, NULL for system bots
    bot_name TEXT UNIQUE NOT NULL,
    cash_balance_cents BIGINT DEFAULT 10000000 CHECK (cash_balance_cents >= 0),
    cash_hold_cents BIGINT DEFAULT 0 CHECK (cash_hold_cents >= 0),
    total_portfolio_value_cents BIGINT DEFAULT 10000000,
    is_active BOOLEAN DEFAULT TRUE,
    trading_strategy TEXT DEFAULT 'RANDOM' CHECK (
        trading_strategy IN (
            'RANDOM',
            'VALUE_INVESTOR',
            'TREND_FOLLOWER',
            'CONTRARIAN'
        )
    ),
    risk_tolerance TEXT DEFAULT 'MEDIUM' CHECK (
        risk_tolerance IN ('CONSERVATIVE', 'MEDIUM', 'AGGRESSIVE')
    ),
    last_trade_at TIMESTAMPTZ,
    total_trades_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_bots_owner ON bots(owner_user_id);
CREATE INDEX idx_bots_active ON bots(is_active)
WHERE is_active = TRUE;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS bots CASCADE;
-- +goose StatementEnd