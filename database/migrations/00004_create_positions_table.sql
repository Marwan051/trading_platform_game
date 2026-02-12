-- +goose Up
-- +goose StatementBegin
CREATE TABLE positions (
    id BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    trader_id BIGINT NOT NULL REFERENCES traders(id) ON DELETE CASCADE,
    stock_ticker TEXT NOT NULL REFERENCES stocks(ticker) ON DELETE CASCADE,
    quantity BIGINT NOT NULL CHECK (quantity >= 0),
    quantity_hold BIGINT DEFAULT 0 CHECK (quantity_hold >= 0),
    average_cost_cents BIGINT NOT NULL,
    total_cost_cents BIGINT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(trader_id, stock_ticker)
);
CREATE INDEX idx_positions_trader ON positions(trader_id);
CREATE INDEX idx_positions_stock ON positions(stock_ticker);
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS positions CASCADE;
-- +goose StatementEnd