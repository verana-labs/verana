# Verana Transaction Test Coverage Plan

This document lists all transaction types in Verana and tracks test implementation progress.

## Trust Registry (tr) Module

### ✅ Completed
- [x] `MsgCreateTrustRegistry` - Create a new trust registry
- [x] `MsgUpdateTrustRegistry` - Update trust registry DID/AKA
- [x] `MsgArchiveTrustRegistry` - Archive/unarchive a trust registry
- [x] `MsgAddGovernanceFrameworkDocument` - Add a governance framework document
- [x] `MsgIncreaseActiveGovernanceFrameworkVersion` - Increase active version

## DID Directory (dd) Module

### ✅ Completed
- [x] `MsgAddDID` - Add a new DID to the directory
- [x] `MsgRenewDID` - Renew an existing DID
- [x] `MsgRemoveDID` - Remove a DID from the directory
- [x] `MsgTouchDID` - Touch/update a DID's modified timestamp

## Credential Schema (cs) Module

### ✅ Completed
- [x] `MsgCreateCredentialSchema` - Create a new credential schema
- [x] `MsgUpdateCredentialSchema` - Update a credential schema
- [x] `MsgArchiveCredentialSchema` - Archive/unarchive a credential schema

## Permission (perm) Module

### ✅ Completed
- [x] `MsgCreatePermission` - Create a permission
- [x] `MsgCreateRootPermission` - Create a root permission
- [x] `MsgExtendPermission` - Extend a permission
- [x] `MsgRevokePermission` - Revoke a permission
- [x] `MsgStartPermissionVP` - Start permission validation process
- [x] `MsgRenewPermissionVP` - Renew permission validation process
- [x] `MsgSetPermissionVPToValidated` - Set permission VP to validated
- [x] `MsgCancelPermissionVPLastRequest` - Cancel permission VP last request
- [x] `MsgCreateOrUpdatePermissionSession` - Create or update permission session

## Trust Deposit (td) Module

### 📋 Planned (Future)
- [ ] `MsgReclaimTrustDeposit` - Reclaim trust deposit
- [ ] `MsgReclaimTrustDepositYield` - Reclaim trust deposit yield
- [ ] `MsgSlashTrustDeposit` - Slash trust deposit
- [ ] `MsgRepaySlashedTrustDeposit` - Repay slashed trust deposit

## Implementation Order

1. ✅ **Trust Registry (tr)** - COMPLETE
   - ✅ CreateTrustRegistry
   - ✅ UpdateTrustRegistry
   - ✅ ArchiveTrustRegistry
   - ✅ AddGovernanceFrameworkDocument
   - ✅ IncreaseActiveGovernanceFrameworkVersion

2. ✅ **DID Directory (dd)** - COMPLETE
   - ✅ AddDID
   - ✅ RenewDID
   - ✅ RemoveDID
   - ✅ TouchDID

3. ✅ **Credential Schema (cs)** - COMPLETE
   - ✅ CreateCredentialSchema
   - ✅ UpdateCredentialSchema
   - ✅ ArchiveCredentialSchema

4. ✅ **Permission (perm)** - COMPLETE
   - ✅ CreatePermission
   - ✅ CreateRootPermission
   - ✅ ExtendPermission
   - ✅ RevokePermission
   - ✅ StartPermissionVP
   - ✅ RenewPermissionVP
   - ✅ SetPermissionVPToValidated
   - ✅ CancelPermissionVPLastRequest
   - ✅ CreateOrUpdatePermissionSession
   - ✅ SlashPermissionTrustDeposit
   - ✅ RepayPermissionSlashedTrustDeposit

5. **Trust Deposit (td)** - Later
   - All 4 trust deposit messages

## Notes

- Focus on transaction signing validation, not business logic
- Each test should be simple: create message, sign, broadcast, verify success
- Use the same test account (`cooluser`) for consistency
- Follow the pattern from `createTrustRegistry.ts`

