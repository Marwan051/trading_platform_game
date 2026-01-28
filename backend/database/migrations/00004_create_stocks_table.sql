-- +goose Up
-- +goose StatementBegin
CREATE TABLE stocks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticker TEXT UNIQUE NOT NULL,
    company_name TEXT NOT NULL,
    sector TEXT,
    description TEXT,
    current_price_cents BIGINT NOT NULL CHECK (current_price_cents > 0),
    previous_close_cents BIGINT,
    market_cap_cents BIGINT,
    total_shares BIGINT DEFAULT 1000000,
    shares_outstanding BIGINT DEFAULT 1000000,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_stocks_ticker ON stocks(ticker);
CREATE INDEX idx_stocks_is_active ON stocks(is_active);
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS stocks CASCADE;
-- +goose StatementEnd