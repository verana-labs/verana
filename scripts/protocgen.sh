#!/usr/bin/env bash

set -eo pipefail

echo "Generating protobuf files..."

# Change to project root
cd "$(dirname "$0")/.."

# Check if ignite is installed
if ! command -v ignite &> /dev/null; then
    echo "Error: ignite is not installed"
    echo "Install from: https://github.com/ignite/cli"
    exit 1
fi

# Generate proto files using Ignite
echo "Using Ignite to generate proto files..."
ignite generate proto-go --yes

echo "✅ Protobuf generation completed!"
echo ""
echo "Generated:"
echo "  • Go files (.pb.go, .pb.gw.go) in x/*/types/"
echo "  • Pulsar files (.pulsar.go, _grpc.pb.go) in api/verana/*/"
