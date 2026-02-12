-- +goose Up
-- +goose StatementBegin
-- Seed sentinel trader for orphaned bots (when owner is deleted)
INSERT INTO traders (
        id,
        trader_type,
        display_name,
        cash_balance_cents,
        total_portfolio_value_cents,
        is_active
    ) OVERRIDING SYSTEM VALUE
VALUES (
        -1,
        'BOT',
        '_ORPHANED_BOTS_OWNER_',
        0,
        0,
        FALSE
    );
-- Seed demo users
INSERT INTO traders (
        trader_type,
        auth_user_id,
        display_name,
        cash_balance_cents,
        total_portfolio_value_cents
    )
VALUES (
        'USER',
        'demo_user_1',
        'AliceTrader',
        10000000,
        10000000
    ),
    -- $100,000
    (
        'USER',
        'demo_user_2',
        'BobInvestor',
        5000000,
        5000000
    ),
    -- $50,000
    (
        'USER',
        'demo_user_3',
        'CharlieHODL',
        25000000,
        25000000
    ),
    -- $250,000
    (
        'USER',
        'demo_user_4',
        'DianaDay',
        15000000,
        15000000
    ),
    -- $150,000
    (
        'USER',
        'demo_user_5',
        'EthanSwing',
        7500000,
        7500000
    ),
    -- $75,000
    (
        'USER',
        'demo_user_6',
        'FionaValue',
        20000000,
        20000000
    ),
    -- $200,000
    (
        'USER',
        'demo_user_7',
        'GeorgeQuant',
        30000000,
        30000000
    ),
    -- $300,000
    (
        'USER',
        'demo_user_8',
        'HannahScalp',
        5000000,
        5000000
    );
-- $50,000
-- Seed trading bots with different strategies
INSERT INTO traders (
        trader_type,
        owner_trader_id,
        display_name,
        cash_balance_cents,
        total_portfolio_value_cents
    )
SELECT 'BOT',
    owner.id,
    seeds.display_name,
    seeds.cash_balance_cents,
    seeds.total_portfolio_value_cents
FROM (
        VALUES -- System bots (no owner)
            (
                NULL,
                'MarketMaker_Alpha',
                50000000,
                50000000
            ),
            -- $500,000
            (
                NULL,
                'MarketMaker_Beta',
                50000000,
                50000000
            ),
            -- $500,000
            (
                NULL,
                'TrendBot_Gamma',
                25000000,
                25000000
            ),
            -- $250,000
            (
                NULL,
                'ValueBot_Delta',
                30000000,
                30000000
            ),
            -- $300,000
            (
                NULL,
                'ContraBotEpsilon',
                20000000,
                20000000
            ),
            -- $200,000
            -- User-owned bots
            (
                'demo_user_1',
                'Alice_Bot_1',
                10000000,
                10000000
            ),
            -- $100,000
            (
                'demo_user_3',
                'Charlie_Bot_1',
                15000000,
                15000000
            ),
            -- $150,000
            (
                'demo_user_3',
                'Charlie_Bot_2',
                10000000,
                10000000
            ),
            -- $100,000
            (
                'demo_user_6',
                'Fiona_Bot_1',
                20000000,
                20000000
            ),
            -- $200,000
            (
                'demo_user_7',
                'George_Bot_1',
                25000000,
                25000000
            )
    ) AS seeds(
        owner_auth_user_id,
        display_name,
        cash_balance_cents,
        total_portfolio_value_cents
    )
    LEFT JOIN traders owner ON owner.auth_user_id = seeds.owner_auth_user_id;
-- $250,000
-- Seed stocks from various sectors
INSERT INTO stocks (
        ticker,
        company_name,
        sector,
        description,
        current_price_cents,
        previous_close_cents,
        total_shares,
        is_active
    )
