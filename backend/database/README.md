# Database Migrations

This folder contains PostgreSQL migrations for the Trading Platform Game using [Goose](https://github.com/pressly/goose).

## Structure

```
database/
├── Dockerfile.migrator    # Docker image for running migrations
├── migrations/            # SQL migration files
│   ├── 00001_create_extensions.sql
│   ├── 00002_create_users_table.sql
│   └── ...
└── README.md
```

## Running Migrations

### With Docker Compose (Recommended)

From the project root:

```bash
# Start database and run migrations
docker compose up db migrator

# Run all services (db + migrations + backend)
docker compose up

# Reset database and re-run migrations
docker compose down -v
docker compose up
```

### Locally with Goose CLI

Install goose:

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
```

Run migrations:

```bash
cd backend/database/migrations

# Apply all pending migrations
goose postgres "postgres://trading:trading_secret@localhost:5432/trading_game?sslmode=disable" up

# Rollback last migration
goose postgres "postgres://trading:trading_secret@localhost:5432/trading_game?sslmode=disable" down

# Check migration status
goose postgres "postgres://trading:trading_secret@localhost:5432/trading_game?sslmode=disable" status
```

## Creating New Migrations

```bash
cd backend/database/migrations
goose create -s <migration_name> sql
```

This creates a new file like `00015_<migration_name>.sql` with Up/Down stubs.

## Migration Guidelines

1. **Always include Down migrations** - Use `DROP TABLE IF EXISTS ... CASCADE`
2. **Use transactions** - Wrap complex migrations in `-- +goose StatementBegin` / `-- +goose StatementEnd`
3. **Test both up and down** - Verify rollbacks work correctly
4. **Keep migrations small** - One logical change per migration

## sqlc Integration

After modifying the schema, regenerate Go code:

```bash
cd backend
sqlc generate
```

Ensure your `sqlc.yaml` points to the migrations folder for schema discovery.
