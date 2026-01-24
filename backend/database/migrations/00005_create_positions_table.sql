-- +goose Up
-- +goose StatementBegin
CREATE TABLE positions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT,
    -- References Better Auth user.id
    bot_id UUID REFERENCES bots(id) ON DELETE CASCADE,
    stock_id UUID NOT NULL REFERENCES stocks(id) ON DELETE CASCADE,
    quantity BIGINT NOT NULL CHECK (quantity > 0),
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
    UNIQUE(user_id, stock_id),
    UNIQUE(bot_id, stock_id)
);
CREATE INDEX idx_positions_user ON positions(user_id)
WHERE user_id IS NOT NULL;
CREATE INDEX idx_positions_bot ON positions(bot_id)
WHERE bot_id IS NOT NULL;
CREATE INDEX idx_positions_stock ON positions(stock_id);
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS positions CASCADE;
-- +goose StatementEnd