-- name: HandleLimitBuyOrderPlaced :exec
WITH inserted_order AS (
    INSERT INTO orders (
            id,
            trader_id,
            stock_ticker,
            order_type,
            side,
            quantity,
            remaining_quantity,
            limit_price_cents,
            status
        )
    VALUES (
            $1,
            $2,
            $3,
            'LIMIT',
            'BUY',
            $4,
            $4,
            $5,
            'PENDING'
        )
    RETURNING id,
        trader_id,
        quantity,
        limit_price_cents
),
-- Lock cash at limit price
lock_trader_cash AS (
    UPDATE traders
    SET cash_balance_cents = traders.cash_balance_cents - (io.quantity * io.limit_price_cents),
        cash_hold_cents = traders.cash_hold_cents + (io.quantity * io.limit_price_cents),
        updated_at = NOW()
    FROM inserted_order io
    WHERE traders.id = io.trader_id
)
SELECT 1;
-- name: HandleMarketBuyOrderPlaced :exec
INSERT INTO orders (
        id,
        trader_id,
        stock_ticker,
        order_type,
        side,
        quantity,
        remaining_quantity,
        limit_price_cents,
        status
    )
VALUES (
        $1,
        $2,
        $3,
        'MARKET',
        'BUY',
        $4,
        $4,
        NULL,
        'PENDING'
    );
-- name: HandleSellOrderPlaced :exec
WITH inserted_order AS (
    INSERT INTO orders (
            id,
            trader_id,
            stock_ticker,
            order_type,
            side,
            quantity,
            remaining_quantity,
            limit_price_cents,
            status
        )
    VALUES ($1, $2, $3, $4, 'SELL', $5, $5, $6, 'PENDING')
    RETURNING id,
        trader_id,
        stock_ticker,
        quantity
),
-- Lock shares for sell
lock_trader_shares AS (
    UPDATE positions
    SET quantity = positions.quantity - io.quantity,
        quantity_hold = positions.quantity_hold + io.quantity,
        updated_at = NOW()
    FROM inserted_order io
    WHERE positions.trader_id = io.trader_id
        AND positions.stock_ticker = io.stock_ticker
)
SELECT 1;
-- name: HandleLimitBuyTradeExecuted :exec
WITH trade_info AS (
    INSERT INTO trades (
            stock_ticker,
            buyer_order_id,
            seller_order_id,
            buyer_trader_id,
            seller_trader_id,
            quantity,
            price_cents,
            total_value_cents
        )
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    RETURNING buyer_order_id,
        seller_order_id,
        buyer_trader_id,
        seller_trader_id,
        stock_ticker,
        quantity,
        price_cents,
        total_value_cents
),
-- Get buyer's limit price for hold release calculation
buyer_order AS (
    SELECT o.limit_price_cents
    FROM orders o
        INNER JOIN trade_info ti ON o.id = ti.buyer_order_id
),
-- Release buyer's cash hold at limit price, refund price improvement
release_buyer_cash_hold AS (
    UPDATE traders
    SET cash_hold_cents = traders.cash_hold_cents - (ti.quantity * bo.limit_price_cents),
        cash_balance_cents = traders.cash_balance_cents + (
            (ti.quantity * bo.limit_price_cents) - ti.total_value_cents
        ),
        updated_at = NOW()
    FROM trade_info ti
        CROSS JOIN buyer_order bo
    WHERE traders.id = ti.buyer_trader_id
),
-- Add shares to buyer's position
buyer_add_position AS (
    INSERT INTO positions (
            trader_id,
            stock_ticker,
            quantity,
            total_cost_cents
        )
    SELECT ti.buyer_trader_id,
        ti.stock_ticker,
        ti.quantity,
        ti.total_value_cents
    FROM trade_info ti ON CONFLICT (trader_id, stock_ticker) DO
    UPDATE
    SET quantity = positions.quantity + EXCLUDED.quantity,
        total_cost_cents = positions.total_cost_cents + EXCLUDED.total_cost_cents,
        updated_at = NOW()
),
-- Release seller's share hold
seller_release_hold AS (
    UPDATE positions
    SET quantity_hold = positions.quantity_hold - ti.quantity,
        total_cost_cents = GREATEST(
            0,
            positions.total_cost_cents - (
                (positions.total_cost_cents * ti.quantity) / NULLIF(positions.quantity, 0)
            )
        ),
        updated_at = NOW()
    FROM trade_info ti
    WHERE positions.trader_id = ti.seller_trader_id
        AND positions.stock_ticker = ti.stock_ticker
),
-- Add cash to seller
seller_add_cash AS (
    UPDATE traders
    SET cash_balance_cents = traders.cash_balance_cents + ti.total_value_cents,
        updated_at = NOW()
    FROM trade_info ti
    WHERE traders.id = ti.seller_trader_id
),
-- Update stock price
update_stock_price AS (
    UPDATE stocks
    SET current_price_cents = ti.price_cents,
        updated_at = NOW()
    FROM trade_info ti
    WHERE stocks.ticker = ti.stock_ticker
),
-- Update buyer's portfolio value
update_buyer_portfolio AS (
    SELECT update_trader_portfolio_value(ti.buyer_trader_id)
    FROM trade_info ti
),
-- Update seller's portfolio value
update_seller_portfolio AS (
    SELECT update_trader_portfolio_value(ti.seller_trader_id)
    FROM trade_info ti
)
SELECT 1;
-- name: HandleMarketBuyTradeExecuted :exec
WITH trade_info AS (
    INSERT INTO trades (
            stock_ticker,
            buyer_order_id,
            seller_order_id,
            buyer_trader_id,
            seller_trader_id,
            quantity,
            price_cents,
            total_value_cents
        )
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    RETURNING buyer_order_id,
        seller_order_id,
        buyer_trader_id,
        seller_trader_id,
        stock_ticker,
        quantity,
        price_cents,
        total_value_cents
),
-- Deduct cash directly from buyer's balance (no hold exists)
deduct_buyer_cash AS (
    UPDATE traders
    SET cash_balance_cents = traders.cash_balance_cents - ti.total_value_cents,
        updated_at = NOW()
    FROM trade_info ti
    WHERE traders.id = ti.buyer_trader_id
),
-- Add shares to buyer's position
buyer_add_position AS (
    INSERT INTO positions (
            trader_id,
            stock_ticker,
            quantity,
            total_cost_cents
        )
    SELECT ti.buyer_trader_id,
        ti.stock_ticker,
        ti.quantity,
        ti.total_value_cents
    FROM trade_info ti ON CONFLICT (trader_id, stock_ticker) DO
    UPDATE
    SET quantity = positions.quantity + EXCLUDED.quantity,
        total_cost_cents = positions.total_cost_cents + EXCLUDED.total_cost_cents,
        updated_at = NOW()
),
-- Release seller's share hold
seller_release_hold AS (
    UPDATE positions
    SET quantity_hold = positions.quantity_hold - ti.quantity,
        total_cost_cents = GREATEST(
            0,
            positions.total_cost_cents - (
                (positions.total_cost_cents * ti.quantity) / NULLIF(positions.quantity, 0)
            )
        ),
        updated_at = NOW()
    FROM trade_info ti
    WHERE positions.trader_id = ti.seller_trader_id
        AND positions.stock_ticker = ti.stock_ticker
),
-- Add cash to seller
seller_add_cash AS (
    UPDATE traders
    SET cash_balance_cents = traders.cash_balance_cents + ti.total_value_cents,
        updated_at = NOW()
    FROM trade_info ti
    WHERE traders.id = ti.seller_trader_id
),
-- Update stock price
update_stock_price AS (
    UPDATE stocks
    SET current_price_cents = ti.price_cents,
        updated_at = NOW()
    FROM trade_info ti
    WHERE stocks.ticker = ti.stock_ticker
),
-- Update buyer's portfolio value
update_buyer_portfolio AS (
    SELECT update_trader_portfolio_value(ti.buyer_trader_id)
    FROM trade_info ti
),
-- Update seller's portfolio value
update_seller_portfolio AS (
    SELECT update_trader_portfolio_value(ti.seller_trader_id)
    FROM trade_info ti
)
SELECT 1;
-- name: HandleOrderFilled :exec
UPDATE orders
SET status = 'FILLED',
    filled_quantity = orders.quantity,
    remaining_quantity = 0,
    filled_at = NOW(),
    updated_at = NOW()
