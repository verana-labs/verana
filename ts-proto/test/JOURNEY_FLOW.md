# Journey Flow Document

This document defines the transaction sequences, prerequisites, and account strategy for all 21 TypeScript client journey tests.

## Overview

**Strategy**: Use multiple accounts to avoid sequence number conflicts. Each journey uses a dedicated account (or reuses an account where appropriate) to ensure one transaction per journey (maximum 2 if funding is needed).

**Master Account**: `cooluser` (mnemonic: "pink glory help gown abstract eight nice crazy forward ketchup skill cheese")
- Used for: TR operations, DD operations, CS operations, and funding other accounts

**Account Creation Pattern**:
1. Generate new account from mnemonic (derived from master account with different derivation paths)
2. Fund account from `cooluser` (if needed)
3. Wait 10 seconds for balance to be reflected
4. Execute transaction

## Journey List (1-21)

### Trust Registry (TR) Module (Journeys 1-5)

#### Journey 1: Create Trust Registry
- **Account**: `cooluser` (master account)
- **Prerequisites**: None
- **Transactions**:
  1. Create Trust Registry (MOD-TR-MSG-1)
- **Outputs**: `trustRegistryId`, `did`, `controller` (saved to `active-tr.json`)

#### Journey 2: Update Trust Registry
- **Account**: `cooluser` (reuse)
- **Prerequisites**: Active TR (from Journey 1)
- **Transactions**:
  1. Update Trust Registry (MOD-TR-MSG-4)
- **Inputs**: Load `active-tr.json` for `trustRegistryId`

#### Journey 3: Archive Trust Registry
- **Account**: `cooluser` (reuse)
- **Prerequisites**: Active TR (from Journey 1)
- **Transactions**:
  1. Archive Trust Registry (MOD-TR-MSG-5)
- **Inputs**: Load `active-tr.json` for `trustRegistryId`
- **Note**: After archiving, Journey 4 will create a new TR

