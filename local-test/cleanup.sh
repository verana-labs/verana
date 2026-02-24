#!/bin/bash

echo "Cleaning up Verana development environment..."

# Stop and remove containers
docker stop validator1 validator2 validator3 validator4 validator5 2>/dev/null || true
docker rm validator1 validator2 validator3 validator4 validator5 2>/dev/null || true

# Remove data directories
rm -rf validator1 validator2 validator3 validator4 validator5

echo "✅ Cleanup complete!"