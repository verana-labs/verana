#!/bin/bash
# scripts/run_all.sh

# Exit immediately if a command exits with a non-zero status
set -e

# Move to the script's parent directory (root of the module)
# cd "$(dirname "$0")/.."

for i in {1..19}; do
  echo "*****************************************************************************************"
  echo "**************************** Running test-harness journey $i ****************************"
  echo "*****************************************************************************************"
  go run cmd/main.go "$i"
done

# Run Journey 23: Error Scenario Tests (Issues #191, #193, #196)
echo "*****************************************************************************************"
echo "**************************** Running test-harness journey 23 ****************************"
echo "*****************************************************************************************"
go run cmd/main.go 23