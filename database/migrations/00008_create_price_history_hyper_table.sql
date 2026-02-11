-- +goose Up
-- +goose StatementBegin
CREATE TABLE price_history (
    stock_ticker UUID NOT NULL REFERENCES stocks(id) ON DELETE CASCADE,
    timestamp TIMESTAMPTZ NOT NULL,
    open_cents BIGINT NOT NULL,
    high_cents BIGINT NOT NULL,
    low_cents BIGINT NOT NULL,
    close_cents BIGINT NOT NULL,
    volume BIGINT DEFAULT 0,
    trade_count INTEGER DEFAULT 0,
    PRIMARY KEY (stock_ticker, timestamp)
);
SELECT create_hypertable(
        'price_history',
        'timestamp',
        chunk_time_interval => INTERVAL '7 days',
        if_not_exists => TRUE
    );
CREATE INDEX idx_price_history_stock ON price_history(stock_ticker, timestamp DESC);
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS price_history CASCADE;
-- +goose StatementEnd