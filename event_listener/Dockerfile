# multi-stage build - use Debian instead of Alpine for glibc (required by glide)
FROM golang:1.26-bookworm AS builder
WORKDIR /src

# cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# copy sources and build with CGO enabled
COPY . .
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
    go build -o /server ./cmd/server

# final image - use distroless or slim Debian
FROM debian:trixie-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /server /server
ENTRYPOINT ["/server"]
