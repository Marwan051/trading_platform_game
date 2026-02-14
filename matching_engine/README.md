# Matching Engine

gRPC service that matches buy and sell orders using price-time priority. Maintains separate order books per stock and publishes trade events to a streaming service.

```
matching_engine/
├── cmd/server/             # Server entry point
├── internal/
│   ├── config/            # Environment configuration
│   ├── interceptors/      # gRPC middleware
│   ├── lib/
│   │   ├── events/        # Event publishing
│   │   ├── matching_engine/ # Core matching logic
│   │   └── types/         # Order and event types
│   ├── server/            # gRPC server setup
│   └── service/           # gRPC service implementation
└── Dockerfile
```

## How It Works

### Order Matching

- **Price-Time Priority** - Better prices execute first, earlier orders break ties
- **Continuous Matching** - Orders match immediately when opposite side is available
- **Partial Fills** - Large orders can match against multiple smaller orders
- **Order Types** - LIMIT (specific price) and MARKET (best available price)

### Order Book Structure

Each stock maintains separate buy/sell queues:

- Buy orders sorted by price (descending), then time (ascending)
- Sell orders sorted by price (ascending), then time (ascending)
- Thread-safe concurrent access with sync.Map per stock

### Event Publishing

All order lifecycle events stream to Valkey:

- `OrderPlaced` - Order accepted and added to book
- `OrderMatched` - Full or partial fill executed
- `OrderCancelled` - Order removed from book
- `OrderRejected` - Invalid order rejected

## gRPC API

### PlaceOrder

Submit a new order for matching:

```protobuf
message PlaceOrderRequest {
  int64 trader_id = 1;
  string stock_ticker = 2;
  OrderType order_type = 3;        // MARKET or LIMIT
  OrderSide side = 4;               // BUY or SELL
  int64 quantity = 5;
  int64 limit_price_cents = 6;      // Required for LIMIT orders
  int64 available_balance_cents = 7;
}
```

Returns matched trades and remaining unmatched quantity.

### CancelOrder

Cancel an existing order:

```protobuf
message CancelOrderRequest {
  string order_id = 1;
  int64 trader_id = 2;
  string stock_ticker = 3;
}
```

### HealthCheck

Service health status.

## Configuration

Set via environment variables:

```bash
GRPC_ADDR=:50051                          # gRPC listen address
ENVIRONMENT=development                    # dev/staging/production
SHUTDOWN_TIMEOUT=30s                       # Graceful shutdown timeout
VALKEY_HOST=localhost                      # Valkey/Redis host
VALKEY_PORT=6379                           # Valkey/Redis port
VALKEY_STREAM_NAME=matching_engine_stream  # Event stream key
```

## Running

### With Docker Compose

```bash
docker compose up matching-engine
```

Service starts on `:50051` and connects to Valkey automatically.

### Local Development

```bash
cd matching_engine
go run cmd/server/main.go
```

Requires Valkey running locally on port 6379.