VALUES -- Technology Sector
    (
        'TECH',
        'TechCorp Inc',
        'Technology',
        'Leading cloud computing and AI services provider',
        15000,
        14800,
        10000000,
        TRUE
    ),
    (
        'NOVA',
        'NovaSoft Systems',
        'Technology',
        'Enterprise software and cybersecurity solutions',
        8500,
        8450,
        5000000,
        TRUE
    ),
    (
        'DIGI',
        'Digital Ventures',
        'Technology',
        'Mobile app development and digital services',
        4200,
        4100,
        8000000,
        TRUE
    ),
    -- Financial Sector
    (
        'FIN',
        'FinanceFirst Bank',
        'Financial',
        'Global investment banking and wealth management',
        12000,
        12100,
        15000000,
        TRUE
    ),
    (
        'PAY',
        'PayFlow Systems',
        'Financial',
        'Digital payment processing and fintech solutions',
        9800,
        9750,
        7000000,
        TRUE
    ),
    -- Healthcare Sector
    (
        'HEAL',
        'HealthPlus Medical',
        'Healthcare',
        'Pharmaceutical research and medical devices',
        18500,
        18200,
        6000000,
        TRUE
    ),
    (
        'BIO',
        'BioGenesis Labs',
        'Healthcare',
        'Biotechnology and gene therapy innovations',
        22000,
        21800,
        4000000,
        TRUE
    ),
    -- Energy Sector
    (
        'ENRG',
        'EnergyCore Resources',
        'Energy',
        'Renewable energy and solar power systems',
        11500,
        11600,
        12000000,
        TRUE
    ),
    (
        'FUEL',
        'FuelTech Solutions',
        'Energy',
        'Advanced battery technology and energy storage',
        6700,
        6650,
        9000000,
        TRUE
    ),
    -- Consumer Goods
    (
        'SHOP',
        'ShopSmart Retail',
        'Consumer',
        'E-commerce platform and retail chain',
        7800,
        7750,
        20000000,
        TRUE
    ),
    (
        'FOOD',
        'FoodFresh Corp',
        'Consumer',
        'Organic food products and sustainable agriculture',
        5400,
        5350,
        15000000,
        TRUE
    ),
    -- Industrial Sector
    (
        'MANU',
        'Manufacturing United',
        'Industrial',
        'Advanced manufacturing and automation systems',
        9200,
        9300,
        8000000,
        TRUE
    ),
    (
        'AUTO',
        'AutoDrive Systems',
        'Industrial',
        'Electric vehicle components and autonomous tech',
        14500,
        14400,
        6000000,
        TRUE
    ),
    -- Real Estate
    (
        'PROP',
        'PropertyPro REIT',
        'Real Estate',
        'Commercial and residential real estate investment',
        10500,
        10450,
        10000000,
        TRUE
    ),
    -- Telecommunications
    (
        'LINK',
        'LinkNet Telecom',
        'Telecom',
        '5G infrastructure and telecommunications services',
        8900,
        8850,
        12000000,
        TRUE
    );
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
-- Remove seed data in reverse order (respecting foreign keys)
DELETE FROM stocks
WHERE ticker IN (
        'TECH',
        'NOVA',
        'DIGI',
        'FIN',
        'PAY',
        'HEAL',
        'BIO',
        'ENRG',
        'FUEL',
        'SHOP',
        'FOOD',
        'MANU',
        'AUTO',
        'PROP',
        'LINK'
    );
DELETE FROM traders
WHERE display_name IN (
        'MarketMaker_Alpha',
        'MarketMaker_Beta',
        'TrendBot_Gamma',
        'ValueBot_Delta',
        'ContraBotEpsilon',
        'Alice_Bot_1',
        'Charlie_Bot_1',
        'Charlie_Bot_2',
        'Fiona_Bot_1',
        'George_Bot_1'
    );
DELETE FROM traders
WHERE auth_user_id IN (
        'demo_user_1',
        'demo_user_2',
        'demo_user_3',
        'demo_user_4',
        'demo_user_5',
        'demo_user_6',
        'demo_user_7',
        'demo_user_8'
    );
-- +goose StatementEnd