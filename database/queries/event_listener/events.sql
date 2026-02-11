-- name: HandleOrderPlaced :exec
WITH inserted_order AS (
    INSERT INTO orders (
            id,
            user_id,
            bot_id,
            stock_ticker,
            order_type,
            side,
            quantity,
            remaining_quantity,
            limit_price_cents,
            status
        )
    VALUES ($1, $2, $3, $4, $5, $6, $7, $7, $8, 'PENDING')
    RETURNING id,
        user_id,
        bot_id,
        side,
        quantity,
        limit_price_cents,
        stock_ticker
),
-- For BUY orders: lock cash
lock_user_cash AS (
    UPDATE user_profile
    SET cash_balance_cents = cash_balance_cents - (io.quantity * io.limit_price_cents),
        cash_hold_cents = cash_hold_cents + (io.quantity * io.limit_price_cents),
        updated_at = NOW()
    FROM inserted_order io
    WHERE user_profile.user_id = io.user_id
        AND io.side = 'BUY'
        AND io.user_id IS NOT NULL
),
lock_bot_cash AS (
    UPDATE bots
    SET cash_balance_cents = cash_balance_cents - (io.quantity * io.limit_price_cents),
        cash_hold_cents = cash_hold_cents + (io.quantity * io.limit_price_cents),
        updated_at = NOW()
    FROM inserted_order io
    WHERE bots.id = io.bot_id
        AND io.side = 'BUY'
        AND io.bot_id IS NOT NULL
),
-- For SELL orders: lock shares (reduce available, increase hold)
lock_user_shares AS (
    UPDATE positions
    SET quantity = quantity - io.quantity,
        quantity_hold = quantity_hold + io.quantity,
        updated_at = NOW()
    FROM inserted_order io
    WHERE positions.user_id = io.user_id
        AND positions.stock_ticker = io.stock_ticker
        AND io.side = 'SELL'
        AND io.user_id IS NOT NULL
),
lock_bot_shares AS (
    UPDATE positions
    SET quantity = quantity - io.quantity,
        quantity_hold = quantity_hold + io.quantity,
        updated_at = NOW()
    FROM inserted_order io
    WHERE positions.bot_id = io.bot_id
        AND positions.stock_ticker = io.stock_ticker
        AND io.side = 'SELL'
        AND io.bot_id IS NOT NULL
)
SELECT 1;
-- name: HandleTradeExecuted :exec
WITH trade_info AS (
    INSERT INTO trades (
            id,
            stock_ticker,
            buyer_order_id,
            seller_order_id,
            buyer_user_id,
            buyer_bot_id,
            seller_user_id,
            seller_bot_id,
            quantity,
            price_cents,
            total_value_cents
        )
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
    RETURNING buyer_user_id,
        buyer_bot_id,
        seller_user_id,
        seller_bot_id,
        stock_ticker,
        quantity,
        price_cents,
        total_value_cents
),
-- BUYER: Release cash from hold (it's being spent now)
release_buyer_user_cash_hold AS (
    UPDATE user_profile
    SET cash_hold_cents = cash_hold_cents - ti.total_value_cents,
        updated_at = NOW()
    FROM trade_info ti
    WHERE user_profile.user_id = ti.buyer_user_id
        AND ti.buyer_user_id IS NOT NULL
),
release_buyer_bot_cash_hold AS (
    UPDATE bots
    SET cash_hold_cents = cash_hold_cents - ti.total_value_cents,
        updated_at = NOW()
    FROM trade_info ti
    WHERE bots.id = ti.buyer_bot_id
        AND ti.buyer_bot_id IS NOT NULL
),
-- BUYER: Add shares to position
buyer_user_add_position AS (
    INSERT INTO positions (
            user_id,
            stock_ticker,
            quantity,
            average_cost_cents,
            total_cost_cents
        )
    SELECT ti.buyer_user_id,
        ti.stock_ticker,
        ti.quantity,
        ti.price_cents,
        ti.total_value_cents
    FROM trade_info ti
    WHERE ti.buyer_user_id IS NOT NULL ON CONFLICT (user_id, stock_ticker)
    WHERE user_id IS NOT NULL DO
    UPDATE
    SET quantity = positions.quantity + EXCLUDED.quantity,
        total_cost_cents = positions.total_cost_cents + EXCLUDED.total_cost_cents,
        average_cost_cents = (
            positions.total_cost_cents + EXCLUDED.total_cost_cents
        ) / (positions.quantity + EXCLUDED.quantity),
        updated_at = NOW()
),
buyer_bot_add_position AS (
    INSERT INTO positions (
            bot_id,
            stock_ticker,
            quantity,
            average_cost_cents,
            total_cost_cents
        )
    SELECT ti.buyer_bot_id,
        ti.stock_ticker,
        ti.quantity,
        ti.price_cents,
        ti.total_value_cents
    FROM trade_info ti
    WHERE ti.buyer_bot_id IS NOT NULL ON CONFLICT (bot_id, stock_ticker)
    WHERE bot_id IS NOT NULL DO
    UPDATE
    SET quantity = positions.quantity + EXCLUDED.quantity,
        total_cost_cents = positions.total_cost_cents + EXCLUDED.total_cost_cents,
        average_cost_cents = (
            positions.total_cost_cents + EXCLUDED.total_cost_cents
        ) / (positions.quantity + EXCLUDED.quantity),
        updated_at = NOW()
),
-- SELLER: Release quantity_hold (shares were already in hold, now they're gone)
seller_user_release_hold AS (
    UPDATE positions
    SET quantity_hold = quantity_hold - ti.quantity,
        total_cost_cents = GREATEST(
            0,
            total_cost_cents - (ti.quantity * average_cost_cents)
        ),
        updated_at = NOW()
    FROM trade_info ti
    WHERE positions.user_id = ti.seller_user_id
        AND positions.stock_ticker = ti.stock_ticker
        AND ti.seller_user_id IS NOT NULL
),
seller_bot_release_hold AS (
    UPDATE positions
    SET quantity_hold = quantity_hold - ti.quantity,
        total_cost_cents = GREATEST(
            0,
            total_cost_cents - (ti.quantity * average_cost_cents)
        ),
        updated_at = NOW()
    FROM trade_info ti
    WHERE positions.bot_id = ti.seller_bot_id
        AND positions.stock_ticker = ti.stock_ticker
        AND ti.seller_bot_id IS NOT NULL
),
-- SELLER: Add cash from sale
seller_user_add_cash AS (
    UPDATE user_profile
    SET cash_balance_cents = cash_balance_cents + ti.total_value_cents,
        updated_at = NOW()
    FROM trade_info ti
    WHERE user_profile.user_id = ti.seller_user_id
        AND ti.seller_user_id IS NOT NULL
),
seller_bot_add_cash AS (
    UPDATE bots
    SET cash_balance_cents = cash_balance_cents + ti.total_value_cents,
        updated_at = NOW()
    FROM trade_info ti
    WHERE bots.id = ti.seller_bot_id
        AND ti.seller_bot_id IS NOT NULL
),
-- Cleanup: Delete empty positions
delete_empty_positions AS (
    DELETE FROM positions
    WHERE quantity = 0
        AND quantity_hold = 0
)
SELECT 1;
-- name: HandleOrderFilled :exec
UPDATE orders
SET status = 'FILLED',
    filled_quantity = quantity,
    remaining_quantity = 0,
    filled_at = NOW(),
    updated_at = NOW()
WHERE id = $1;
-- name: HandleOrderPartiallyFilled :exec
UPDATE orders
SET filled_quantity = $2,
    remaining_quantity = $3,
    status = 'PARTIAL',
    updated_at = NOW()
WHERE id = $1;
-- name: HandleOrderCancelled :exec
WITH cancelled_order AS (
    UPDATE orders
    SET status = 'CANCELLED',
        cancelled_at = NOW(),
        updated_at = NOW()
    WHERE orders.id = $1
        AND status IN ('PENDING', 'PARTIAL')
    RETURNING id,
        user_id,
        bot_id,
        side,
        remaining_quantity,
        limit_price_cents,
        stock_ticker
),
-- For BUY orders: release cash hold
return_user_cash AS (
    UPDATE user_profile
    SET cash_balance_cents = cash_balance_cents + (co.remaining_quantity * co.limit_price_cents),
        cash_hold_cents = cash_hold_cents - (co.remaining_quantity * co.limit_price_cents),
        updated_at = NOW()
    FROM cancelled_order co
    WHERE user_profile.user_id = co.user_id
        AND co.side = 'BUY'
        AND co.user_id IS NOT NULL
),
return_bot_cash AS (
    UPDATE bots
    SET cash_balance_cents = cash_balance_cents + (co.remaining_quantity * co.limit_price_cents),
        cash_hold_cents = cash_hold_cents - (co.remaining_quantity * co.limit_price_cents),
        updated_at = NOW()
    FROM cancelled_order co
    WHERE bots.id = co.bot_id
        AND co.side = 'BUY'
        AND co.bot_id IS NOT NULL
),
-- For SELL orders: release share hold
return_user_shares AS (
    UPDATE positions
    SET quantity = quantity + co.remaining_quantity,
        quantity_hold = quantity_hold - co.remaining_quantity,
        updated_at = NOW()
    FROM cancelled_order co
    WHERE positions.user_id = co.user_id
        AND positions.stock_ticker = co.stock_ticker
        AND co.side = 'SELL'
        AND co.user_id IS NOT NULL
),
return_bot_shares AS (
    UPDATE positions
    SET quantity = quantity + co.remaining_quantity,
        quantity_hold = quantity_hold - co.remaining_quantity,
        updated_at = NOW()
    FROM cancelled_order co
    WHERE positions.bot_id = co.bot_id
        AND positions.stock_ticker = co.stock_ticker
        AND co.side = 'SELL'
        AND co.bot_id IS NOT NULL
)
SELECT 1;
-- name: HandleOrderRejected :exec
INSERT INTO orders (
        id,
        user_id,
        bot_id,
        stock_ticker,
        order_type,
        side,
        quantity,
        remaining_quantity,
        limit_price_cents,
        status
    )
VALUES ($1, $2, $3, $4, $5, $6, $7, 0, $8, 'REJECTED');