WHERE orders.id = $1;
-- name: HandleOrderPartiallyFilled :exec
UPDATE orders
SET filled_quantity = orders.filled_quantity + $2,
    remaining_quantity = orders.remaining_quantity - $2,
    status = 'PARTIAL',
    updated_at = NOW()
WHERE orders.id = $1;
-- name: HandleLimitBuyOrderCancelled :exec
WITH cancelled_order AS (
    UPDATE orders
    SET status = 'CANCELLED',
        cancelled_at = NOW(),
        updated_at = NOW()
    WHERE orders.id = $1
        AND status IN ('PENDING', 'PARTIAL')
    RETURNING id,
        trader_id,
        remaining_quantity,
        limit_price_cents
),
-- Release cash hold for limit buy
return_trader_cash AS (
    UPDATE traders
    SET cash_balance_cents = traders.cash_balance_cents + (co.remaining_quantity * co.limit_price_cents),
        cash_hold_cents = traders.cash_hold_cents - (co.remaining_quantity * co.limit_price_cents),
        updated_at = NOW()
    FROM cancelled_order co
    WHERE traders.id = co.trader_id
)
SELECT 1;
-- name: HandleMarketBuyOrderCancelled :exec
UPDATE orders
SET status = 'CANCELLED',
    cancelled_at = NOW(),
    updated_at = NOW()
WHERE orders.id = $1
    AND status IN ('PENDING', 'PARTIAL');
-- name: HandleSellOrderCancelled :exec
WITH cancelled_order AS (
    UPDATE orders
    SET status = 'CANCELLED',
        cancelled_at = NOW(),
        updated_at = NOW()
    WHERE orders.id = $1
        AND status IN ('PENDING', 'PARTIAL')
    RETURNING id,
        trader_id,
        stock_ticker,
        remaining_quantity
),
-- Release share hold for sell orders
return_trader_shares AS (
    UPDATE positions
    SET quantity = positions.quantity + co.remaining_quantity,
        quantity_hold = positions.quantity_hold - co.remaining_quantity,
        updated_at = NOW()
    FROM cancelled_order co
    WHERE positions.trader_id = co.trader_id
        AND positions.stock_ticker = co.stock_ticker
)
SELECT 1;
-- name: HandleOrderRejected :exec
INSERT INTO orders (
        id,
        trader_id,
        status
    )
VALUES ($1, $2, 'REJECTED');