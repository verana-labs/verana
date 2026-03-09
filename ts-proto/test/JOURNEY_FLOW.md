# Journey Flow Document

This document defines the transaction sequences, prerequisites, and account strategy for all TypeScript client journey tests.

## Overview

**Strategy**: Use the `cooluser` master account for all transactions. Each journey tests a single transaction type (or a small set of related transactions) to validate that the TypeScript protobuf types align with the blockchain.

**Master Account**: `cooluser` (mnemonic: "pink glory help gown abstract eight nice crazy forward ketchup skill cheese")
- Used for: DE authorization grants, TR operations, CS operations

**State Sharing**: Journeys share state via `journey_results/*.json` files. Earlier journeys save IDs (trust registry, schema, etc.) that later journeys load.

## Journey List

### Delegation Engine (DE) Module

#### DE: Grant TR Operator Authorization
- **Script**: `test:de-grant-auth`
- **File**: `deGrantOperatorAuthorization.ts`
- **Prerequisites**: None
- **Transactions**:
  1. Grant operator authorization for Trust Registry messages
- **Outputs**: Operator authorization saved to journey results

#### DE: Grant CS Operator Authorization
- **Script**: `test:de-grant-cs-auth`
- **File**: `deGrantCsOperatorAuthorization.ts`
- **Prerequisites**: None
- **Transactions**:
  1. Grant operator authorization for Credential Schema messages
- **Outputs**: Operator authorization saved to journey results

---

### Trust Registry (TR) Module

All TR journeys use operator-signed transactions via the delegation engine.

#### TR: Create Trust Registry
- **Script**: `test:tr-create`
- **File**: `trCreateTrustRegistry.ts`
- **Prerequisites**: DE authorization (from DE Grant TR Operator Authorization)
- **Transactions**:
  1. Create Trust Registry (MOD-TR-MSG-1)
- **Outputs**: `trustRegistryId`, `did` (saved to journey results)

#### TR: Add Governance Framework Document
- **Script**: `test:tr-add-gfd`
- **File**: `trAddGovernanceFrameworkDocument.ts`
- **Prerequisites**: Active TR (from Create Trust Registry)
- **Transactions**:
  1. Add Governance Framework Document (MOD-TR-MSG-2)

#### TR: Increase Active Governance Framework Version
- **Script**: `test:tr-increase-gf-version`
- **File**: `trIncreaseActiveGovernanceFrameworkVersion.ts`
- **Prerequisites**: TR with GF documents (from Add GF Document)
- **Transactions**:
  1. Increase Active Governance Framework Version (MOD-TR-MSG-3)

#### TR: Update Trust Registry
- **Script**: `test:tr-update`
- **File**: `trUpdateTrustRegistry.ts`
- **Prerequisites**: Active TR
- **Transactions**:
  1. Update Trust Registry (MOD-TR-MSG-4)

#### TR: Archive Trust Registry
- **Script**: `test:tr-archive`
- **File**: `trArchiveTrustRegistry.ts`
- **Prerequisites**: Active TR
- **Transactions**:
  1. Archive Trust Registry (MOD-TR-MSG-5)

---

### Credential Schema (CS) Module

All CS journeys use operator-signed transactions via the delegation engine.

#### CS: Create Credential Schema
- **Script**: `test:cs-create`
- **File**: `csCreateCredentialSchema.ts`
- **Prerequisites**: DE CS authorization, active TR
- **Transactions**:
  1. Create Credential Schema (MOD-CS-MSG-1)
- **Outputs**: `schemaId` (saved to journey results)

#### CS: Update Credential Schema
- **Script**: `test:cs-update`
- **File**: `csUpdateCredentialSchema.ts`
- **Prerequisites**: Active CS (from Create Credential Schema)
- **Transactions**:
  1. Update Credential Schema (MOD-CS-MSG-2)

#### CS: Archive Credential Schema
- **Script**: `test:cs-archive`
- **File**: `csArchiveCredentialSchema.ts`
- **Prerequisites**: Active CS
- **Transactions**:
  1. Archive Credential Schema (MOD-CS-MSG-3)

---

## Execution Order

The `runAll.ts` script runs all 10 tests sequentially:

1. DE: Grant TR Operator Authorization
2. TR: Create Trust Registry
3. TR: Add GF Document
4. TR: Increase Active GF Version
5. TR: Update Trust Registry
6. TR: Archive Trust Registry
7. DE: Grant CS Operator Authorization
8. CS: Create Credential Schema
9. CS: Update Credential Schema
10. CS: Archive Credential Schema

## Resource Reuse Strategy

### Journey Results Storage
- Results saved as JSON files in `journey_results/`
- Each journey saves IDs needed by subsequent journeys
- `journey_results/` is gitignored

### Transaction Count
- Each journey executes 1 transaction
- Total: 10 transactions across all journeys

## References

- [Verana VPR Specification](https://verana-labs.github.io/verifiable-trust-vpr-spec/)
- MOD-TR-MSG-1 through MOD-TR-MSG-5: Trust Registry messages
- MOD-CS-MSG-1 through MOD-CS-MSG-3: Credential Schema messages
