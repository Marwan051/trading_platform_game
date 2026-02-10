-- +goose Up
-- +goose StatementBegin
CREATE TABLE bots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id TEXT,
    -- References Better Auth user.id, NULL for system bots
    bot_name TEXT UNIQUE NOT NULL,
    -- Financial state (persisted to DB)
    cash_balance_cents BIGINT DEFAULT 10000000 CHECK (cash_balance_cents >= 0),
    total_portfolio_value_cents BIGINT DEFAULT 10000000,
    -- Bot configuration (loaded into memory on startup)
    is_active BOOLEAN DEFAULT TRUE,
    trading_strategy TEXT DEFAULT 'RANDOM' CHECK (
        trading_strategy IN (
            'RANDOM',
            'VALUE_INVESTOR',
            'NEWS_FOLLOWER',
            'TREND_FOLLOWER',
            'CONTRARIAN',
            'DIVERSIFIED',
            'CUSTOM'
        )
    ),
    risk_tolerance TEXT DEFAULT 'MEDIUM' CHECK (
        risk_tolerance IN ('CONSERVATIVE', 'MEDIUM', 'AGGRESSIVE')
    ),
    -- Trading parameters (used by in-memory bot service)
    min_trade_interval_seconds INTEGER DEFAULT 1000,
    max_position_size_pct INTEGER DEFAULT 1000,
    -- 10.00% in basis points
    max_order_value_cents BIGINT DEFAULT 1000000,
    -- $10,000.00 in cents
    preferred_stocks JSONB,
    -- Array of stock tickers bot prefers
    trading_hours_start TIME DEFAULT '09:30:00',
    trading_hours_end TIME DEFAULT '16:00:00',
    -- Bot performance metrics (updated periodically from in-memory service)
    last_trade_at TIMESTAMPTZ,
    total_trades_count INTEGER DEFAULT 0,
    win_rate_basis_points INTEGER DEFAULT 0,
    -- Custom strategy code (for user-created bots)
    custom_strategy_code TEXT,
    -- JSON or code for custom strategies
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_bots_owner ON bots(owner_user_id);
CREATE INDEX idx_bots_active ON bots(is_active)
WHERE is_active = TRUE;
CREATE INDEX idx_bots_strategy ON bots(trading_strategy);
CREATE INDEX idx_bots_last_trade ON bots(last_trade_at)
WHERE is_active = TRUE;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS bots CASCADE;
-- +goose StatementEnd