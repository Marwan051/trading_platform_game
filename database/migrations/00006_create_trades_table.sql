-- +goose Up
-- +goose StatementBegin
CREATE TABLE trades (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    stock_ticker TEXT NOT NULL REFERENCES stocks(ticker) ON DELETE CASCADE,
    buyer_order_id UUID NOT NULL REFERENCES orders(id),
    seller_order_id UUID NOT NULL REFERENCES orders(id),
    buyer_trader_id BIGINT NOT NULL REFERENCES traders(id),
    seller_trader_id BIGINT NOT NULL REFERENCES traders(id),
    quantity BIGINT NOT NULL CHECK (quantity > 0),
    price_cents BIGINT NOT NULL CHECK (price_cents > 0),
    total_value_cents BIGINT NOT NULL,
    executed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, executed_at)
);
SELECT create_hypertable(
        'trades',
        'executed_at',
        chunk_time_interval => INTERVAL '1 day',
        if_not_exists => TRUE
    );
CREATE INDEX idx_trades_stock_time ON trades(stock_ticker, executed_at DESC);
CREATE INDEX idx_trades_buyer_trader ON trades(buyer_trader_id, executed_at DESC);
CREATE INDEX idx_trades_seller_trader ON trades(seller_trader_id, executed_at DESC);
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS trades CASCADE;
-- +goose StatementEnd