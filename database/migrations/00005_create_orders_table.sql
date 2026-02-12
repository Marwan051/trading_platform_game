-- +goose Up
-- +goose StatementBegin
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trader_id BIGINT NOT NULL REFERENCES traders(id) ON DELETE CASCADE,
    stock_ticker TEXT NOT NULL REFERENCES stocks(ticker) ON DELETE CASCADE,
    order_type TEXT NOT NULL CHECK (order_type IN ('MARKET', 'LIMIT')),
    side TEXT NOT NULL CHECK (side IN ('BUY', 'SELL')),
    quantity BIGINT NOT NULL CHECK (quantity > 0),
    filled_quantity BIGINT DEFAULT 0 CHECK (filled_quantity >= 0),
    remaining_quantity BIGINT NOT NULL CHECK (remaining_quantity >= 0),
    limit_price_cents BIGINT,
    status TEXT DEFAULT 'PENDING' CHECK (
        status IN (
            'PENDING',
            'PARTIAL',
            'FILLED',
            'CANCELLED',
            'REJECTED'
        )
    ),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    filled_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ
);
CREATE INDEX idx_orders_trader ON orders(trader_id);
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