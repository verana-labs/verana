# Step 20: DE RevokeVsOperatorAuthorization — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Document that `RevokeVsOperatorAuthorization` is not a standalone DE message and is fully covered by PERM tests.

**Architecture:** Same as step 19 — VS operator authorization revocation is orchestrated internally from PERM messages (RevokePermission, SlashPermissionTrustDeposit) via the `delegationKeeper` interface. No standalone DE message exists.

**Tech Stack:** N/A — no new test files needed.

---

## Status: No Implementation Required

See step 19 for full explanation. Coverage for the revoke path comes from:
- Step 27 (RevokePermission): `revokeVSOperatorAuthorization` is called for ISSUER/VERIFIER perms
- Step 29 (SlashPermissionTrustDeposit): `revokeVSOperatorAuthorization` is called for ISSUER/VERIFIER perms

- [ ] **Action**: Verify the DE proto does not define `RevokeVsOperatorAuthorization`.

```bash
grep -r "RevokeVsOperator" proto/verana/de/
```

Expected: no output.

- [ ] **Mark done**: No code changes required. Covered by PERM steps 27 and 29.
