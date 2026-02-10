-- +goose Up
-- +goose StatementBegin
CREATE TABLE transaction_log (
    id UUID DEFAULT gen_random_uuid(),
    user_id TEXT,
    -- References Better Auth user.id
    bot_id UUID REFERENCES bots(id) ON DELETE CASCADE,
    transaction_type TEXT NOT NULL CHECK (
        transaction_type IN (
            'DEPOSIT',
            'WITHDRAWAL',
            'BUY',
            'SELL',
            'DIVIDEND',
            'FEE'
        )
    ),
    amount_cents BIGINT NOT NULL,
    balance_after_cents BIGINT NOT NULL,
    reference_id UUID,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at),
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
SELECT create_hypertable(
        'transaction_log',
        'created_at',
        chunk_time_interval => INTERVAL '30 days',
        if_not_exists => TRUE
    );
CREATE INDEX idx_transaction_log_user ON transaction_log(user_id, created_at DESC)
WHERE user_id IS NOT NULL;
CREATE INDEX idx_transaction_log_bot ON transaction_log(bot_id, created_at DESC)
WHERE bot_id IS NOT NULL;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS transaction_log CASCADE;
-- +goose StatementEnd