#### Journey 4: Add Governance Framework Document
- **Account**: `cooluser` (reuse)
- **Prerequisites**: TR (can be archived - code doesn't check archived status)
- **Transactions**:
  1. Add Governance Framework Document (MOD-TR-MSG-2)
- **Inputs**: Load `active-tr.json` for `trustRegistryId` (reuse even if archived)
- **Outputs**: None (TR remains the same)

#### Journey 5: Increase Active Governance Framework Version
- **Account**: `cooluser` (reuse)
- **Prerequisites**: TR with GF documents (from Journey 4, can be archived)
- **Transactions**:
  1. Increase Active Governance Framework Version (MOD-TR-MSG-3)
- **Inputs**: Load `active-tr.json` for `trustRegistryId` (reuse even if archived)
- **Note**: Requires TR from Journey 4 that has at least one governance framework document added.

---

### DID Directory (DD) Module (Journeys 6-9)

#### Journey 6: Add DID
- **Account**: `cooluser` (reuse - no conflicts with TR/CS)
- **Prerequisites**: None (DD is independent)
- **Transactions**:
  1. Add DID (MOD-DD-MSG-1)

#### Journey 7: Renew DID
- **Account**: `cooluser` (reuse)
- **Prerequisites**: DID from Journey 6
- **Transactions**:
  1. Renew DID (MOD-DD-MSG-2)
- **Inputs**: DID from Journey 6 (or load from journey results)

#### Journey 8: Remove DID
- **Account**: `cooluser` (reuse)
- **Prerequisites**: DID from Journey 6 or 7
- **Transactions**:
  1. Remove DID (MOD-DD-MSG-3)
- **Inputs**: DID from previous journey

#### Journey 9: Touch DID
- **Account**: `cooluser` (reuse)
- **Prerequisites**: DID from Journey 6, 7, or 8
- **Transactions**:
  1. Touch DID (MOD-DD-MSG-4)
- **Inputs**: DID from previous journey

---

### Credential Schema (CS) Module (Journeys 10-12)

#### Journey 10: Create Credential Schema
- **Account**: `cooluser` (reuse)
- **Prerequisites**: TR (can be archived - code doesn't check archived status, only checks controller)
- **Transactions**:
  1. Create Credential Schema (MOD-CS-MSG-1)
     - **Critical**: Set `issuerPermManagementMode` and `verifierPermManagementMode` to `OPEN` (1)
     - This allows direct permission creation without grantor validation
- **Inputs**: Load `active-tr.json` for `trustRegistryId` (reuse even if archived)
- **Outputs**: `schemaId`, `trustRegistryId`, `did` (saved to `active-cs.json` and `active-tr.json`)

#### Journey 11: Update Credential Schema
- **Account**: `cooluser` (reuse)
- **Prerequisites**: Active CS (from Journey 10)
- **Transactions**:
  1. Update Credential Schema (MOD-CS-MSG-2)
- **Inputs**: Load `active-cs.json` for `schemaId`

#### Journey 12: Archive Credential Schema
- **Account**: `cooluser` (reuse)
- **Prerequisites**: Active CS (from Journey 10 or 11)
- **Transactions**:
  1. Archive Credential Schema (MOD-CS-MSG-3)
- **Inputs**: Load `active-cs.json` for `schemaId`
- **Note**: After archiving, Journey 13 will create new TR/CS if needed

---

### Permission (PERM) Module (Journeys 13-21)

**Important**: All permission journeys use NEW accounts to avoid sequence conflicts. Each account is funded by `cooluser` before use.

#### Journey 13: Create Root Permission
- **Account**: `account_13` (NEW - derived from master mnemonic with path `m/44'/118'/0'/0/13`)
- **Prerequisites**: 
  - Active TR (from Journey 1, 2, or 4)
  - Active CS with OPEN permission mode (from Journey 10 or 11)
- **Setup**:
  1. Create account from mnemonic (derivation path 13)
  2. Fund account from `cooluser` (send sufficient tokens)
  3. Wait 10 seconds for balance reflection
- **Transactions**:
  1. Create Root Permission (MOD-PERM-MSG-7)
     - Type: ECOSYSTEM
     - This is the validator permission required for the schema
- **Inputs**: Load `active-tr.json` and `active-cs.json` for `trustRegistryId`, `schemaId`, `did`
- **Outputs**: `rootPermissionId` (save to journey results for reuse)

#### Journey 14: Create Permission
- **Account**: `account_14` (NEW - derived from master mnemonic with path `m/44'/118'/0'/0/14`)
- **Prerequisites**: 
  - Active TR (from Journey 1, 2, or 4)
  - Active CS with OPEN permission mode (from Journey 10 or 11)
  - Root Permission (from Journey 13) - **REQUIRED**
- **Setup**:
  1. Create account from mnemonic (derivation path 14)
  2. Fund account from `cooluser`
  3. Wait 10 seconds for balance reflection
- **Transactions**:
  1. Create Permission (MOD-PERM-MSG-14)
     - Type: ISSUER (default)
     - Uses root permission ID from Journey 13 as `validatorPermId`
- **Inputs**: Load `active-tr.json`, `active-cs.json`, and root permission ID from Journey 13
- **Outputs**: `permissionId` (save to journey results for reuse in Journeys 15-16)

#### Journey 15: Extend Permission
- **Account**: `account_14` (REUSE from Journey 14)
- **Prerequisites**: 
  - Active TR and CS
  - Permission from Journey 14 (must be effective)
- **Setup**:
  1. Wait for permission from Journey 14 to become effective (effectiveFrom is 10 seconds in future)
  2. Wait additional time if needed
- **Transactions**:
  1. Extend Permission (MOD-PERM-MSG-8)
- **Inputs**: Load permission ID from Journey 14

#### Journey 16: Revoke Permission
- **Account**: `account_14` (REUSE from Journey 14)
- **Prerequisites**: 
  - Active TR and CS
  - Permission from Journey 14 (must be effective)
- **Setup**:
  1. Ensure permission from Journey 14 is effective
- **Transactions**:
  1. Revoke Permission (MOD-PERM-MSG-9)
- **Inputs**: Load permission ID from Journey 14

#### Journey 17: Start Permission VP
- **Account**: `account_17` (NEW - derived from master mnemonic with path `m/44'/118'/0'/0/17`)
- **Prerequisites**: 
  - Active TR and CS
  - Root Permission (from Journey 13) - used as validator permission
- **Setup**:
  1. Create account from mnemonic (derivation path 17)
  2. Fund account from `cooluser`
  3. Wait 10 seconds for balance reflection
- **Transactions**:
  1. Start Permission VP (MOD-PERM-MSG-1)
     - Type: ISSUER (default)
     - Uses root permission ID from Journey 13 as `validatorPermId`
     - **Note**: The creator (`account_17`) becomes the `grantee` of the created permission
- **Inputs**: Load `active-tr.json`, `active-cs.json`, and root permission ID from Journey 13
- **Outputs**: `permissionId` (save for Journeys 18 and 20)

#### Journey 18: Renew Permission VP
- **Account**: `account_17` (REUSE from Journey 17 - **MUST be the same account**)
- **Prerequisites**: 
  - Active TR and CS
  - Root Permission (from Journey 13)
  - Permission with active VP (from Journey 17)
- **Code Check**: `applicantPerm.Grantee != msg.Creator` - **ONLY the grantee can renew**
- **Setup**:
  1. Reuse `account_17` (same account that started the VP in Journey 17)
  2. Ensure permission from Journey 17 is in VALIDATED state (not PENDING)
- **Transactions**:
  1. Renew Permission VP (MOD-PERM-MSG-2)
- **Inputs**: Load permission ID from Journey 17 (must have active VP, grantee must be account_17)

#### Journey 19: Set Permission VP To Validated
- **Account**: `account_13` (REUSE from Journey 13 - **MUST be the validator grantee**)
- **Prerequisites**: 
  - Active TR and CS
  - Root Permission (from Journey 13) - **account_13 is the grantee of this validator permission**
  - Permission VP started (from Journey 17) - must be in PENDING state
- **Code Check**: `validatorPerm.Grantee != msg.Creator` - **ONLY the validator permission grantee can validate**
- **Setup**:
  1. Reuse `account_13` (the account that created the root permission in Journey 13, which is the validator)
  2. Ensure permission from Journey 17 is in PENDING state
- **Transactions**:
  1. Set Permission VP To Validated (MOD-PERM-MSG-3)
- **Inputs**: Load permission ID from Journey 17 (must be PENDING), root permission ID from Journey 13

#### Journey 20: Cancel Permission VP Last Request
- **Account**: `account_17` (REUSE from Journey 17 - **MUST be the same account**)
- **Prerequisites**: 
  - Active TR and CS
  - Root Permission (from Journey 13)
  - Permission VP started (from Journey 17) - must be in PENDING state
- **Code Check**: `applicantPerm.Grantee != msg.Creator` - **ONLY the grantee can cancel**
- **Setup**:
  1. Reuse `account_17` (same account that started the VP in Journey 17)
  2. Ensure permission from Journey 17 is in PENDING state
  3. **Alternative**: Start a new permission VP with account_17 if needed
- **Transactions**:
  1. Cancel Permission VP Last Request (MOD-PERM-MSG-6)
- **Inputs**: Load permission ID from Journey 17 (must be PENDING, grantee must be account_17)

#### Journey 21: Create Or Update Permission Session
- **Account**: `account_21` (NEW - derived from master mnemonic with path `m/44'/118'/0'/0/21`)
- **Prerequisites**: 
  - Active TR and CS
  - Root Permission (from Journey 13)
  - Issuer Permission (from Journey 14)
  - Verifier Permission (create new with type VERIFIER)
  - Agent Permission (create new with type ISSUER or VERIFIER)
- **Setup**:
  1. Create account from mnemonic (derivation path 21)
  2. Fund account from `cooluser`
  3. Wait 10 seconds for balance reflection
  4. Create issuer permission (reuse Journey 14 logic) OR use from Journey 14
  5. Create verifier permission (new, type VERIFIER)
  6. Create agent permission (new, type ISSUER or VERIFIER)
- **Transactions**:
  1. Create Or Update Permission Session (MOD-PERM-MSG-10)
- **Inputs**: Load issuer, verifier, and agent permission IDs

---

## Account Derivation Strategy

All accounts are derived from the master mnemonic using HD wallet derivation:

```
Master Mnemonic: "pink glory help gown abstract eight nice crazy forward ketchup skill cheese"

Account Derivation:
- cooluser (Journeys 1-12): m/44'/118'/0'/0/0 (default)
- account_13: m/44'/118'/0'/0/13 (Journey 13, reused in Journey 19)
- account_14: m/44'/118'/0'/0/14 (Journey 14, reused in Journeys 15-16)
- account_15: m/44'/118'/0'/0/14 (reuse 14)
- account_16: m/44'/118'/0'/0/14 (reuse 14)
- account_17: m/44'/118'/0'/0/17 (Journey 17, reused in Journeys 18 and 20)
- account_18: m/44'/118'/0'/0/17 (reuse 17 for Journey 18)
- account_19: m/44'/118'/0'/0/13 (reuse 13 for Journey 19)
- account_20: m/44'/118'/0'/0/17 (reuse 17 for Journey 20)
- account_21: m/44'/118'/0'/0/21
```

## Funding Pattern

For journeys 13-21 (new accounts):

1. **Funding Transaction** (from `cooluser`):
   - Send sufficient tokens (e.g., 1000000000uvna) to new account
   - Use `cooluser` account to send funds

2. **Wait Period**:
   - Wait 10 seconds after funding for balance to be reflected on-chain
   - This ensures the account has funds before executing the main transaction

3. **Main Transaction**:
   - Execute the journey's main transaction using the new account

## Resource Reuse Strategy

### Active TR/CS Management

- **Storage**: `active-tr.json` and `active-cs.json` in `journey_results/`
- **Update Rules**:
  - When a new TR is created → update `active-tr.json`
  - When a new CS is created → update `active-cs.json` and `active-tr.json` (if TR was created)
  - When TR/CS is archived → next journey that needs it will create new ones

### Permission ID Storage

- **Root Permission**: Stored in journey results after Journey 13
- **Regular Permissions**: Stored in journey results after creation
- **Reuse**: Later journeys load permission IDs from journey results

## Transaction Count Per Journey

- **Journeys 1-12**: 1 transaction (using `cooluser`)
- **Journeys 13-21**: 2 transactions (1 funding + 1 main transaction)
  - Exception: Journey 21 may need 4 transactions (funding + 3 permission creations + 1 session creation)

## Sequence Number Management

By using different accounts for each journey (especially 13-21), we eliminate sequence number conflicts:
- Each account starts with sequence 0
- Each account only sends 1-2 transactions
- No race conditions between journeys

## Implementation Notes

1. **Account Creation**: Use `DirectSecp256k1HdWallet.fromMnemonic()` or `Secp256k1HdWallet.fromMnemonic()` with custom derivation path
2. **Funding**: Use `cooluser` account to send tokens to new accounts via bank transfer
3. **Waiting**: Use `setTimeout` or query client to wait for balance/transaction confirmation
4. **Resource Loading**: Use `getActiveTRAndSchema()` helper to load active TR/CS
5. **Permission Storage**: Create helper functions to save/load permission IDs from journey results

## References

- [Verana VPR Specification](https://verana-labs.github.io/verifiable-trust-vpr-spec/)
- MOD-TR-MSG-1 through MOD-TR-MSG-6: Trust Registry messages
- MOD-CS-MSG-1 through MOD-CS-MSG-4: Credential Schema messages
- MOD-PERM-MSG-1 through MOD-PERM-MSG-14: Permission messages
- MOD-DD-MSG-1 through MOD-DD-MSG-5: DID Directory messages

