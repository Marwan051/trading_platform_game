# Event Listener Service

The event listener acts as a bridge between the matching engine and the database. It consumes events from the Valkey stream, processes them, and persists them to PostgreSQL.

## Architecture

### Key Components

- **Streaming Client**: Manages connection to the stream and consumes events from the stream
- **Database Layer**: Uses SQLC-generated queries for type-safe database operations

## Configuration

The service is configured via environment variables:

| Variable             | Description                  | Default                                                                        |
| -------------------- | ---------------------------- | ------------------------------------------------------------------------------ |
| `ENVIRONMENT`        | Deployment environment       | `development`                                                                  |
| `SHUTDOWN_TIMEOUT`   | Graceful shutdown duration   | `30s`                                                                          |
| `VALKEY_HOST`        | Valkey server hostname       | `localhost`                                                                    |
| `VALKEY_PORT`        | Valkey server port           | `6379`                                                                         |
| `VALKEY_STREAM_NAME` | Stream name to consume from  | `matching_engine_stream`                                                       |
| `DATABASE_URL`       | PostgreSQL connection string | `postgres://postgres:postgres@localhost:5432/trading_platform?sslmode=disable` |

## Development

### Prerequisites

- Go 1.25 or later
- PostgreSQL 14+ with TimescaleDB extension
- Valkey (Redis-compatible) server

### Local Setup

1. **Install dependencies:**

   ```bash
   go mod download
   ```

2. **Start dependencies:**

   ```bash
   docker-compose up -d postgres valkey
   ```

3. **Run migrations:**

   ```bash
   cd ../database && make migrate-up
   ```

4. **Start the service:**
   ```bash
   go run cmd/server/main.go
   ```

### Running Tests

```bash
go test ./... -v
```

### Code Generation

If database schema changes, regenerate SQLC queries:

```bash
sqlc generate
```

## Deployment

### Docker

Build the container:

```bash
docker build -t event-listener:latest .
```

Run with environment variables:

```bash
docker run --rm \
  -e DATABASE_URL=postgres://user:pass@db:5432/trading_platform \
  -e VALKEY_HOST=valkey \
  -e VALKEY_PORT=6379 \
  event-listener:latest
```

### Docker Compose

The service is included in the root `docker-compose.yml`:

```bash
docker-compose up event_listener
```
