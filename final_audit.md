# Verana Chain — Final Spec Compliance Audit Report

**Spec:** VPR v4 — https://verana-labs.github.io/verifiable-trust-vpr-spec/  
**Branch:** `feat/align-spec-v4-draft-13`  
**Date:** 2026-04-16  
**Method:** Parallel swarm audit — one agent per module, reading proto + keeper + types + codec + ts-proto  
**Confidence threshold:** All findings confirmed at ≥ 0.80 from actual code

---

## Executive Summary

| Severity | Count |
|---|---|
| CRITICAL | 3 |
| HIGH | 16 |
| MEDIUM | 20 |
| LOW | 21 |
| **Total** | **60** |

| Module | Critical | High | Medium | Low | Total |
|---|---|---|---|---|---|
| TR — Trust Registry | 0 | 1 | 3 | 3 | 7 |
| CS — Credential Schema | **3** | 3 | 1 | 2 | 9 |
| PERM — Permission | 0 | 5 | 7 | 3 | 15 |
| TD — Trust Deposit | 0 | 1 | 1 | 5 | 7 |
| DE — Delegation | 0 | 2 | 3 | 3 | 8 |
| DI — Digests | 0 | 0 | 2 | 4 | 6 |
| XR — Exchange Rate | 0 | 4 | 3 | 1 | 8 |

### Top Priorities

1. **[CS-5,6,7] Schema Authorization Policy is entirely unimplemented** — 3 critical, core feature missing
2. **[XR-MSG-1] Exchange rates created in disabled state** — spec mandates `state=true` on creation
3. **[TD-MSG-2] Amino converter drops `amount` field** — ReclaimTrustDepositYield un-signable via amino/Ledger
4. **[DI-MSG-1] Amino converter drops `digest_algorithm` + wrong aminoType** — StoreDigest un-signable
5. **[DE-MSG-3] `fee_spend_limit` never stored** — fee spend limits silently dropped on every grant
6. **[PERM-MSG-12] Slash authorization uses ancestor logic, not governance authority**
7. **[TR-MSG-5 / CS-MSG-3] archive=false path hard-rejected** — unarchive permanently impossible in both modules
8. **[XR-MSG-2] rate_scale and validity_duration not updatable** — proto missing those fields entirely
9. **[PERM-MSG-7] permission_type hardcoded to ECOSYSTEM; vs_operator missing**
10. **[PERM-MSG-9] Trust deposit not freed on permission revocation**

---

## Module: Trust Registry (TR)

