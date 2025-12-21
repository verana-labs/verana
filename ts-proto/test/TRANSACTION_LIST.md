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

### 📋 Planned
- [ ] `MsgAddDID` - Add a new DID to the directory
- [ ] `MsgRenewDID` - Renew an existing DID
- [ ] `MsgRemoveDID` - Remove a DID from the directory
- [ ] `MsgTouchDID` - Touch/update a DID's modified timestamp

## Credential Schema (cs) Module

### 📋 Planned
- [ ] `MsgCreateCredentialSchema` - Create a new credential schema
- [ ] `MsgUpdateCredentialSchema` - Update a credential schema
- [ ] `MsgArchiveCredentialSchema` - Archive/unarchive a credential schema

## Permission (perm) Module

### 📋 Planned (Future)
- [ ] `MsgCreatePermission` - Create a permission
- [ ] `MsgCreateRootPermission` - Create a root permission
- [ ] `MsgExtendPermission` - Extend a permission
- [ ] `MsgRevokePermission` - Revoke a permission
- [ ] `MsgStartPermissionVP` - Start permission validation process
- [ ] `MsgRenewPermissionVP` - Renew permission validation process
- [ ] `MsgSetPermissionVPToValidated` - Set permission VP to validated
- [ ] `MsgCancelPermissionVPLastRequest` - Cancel permission VP last request
- [ ] `MsgCreateOrUpdatePermissionSession` - Create or update permission session
- [ ] `MsgSlashPermissionTrustDeposit` - Slash permission trust deposit
- [ ] `MsgRepayPermissionSlashedTrustDeposit` - Repay permission slashed trust deposit

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

2. **DID Directory (dd)** - Next
   - AddDID
   - RenewDID
   - RemoveDID
   - TouchDID

3. **Credential Schema (cs)** - After DID
   - CreateCredentialSchema
   - UpdateCredentialSchema
   - ArchiveCredentialSchema

4. **Permission (perm)** - Later
   - All 11 permission messages

5. **Trust Deposit (td)** - Later
   - All 4 trust deposit messages

## Notes

- Focus on transaction signing validation, not business logic
- Each test should be simple: create message, sign, broadcast, verify success
- Use the same test account (`cooluser`) for consistency
- Follow the pattern from `createTrustRegistry.ts`

