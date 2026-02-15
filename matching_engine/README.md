# Matching Engine

The Matching Engine is the core component of the trading platform responsible for matching buy and sell orders for stocks. It is designed to be high-performance, concurrent, and correct, ensuring fair execution of trades based on **Price-Time Priority (FIFO)**.

## Core Concepts & Data Structures

Understanding the matching engine requires understanding its internal data structures defined in `internal/lib/types/order_types.go`. These structures are optimized for $O(1)$ access and $O(\log N)$ updates.

### 1. The Order Book (`StockOrderBook`)

Each stock (e.g., "AAPL") has its own independent `StockOrderBook`. This allows the engine to process trades for different stocks in parallel without blocking each other.

- **Concurrency**: Protected by a `sync.RWMutex` to ensure thread-safety.
- **Components**: Contains a `BuySide` (Bids) and a `SellSide` (Asks).

### 2. PriceLevel (The Queue)

- **What it is**: A group of all orders at a single specific price.
- **Structure**: Uses a **Doubly-Linked List**.
- **Logic**: This ensures **FIFO (First-In-First-Out)** execution. When the engine matches orders at a specific price (e.g., $150.00), it matches the oldest order (at the front of the list) first.

### 3. PriceHeap (The Sorter)

- **What it is**: A priority queue that keeps track of active price levels.
- **Buy Side**: Implemented as a **Max-Heap** (Highest Buy Price is at the top).
- **Sell Side**: Implemented as a **Min-Heap** (Lowest Sell Price is at the top).
- **Logic**: Allows the engine to find the "Best Bid" or "Best Ask" in $O(1)$ time.

### 4. OrderBookSide (The Manager)

- **What it is**: Manages one side of the market (all Buys or all Sells).
- **Lookups**: Contains a Hash Map (`orderID -> Order`) to allow for $O(1)$ order cancellation.
- **Lazy Deletion**: When a price level becomes empty, it is removed from the map immediately, but removed from the Heap only when it bubbles to the top. This keeps the critical path fast.

---

## Matching Logic (`matching_engine.go`)

The core matching loop happens in `SubmitOrder` and its helper functions `matchBuyOrder` and `matchSellOrder`.

### 1. Order Submission

When a user places an order:

1.  **Validation**: checks quantity > 0, price > 0, etc.
2.  **Locking**: The specific stock's order book is locked.
3.  **Event**: An `OrderPlaced` event is emitted.

### 2. The Matching Loop

The engine attempts to match the incoming order immediately against the **opposite side** of the book.

**For a BUY Order:**

1.  **Check Best Ask**: The engine peeks at the top of the `SellSide` heap (Lowest Sell Price).
2.  **Price Check**:
    - **Limit Order**: Is `Buy Limit Price >= Best Ask Price`?
    - **Market Order**: Is there any sell order available? (And does the buyer have enough funds?)
3.  **Match**: If the price crosses, the engine matches against the orders at that price level (FIFO).
4.  **Repeat**: This continues until:
    - The buy order is fully filled.
    - There are no more sell orders.
    - The next best sell price is too high (for Limit orders).
    - The buyer runs out of funds (for Market orders).

**For a SELL Order:**

1.  **Check Best Bid**: The engine peeks at the top of the `BuySide` heap (Highest Buy Price).
2.  **Price Check**: Is `Sell Limit Price <= Best Bid Price`?
3.  **Match**: Executes trades against the buy orders.

### 3. Post-Match Handling

- **Filled**: If the incoming order is fully executed, `OrderFilled` events are emitted.
- **Partial/Unfilled**:
  - **Limit Orders**: Any remaining quantity is added to the `OrderBook` as a new resting order.
  - **Market Orders**: Any unfilled quantity is typically cancelled (Immediate-Or-Cancel behavior) if liquidity runs out.

---

## Code Structure

```
matching_engine/
├── cmd/server/             # Server entry point
├── internal/
│   ├── config/            # Environment configuration
│   ├── interceptors/      # gRPC middleware
│   ├── lib/
│   │   ├── events/        # Event publishing
│   │   ├── matching_engine/ # Core matching logic
│   │   └── types/         # Order and event types (Order, PriceLevel, Heap)
│   ├── server/            # gRPC server setup
│   └── service/           # gRPC service implementation
└── Dockerfile
```

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
