# Database

PostgreSQL with TimescaleDB extension for time-series data. Uses Goose for migrations and sqlc for type-safe query generation.

```
database/
├── migrations/          # Goose migration files (numbered)
├── queries/            # sqlc queries organized by service
│   ├── event_listener/
│   └── market_and_user_data/
├── Dockerfile
└── Dockerfile.migrator
```

## Technologies

- PostgreSQL
- TimescaleDB for time-series processing
- Goose for schema migrations
- sqlc for type-safe SQL → Go code generation

## Schema Overview

### Core Tables

- `traders` - User and bot trader accounts
- `stocks` - Available stocks with current prices
- `orders` - Buy/sell orders (LIMIT/MARKET)
- `positions` - Trader holdings per stock
- `trades` - Executed trades (hypertable partitioned by `executed_at`)

### Views

- `leaderboard` - Ranked traders by total portfolio value
- `price_1min` - 1-minute OHLCV continuous aggregate
- `price_1hour` - 1-hour OHLCV continuous aggregate

## Setup

### Prerequisites

- Docker
- (Optional) [Goose](https://github.com/pressly/goose) CLI for local migrations
- (Optional) [sqlc](https://github.com/sqlc-dev/sqlc) CLI for query generation

### Quick Start

The database starts automatically with the project:

```bash
docker compose up db migrator
```

This will:

1. Start the database container on port `5432`
2. Run all migrations via the migrator service
3. Create all tables, views, and seed initial data

### Manual Migration

Install Goose locally:

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
```

Run migrations manually:

```bash
cd database
export GOOSE_DRIVER=postgres
export GOOSE_DBSTRING="postgres://postgres:postgres@localhost:5432/trading_platform?sslmode=disable"

goose -dir migrations up        # Run all pending migrations
goose -dir migrations status    # Check migration status
goose -dir migrations version   # Show current version
```

## Query Development

### sqlc Configuration

Query files are organized by service:

- `queries/event_listener/events.sql` - Order placement, trade execution, cancellations
- `queries/market_and_user_data/*.sql` - Read queries for market data service

### Generating Go Code

Event listener queries:

```bash
cd ../event_listener
sqlc generate
```

Backend queries:

```bash
cd ../backend
sqlc generate
```

## Connection Details

| Env var      | Default                                                                      | Description                                           |
| ------------ | ---------------------------------------------------------------------------- | ----------------------------------------------------- |
| DB_HOST      | localhost                                                                    | Database host (use `db` for internal Docker services) |
| DB_PORT      | 5432                                                                         | Database port                                         |
| DB_NAME      | trading_platform                                                             | Database name                                         |
| DB_USER      | postgres                                                                     | Database user                                         |
| DB_PASSWORD  | postgres                                                                     | Database password                                     |
| DB_SSLMODE   | disable                                                                      | SSL mode for connection                               |
| DATABASE_URL | postgres://postgres:postgres@localhost:5432/trading_platform?sslmode=disable | Full connection string (local)                        |

## Seed Data

Initial data is in `migrations/00011_seed_initial_data.sql` - sample stocks, test accounts, and market prices. Modify as needed.

## Manual Queries

```bash
docker exec -it trading_db psql -U postgres -d trading_platform -c "SELECT * FROM traders LIMIT 5;"
```
