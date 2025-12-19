# Verana Test Harness & Simulations

This directory contains a comprehensive test harness for the Verana blockchain, including end-to-end journey tests and economic simulations. The test harness validates all Verana modules and their interactions through realistic user journeys.

## Overview

The test harness provides:

- **19 Journey Tests**: End-to-end tests covering all Verana features and workflows
- **TD Yield Simulations**: Economic simulations testing Trust Deposit yield distribution under different funding scenarios
- **Automated Test Execution**: Scripts to run individual journeys or all journeys sequentially

## Prerequisites

- **Go 1.22+** installed
- A running Verana blockchain node (local or remote)
- Environment variables configured (see Configuration section)
- Necessary Account created


## Account Creation


This script will create the necessary account for the testharness execution

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
export GAS="200000"
export FEES="750000uvna"
```

### Devnet

```bash
export ADDRESS_PREFIX="verana"
export HOME_DIR="~/.verana"
export NODE_RPC="http://node1.devnet.verana.network:26657"
export GAS="300000"
export FEES="750000uvna"
```

### Testnet

```bash
export ADDRESS_PREFIX="verana"
export HOME_DIR="~/.verana"
export NODE_RPC="http://node1.testnet.verana.network:26657"
export GAS="300000"
export FEES="750000uvna"
```

**Note:** For TD Yield simulations, set fees to 0 for accurate balance tracking:

```bash
export FEES="0uvna"
```

## Usage

### Running Individual Journeys

```bash
# Run a specific journey by ID
go run cmd/main.go [journey_id]

