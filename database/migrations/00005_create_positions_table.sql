-- +goose Up
-- +goose StatementBegin
CREATE TABLE positions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT,
    -- References Better Auth user.id
    bot_id BIGINT REFERENCES bots(id) ON DELETE CASCADE,
    stock_ticker TEXT NOT NULL REFERENCES stocks(ticker) ON DELETE CASCADE,
    quantity BIGINT NOT NULL CHECK (quantity >= 0),
    quantity_hold BIGINT DEFAULT 0 CHECK (quantity_hold >= 0),
    average_cost_cents BIGINT NOT NULL,
    total_cost_cents BIGINT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CHECK (
        (
            user_id IS NOT NULL
            AND bot_id IS NULL
        )
        OR (
            user_id IS NULL
            AND bot_id IS NOT NULL
        )
    ),
    UNIQUE(user_id, stock_ticker),
    UNIQUE(bot_id, stock_ticker)
);
CREATE INDEX idx_positions_user ON positions(user_id)
WHERE user_id IS NOT NULL;
CREATE INDEX idx_positions_bot ON positions(bot_id)
WHERE bot_id IS NOT NULL;
CREATE INDEX idx_positions_stock ON positions(stock_ticker);
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS positions CASCADE;
-- +goose StatementEnd