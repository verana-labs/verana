# Journey Flow Document

This document defines the transaction sequences, prerequisites, and account strategy for all TypeScript client journey tests.

## Overview

**Strategy**: Use the `cooluser` master account for all transactions. Each journey tests a single transaction type (or a small set of related transactions) to validate that the TypeScript protobuf types align with the blockchain.

**Master Account**: `cooluser` (mnemonic: "pink glory help gown abstract eight nice crazy forward ketchup skill cheese")
- Used for: DE authorization grants, TR operations, CS operations, PERM operations

**Account Derivation Indices**:
- Index 10: Authority (shared across TR, CS, PERM)
- Index 11: TR Operator
- Index 13: CS Operator
- Index 15: PERM Operator

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

### Permission (PERM) Module

All PERM journeys use operator-signed transactions (authority=index 10, operator=index 15).

#### DE: Grant PERM Operator Authorization
- **Script**: `test:de-grant-perm-auth`
- **File**: `deGrantPermOperatorAuthorization.ts`
- **Prerequisites**: DE TR authorization (from DE Grant TR Operator Authorization)
- **Transactions**: Grant operator auth for all PERM + TR + CS message types

#### PERM: Create Root Permission
- **Script**: `test:perm-create-root`
- **File**: `permCreateRootPermission.ts`
- **Prerequisites**: PERM authorization
- **Transactions**: Create TR, CS (GRANTOR_VALIDATION), Root Permission (MOD-PERM-MSG-7)
- **Outputs**: perm-root-setup

#### PERM: Create Permission (Self-Create)
- **Script**: `test:perm-create`
- **File**: `permCreatePermission.ts`
- **Prerequisites**: PERM authorization
- **Transactions**: Create OPEN CS + Root, then MsgCreatePermission (MOD-PERM-MSG-14)

#### PERM: Adjust Permission
- **Script**: `test:perm-adjust`
- **File**: `permAdjustPermission.ts`
- **Prerequisites**: Root permission (effective)
- **Transactions**: MsgAdjustPermission (MOD-PERM-MSG-8)

#### PERM: Revoke Permission
- **Script**: `test:perm-revoke`
- **File**: `permRevokePermission.ts`
- **Prerequisites**: PERM authorization (creates fresh)
- **Transactions**: Fresh root + MsgRevokePermission (MOD-PERM-MSG-9)

#### PERM: Start Permission VP
- **Script**: `test:perm-start-vp`
- **File**: `permStartPermissionVP.ts`
- **Prerequisites**: Root permission (effective)
- **Transactions**: MsgStartPermissionVP (MOD-PERM-MSG-1)
- **Outputs**: perm-vp-setup

#### PERM: Set Permission VP To Validated
- **Script**: `test:perm-validate-vp`
- **File**: `permSetPermissionVPToValidated.ts`
- **Prerequisites**: VP from Start VP
- **Transactions**: MsgSetPermissionVPToValidated (MOD-PERM-MSG-2)
- **Outputs**: perm-validated-setup

#### PERM: Renew + Cancel Permission VP
- **Script**: `test:perm-cancel-vp`
- **File**: `permCancelPermissionVPLastRequest.ts`
- **Prerequisites**: Validated VP
- **Transactions**: MsgRenewPermissionVP (MOD-PERM-MSG-3) + MsgCancelPermissionVPLastRequest (MOD-PERM-MSG-6)

#### PERM: Create/Update Permission Session
- **Script**: `test:perm-csps`
- **File**: `permCreateOrUpdatePermissionSession.ts`
- **Prerequisites**: PERM authorization (creates fresh chain)
- **Transactions**: Full VP chain + MsgCreateOrUpdatePermissionSession (MOD-PERM-MSG-10) x2

#### PERM: Slash Permission Trust Deposit
- **Script**: `test:perm-slash`
- **File**: `permSlashPermissionTrustDeposit.ts`
- **Prerequisites**: PERM authorization (creates fresh chain)
- **Transactions**: Full VP chain + MsgSlashPermissionTrustDeposit (MOD-PERM-MSG-11)
- **Outputs**: perm-slash-setup

#### PERM: Repay Slashed Trust Deposit
- **Script**: `test:perm-repay`
- **File**: `permRepayPermissionSlashedTrustDeposit.ts`
- **Prerequisites**: Slashed permission from Slash journey
- **Transactions**: MsgRepayPermissionSlashedTrustDeposit (MOD-PERM-MSG-12)

---

## Execution Order

The `runAll.ts` script runs all 21 tests sequentially:

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
11. DE: Grant PERM Operator Authorization
12. PERM: Create Root Permission
13. PERM: Create Permission (Self-Create)
14. PERM: Adjust Permission
15. PERM: Revoke Permission
16. PERM: Start Permission VP
17. PERM: Set Permission VP To Validated
18. PERM: Renew + Cancel Permission VP
19. PERM: Create/Update Permission Session
20. PERM: Slash Permission Trust Deposit
21. PERM: Repay Slashed Trust Deposit

## Resource Reuse Strategy

### Journey Results Storage
- Results saved as JSON files in `journey_results/`
- Each journey saves IDs needed by subsequent journeys
- `journey_results/` is gitignored

### Transaction Count
- TR/CS journeys: ~10 transactions
- PERM journeys: ~30+ transactions (including prerequisites)
- Total: ~40+ transactions across all journeys

## References

- [Verana VPR Specification](https://verana-labs.github.io/verifiable-trust-vpr-spec/)
- MOD-TR-MSG-1 through MOD-TR-MSG-5: Trust Registry messages
- MOD-CS-MSG-1 through MOD-CS-MSG-3: Credential Schema messages
- MOD-PERM-MSG-1 through MOD-PERM-MSG-14: Permission messages
