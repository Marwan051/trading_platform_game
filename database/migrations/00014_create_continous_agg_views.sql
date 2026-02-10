-- +goose Up
-- +goose StatementBegin
CREATE MATERIALIZED VIEW price_1min WITH (timescaledb.continuous) AS
SELECT stock_id,
    time_bucket('1 minute', executed_at) AS bucket,
    FIRST(price_cents, executed_at) AS open_cents,
    MAX(price_cents) AS high_cents,
    MIN(price_cents) AS low_cents,
    LAST(price_cents, executed_at) AS close_cents,
    SUM(quantity) AS volume,
    COUNT(*) AS trade_count
FROM trades
GROUP BY stock_id,
    bucket WITH NO DATA;
SELECT add_continuous_aggregate_policy(
        'price_1min',
        start_offset => INTERVAL '3 hours',
        end_offset => INTERVAL '1 minute',
        schedule_interval => INTERVAL '1 minute'
    );
-- 1-hour OHLCV aggregate
CREATE MATERIALIZED VIEW price_1hour WITH (timescaledb.continuous) AS
SELECT stock_id,
    time_bucket('1 hour', executed_at) AS bucket,
    FIRST(price_cents, executed_at) AS open_cents,
    MAX(price_cents) AS high_cents,
    MIN(price_cents) AS low_cents,
    LAST(price_cents, executed_at) AS close_cents,
    SUM(quantity) AS volume,
    COUNT(*) AS trade_count
FROM trades
GROUP BY stock_id,
    bucket WITH NO DATA;
SELECT add_continuous_aggregate_policy(
        'price_1hour',
        start_offset => INTERVAL '1 week',
        end_offset => INTERVAL '1 hour',
        schedule_interval => INTERVAL '1 hour'
    );
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP MATERIALIZED VIEW IF EXISTS price_1min CASCADE;
DROP MATERIALIZED VIEW IF EXISTS price_1hour CASCADE;
-- +goose StatementEnd