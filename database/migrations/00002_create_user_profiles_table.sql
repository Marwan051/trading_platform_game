-- +goose Up
-- +goose StatementBegin
CREATE TABLE user_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT UNIQUE NOT NULL,
    -- References Better Auth's user.id (TEXT type)
    username TEXT UNIQUE NOT NULL,
    -- Game display name
    cash_balance_cents BIGINT DEFAULT 1000000 CHECK (cash_balance_cents >= 0),
    -- $100,000.00 in cents
    cash_hold_cents BIGINT DEFAULT 0 CHECK (cash_hold_cents >= 0),
    -- Cash locked in pending buy orders
    total_portfolio_value_cents BIGINT DEFAULT 1000000,
    last_active_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_user_profile_user_id ON user_profile(user_id);
CREATE INDEX idx_user_profile_username ON user_profile(username);
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS user_profile CASCADE;
-- +goose StatementEnd