# Examples:
go run cmd/main.go 1   # Trust Registry Controller Journey
go run cmd/main.go 2   # Issuer Grantor Validation Journey
go run cmd/main.go 19  # Trust Deposit Yield Journey
```

### Running All Journeys

```bash
# Run all 19 journey tests sequentially
./scripts/run_all.sh
```

This script runs journeys 1-19 in order. Each journey is independent and can be run multiple times.

### Available Journeys

| ID | Journey Name | Description |
|----|--------------|-------------|
| 1 | Trust Registry Controller | Create trust registry, credential schema, root permission, and DID |
| 2 | Issuer Grantor Validation | Issuer grantor validation workflow |
| 3 | Issuer Validation | Issuer validation via issuer grantor |
| 4 | Verifier Validation | Verifier validation via trust registry |
| 5 | Credential Issuance | Issue credentials to holders |
| 6 | Credential Verification | Verify credentials |
| 7 | Permission Renewal | Renew expiring permissions |
| 8 | Permission Termination | Void |
| 9 | Governance Framework Update | Update governance framework documents |
| 10 | Trust Deposit Management | Manage trust deposits and reclaim yield |
| 11 | DID Management | Renew, touch, and remove DIDs |
| 12 | Permission Revocation | Revoke permissions for non-compliance |
| 13 | Permission Extension | Extend permission validity periods |
| 14 | Credential Schema Update | Update and archive credential schemas |
| 15 | Failed Validation | Handle failed validation scenarios |
| 16 | Slash Permission Trust Deposit | Slash trust deposits for violations |
| 17 | Repay Permission Slashed Trust Deposit | Repay slashed deposits |
| 18 | Create Permission | Create new permissions |
| 19 | Trust Deposit Yield Accumulation | Accumulate and reclaim yield |

## TD Yield Simulations

The TD Yield simulations test Trust Deposit yield distribution under different funding scenarios. These simulations verify protocol health and economic invariants.

### Overview

The simulations test how yield is distributed from the Yield Intermediate Pool (YIP) to the Trust Deposit module and verify that the protocol remains healthy (doesn't go into debt).

### Prerequisites

**Important:** Set fees to 0 for accurate balance tracking:

```bash
export FEES="0uvna"
```

### Simulation Setup

#### Step 1: Initialize Chain for Simulations

If starting fresh, initialize the chain with simulation-optimized parameters:

```bash
./scripts/init_chain_for_simulations.sh
```

This script sets up:
- Fast voting periods (30s voting, 20s expedited)
- Reduced `blocks_per_year` (17280 = 1 day = 1 year) for faster yield accumulation
- All necessary configuration for running simulations

#### Step 2: Setup Funding Proposal (Journey 20)

Create a continuous funding proposal for the Yield Intermediate Pool:

```bash
go run cmd/main.go 20
```

This creates a governance proposal that funds YIP with a percentage of block rewards. **This only needs to be run once per chain setup.** If a proposal already exists, you can skip this step.

**Note:** The default funding percentage is very small (~1.27e-10%) to ensure simulations complete quickly. You can customize this in `cmd/main.go`.

### Running Simulations

#### Simulation 1: Sufficient Funding (Journey 21)

**Scenario:** `allowance < YIP per-block funding`

Tests the scenario where the YIP has more funds available than needed per block.

```bash
go run cmd/main.go 21
```

**What it does:**
1. Sets up a test account (Issuer_Applicant)
2. Funds the account and creates a DID to generate trust deposit
3. Monitors up to 50 blocks to track transfers from YIP to TD module
4. Performs up to 20 yield reclaims (exits early when max reclaims reached)
5. Shows detailed balance breakdown for each reclaim:
   - User balance at block N-1 (end) and block N (after reclaim)
   - TD module balance at block N-1 (end), after BeginBlock, and after reclaim
   - BeginBlock addition amount
   - Reclaim removal amount
   - Net change between blocks
6. Tracks metrics:
   - Total sent to TD module
   - Total reclaimed by users
   - Net change in TD module
7. Verifies protocol health and invariants at each reclaim block

**Expected behavior:**
- Only `allowance` amount is transferred per block
- Excess is returned to protocol pool
- YIP balance remains stable (not accumulating)
- Simulation exits early when 20 reclaims are completed

#### Simulation 2: Insufficient Funding (Journey 22)

**Scenario:** `allowance > YIP per-block funding`

Tests the scenario where the YIP has less funds available than needed per block.

```bash
go run cmd/main.go 22
```

**What it does:**
1. Sets up a test account (Issuer_Applicant - same as Simulation 1)
2. Funds the account and creates a DID with multiple years to grow trust deposit
3. Monitors up to 50 blocks to track transfers
4. Verifies YIP stays near-empty (all available is transferred)
5. Performs up to 20 yield reclaims (exits early when max reclaims reached)
6. Shows same detailed balance breakdown as Simulation 1
7. Tracks same metrics as Simulation 1

**Expected behavior:**
- All YIP balance is transferred per block
- YIP stays empty/near-empty
- No excess to return to protocol pool
- Simulation exits early when 20 reclaims are completed

### Understanding Simulation Results

#### Metrics Tracked

Both simulations track:

- **Total Sent to TD Module**: Sum of all transfers from YIP to TD module over monitored blocks
- **Total Reclaimed**: Sum of all yield reclaimed by users
- **Net Change in TD Module**: Final balance - Initial balance
- **Protocol Health**: Verifies that `Total Sent - Total Reclaimed ≈ Net Change`
- **Invariants** (checked at each reclaim block):
  - `module_balance >= sum(share * shareValue)`
  - `module_balance >= sum(amount)`

#### Detailed Reclaim Logging

For each reclaim transaction, the simulation shows:

- **User Balance**: Balance at block N-1 (end) and block N (after reclaim), with change
- **TD Module Balance**:
  - Balance at block N-1 (end)
  - Balance at block N (after BeginBlock, before reclaim) - calculated
  - Balance at block N (after reclaim) - queried
  - BeginBlock addition amount (from block events)
  - Reclaim removal amount (calculated)
  - Net change from block N-1 to block N

This detailed breakdown helps identify any discrepancies between expected and actual balance changes, accounting for BeginBlock operations that occur in the same block as reclaim transactions.

#### Protocol Health Check

The simulation calculates:

```
Expected Net Change = Total Sent to TD Module - Total Reclaimed
Actual Net Change = Final TD Module Balance - Initial TD Module Balance
Difference = |Actual - Expected|
```

**Success Criteria:**

✅ **Protocol is healthy if:**
- Difference < 1000 uvna (within tolerance for other operations, dust, etc.)
- Invariants hold throughout the simulation
- No negative balances detected

⚠️ **Warning signs:**
- Large difference between expected and actual net change
- Invariant violations
- Module balance < sum of all deposits

### Customization

#### Change Funding Percentage

Edit `cmd/main.go` line with journey 20:

```go
case 20:
    // Change "0.000000000001265823" to your desired percentage
    _, err := td_yield.SetupFundingProposal(ctx, client, "0.001000000000000000") // 0.1%
    return err
