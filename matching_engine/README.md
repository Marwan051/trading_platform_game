# ⚡ Matching Engine

The **Matching Engine** is the high-performance core of the Trading Platform Game. It is a concurrent, in-memory order matching system responsible for executing buy and sell orders with sub-millisecond latency.

Built with **Go**, it prioritizes correctness and speed, ensuring fair execution based on **Price-Time Priority (FIFO)**.

## Key Features

- **High Concurrency**: Independent order books for each stock allow parallel processing without lock contention.
- **Price-Time Priority**: Strictly follows FIFO rules—orders at the same price are matched based on arrival time.
- **Microsecond Latency**: Optimized internal data structures ($O(1)$ lookups, $O(\log N)$ updates).
- **Event-Driven**: Emits real-time trade execution and order status events via Valkey (Redis) Streams.
- **gRPC API**: Strongly typed communication with other services.

## 🛠 Tech Stack

- **Language**: Go 1.22+
- **Communication**: gRPC / Protobuf
- **Event Streaming**: Valkey (Redis) Streams

## 🏗 Architecture & Internal Logic

The engine uses special data structures to maintain high throughput.

### Data Structures (`internal/lib/types`)

| Component       | Structure           | Purpose                                                            |
| --------------- | ------------------- | ------------------------------------------------------------------ |
| **Order Book**  | `sync.RWMutex`      | Thread-safe container for a single stock's bids and asks.          |
| **Price Level** | Doubly-Linked List  | Stores all orders at a specific price. Ensures **FIFO** execution. |
| **Price Heap**  | Max-Heap / Min-Heap | Keeps track of best prices ($O(1)$ access to best bid/ask).        |
| **Order Map**   | Hash Map            | Quick access ($O(1)$) for order cancellation by ID.                |

### Matching Algorithm

When an order is submitted via `SubmitOrder`:

1.  **Validation**: Basic checks (quantity, price, balance).
2.  **Locking**: The specific stock's book is locked (granular locking).
3.  **Crossing**: The engine checks if the order matches against the _opposite_ side of the book.
    - **Buy Order**: Matched against lowest `Sell` prices (Min-Heap).
    - **Sell Order**: Matched against highest `Buy` prices (Max-Heap).
4.  **Execution**: Matches are generated until the order is filled or liquidity runs out.
5.  **Resting**: Unfilled limit orders are added to the book.

## ⚙️ Configuration

| Variable             | Description                                       | Default                  |
| -------------------- | ------------------------------------------------- | ------------------------ |
| `GRPC_ADDR`          | Address for the gRPC server to listen on          | `:50051`                 |
| `ENVIRONMENT`        | Runtime environment (`development`, `production`) | `development`            |
| `VALKEY_HOST`        | Hostname of the Valkey/Redis instance             | `localhost`              |
| `VALKEY_PORT`        | Port of the Valkey/Redis instance                 | `6379`                   |
| `VALKEY_STREAM_NAME` | Key for the event stream                          | `matching_engine_stream` |
| `SHUTDOWN_TIMEOUT`   | Time to wait for graceful shutdown                | `30s`                    |

## Getting Started

### Prerequisites

- Go 1.22+
- Docker
- Valkey or Redis instance

### Running with Docker

The service is part of the main `docker-compose.yml` stack.

```bash
docker compose up matching-engine
```

## API Reference

The service exposes a gRPC interface defined in `proto/v1/matching_engine/matching_engine.proto`.

### `PlaceOrder`

Submits a market or limit order.

```protobuf
message PlaceOrderRequest {
  int64 trader_id = 1;
  string stock_ticker = 2;
  OrderType order_type = 3;  // MARKET or LIMIT
  OrderSide side = 4;        // BUY or SELL
  int64 quantity = 5;
  int64 limit_price_cents = 6;
  int64 available_balance_cents = 7;
}
```

### `CancelOrder`

Removes a resting order from the book.

```protobuf
message CancelOrderRequest {
  string order_id = 1;
  int64 trader_id = 2;
  string stock_ticker = 3;
}
```

## Project Structure

```
matching_engine/
├── cmd/
│   └── server/            # Main entry point
├── internal/
│   ├── config/            # Configuration management
│   ├── interceptors/      # gRPC logging/recovery middleware
│   ├── lib/
│   │   ├── events/        # Event streaming logic
│   │   ├── matching_engine/ # Core domain logic (The Engine)
│   │   └── types/         # Data structures (Heaps, Lists, Types)
│   ├── server/            # gRPC server definition
│   └── service/           # Implementation of the gRPC interface
└── Dockerfile
```
