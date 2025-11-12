#!/usr/bin/env bash

set -eo pipefail

echo "Formatting protobuf files"
find ./ -name "*.proto" -exec clang-format -i {} \; 2>/dev/null || echo "clang-format not found, skipping formatting"

home=$PWD

echo "Generating proto code"
proto_dirs=$(find ./ -name 'buf.yaml' -print0 | xargs -0 -n1 dirname | sort | uniq)

for dir in $proto_dirs; do
  echo "Processing proto directory: $dir"
  cd "$dir"

  # Generate pulsar proto code (for api directory)
  if [ -f "buf.gen.pulsar.yaml" ]; then
    echo "  Generating pulsar proto code..."
    buf generate --template buf.gen.pulsar.yaml

    # Move generated files to the right places
    if [ -d "../api" ]; then
      echo "  Moving pulsar generated files to api directory..."
      # The pulsar files should already be in the right place based on buf.gen.pulsar.yaml config
    fi
  fi

  # Generate gogo proto code (for x/ modules - types.pb.go files)
  if [ -f "buf.gen.gogo.yaml" ]; then
    echo "  Generating gogo proto code..."

    for file in $(find . -maxdepth 5 -name '*.proto'); do
      if grep -q "option go_package" "$file" && \
         ! grep -q "option go_package.*cosmossdk.io/api" "$file"; then
        buf generate --template buf.gen.gogo.yaml "$file"
      fi
    done

    # Move generated files from nested structure to correct location
    if [ -d "github.com/verana-labs/verana/x" ]; then
      echo "  Moving gogo generated files to x/ directory..."
      cp -r github.com/verana-labs/verana/x/* ../x/
      rm -rf github.com
    fi
  fi

  # Generate swagger/OpenAPI documentation
  if [ -f "buf.gen.swagger.yaml" ]; then
    echo "  Generating swagger documentation..."
    buf generate --template buf.gen.swagger.yaml
  fi

  cd "$home"
done

echo "✓ Proto generation complete!"