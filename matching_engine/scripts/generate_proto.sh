#!/bin/bash

# Generate Go code from proto files
# Requires: protoc, protoc-gen-go, protoc-gen-go-grpc

set -e

echo "Checking for required tools..."
command -v protoc >/dev/null 2>&1 || { echo "protoc is required but not installed. Install it first."; exit 1; }
command -v protoc-gen-go >/dev/null 2>&1 || { echo "protoc-gen-go is required. Install: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"; exit 1; }
command -v protoc-gen-go-grpc >/dev/null 2>&1 || { echo "protoc-gen-go-grpc is required. Install: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"; exit 1; }

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
PROTO_BASE="api/proto/v1"

cd "${PROJECT_ROOT}"

echo "Generating common types..."
protoc \
  --proto_path=. \
  --go_out=. \
  --go_opt=paths=source_relative \
  ${PROTO_BASE}/common/types.proto

echo "Generating common events..."
protoc \
  --proto_path=. \
  --go_out=. \
  --go_opt=paths=source_relative \
  ${PROTO_BASE}/common/events.proto

echo "Generating matching engine service..."
protoc \
  --proto_path=. \
  --go_out=. \
  --go_opt=paths=source_relative \
  --go-grpc_out=. \
  --go-grpc_opt=paths=source_relative \
  ${PROTO_BASE}/matching_engine/matching_engine.proto

echo "✓ Proto files generated successfully"