```

#### Adjust Monitoring Duration

Edit the simulation files (`journeys/simulations/td_yield/02_simulation_sufficient_funding.go` and `03_simulation_insufficient_funding.go`):

- `monitorBlocks := 50` - Maximum blocks to monitor (simulation exits early if max reclaims reached)
- `maxReclaims := 20` - Maximum number of reclaims to perform (simulation exits monitoring loop when reached)

**Note:** The simulation uses block-height-based queries for deterministic state verification. All balance queries use the `--height` flag to ensure accurate comparisons between blocks, avoiding race conditions from asynchronous block production.

## Project Structure

```
testharness/
├── cmd/
│   └── main.go                    # Entry point for running journeys
├── lib/
│   ├── client.go                  # Cosmos client setup
│   ├── fixtures.go                # Test account fixtures
│   ├── helpers.go                 # Helper functions
│   ├── queries.go                 # Query operations
│   ├── transactions.go            # Transaction operations
│   └── utils.go                   # Utility functions
├── journeys/
│   ├── journey01_trust_registry.go
│   ├── journey02_issuer_grantor.go
│   ├── journey03_issuer_validation.go
│   ├── journey04_verifier_validation.go
│   ├── journey05_credential_issuance.go
│   ├── journey06_credential_verification.go
│   ├── journey07_permission_renewal.go
│   ├── journey08_permission_termination.go
│   ├── journey09_gov_framework_update.go
│   ├── journey10_trust_deposit_management.go
│   ├── journey11_did_management.go
│   ├── journey12_permission_revocation.go
│   ├── journey13_permission_extension.go
│   ├── journey14_credential_schema_update.go
│   ├── journey15_failed_validation.go
│   ├── journey16_slash_permission_trust_deposit.go
│   ├── journey17_repay_permission_slashed_trust_deposit.go
│   ├── journey18_create_permission.go
│   ├── journey19_trust_deposit_yield.go
│   └── simulations/
│       └── td_yield/
│           ├── 01_proposal_setup.go
│           ├── 02_simulation_sufficient_funding.go
│           ├── 03_simulation_insufficient_funding.go
│           └── README.md
├── scripts/
│   ├── create_test_accounts.sh     # Create test accounts
│   ├── init_chain_for_simulations.sh  # Initialize chain for simulations
│   ├── run_all.sh                  # Run all journeys
│   └── setup_accounts.sh           # Setup accounts
├── journey_results/                # Journey execution results (JSON)
└── README.md                       # This file
```

## Journey Details

### Journey 1: Trust Registry Controller Journey

1. `Trust_Registry_Controller` account is created with sufficient funds
2. `Trust_Registry_Controller` creates a trust registry:
   - Transaction: `Create New Trust Registry` (MOD-TR-MSG-1)
   - Parameters: DID, governance framework document URL, language
3. `Trust_Registry_Controller` creates a credential schema:
   - Transaction: `Create a Credential Schema` (MOD-CS-MSG-1)
   - Parameters: trust registry ID, JSON schema, issuer/verifier permission management modes
4. `Trust_Registry_Controller` creates root permission:
   - Transaction: `Create Root Permission` (MOD-PERM-MSG-7)
   - Parameters: schema ID, validation service DID, validation/issuance/verification fees
5. `Trust_Registry_Controller` adds DID to directory:
   - Transaction: `Add a DID` (MOD-DD-MSG-1)
   - Parameters: DID, registration period

### Journey 2: Issuer Grantor Validation Journey

1. `Issuer_Grantor_Applicant` account is created with sufficient funds
2. `Trust_Registry_Controller` already exists with trust registry, credential schema, and root permission
3. `Issuer_Grantor_Applicant` starts validation process:
   - Transaction: `Start Permission VP` (MOD-PERM-MSG-1)
   - Parameters: type=ISSUER_GRANTOR, Trust Registry's validator permission ID, country
4. `Issuer_Grantor_Applicant` connects to `Trust_Registry_Controller`'s validation service
5. `Trust_Registry_Controller` validates the applicant:
   - Transaction: `Set Permission VP to Validated` (MOD-PERM-MSG-3)
   - Parameters: permission ID, effective until, validation/issuance/verification fees
6. `Issuer_Grantor_Applicant` adds their DID to directory:
   - Transaction: `Add a DID` (MOD-DD-MSG-1)
   - Parameters: DID, registration period

### Journey 3-19: Additional Journeys

See the individual journey files in `journeys/` for detailed descriptions of each journey's workflow.

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
  - Invalid parameters (e.g., AKA must be a valid URI)
  - Insufficient permissions
  - Invalid state transitions

### Simulation-Specific Issues

**Simulation exits immediately**
- Ensure the funding proposal (Journey 20) has been created and passed.
- Check that `blocks_per_year` is configured correctly for your chain.

**Balance discrepancies**
- Ensure `FEES="0uvna"` is set for accurate balance tracking.
- Verify the chain is producing blocks consistently.

**Invariant violations**
- Check the detailed logs for which invariant failed and at which block.
- Verify the TD module state matches expected values.

## Notes

- Each journey is **independent** - they can be run multiple times
- Journeys may depend on previous journeys (e.g., Journey 2 requires Journey 1)
- The `run_all.sh` script runs journeys 1-19 in order
- Simulation journeys (20-22) are separate and can be run independently
- Journey results are saved to `journey_results/` directory as JSON files

## Contributing

When adding new journeys:

1. Create a new file `journeyXX_description.go` in the `journeys/` directory
2. Implement the journey function following the pattern of existing journeys
3. Add the journey to `cmd/main.go` switch statement
4. Update this README with journey description
5. Add journey to `scripts/run_all.sh` if it should be part of the full test suite