### TR-HIGH-1 — MOD-TR-MSG-5: Unarchive path permanently blocked
- **File:** [x/tr/keeper/msg_server.go:245](x/tr/keeper/msg_server.go#L245)
- **Spec says:** `if archive=false: set archived=null; set modified=now`
- **Code does:** Hard-returns error for any `msg.Archive == false`: `"archive cannot be set to false; archiving is a terminal operation"` before any state check
- **Impact:** TrustRegistry can never be unarchived. The spec defines archive as a toggle; the code treats it as permanent.

### TR-MEDIUM-1 — MOD-TR-MSG-5: Wrong error semantics for archive=false
- **File:** [x/tr/keeper/msg_server.go:245](x/tr/keeper/msg_server.go#L245)
- **Spec says:** MUST abort if `archive=false` but not archived (stateful check)
- **Code does:** Rejects archive=false unconditionally before the stateful check, producing misleading error message
- **Impact:** Clients receive wrong error ("terminal operation" vs "not currently archived")

### TR-MEDIUM-2 — MOD-TR-MSG-2: O(n) GFDocument table walk, no secondary index
- **File:** [x/tr/keeper/gfd.go:106](x/tr/keeper/gfd.go#L106)
- **Spec says:** Create or replace GFD with (gfv_id, doc_language) key
- **Code does:** Full `GFDocument.Walk` to find existing document by (gfv_id, language) — no index
- **Impact:** Gas exhaustion on large chains, blocking legitimate AddGovernanceFrameworkDocument transactions

### TR-MEDIUM-3 — MOD-TR-MSG-3: O(n) GFDocument walk in version increment
- **File:** [x/tr/keeper/gfv.go:38](x/tr/keeper/gfv.go#L38)
- **Spec says:** MUST abort if no GFD exists for target version with language matching tr.language
- **Code does:** Full Walk over all GFDocuments to find matching (gfv_id, language) pair
- **Impact:** Same gas exhaustion risk as TR-MEDIUM-2

### TR-LOW-1 — MOD-TR-MSG-6: MsgUpdateParams missing from legacy amino codec
- **File:** [x/tr/types/codec.go:12](x/tr/types/codec.go#L12)
- **Spec says:** Every message registered in amino codec (required for Ledger/governance signing)
- **Code does:** Registers 5 user-facing messages but omits MsgUpdateParams despite its `amino.name` annotation in proto
- **Impact:** MsgUpdateParams cannot be signed via Ledger or amino-based governance flows

### TR-LOW-2 — MOD-TR-MSG-6: MsgUpdateParams has no TypeScript amino converter
- **File:** [ts-proto/src/amino-converter/tr.ts](ts-proto/src/amino-converter/tr.ts)
- **Spec says:** Every message should have amino converter
- **Code does:** No entry for MsgUpdateParams; also absent from `signing.ts` veranaTypeUrls/createVeranaAminoTypes
- **Impact:** TypeScript clients cannot construct amino-encoded MsgUpdateParams governance proposals

### TR-LOW-3 — MOD-TR-MSG-2: Version=0 conflated with missing version in ValidateBasic
- **File:** [x/tr/types/types.go:56](x/tr/types/types.go#L56)
- **Spec says:** Mandatory param: version. MUST abort if missing.
- **Code does:** `msg.Version == 0` used as missing sentinel; produces "missing mandatory parameter" for explicitly-sent version=0
- **Impact:** Misleading error message; no security impact

---

## Module: Credential Schema (CS)

### CS-CRITICAL-1 — MOD-CS-MSG-5: Create Schema Authorization Policy entirely unimplemented
- **File:** [proto/verana/cs/v1/tx.proto:14](proto/verana/cs/v1/tx.proto#L14)
- **Spec says:** Create SchemaAuthorizationPolicy with auto-id, schema_id, created=now, version=1, role, url, digest_sri, effective_from, effective_until, revoked=false
- **Code does:** No RPC method, no proto message, no keeper handler, no type for MsgCreateSchemaAuthorizationPolicy exists
- **Impact:** Schema Authorization Policies cannot be created. Role-based policy workflows are entirely blocked.

### CS-CRITICAL-2 — MOD-CS-MSG-6: Increase Active Schema Authorization Policy Version entirely unimplemented
- **File:** [proto/verana/cs/v1/tx.proto:14](proto/verana/cs/v1/tx.proto#L14)
- **Spec says:** Identify next policy version, mark as active, update effective_from if needed
- **Code does:** No RPC, no proto, no handler exists
- **Impact:** Schema Authorization Policy versioning non-functional

### CS-CRITICAL-3 — MOD-CS-MSG-7: Revoke Schema Authorization Policy entirely unimplemented
- **File:** [proto/verana/cs/v1/tx.proto:14](proto/verana/cs/v1/tx.proto#L14)
- **Spec says:** Load policy, set revoked=true, persist
- **Code does:** No RPC, no proto, no handler exists
- **Impact:** Schema Authorization Policies can never be revoked

### CS-HIGH-1 — MOD-CS-MSG-2: Proto missing 6 of 13 updatable fields
- **File:** [proto/verana/cs/v1/tx.proto:77](proto/verana/cs/v1/tx.proto#L77)
- **Spec says:** Optional updatable: json_schema, validity periods (5), onboarding modes (3), pricing_asset_type, pricing_asset, digest_algorithm
- **Code does:** MsgUpdateCredentialSchema only contains 5 validity period fields. json_schema, onboarding modes, pricing_asset_type, pricing_asset, digest_algorithm absent.
- **Impact:** json_schema, onboarding modes, pricing asset, and digest_algorithm are frozen at creation with no update path

### CS-HIGH-2 — MOD-CS-MSG-2: ValidateBasic mandates all 5 validity period fields; spec says optional
- **File:** [x/cs/types/types.go:702](x/cs/types/types.go#L702)
- **Spec says:** Mandatory: id only. Optional (at-least-one): validity periods, onboarding modes, etc.
- **Code does:** Returns error if any of the 5 validity period optional wrappers is nil; no "at least one field" check
- **Impact:** Clients must supply all 5 validity periods even when only updating one; partial-update semantics not honored

### CS-HIGH-3 — MOD-CS-MSG-3: archive=false (unarchive) path hard-rejected
- **File:** [x/cs/keeper/msg_server.go:215](x/cs/keeper/msg_server.go#L215)
- **Spec says:** `if archive=false: set archived=null; set modified=now`
- **Code does:** `if !msg.Archive { return nil, fmt.Errorf("archive cannot be set to false; use credential schema lifecycle management") }`
- **Impact:** Credential schemas can never be unarchived. Once archived, permanently locked.

### CS-MEDIUM-1 — MOD-CS-MSG-3: Missing precondition: archive=false but not archived
- **File:** [x/cs/keeper/msg_server.go:181](x/cs/keeper/msg_server.go#L181)
- **Spec says:** MUST abort if archive=false but not archived
- **Code does:** Unarchive branch is rejected wholesale (CS-HIGH-3); no check that schema is currently archived
- **Impact:** If unarchive were implemented, attempting to unarchive a non-archived schema would silently succeed

### CS-LOW-1 — MOD-CS-MSG-4: UpdateParams does not validate per-key existence
- **File:** [x/cs/keeper/msg_update_params.go:12](x/cs/keeper/msg_update_params.go#L12)
- **Spec says:** MUST abort if any key not exist
- **Code does:** Calls `ms.SetParams(ctx, req.Params)` directly with no per-key existence check
- **Impact:** Malformed governance proposals with unknown keys silently accepted

### CS-LOW-2 — MOD-CS-MSG-1: ID counter incremented before precondition checks
- **File:** [x/cs/keeper/msg_server.go:43](x/cs/keeper/msg_server.go#L43)
- **Spec says:** Precondition checks first, then create with auto-id
- **Code does:** `GetNextID` called before `validateCreateCredentialSchemaParams`; ID gaps on failures
- **Impact:** Sparse ID sequence when creates fail after auth but before validation

---

## Module: Permission (PERM)

### PERM-HIGH-1 — MOD-PERM-MSG-2: permission_type field missing from proto; match check absent — RETRACTED 2026-04-24
- **Retraction reason:** Spec v4 draft 13 [MOD-PERM-MSG-2-1] lists only `corporation, operator, id` as parameters. `permission_type` is not a parameter of MsgRenewPermissionVP. Original finding misread the spec.
- **Original text preserved:** File [proto/verana/perm/v1/tx.proto:98], claimed spec requires permission_type mandatory with match check.
- **Action:** PR #280 added the spurious field based on this misread; reverted in branch `fix/perm-spec-v4-revert`.

### PERM-HIGH-2 — MOD-PERM-MSG-7: permission_type hardcoded to ECOSYSTEM — RETRACTED 2026-04-24
- **Retraction reason:** Spec v4 draft 13 [MOD-PERM-MSG-7-3] explicitly states `perm.type: ECOSYSTEM` is hardcoded. The original implementation was spec-correct.
- **Original text preserved:** File [x/perm/keeper/msg_server.go:706], claimed spec requires permission_type ∈ {ISSUER, VERIFIER, ISSUER_GRANTOR, VERIFIER_GRANTOR}. This is wrong; MSG-7 creates an ECOSYSTEM root permission only.
- **Action:** PR #280 implemented this incorrect finding; reverted in branch `fix/perm-spec-v4-revert`. Handler restored to hardcoded ECOSYSTEM.

### PERM-HIGH-3 — MOD-PERM-MSG-7: vs_operator field missing from MsgCreateRootPermission — RETRACTED 2026-04-24
- **Retraction reason:** Spec v4 draft 13 [MOD-PERM-MSG-7-1] does not include `vs_operator` in MSG-7 parameters, and [MOD-PERM-MSG-7-3] execution does not set `perm.vs_operator`. ECOSYSTEM root permissions are never used as Permission Session targets; only perms created via MSG-1 (StartPermissionVP) and MSG-14 (SelfCreatePermission) carry vs_operator.
- **Original text preserved:** File [proto/verana/perm/v1/tx.proto:150], claimed spec requires vs_operator mandatory on MSG-7.
- **Action:** PR #280 added the spurious field based on this misread; reverted in branch `fix/perm-spec-v4-revert`.

### PERM-HIGH-4 — MOD-PERM-MSG-12: reason field missing from SlashPermissionTrustDeposit
- **File:** [proto/verana/perm/v1/tx.proto:221](proto/verana/perm/v1/tx.proto#L221)
- **Spec says:** Mandatory: corporation, operator, permission_id, slash_amount, reason
- **Code does:** Proto has corporation, operator, id, amount — no reason field. ValidateBasic also has no reason check.
- **Impact:** Governance/validator slashes carry no on-chain justification; auditability broken

### PERM-HIGH-5 — MOD-PERM-MSG-12: Slash uses validator-ancestor authorization, not governance authority
- **File:** [x/perm/keeper/msg_server.go:1262](x/perm/keeper/msg_server.go#L1262)
- **Spec says:** MUST abort if corporation not governance authority
- **Code does:** `validateSlashPermissionValidatorPerms` checks `checkValidatorAncestorOption` OR `checkTrustRegistryControllerOption` — any ancestor or TR controller can slash
- **Impact:** Significant authorization scope mismatch; slashing can be performed by non-governance actors

### PERM-MEDIUM-1 — MOD-PERM-MSG-2: Only VALIDATED accepted; spec allows PENDING too
- **File:** [x/perm/keeper/msg_server.go:120](x/perm/keeper/msg_server.go#L120)
- **Spec says:** MUST abort if vp_state not VALIDATED or PENDING
- **Code does:** `if applicantPerm.VpState != types.ValidationState_VALIDATED { return nil, ... }` — only VALIDATED accepted
- **Impact:** Permissions in PENDING state cannot be renewed, contra spec

### PERM-MEDIUM-2 — MOD-PERM-MSG-8: AdjustPermission missing validation_fees, issuance_fees, verification_fees, effective_from, and discounts
- **File:** [proto/verana/perm/v1/tx.proto:175](proto/verana/perm/v1/tx.proto#L175)
- **Spec says:** Adjustment fields include: validation_fees, issuance_fees, verification_fees, effective_from, effective_until, discounts
- **Code does:** Proto only has corporation, operator, id, effective_until. Execution only updates effective_until.
- **Impact:** Fees and discounts cannot be adjusted; spec-mandated adjustment fields are a partial stub

### PERM-MEDIUM-3 — MOD-PERM-MSG-13: Checks SlashedDeposit==0 instead of Slashed!=nil
- **File:** [x/perm/keeper/msg_server.go:1338](x/perm/keeper/msg_server.go#L1338)
- **Spec says:** MUST abort if permission not exist with slashed not null
- **Code does:** `if applicantPerm.SlashedDeposit == 0` — checks amount, not timestamp
- **Impact:** Edge case where slashed timestamp and slashed_deposit could diverge; technically violates spec precondition

### PERM-MEDIUM-4 — MOD-PERM-MSG-10: AUTHZ-CHECK-3 does not pass agent_perm_id for membership check
- **File:** [x/perm/keeper/csps.go:127](x/perm/keeper/csps.go#L127)
- **Spec says:** AUTHZ-CHECK-3: vso.permissions includes agent_perm_id
- **Code does:** `CheckVSOperatorAuthorization` takes only (authority, vsOperator) — no agent_perm_id parameter; membership check may not be enforced
- **Impact:** vs_operator may create sessions for permissions they are not authorized to service

### PERM-MEDIUM-5 — MOD-PERM-MSG-9: Trust deposit not freed on permission revocation
- **File:** [x/perm/keeper/msg_server.go:1083](x/perm/keeper/msg_server.go#L1083)
- **Spec says:** Execution: set revoked=now, free associated trust deposit, set modified=now, persist
- **Code does:** Sets revoked and modified, persists. No call to AdjustTrustDeposit or any deposit release mechanism.
- **Impact:** Trust deposit locked in revoked permission is never freed; corporations lose access to deposit funds

### PERM-MEDIUM-6 — MOD-PERM-MSG-14: validator_perm_id set to ECOSYSTEM permission, not null
- **File:** [x/perm/keeper/msg_server.go:1518](x/perm/keeper/msg_server.go#L1518)
- **Spec says:** Execution: validator_perm_id=null (self-created, no parent validator)
- **Code does:** ValidateBasic enforces non-zero validator_perm_id; execution sets ValidatorPermId: msg.ValidatorPermId
- **Impact:** Self-created permissions have a non-null validator_perm_id, potentially triggering incorrect fee distribution or revocation chains

### PERM-MEDIUM-7 — MOD-PERM-MSG-3: Full fees sent to validator wallet; no trust_deposit_rate% split
- **File:** [x/perm/keeper/perm_validated.go:143](x/perm/keeper/perm_validated.go#L143)
- **Spec says:** Distribute escrowed fees: trust_deposit_rate% → validator deposit, remainder → validator wallet
- **Code does:** Full vp_current_fees amount sent to validator wallet; vp_current_deposit handled separately from pre-computed amount — no trust_deposit_rate% fee split
- **Impact:** Fee distribution formula not followed; validators receive full escrowed fees as wallet payment

### PERM-LOW-1 — MOD-PERM-MSG-1: schema_id and applicant_corporation not explicit proto fields
- **File:** [proto/verana/perm/v1/tx.proto:61](proto/verana/perm/v1/tx.proto#L61)
- **Spec says:** Mandatory: schema_id, applicant_corporation
- **Code does:** schema_id derived from validator_perm.SchemaId; applicant_corporation implicit as msg.Corporation
- **Impact:** External verifiers cannot validate these at transaction level; design deviation from spec

### PERM-LOW-2 — MOD-PERM-MSG-11: MsgUpdateParams not in legacy amino codec
- **File:** [x/perm/types/codec.go:12](x/perm/types/codec.go#L12)
- **Code does:** Registers all 10 user messages but omits MsgUpdateParams
- **Impact:** MsgUpdateParams cannot be signed via Ledger or amino governance flows

### PERM-LOW-3 — MOD-PERM-MSG-11: No TypeScript amino converter for MsgUpdateParams
- **File:** [ts-proto/src/amino-converter/perm.ts](ts-proto/src/amino-converter/perm.ts)
- **Code does:** All 10 user-facing PERM messages have converters; MsgUpdateParams absent
- **Impact:** TypeScript clients cannot sign MsgUpdateParams via amino path

---

## Module: Trust Deposit (TD)

### TD-HIGH-1 — MOD-TD-MSG-2: Amino converter drops mandatory `amount` field
- **File:** [ts-proto/src/amino-converter/td.ts:10](ts-proto/src/amino-converter/td.ts#L10)
- **Spec says:** amount is a mandatory field (ValidateBasic: amount > 0)
- **Code does:** `toAmino: ({ corporation, operator }) => ({ corporation, operator })` — amount destructured but never included. `fromAmino` also never reads `value.amount`, so amount is always 0 after amino round-trip.
- **Impact:** Any Ledger or amino-signed ReclaimTrustDepositYield fails ValidateBasic ("amount must be > 0"). Message effectively un-signable via amino wallets.

### TD-MEDIUM-1 — MOD-TD-MSG-5: reason field missing from MsgSlashTrustDeposit
- **File:** [proto/verana/td/v1/tx.proto:67](proto/verana/td/v1/tx.proto#L67)
- **Spec says:** Mandatory: corporation, slash_amount, reason
- **Code does:** Proto has authority, corporation, deposit — no reason field. ValidateBasic has no reason check.
- **Impact:** Governance slash proposals carry no justification string; on-chain auditability broken

### TD-LOW-1 — MOD-TD-MSG-2: Partial yield claims allowed; spec says set claimable to 0
- **File:** [x/td/keeper/msg_server.go:73](x/td/keeper/msg_server.go#L73)
- **Spec says:** transfer claimable amount to corporation wallet, set claimable balance to 0
- **Code does:** `td.Claimable -= msg.Amount` — if amount < claimable, residual remains
- **Impact:** Partial yield withdrawal enabled; spec mandates full drain

### TD-LOW-2 — MOD-TD-MSG-2: No last_claimed timestamp persisted; spec says record timestamp
- **File:** [x/td/keeper/msg_server.go:101](x/td/keeper/msg_server.go#L101)
- **Spec says:** record timestamp (of claim)
- **Code does:** Emits timestamp in event only; TrustDeposit struct has no last_claimed field; not persisted to store
- **Impact:** Last yield claim not queryable from state; requires reconstructing from event logs

### TD-LOW-3 — MOD-TD-MSG-6: Zeroes slashed_deposit and repaid_deposit instead of decrementing
- **File:** [x/td/keeper/msg_server.go:256](x/td/keeper/msg_server.go#L256)
- **Spec says:** decrement slashed_deposit, increment repaid_deposit (cumulative tracker)
- **Code does:** On full repay: sets both `td.SlashedDeposit = 0` and `td.RepaidDeposit = 0`, destroying historical record
- **Impact:** After repayment, historical slash accounting reset; second slash starts fresh rather than accumulating

### TD-LOW-4 — MOD-TD-MSG-7: BurnEcosystemSlashedTrustDeposit does not zero slashed_deposit
- **File:** [x/td/keeper/burn_slashed_td.go:29](x/td/keeper/burn_slashed_td.go#L29)
- **Spec says:** burn slashed_deposit amount, zero out slashed_deposit field
- **Code does:** Decrements td.Deposit and td.Share; comment explicitly states "SlashedDeposit/LastSlashed/SlashCount are NOT updated here"
- **Impact:** td.SlashedDeposit retains its value after burn; corporations may be permanently blocked from yield reclaim if slashed_deposit guard check fires

### TD-LOW-5 — MOD-TD-MSG-2: Field number gap in MsgReclaimTrustDepositYield proto
- **File:** [proto/verana/td/v1/tx.proto:53](proto/verana/td/v1/tx.proto#L53)
- **Code does:** Fields: corporation=1, operator=2, amount=4. Field number 3 is skipped, indicating a removed field without deprecation.
- **Impact:** Stale serialized messages with field 3 will be silently ignored; wire compatibility caveat

---

## Module: Delegation (DE)

### DE-HIGH-1 — MOD-DE-MSG-3: fee_spend_limit never populated during GrantOperatorAuthorization
- **File:** [x/de/keeper/msg_grant_operator_authorization.go:78](x/de/keeper/msg_grant_operator_authorization.go#L78)
- **Spec says:** Execution: set corporation, operator, msg_types, spend_limit, fee_spend_limit, expiration, period, persist
- **Code does:** Struct built setting Corporation, Operator, MsgTypes, SpendLimit, Expiration, Period. FeeSpendLimit (proto field 5) never set — always zero.
- **Impact:** Fee spend limit dimension silently dropped on every grant; fee spend limit enforcement impossible

### DE-HIGH-2 — MOD-DE-MSG-5: AddPermToVSOA does not validate permission ownership
- **File:** [x/de/keeper/keeper.go:83](x/de/keeper/keeper.go#L83)
- **Spec says:** Preconditions: all permission IDs valid and owned by corporation
- **Code does:** Only checks mutual exclusivity with OperatorAuthorization, then appends permID directly with no keeper call to verify existence or ownership
- **Impact:** Arbitrary permission IDs (non-existent or belonging to different corporations) can be registered in VSOperatorAuthorization

### DE-MEDIUM-1 — MOD-DE-MSG-3: No existence check for corporation group or grantee account
- **File:** [x/de/keeper/msg_grant_operator_authorization.go:15](x/de/keeper/msg_grant_operator_authorization.go#L15)
- **Spec says:** Preconditions: corporation group exists, operator account exists
- **Code does:** Address bech32 validation in ValidateBasic only; no group keeper or account keeper call to verify on-chain existence
- **Impact:** Grants issued to non-existent accounts or non-group addresses create orphaned authorization records

### DE-MEDIUM-2 — MOD-DE-MSG-2: RevokeFeeAllowance silently no-ops when FeeGrant not found
- **File:** [x/de/keeper/fee_grant.go:100](x/de/keeper/fee_grant.go#L100)
- **Spec says:** Precondition: FeeGrant entry must exist
- **Code does:** `if !has { return nil }` — returns success when FeeGrant does not exist
- **Impact:** Callers cannot detect missing FeeGrant during revocation; state inconsistencies masked

### DE-MEDIUM-3 — MOD-DE-MSG-7/8: Grant/Revoke Exchange Rate Authorization not implemented in DE module
- **File:** [proto/verana/de/v1/tx.proto:16](proto/verana/de/v1/tx.proto#L16)
- **Spec says:** MOD-DE-MSG-7: Create ExchangeRate entry; MOD-DE-MSG-8: set state=false
- **Code does:** DE module has no proto message, no keeper state, no handler for these messages. Exchange rate logic lives in XR module.
- **Impact:** If spec assigns these to DE, they are entirely absent. If correctly placed in XR, this is a spec naming discrepancy.

### DE-LOW-1 — MOD-DE-MSG-3/4: amino.name uses abbreviated form
- **File:** [proto/verana/de/v1/tx.proto:65](proto/verana/de/v1/tx.proto#L65)
- **Spec says:** amino.name follows `verana/x/de/MsgXxx` with full message name
- **Code does:** `amino.name = "verana/x/de/MsgGrantOpAuthorization"` (abbreviated) vs full `MsgGrantOperatorAuthorization`
- **Impact:** Amino signing may break wallets/tooling expecting canonical full-name pattern

### DE-LOW-2 — MOD-DE-MSG-3: feegrant_spend_limit positive check runs without with_feegrant guard
- **File:** [x/de/types/types.go:112](x/de/types/types.go#L112)
- **Code does:** Positive-amount check runs regardless of `msg.WithFeegrant` state; redundant with earlier guard but inconsistent
- **Impact:** Minor inconsistency; functionally caught by earlier check at line 103

### DE-LOW-3 — MOD-DE-MSG-1: GrantFeeAllowance has no grantor group or grantee account existence check
- **File:** [x/de/keeper/fee_grant.go:14](x/de/keeper/fee_grant.go#L14)
- **Spec says:** Preconditions: grantor group exists, grantee account exists, valid delegation exists
- **Code does:** Static validation only; no group keeper or account keeper called
- **Impact:** Orphaned FeeGrant records for non-existent accounts

---

## Module: Digests (DI)

### DI-MEDIUM-1 — MOD-DI-MSG-1: Amino converter aminoType uses proto URL, not amino.name
- **File:** [ts-proto/src/amino-converter/di.ts:6](ts-proto/src/amino-converter/di.ts#L6)
- **Spec says:** amino.name in proto is `"verana/x/di/MsgStoreDigest"`
- **Code does:** `aminoType: "/verana.di.v1.MsgStoreDigest"` — leading slash, dot-separated package path format
- **Impact:** Amino/Ledger signing fails because aminoType doesn't match the proto's amino.name declaration

### DI-MEDIUM-2 — MOD-DI-MSG-1: Amino converter drops digest_algorithm field
- **File:** [ts-proto/src/amino-converter/di.ts:7](ts-proto/src/amino-converter/di.ts#L7)
- **Spec says:** digest_algorithm is mandatory (ValidateBasic enforces non-empty)
- **Code does:** `toAmino` serializes only {authority, operator, digest}; digest_algorithm dropped. fromAmino reconstructs same three fields; round-trip leaves DigestAlgorithm as empty string.
- **Impact:** Any amino-encoded StoreDigest transaction (Ledger, Keplr amino mode) fails ValidateBasic on-chain with "digest_algorithm is required"

### DI-LOW-1 — MOD-DI-MSG-1: No SRI format validation on digest field
- **File:** [x/di/types/types.go:31](x/di/types/types.go#L31)
- **Spec says:** MUST abort if digest empty or invalid format (SRI hash: e.g. `sha256-<base64>`)
- **Code does:** Checks `digest != ""` and `len(digest) <= 256` only; no structural SRI format validation
- **Impact:** Malformed non-SRI digest strings stored on-chain; downstream SRI consumers may break

### DI-LOW-2 — MOD-DI-MSG-1: No module-level fee check in StoreDigest
- **File:** [x/di/keeper/msg_store_digest.go:14](x/di/keeper/msg_store_digest.go#L14)
- **Spec says:** MUST abort if fee insufficient
- **Code does:** Relies entirely on ante-handler for fee enforcement; no module-level fee validation
- **Impact:** If spec requires a module-defined service fee beyond gas, it is not enforced

### DI-LOW-3 — MOD-DI-QRY-1: digest_id (uint64) lookup not implemented
- **File:** [proto/verana/di/v1/query.proto:39](proto/verana/di/v1/query.proto#L39)
- **Spec says:** Parameters: digest_id (uint64) OR digest value
- **Code does:** QueryGetDigestRequest only has `digest string` field; no digest_id field; only value-based lookup
- **Impact:** Clients cannot look up digests by integer ID

### DI-LOW-4 — MOD-DI-MSG-1: No auto-incremented integer ID assigned to Digest records
- **File:** [x/di/keeper/msg_store_digest.go:47](x/di/keeper/msg_store_digest.go#L47)
- **Spec says:** Execution: persist with auto-incremented ID
- **Code does:** Records stored keyed by digest string value; no uint64 ID field, no sequence counter, no ID in response
- **Impact:** Spec-mandated auto-incremented ID absent; explains DI-LOW-3 (lookup by ID impossible)

---

## Module: Exchange Rate (XR)

### XR-HIGH-1 — MOD-XR-MSG-1: Initial state defaults to false; spec mandates true
- **File:** [x/xr/keeper/msg_create_exchange_rate.go:68](x/xr/keeper/msg_create_exchange_rate.go#L68)
- **Spec says:** Execution: state=true (enabled)
- **Code does:** Proto carries `bool state = 9`; code sets `State: msg.State`. Happy-path test asserts `require.False(t, xr.State)` confirming default is false.
- **Impact:** Newly created exchange rates are disabled by default, requiring a separate SetExchangeRateState call before use

### XR-HIGH-2 — MOD-XR-MSG-2: No asset-pair matching check
- **File:** [x/xr/keeper/msg_update_exchange_rate.go:29](x/xr/keeper/msg_update_exchange_rate.go#L29)
- **Spec says:** MUST abort if base and quote assets don't match existing entry
- **Code does:** MsgUpdateExchangeRate proto has only {authority, operator, id, rate} — no asset fields; handler loads by ID and updates rate with no asset comparison
- **Impact:** Spec precondition entirely absent

### XR-HIGH-3 — MOD-XR-MSG-2: rate_scale and validity_duration not updatable
- **File:** [proto/verana/xr/v1/tx.proto:72](proto/verana/xr/v1/tx.proto#L72)
- **Spec says:** Execution: update rate_scale if provided, update validity_duration if provided, recalculate expires
- **Code does:** MsgUpdateExchangeRate proto only has {authority, operator, id, rate}; rate_scale and validity_duration absent from message. Handler updates only Rate and recalculates expires using existing xr.ValidityDuration.
- **Impact:** rate_scale and validity_duration permanently frozen after creation

### XR-HIGH-4 — MOD-XR-MSG-1/2/3: Amino converters use proto URL instead of amino.name
- **File:** [ts-proto/src/amino-converter/xr.ts:17](ts-proto/src/amino-converter/xr.ts#L17)
- **Spec says:** amino.name declared in proto as `"verana/x/xr/MsgCreateExchangeRate"` (and same pattern for others)
- **Code does:** Converters use `aminoType: "/verana.xr.v1.MsgCreateExchangeRate"` — leading slash, package path format, different string
- **Impact:** Ledger hardware wallet signing and amino-based signing fail because aminoType doesn't match proto's amino.name

### XR-MEDIUM-1 — MOD-XR-MSG-1: base_asset unconditionally required even for TU type
- **File:** [x/xr/types/types.go:34](x/xr/types/types.go#L34)
- **Spec says:** base_asset: required if COIN or FIAT, null if TU
- **Code does:** `if msg.BaseAsset == "" { return error }` unconditionally; for TU type, enforces literal string "TU" rather than allowing null/empty
- **Impact:** Clients following the spec (omitting base_asset for TU) are rejected; literal "TU" required

### XR-MEDIUM-2 — MOD-XR-MSG-3: Explicit state set instead of toggle
- **File:** [x/xr/keeper/msg_set_exchange_rate_state.go:41](x/xr/keeper/msg_set_exchange_rate_state.go#L41)
- **Spec says:** toggle state (true↔false)
- **Code does:** `xr.State = msg.State` — direct set from caller-supplied boolean
- **Impact:** Concurrent toggle calls may set same value; clients must track current state themselves

### XR-MEDIUM-3 — MOD-XR-QRY-2: ListExchangeRates has no pagination
- **File:** [proto/verana/xr/v1/query.proto:74](proto/verana/xr/v1/query.proto#L74)
- **Spec says:** paginated list
- **Code does:** No PageRequest in request, no PageResponse in response; returns all records unbounded
- **Impact:** Unbounded result sets as exchange rate count grows; no cursor for clients

### XR-LOW-1 — MOD-XR-MSG-1/2/3: No module-level fee check
- **File:** [x/xr/keeper/msg_create_exchange_rate.go](x/xr/keeper/msg_create_exchange_rate.go), [msg_update_exchange_rate.go](x/xr/keeper/msg_update_exchange_rate.go), [msg_set_exchange_rate_state.go](x/xr/keeper/msg_set_exchange_rate_state.go)
- **Spec says:** MUST abort if fee insufficient (for all three messages)
- **Code does:** No module-level fee validation; relies on ante-handler only
- **Impact:** Module-defined service fees (if any) not enforced

---

## Cross-Module Patterns

These patterns appear across multiple modules and should be fixed systematically:

### Pattern 1: archive=false (unarchive) hard-rejected
Affects: TR (MSG-5), CS (MSG-3)  
Both modules return an unconditional error for `archive=false` instead of implementing the spec's bidirectional toggle. Fix both together.

### Pattern 2: amino.name mismatch in TypeScript converters
Affects: DI (MSG-1), XR (MSG-1,2,3), DE (MSG-3,4)  
TS converters use the protobuf package URL format (`/verana.{module}.v1.MsgXxx`) instead of the amino.name declared in proto (`verana/x/{module}/MsgXxx`). This breaks Ledger signing across these modules. The Go codec uses the correct amino.name; the TS side is out of sync.

### Pattern 3: MsgUpdateParams missing from amino codec
Affects: TR, CS (implicit), PERM  
Governance messages need amino registration for Ledger/governance signing flows. All three modules omit this.

### Pattern 4: reason field missing from slash messages
Affects: PERM (MSG-12), TD (MSG-5)  
Both slash messages are missing a mandatory reason field specified in the spec. Fix both proto files together.

### Pattern 5: No per-key validation in UpdateParams
Affects: CS (MSG-4), and likely TR, PERM, TD, XR (not fully verified)  
The pattern of accepting Params struct wholesale without verifying each supplied key matches a known parameter key appears to be consistent across modules.

---

## Verified Clean (No Findings)

The following areas were audited and confirmed compliant:

- **XR price calculation (QRY-3):** Multiply before divide to avoid precision loss — correct
- **XR governance authority checks (MSG-1, MSG-3):** `bytes.Equal(ms.GetAuthority(), authority)` — correct
- **XR ExchangeRate existence checks (MSG-2, MSG-3):** proper not-found errors returned
- **XR expires calculation:** `blockTime.Add(msg.ValidityDuration)` using ctx.BlockTime() — correct
- **XR state=false handling in GetPrice:** Checks `!xr.State` returns FailedPrecondition — correct
- **TR proto signer annotations:** `cosmos.msg.v1.signer = "operator"` for corporation+operator messages — correct
- **TR ValidateBasic address validation:** AccAddressFromBech32 used for operator and corporation
- **TD CheckOperatorAuthorization:** Nil delegation keeper is hard error, not bypass — correct
- **DE amino converters:** GrantOperatorAuthorization and RevokeOperatorAuthorization converters correctly map all non-signer fields
- **PERM AUTHZ-CHECK pattern:** Standard corporation+operator messages call CheckOperatorAuthorization correctly
- **DI duplicate digest check:** StoreDigest correctly prevents storing the same digest twice
- **DI query not-found handling:** GetDigest returns proper not-found error, no panic
