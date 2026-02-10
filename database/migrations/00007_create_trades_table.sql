-- +goose Up
-- +goose StatementBegin
CREATE TABLE trades (
    id UUID DEFAULT gen_random_uuid(),
    stock_id UUID NOT NULL REFERENCES stocks(id) ON DELETE CASCADE,
    buyer_order_id UUID NOT NULL REFERENCES orders(id),
    seller_order_id UUID NOT NULL REFERENCES orders(id),
    buyer_user_id TEXT,
    -- References Better Auth user.id
    buyer_bot_id UUID REFERENCES bots(id),
    seller_user_id TEXT,
    -- References Better Auth user.id
    seller_bot_id UUID REFERENCES bots(id),
    quantity BIGINT NOT NULL CHECK (quantity > 0),
    price_cents BIGINT NOT NULL CHECK (price_cents > 0),
    total_value_cents BIGINT NOT NULL,
    executed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, executed_at),
    CHECK (
        (
            buyer_user_id IS NOT NULL
            AND buyer_bot_id IS NULL
        )
        OR (
            buyer_user_id IS NULL
            AND buyer_bot_id IS NOT NULL
        )
    ),
    CHECK (
        (
            seller_user_id IS NOT NULL
            AND seller_bot_id IS NULL
        )
        OR (
            seller_user_id IS NULL
            AND seller_bot_id IS NOT NULL
        )
    )
);
SELECT create_hypertable(
        'trades',
        'executed_at',
        chunk_time_interval => INTERVAL '1 day',
        if_not_exists => TRUE
    );
CREATE INDEX idx_trades_stock_time ON trades(stock_id, executed_at DESC);
CREATE INDEX idx_trades_buyer_user ON trades(buyer_user_id, executed_at DESC)
WHERE buyer_user_id IS NOT NULL;
CREATE INDEX idx_trades_buyer_bot ON trades(buyer_bot_id, executed_at DESC)
WHERE buyer_bot_id IS NOT NULL;
CREATE INDEX idx_trades_seller_user ON trades(seller_user_id, executed_at DESC)
WHERE seller_user_id IS NOT NULL;
CREATE INDEX idx_trades_seller_bot ON trades(seller_bot_id, executed_at DESC)
WHERE seller_bot_id IS NOT NULL;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS trades CASCADE;
-- +goose StatementEnd