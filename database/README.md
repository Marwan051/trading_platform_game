# Database

PostgreSQL 18 with TimescaleDB extension for time-series data. Uses Goose for migrations and sqlc for type-safe query generation.

```
database/
├── migrations/          # Goose migration files (numbered)
├── queries/            # sqlc queries organized by service
│   ├── event_listener/
│   └── market_and_user_data/
├── Dockerfile
└── Dockerfile.migrator
```

## Stack

- PostgreSQL 18
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
- (Optional) Goose CLI for local migrations
- (Optional) sqlc CLI for query generation

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

## Migrations

### Naming Convention

Migrations follow the pattern: `{version}_{description}.sql`

Example: `00012_add_trader_statistics.sql`

### Creating a New Migration

```bash
cd database/migrations
goose create add_new_feature sql
```

Edit the generated file with both `-- +goose Up` and `-- +goose Down` sections.

### Migration Guidelines

- Always include rollback logic in `-- +goose Down` sections
- Test up and down migrations locally before committing
- Update seed data (00011) if schema changes affect it

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

## Development Workflow

### 1. Schema Changes

1. Create migration: `goose create feature_name sql`
2. Write up/down SQL
3. Test: `goose up` then `goose down`
4. Commit

### 2. Add Queries

1. Edit `.sql` file in `queries/`
2. Run `sqlc generate` in the service
3. Use generated functions in code

### 3. Testing

Run integration tests that use the database:

```bash
cd ../matching_engine && go test ./...
cd ../event_listener && go test ./...
```

### 4. Reset Database

```bash
docker compose down -v  # Remove volumes
docker compose up -d db migrator --build
```

## Connection Details

### Local Development

```
Host: localhost
Port: 5432
Database: trading_platform
User: postgres
Password: postgres
```

Connection string:

```
postgres://postgres:postgres@localhost:5432/trading_platform?sslmode=disable
```

### Docker Internal

Services within docker-compose use:

```
Host: db
Port: 5432
```

## Seed Data

Initial data is in `migrations/00011_seed_initial_data.sql` - sample stocks, test accounts, and market prices. Modify as needed.

## Common Operations

### View Current Schema

```bash
docker exec -it trading_db psql -U postgres -d trading_platform
\dt   # List tables
\dv   # List views
\d table_name  # Describe table
```

### Manual Queries

```bash
docker exec -it trading_db psql -U postgres -d trading_platform -c "SELECT * FROM traders LIMIT 5;"
```
