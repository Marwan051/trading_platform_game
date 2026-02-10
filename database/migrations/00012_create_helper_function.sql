-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_user_portfolio_value(p_user_id TEXT) RETURNS BIGINT AS $
DECLARE v_portfolio_value_cents BIGINT;
BEGIN
SELECT COALESCE(SUM(up.quantity * s.current_price_cents), 0) INTO v_portfolio_value_cents
FROM positions up
    JOIN stocks s ON up.stock_id = s.id
WHERE up.user_id = p_user_id;
UPDATE user_profile
SET total_portfolio_value_cents = cash_balance_cents + v_portfolio_value_cents,
    updated_at = NOW()
WHERE user_id = p_user_id;
RETURN v_portfolio_value_cents;
END;
$ LANGUAGE plpgsql;
-- Function to update bot portfolio value
CREATE OR REPLACE FUNCTION update_bot_portfolio_value(p_bot_id UUID) RETURNS BIGINT AS $
DECLARE v_portfolio_value_cents BIGINT;
BEGIN
SELECT COALESCE(SUM(up.quantity * s.current_price_cents), 0) INTO v_portfolio_value_cents
FROM user_positions up
    JOIN stocks s ON up.stock_id = s.id
WHERE up.bot_id = p_bot_id;
UPDATE bots
SET total_portfolio_value_cents = cash_balance_cents + v_portfolio_value_cents,
    updated_at = NOW()
WHERE id = p_bot_id;
RETURN v_portfolio_value_cents;
END;
$ LANGUAGE plpgsql;
-- Function to get order book depth
CREATE OR REPLACE FUNCTION get_order_book(p_stock_id UUID, p_depth INTEGER DEFAULT 10) RETURNS TABLE(
        side TEXT,
        price_cents BIGINT,
        quantity BIGINT
    ) AS $ BEGIN RETURN QUERY (
        SELECT 'BUY'::TEXT,
            o.limit_price_cents,
            SUM(o.remaining_quantity)
        FROM orders o
        WHERE o.stock_id = p_stock_id
            AND o.side = 'BUY'
            AND o.status IN ('PENDING', 'PARTIAL')
            AND o.order_type = 'LIMIT'
        GROUP BY o.limit_price_cents
        ORDER BY o.limit_price_cents DESC
        LIMIT p_depth
    )
UNION ALL
(
    SELECT 'SELL'::TEXT,
        o.limit_price_cents,
        SUM(o.remaining_quantity)
    FROM orders o
    WHERE o.stock_id = p_stock_id
        AND o.side = 'SELL'
        AND o.status IN ('PENDING', 'PARTIAL')
        AND o.order_type = 'LIMIT'
    GROUP BY o.limit_price_cents
    ORDER BY o.limit_price_cents ASC
    LIMIT p_depth
);
END;
$ LANGUAGE plpgsql;
-- Function to check if bot should trade based on schedule
CREATE OR REPLACE FUNCTION bot_can_trade_now(p_bot_id UUID) RETURNS BOOLEAN AS $
DECLARE v_bot RECORD;
v_current_time TIME;
v_time_since_last_trade INTERVAL;
BEGIN
SELECT * INTO v_bot
FROM bots
WHERE id = p_bot_id;
IF NOT FOUND
OR NOT v_bot.is_active THEN RETURN FALSE;
END IF;
v_current_time := CURRENT_TIME;
IF v_current_time < v_bot.trading_hours_start
OR v_current_time > v_bot.trading_hours_end THEN RETURN FALSE;
END IF;
IF v_bot.last_trade_at IS NOT NULL THEN v_time_since_last_trade := NOW() - v_bot.last_trade_at;
IF EXTRACT(
    EPOCH
    FROM v_time_since_last_trade
) < v_bot.min_trade_interval_seconds THEN RETURN FALSE;
END IF;
END IF;
RETURN TRUE;
END;
$ LANGUAGE plpgsql;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP FUNCTION IF EXISTS update_user_portfolio_value(TEXT) CASCADE;
DROP FUNCTION IF EXISTS update_bot_portfolio_value(UUID) CASCADE;
DROP FUNCTION IF EXISTS get_order_book(UUID, INTEGER) CASCADE;
DROP FUNCTION IF EXISTS bot_can_trade_now(UUID) CASCADE;
-- +goose StatementEnd