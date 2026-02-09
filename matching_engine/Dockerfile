# Builder stage
FROM golang:1.23-bookworm AS builder

# Install build dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    git \
    && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /build

# Copy go mod files first for better layer caching
COPY matching_engine/go.mod matching_engine/go.sum ./matching_engine/
COPY proto/ ./proto/

# Download dependencies
WORKDIR /build/matching_engine
RUN go mod download

# Copy source code
COPY matching_engine/ .

# Build the application with CGO enabled for valkey-glide
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s' \
    -o /build/bin/matching_engine \
    ./cmd/server

# Runner stage
FROM debian:trixie-slim

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    tzdata \
    && rm -rf /var/lib/apt/lists/*

# Create non-root user
RUN groupadd -g 1000 appuser && \
    useradd -r -u 1000 -g appuser -s /sbin/nologin appuser

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/bin/matching_engine /app/matching_engine

# Change ownership
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose gRPC port
EXPOSE 50051

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["grpc_health_probe", "-addr=:50051"] || exit 1

# Run the application
CMD ["/app/matching_engine"]
