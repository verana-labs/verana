# Verana Test Harness

This directory contains the end-to-end test harness for the Verana blockchain. The test harness validates all Verana modules through operator-authorization journeys that test the v4 authority/operator pattern with AUTHZ-CHECK delegation.

## Overview

The test harness provides:

- **10 Journey Tests**: End-to-end tests covering Trust Registry, Credential Schema, and Permission module operations with operator authorization
- **Automated Test Execution**: Scripts to run individual journeys or all journeys sequentially

## Prerequisites

- **Go 1.22+** installed
- A running Verana blockchain node (local or remote)
- Environment variables configured (see Configuration section)
- Necessary accounts created

## Account Creation

This script will create the necessary accounts for the testharness execution:

```bash
cd testharness
./scripts/setup_accounts.sh
```

## Installation

The test harness is part of the Verana repository. No separate installation is needed:

```bash
# From the Verana repository root
cd testharness

# Ensure dependencies are installed
go mod tidy
```

## Configuration

Set up environment variables for the test harness:

### Local Development

```bash
export ADDRESS_PREFIX="verana"
export HOME_DIR="~/.verana"
export NODE_RPC="http://localhost:26657"
export GAS="auto"  # or a fixed value like "500000"
export FEES="750000uvna"
```

### Devnet

```bash
export ADDRESS_PREFIX="verana"
export HOME_DIR="~/.verana"
export NODE_RPC="http://node1.devnet.verana.network:26657"
export GAS="auto"  # auto-estimates with 1.5x adjustment
export FEES="750000uvna"
```

### Testnet

```bash
export ADDRESS_PREFIX="verana"
export HOME_DIR="~/.verana"
export NODE_RPC="http://node1.testnet.verana.network:26657"
export GAS="auto"  # auto-estimates with 1.5x adjustment
export FEES="750000uvna"
```

## Usage

### Running Individual Journeys

```bash
# Run a specific journey by ID
go run cmd/main.go [journey_id]

# Examples:
go run cmd/main.go 101  # TR Operator Authorization Setup
go run cmd/main.go 304  # Permission Create Root Permission
```

### Running All Journeys

```bash
# Run all 10 journey tests sequentially
./scripts/run_all.sh
```

### Available Journeys

| ID  | Journey Name | Description |
|-----|--------------|-------------|
| 101 | TR Operator Authorization Setup | Create group, add members, fund group policy for TR operations |
| 102 | TR Operations with Operator Auth | Test all 5 TR operations: fail without auth, grant auth, succeed with auth |
| 201 | CS Operator Authorization Setup | Create group, add members, fund group policy for CS operations |
| 202 | CS Operations with Operator Auth | Test all 3 CS operations: fail without auth, grant auth, succeed with auth |
| 301 | Perm Operator Authorization Setup | Create group, add members, fund group policy for Permission operations |
| 302 | Perm Operations with Operator Auth | Test CancelVPLastRequest and CreateRootPermission: fail without auth, grant auth, succeed |
| 303 | Perm Cancel VP Last Request | Cancel VP last request with operator authorization |
| 304 | Perm Create Root Permission | Create root permission with operator authorization |
| 305 | Perm Adjust Permission | Adjust permission with operator authorization |
| 306 | Perm Revoke Permission | Revoke permission with operator authorization |

### Journey Dependencies

Journeys must be run in order within each module group:

- **TR**: 101 -> 102
- **CS**: 201 -> 202
- **Perm**: 301 -> 302 -> 303, 304, 305, 306

Each permission journey (303-306) depends on 301 and 302 for group/operator setup. Journeys 304-306 also share results between them (e.g., 306 uses the trust registry from 304).

### Journey Pattern

All journeys follow the same authorization test pattern:

1. **Fail without auth**: Operator tries the operation without delegation authorization (expect failure)
2. **Grant auth**: Admin proposes and members vote to grant operator authorization via group policy
3. **Succeed with auth**: Operator retries the operation with authorization (expect success)
4. **Verify**: Check the on-chain state matches expectations
5. **Unauthorized operator**: A different operator tries the same operation (expect failure)

## Project Structure

```
testharness/
├── cmd/
│   └── main.go                              # Entry point for running journeys
├── lib/
│   ├── client.go                            # Cosmos client setup
│   ├── fixtures.go                          # Test account fixtures
│   ├── helpers.go                           # Helper functions
│   ├── queries.go                           # Query operations
│   ├── transactions.go                      # Transaction operations
│   └── utils.go                             # Utility functions
├── journeys/
│   ├── journey101_tr_authz_setup.go         # TR authorization setup
│   ├── journey102_tr_authz_operations.go    # TR operations with auth
│   ├── journey201_cs_authz_setup.go         # CS authorization setup
│   ├── journey202_cs_authz_operations.go    # CS operations with auth
│   ├── journey301_perm_authz_setup.go       # Perm authorization setup
│   ├── journey302_perm_authz_operations.go  # Perm operations with auth
│   ├── journey303_perm_cancel_vp.go         # Cancel VP last request
│   ├── journey304_perm_create_root.go       # Create root permission
│   ├── journey305_perm_adjust.go            # Adjust permission
│   └── journey306_perm_revoke.go            # Revoke permission
├── scripts/
│   ├── create_test_accounts.sh              # Create test accounts
│   ├── run_all.sh                           # Run all journeys
│   └── setup_accounts.sh                    # Setup accounts
├── journey_results/                         # Journey execution results (JSON)
└── README.md                                # This file
```

## Troubleshooting

### Common Issues

**"Account does not exist on chain"**
- Ensure the test accounts have been funded. Use `scripts/create_test_accounts.sh` or fund manually.

**"Insufficient fees"**
- Increase the `FEES` environment variable or ensure accounts have sufficient balance.

**"Connection refused"**
- Verify the blockchain node is running and accessible at the configured `NODE_RPC` endpoint.

**"Transaction failed"**
- Check the transaction logs for specific error messages. Common issues include:
  - Invalid parameters
  - Insufficient permissions / missing operator authorization
  - Invalid state transitions

## Notes

- Journeys within a module group must be run in order (e.g., 301 before 302)
- The `run_all.sh` script runs all 10 journeys in the correct order
- Journey results are saved to `journey_results/` directory as JSON files
- Results from earlier journeys are loaded by later journeys (group policy address, operator address, trust registry ID, etc.)

## Contributing

When adding new journeys:

1. Create a new file `journeyXXX_description.go` in the `journeys/` directory
2. Implement the journey function following the authorization test pattern
3. Add the journey to `cmd/main.go` switch statement
4. Update `scripts/run_all.sh`
5. Update this README with the journey description
