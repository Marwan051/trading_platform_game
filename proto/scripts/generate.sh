#!/bin/bash
set -e

# Get the directory where the script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Set workspace root (proto folder's parent)
WORKSPACE_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
# Set proto root
PROTO_ROOT="${WORKSPACE_ROOT}/proto"

# Create output directory
mkdir -p "${PROTO_ROOT}/gen/go"

echo "Generating Go code..."

# Run protoc from the workspace root
# This allows imports like "proto/v1/common/types.proto" to work correctly
cd "${WORKSPACE_ROOT}"

protoc \
  --proto_path=. \
  --go_out="${PROTO_ROOT}/gen/go" \
  --go_opt=module=github.com/Marwan051/tradding_platform_game/proto/gen/go \
  --go-grpc_out="${PROTO_ROOT}/gen/go" \
  --go-grpc_opt=module=github.com/Marwan051/tradding_platform_game/proto/gen/go \
  proto/v1/common/*.proto \
  proto/v1/matching_engine/*.proto \
  proto/v1/market_data/*.proto

echo "Done!"
