# Step 19: DE GrantVsOperatorAuthorization — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Document that `GrantVsOperatorAuthorization` is not a standalone DE message and is fully covered by PERM tests.

**Architecture:** The DE module only exposes `GrantOperatorAuthorization` and `RevokeOperatorAuthorization` as standalone MsgServer messages. VS Operator Authorization is orchestrated internally via the `delegationKeeper.GrantFeeAllowance` / `RemovePermFromVSOA` interface, called from within PERM's `AdjustPermission` (step 26) and `SelfCreatePermission` (step 31). Unit test coverage comes from those PERM tests using a `MockDelegationKeeper`.

**Tech Stack:** N/A — no new test files needed.

---

## Status: No Implementation Required

`x/de/types/tx.pb.go` defines the `MsgServer` interface with exactly two messages:
- `GrantOperatorAuthorization` (covered in step 17)
- `RevokeOperatorAuthorization` (covered in step 18)

There is no standalone `GrantVsOperatorAuthorization` message in the DE module. The VS operator authorization logic is handled:
1. **Grant path**: `PERM AdjustPermission` → `grantVSOperatorAuthorization` → `delegationKeeper.GrantFeeAllowance`
2. **Grant path**: `PERM SelfCreatePermission` → `grantVSOperatorAuthorization`
3. **Revoke path**: `PERM RevokePermission` → `revokeVSOperatorAuthorization` → `delegationKeeper.RemovePermFromVSOA`
4. **Revoke path**: `PERM SlashPermissionTrustDeposit` → `revokeVSOperatorAuthorization`

Coverage comes from:
- Step 26 (AdjustPermission): verifies VS operator authz grant side effect
- Step 27 (RevokePermission): verifies VS operator authz revoke side effect
- Step 29 (SlashPermissionTrustDeposit): verifies VS operator authz revoke side effect
- Step 31 (SelfCreatePermission): verifies VS operator authz grant side effect

- [ ] **Action**: Verify the DE proto (`proto/verana/de/v1/tx.proto`) does not define `GrantVsOperatorAuthorization`. If it does in a future version, implement this step then.

```bash
grep -r "GrantVsOperator\|RevokeVsOperator" proto/verana/de/
```

Expected: no output (no such messages defined).

- [ ] **Mark done**: This step requires no code changes. Update the issue tracker to note DE VS operator authorization is covered via PERM test steps 26, 27, 29, 31.
