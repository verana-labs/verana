# TD Yield Simulations

This directory contains simulations for testing Trust Deposit yield distribution under different funding scenarios.

## Overview

The simulations test how yield is distributed from the Yield Intermediate Pool (YIP) to the Trust Deposit module and verify that the protocol remains healthy (doesn't go into debt).

## Files

1. **`01_proposal_setup.go`** - Sets up the continuous funding proposal for YIP
2. **`02_simulation_sufficient_funding.go`** - Tests scenario where `allowance < YIP per-block funding`
3. **`03_simulation_insufficient_funding.go`** - Tests scenario where `allowance > YIP per-block funding`

## How to Run

### Prerequisites

**Important:** Set fees to 0 for accurate balance tracking:
```bash
export FEES=0uvna
```

This ensures that balance changes after reclaiming reflect only the yield amount, not transaction fees.

### Option 1: Using main.go (Recommended)

```bash
# Step 1: Initialize chain for simulations (if starting fresh)
./scripts/init_chain_for_simulations.sh

# Step 2: Setup the funding proposal (run once)
go run cmd/main.go 20

# Step 3a: Run sufficient funding simulation
go run cmd/main.go 21

# Step 3b: Run insufficient funding simulation
go run cmd/main.go 22
```

### Option 2: Direct Package Execution

You can also create a simple runner script or call the functions directly from Go code.

## Simulation Flow

### Step 1: Proposal Setup (Journey 20)
- Creates a continuous funding proposal
- Sets funding to 0.05% of block rewards (configurable)
- Votes and waits for proposal to pass
- **Note:** This only needs to be run once. If a proposal already exists, you can skip this step.

### Step 2a: Sufficient Funding Simulation (Journey 21)
**Scenario:** `allowance < YIP per-block funding`

**What it does:**
1. Sets up a test account (Issuer_Applicant)
2. Funds the account and creates a DID to generate trust deposit
3. Monitors up to 50 blocks to track transfers from YIP to TD module
4. Performs up to 20 yield reclaims (exits early when max reclaims reached)
5. For each reclaim, shows detailed balance breakdown:
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

### Step 2b: Insufficient Funding Simulation (Journey 22)
**Scenario:** `allowance > YIP per-block funding`

**What it does:**
1. Sets up a test account (Issuer_Applicant - same as Step 2a)
2. Funds the account and creates a DID with multiple years to grow trust deposit
3. Monitors up to 50 blocks to track transfers
4. Verifies YIP stays near-empty (all available is transferred)
5. Performs up to 20 yield reclaims (exits early when max reclaims reached)
6. Shows same detailed balance breakdown as Step 2a for each reclaim
7. Tracks same metrics as Step 2a

**Expected behavior:**
- All YIP balance is transferred per block
- YIP stays empty/near-empty
- No excess to return to protocol pool
- Simulation exits early when 20 reclaims are completed

## Metrics Tracked

Both simulations track:

- **Total Sent to TD Module**: Sum of all transfers from YIP to TD module over monitored blocks
- **Total Reclaimed**: Sum of all yield reclaimed by users
- **Net Change in TD Module**: Final balance - Initial balance
- **Protocol Health**: Verifies that `Total Sent - Total Reclaimed ≈ Net Change`
- **Invariants** (checked at each reclaim block): 
  - `module_balance >= sum(share * shareValue)`
  - `module_balance >= sum(amount)`

### Detailed Reclaim Logging

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

## Understanding the Results

### Protocol Health Check

The simulation calculates:
```
Expected Net Change = Total Sent to TD Module - Total Reclaimed
Actual Net Change = Final TD Module Balance - Initial TD Module Balance
Difference = |Actual - Expected|
```

If the difference is > 1000 uvna, it indicates a potential protocol imbalance.

### Success Criteria

✅ **Protocol is healthy if:**
- Difference < 1000 uvna (within tolerance for other operations, dust, etc.)
- Invariants hold throughout the simulation
- No negative balances detected

⚠️ **Warning signs:**
- Large difference between expected and actual net change
- Invariant violations
- Module balance < sum of all deposits

## Customization

### Change Funding Percentage

Edit `cmd/main.go` line with journey 20:
```go
case 20:
    // Change "0.000500000000000000" to your desired percentage
    _, err := td_yield.SetupFundingProposal(ctx, client, "0.001000000000000000") // 0.1%
    return err
```

### Adjust Monitoring Duration

Edit the simulation files:
- `monitorBlocks := 50` - Maximum blocks to monitor (simulation exits early if max reclaims reached)
- `maxReclaims := 20` - Maximum number of reclaims to perform (simulation exits monitoring loop when reached)

**Note:** The simulation uses block-height-based queries for deterministic state verification. All balance queries use the `--height` flag to ensure accurate comparisons between blocks, avoiding race conditions from asynchronous block production.

## Notes

- Each simulation is **independent** - they set up their own accounts and operations
- Simulations can be run multiple times (they don't depend on previous journeys)
- The proposal setup (Journey 20) only needs to be run once per chain setup
- If a proposal already exists, you can skip Journey 20 and go directly to 21 or 22
