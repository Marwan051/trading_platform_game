-- +goose Up
-- +goose StatementBegin
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT,
    -- References Better Auth user.id
    bot_id UUID REFERENCES bots(id) ON DELETE CASCADE,
    stock_ticker UUID NOT NULL REFERENCES stocks(id) ON DELETE CASCADE,
    order_type TEXT NOT NULL CHECK (order_type IN ('MARKET', 'LIMIT')),
    side TEXT NOT NULL CHECK (side IN ('BUY', 'SELL')),
    quantity BIGINT NOT NULL CHECK (quantity > 0),
    filled_quantity BIGINT DEFAULT 0 CHECK (filled_quantity >= 0),
    remaining_quantity BIGINT NOT NULL CHECK (remaining_quantity >= 0),
    limit_price_cents BIGINT,
    time_in_force TEXT DEFAULT 'GTC' CHECK (time_in_force IN ('GTC', 'DAY', 'IOC')),
    expires_at TIMESTAMPTZ,
    status TEXT DEFAULT 'PENDING' CHECK (
        status IN (
            'PENDING',
            'PARTIAL',
            'FILLED',
            'CANCELLED',
            'REJECTED',
            'EXPIRED'
        )
    ),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    filled_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    CHECK (
        (
            user_id IS NOT NULL
            AND bot_id IS NULL
        )
        OR (
            user_id IS NULL
            AND bot_id IS NOT NULL
        )
    )
);
CREATE INDEX idx_orders_user ON orders(user_id)
WHERE user_id IS NOT NULL;
CREATE INDEX idx_orders_bot ON orders(bot_id)
WHERE bot_id IS NOT NULL;
CREATE INDEX idx_orders_stock ON orders(stock_ticker);
CREATE INDEX idx_orders_status ON orders(status)
WHERE status IN ('PENDING', 'PARTIAL');
CREATE INDEX idx_orders_book ON orders(stock_ticker, side, status, limit_price_cents)
WHERE status IN ('PENDING', 'PARTIAL')
    AND order_type = 'LIMIT';
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS orders CASCADE;
-- +goose StatementEnd