# Issue #185: Anchor-Based Trust Deposit POC - Status Report

> **Issue**: [#185 - Anchor-Based Control, Authorization, and Trust Deposit Architecture](https://github.com/verana-labs/verana/issues/185)
> **Branch**: `poc/anchor-based-td`
> **Last Updated**: 2026-01-19

---

## POC Objectives & Status

| # | Objective | Status | Notes |
|---|-----------|--------|-------|
| 1 | Anchor-Linked Trust Deposit | ✅ **Complete** | Trust deposits keyed by `anchor_id` (group policy address) |
| 2 | VS Registration with Hot Keys | ✅ **Complete** | Operators registered via `MsgRegisterVerifiableService` |
| 3 | Operator Spending from Anchor | ✅ **Complete** | `DebitAnchorTrustDeposit` with operator resolution |
| 4 | Per-VS Spend Limits | ✅ **Complete** | `OperatorAllowance` with limit, spent, reset_period |
| 5 | **Module Integration** | ✅ **Complete** | All VPR modules updated with anchor-aware trust deposits |
| 6 | **Auto-Execution** | ✅ **Complete** | Group proposals auto-execute after voting period (EndBlocker) |
| 7 | **Fee Payment (x/feegrant)** | ✅ **Complete** | Documented in POC guide; native Cosmos SDK support |

---

## What Was Implemented

### Proto Definitions
- `proto/verana/td/v1/anchor.proto` - Anchor, VerifiableService, OperatorAllowance messages
- `proto/verana/td/v1/tx.proto` - RegisterAnchor, RegisterVerifiableService, SetOperatorAllowance RPCs
- `proto/verana/td/v1/query.proto` - GetAnchor, GetVerifiableService, GetOperatorAllowance queries

### Keeper Functions (`x/td/keeper/anchor_poc.go`)
- `RegisterAnchor()` - Register group policy as anchor
- `RegisterVerifiableService()` - Register hot operator key
- `SetOperatorAllowance()` - Set spending limits
- `AdjustAnchorTrustDeposit()` - Handle +/- trust deposit adjustments
- `DebitAnchorTrustDeposit()` - Debit with allowance enforcement
- `GetAnchorForOperator()` - Resolve operator → anchor

### Auto-Execution (`x/td/module/module.go`)
- `EndBlocker` - Automatically executes ACCEPTED proposals after voting period
- Queries all groups, policies, proposals each block
- Eliminates manual `veranad tx group exec` commands
### Query Endpoints (`x/td/keeper/query.go`)
- `GetAnchor` - Query anchor by ID
- `GetVerifiableService` - Query VS by operator account
- `GetOperatorAllowance` - Query allowance for anchor/operator pair

### Module Integration (Completed 2026-01-15)
- **x/dd**: `add_did.go`, `renew_did.go`, `remove_did.go` use `adjustTrustDepositAnchorAware`
- **x/perm**: `msg_server.go`, `start_perm_vp.go`, `csps.go`, `perm_validated.go` use anchor-aware helper
- **x/cs**: `credential_schema.go` uses `adjustTrustDepositAnchorAware`
- **x/tr**: `msg_server.go` uses `adjustTrustDepositAnchorAware`

### CLI Commands (AutoCLI)
- `veranad query td get-anchor [anchor_id]`
- `veranad query td get-verifiable-service [operator_account]`
- `veranad query td get-operator-allowance [anchor_id] [operator_account]`

### Transaction Commands (via group proposals)
- `veranad tx td register-anchor`
- `veranad tx td register-verifiable-service`
- `veranad tx td set-operator-allowance`

### Tests
- Comprehensive unit tests in `x/td/keeper/anchor_poc_test.go`
- All tests passing ✅

### Documentation
- `ANCHOR_POC_IMPLEMENTATION.md` - Architecture overview
- `POC_SETUP_GUIDE.md` - Step-by-step CLI testing guide with:
  - Multisend for funding (single tx)
  - Auto-execution (no manual exec commands)
  - x/feegrant setup for operator fee allowances
- `feegrant_implementation.md` - Analysis of x/feegrant for fee payment requirement

---

## What Remains (Future Work)

### Medium Priority

| Item | Description |
|------|-------------|
| **Genesis Export/Import** | Add Anchors, VerifiableServices, OperatorAllowances to genesis |
| **Migrations** | Migrate existing account-based TDs to anchor-based (if needed) |
| **List Queries** | Add `ListAnchors`, `ListVerifiableServicesByAnchor`, `ListOperatorAllowances` |
| **x/authz Documentation** | Document required authz grants for each VPR operation type (no code changes needed; x/authz is built into Cosmos SDK) |

### Medium Priority

| Item | Description |
|------|-------------|
| **Genesis Export/Import** | Add Anchors, VerifiableServices, OperatorAllowances to genesis |
| **Migrations** | Migrate existing account-based TDs to anchor-based (if needed) |
| **List Queries** | Add `ListAnchors`, `ListVerifiableServicesByAnchor`, `ListOperatorAllowances` |
---

## Key Design Decisions

1. **Trust deposits accumulate from operations** - TDs are NOT funded directly; they increase when operations (DID registration, etc.) are performed

2. **Anchor = Group Policy Address** - The `anchor_id` IS the group policy account address, making it a real on-chain account that can hold funds and sign transactions

3. **Group Proposals for Anchor Operations** - All anchor management requires group proposals since `creator == anchor_id`

4. **Operator Allowances Reset** - Spent amount resets to 0 when `reset_period` elapses

---

## Commits on `poc/anchor-based-td`

| Commit | Description |
|--------|-------------|
| Initial | Proto definitions for Anchor, VS, OperatorAllowance |
| ... | Keeper collections and functions |
| ... | MsgServer handlers |
| ... | Unit tests |
| `29957ec` | Remove CreateAnchorTrustDeposit (TDs accumulate from ops) |
| `ad176e2` | Add anchor query endpoints |
| `36b778e` | Configure AutoCLI with positional args |
| `dc298cb` | Add verification queries to POC guide |
| `79405f9` | Add anchor-aware methods to TrustDepositKeeper interfaces |
| `9e0f27d` | x/dd: DID operations use anchor-aware adjustment |
| `0c97ff1` | x/perm: Permission operations use anchor-aware adjustment |
| `de7f1b5` | x/cs, x/tr: Schema and registry use anchor-aware adjustment |

---

## Testing the POC

See [POC_SETUP_GUIDE.md](./POC_SETUP_GUIDE.md) for complete step-by-step instructions.

Quick verification:
```bash
# Query anchor
veranad query td get-anchor $ANCHOR_ID

# Query VS operator
veranad query td get-verifiable-service $OPERATOR1

# Query allowance
veranad query td get-operator-allowance $ANCHOR_ID $OPERATOR1
```
