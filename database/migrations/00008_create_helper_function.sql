-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_trader_portfolio_value(p_trader_id BIGINT) RETURNS BIGINT AS $$
DECLARE v_portfolio_value_cents BIGINT;
BEGIN
SELECT COALESCE(SUM(p.quantity * s.current_price_cents), 0) INTO v_portfolio_value_cents
FROM positions p
    JOIN stocks s ON p.stock_ticker = s.ticker
WHERE p.trader_id = p_trader_id;
UPDATE traders
SET total_portfolio_value_cents = cash_balance_cents + v_portfolio_value_cents,
    updated_at = NOW()
WHERE id = p_trader_id;
RETURN v_portfolio_value_cents;
END;
$$ LANGUAGE plpgsql;
-- Function to get order book depth
CREATE OR REPLACE FUNCTION get_order_book(p_stock_ticker TEXT, p_depth INTEGER DEFAULT 10) RETURNS TABLE(
        side TEXT,
        price_cents BIGINT,
        quantity BIGINT
    ) AS $$ BEGIN RETURN QUERY (
        SELECT 'BUY'::TEXT,
            o.limit_price_cents,
            SUM(o.remaining_quantity)
        FROM orders o
        WHERE o.stock_ticker = p_stock_ticker
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
    WHERE o.stock_ticker = p_stock_ticker
        AND o.side = 'SELL'
        AND o.status IN ('PENDING', 'PARTIAL')
        AND o.order_type = 'LIMIT'
    GROUP BY o.limit_price_cents
    ORDER BY o.limit_price_cents ASC
    LIMIT p_depth
);
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP FUNCTION IF EXISTS update_trader_portfolio_value(BIGINT) CASCADE;
DROP FUNCTION IF EXISTS get_order_book(TEXT, INTEGER) CASCADE;
-- +goose StatementEnd