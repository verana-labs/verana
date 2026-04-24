# PERM Module Spec v4 Revert + Test Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Revert three spec-violating changes PR #280 introduced to the PERM module (Mohammad's devnet bug), and harden the test suite so the same class of bug cannot reach main again.

**Architecture:** Two stages. Stage 1 is a targeted revert: remove `permission_type` and `vs_operator` fields from `MsgCreateRootPermission`, remove `permission_type` from `MsgRenewPermissionVP`, restore hardcoded ECOSYSTEM, fix overlap check, fix tests that assert the bug, and redeploy devnet. Stage 2 adds a ValidateBasic test suite, a CLI-boundary integration test, a "Mohammad regression" end-to-end test, and a static analyzer that fails CI when a proto field marked "mandatory per spec" lacks a corresponding ValidateBasic check.

**Tech Stack:** Go 1.22, Cosmos SDK v0.53.x, `make proto-gen` for proto regeneration, `go test ./x/perm/...` for unit tests, existing `veranad` CLI for integration tests.

**Spec source of truth:** [VPR spec v4-draft13 spec.md](https://github.com/verana-labs/verifiable-trust-vpr-spec/blob/main/spec.md). Every test added in this plan quotes the spec verbatim in a doc comment with the section anchor (e.g. `[MOD-PERM-MSG-7-1]`).

**Out of scope:**
- Changes to modules other than PERM (the cross-module sweep found this bug class concentrated in PERM).
- New proto field additions. This plan only removes fields and restores hardcoded values.
- Renaming or refactoring beyond what is required to make the tests pass.

---

## Stage 1 — Revert to spec-compliant behavior

### Phase 1.1 — `MsgCreateRootPermission` revert

Spec [MOD-PERM-MSG-7-1] lists exactly these parameters: `corporation, operator, schema_id, did, effective_from, effective_until, validation_fees, issuance_fees, verification_fees`. Spec [MOD-PERM-MSG-7-3] hardcodes `perm.type: ECOSYSTEM`. Spec [MOD-PERM-MSG-7-2-4] filters overlap on `(schema_id, ECOSYSTEM, corporation)`.

#### Task 1.1.1 — Flip `TestCreateRootPermission` to assert ECOSYSTEM literal

**Files:**
- Modify: [x/perm/keeper/msg_server_test.go:4665-4830](x/perm/keeper/msg_server_test.go#L4665-L4830)

- [ ] **Step 1: Replace every `PermissionType: types.PermissionType_ISSUER, VsOperator: operator,` line inside the testCases table with nothing**

In the table-driven cases at lines 4701-4795, remove the two lines from each struct literal that set `PermissionType` and `VsOperator`. The Msg struct will build with the proto3 zero value for both; this is intentional because the fields are going away in Task 1.1.4.

- [ ] **Step 2: Replace the post-success assertion block**

Current block at lines 4809-4828 contains:
```go
require.Equal(t, tc.msg.PermissionType, perm.Type)
```

Replace with:
```go
// [MOD-PERM-MSG-7-3] Spec v4 draft 13: perm.type is hardcoded to ECOSYSTEM.
require.Equal(t, types.PermissionType_ECOSYSTEM, perm.Type,
    "Create Root Permission MUST hardcode perm.type to ECOSYSTEM per spec [MOD-PERM-MSG-7-3]")
require.Equal(t, "", perm.VsOperator,
    "Create Root Permission does not set perm.vs_operator per spec [MOD-PERM-MSG-7-3]")
```

- [ ] **Step 3: Run the test to verify it fails**

Run: `go test ./x/perm/keeper/ -run TestCreateRootPermission -v -count=1`

Expected: FAIL on cases 7 and 8 (the happy paths) with `Not equal: expected ECOSYSTEM(5), actual ISSUER(1)` because the handler still stores `msg.PermissionType`. This proves the test now enforces the spec invariant.

- [ ] **Step 4: Commit the failing test**

```bash
git add x/perm/keeper/msg_server_test.go
git commit --no-gpg-sign -s -m "test(perm): assert CreateRootPermission hardcodes ECOSYSTEM per spec"
```

#### Task 1.1.2 — Hardcode ECOSYSTEM in handler and drop `VsOperator` assignment

**Files:**
- Modify: [x/perm/keeper/msg_server.go:697-726](x/perm/keeper/msg_server.go#L697-L726)

- [ ] **Step 1: Rewrite `executeCreateRootPermission`**

Replace the entire body of `executeCreateRootPermission` (lines 697-726) with:

```go
// [MOD-PERM-MSG-7-3] Create Root Permission execution
// Spec v4 draft 13: perm.type is hardcoded to ECOSYSTEM. vs_operator is not
// set on root permissions; only on perms created via StartPermissionVP or
// SelfCreatePermission.
func (ms msgServer) executeCreateRootPermission(ctx sdk.Context, msg *types.MsgCreateRootPermission, now time.Time) (uint64, error) {
	perm := types.Permission{
		SchemaId:         msg.SchemaId,
		Modified:         &now,
		Type:             types.PermissionType_ECOSYSTEM,
		Did:              msg.Did,
		Corporation:      msg.Corporation,
		Created:          &now,
		EffectiveFrom:    msg.EffectiveFrom,
		EffectiveUntil:   msg.EffectiveUntil,
		ValidationFees:   msg.ValidationFees,
		IssuanceFees:     msg.IssuanceFees,
		VerificationFees: msg.VerificationFees,
		Deposit:          0,
	}

	id, err := ms.Keeper.CreatePermission(ctx, perm)
	if err != nil {
		return 0, fmt.Errorf("failed to create perm: %w", err)
	}

	return id, nil
}
```

Key differences from current code: `Type` is the literal enum `ECOSYSTEM`, and `VsOperator` is no longer assigned (defaults to `""`).

- [ ] **Step 2: Run the test to verify it passes**

Run: `go test ./x/perm/keeper/ -run TestCreateRootPermission -v -count=1`

Expected: PASS on all 8 cases.

- [ ] **Step 3: Commit**

```bash
git add x/perm/keeper/msg_server.go
git commit --no-gpg-sign -s -m "fix(perm): hardcode ECOSYSTEM type on CreateRootPermission per spec v4 draft 13"
```

#### Task 1.1.3 — Fix overlap check to filter on literal ECOSYSTEM

**Files:**
- Modify: [x/perm/keeper/msg_server.go:659-695](x/perm/keeper/msg_server.go#L659-L695)
- Test: Existing `TestCreateRootPermission_OverlapChecks` at [msg_server_test.go:4834](x/perm/keeper/msg_server_test.go#L4834)

- [ ] **Step 1: Replace the filter predicate in `checkCreateRootPermissionOverlap`**

Current code at lines 664-670:
```go
err := ms.Permission.Walk(ctx, nil, func(key uint64, perm types.Permission) (bool, error) {
    // Match on schema_id, permission_type, and corporation.
    if perm.SchemaId != msg.SchemaId ||
        perm.Type != msg.PermissionType ||
        perm.Corporation != msg.Corporation {
        return false, nil
    }
```

Replace with:
```go
err := ms.Permission.Walk(ctx, nil, func(key uint64, perm types.Permission) (bool, error) {
    // [MOD-PERM-MSG-7-2-4] Spec v4 draft 13: find all active permissions for
    // (schema_id, ECOSYSTEM, corporation). Type is always ECOSYSTEM because
    // this is a root-permission overlap check.
    if perm.SchemaId != msg.SchemaId ||
        perm.Type != types.PermissionType_ECOSYSTEM ||
        perm.Corporation != msg.Corporation {
        return false, nil
    }
```

- [ ] **Step 2: Update the comment at lines 659-662**

Replace:
```go
// [MOD-PERM-MSG-7-2-4] Create Root Permission overlap checks.
// Find all active permissions (not revoked, not slashed, not repaid) for
// (schema_id, permission_type, corporation). Spec v4 draft 13: permission_type
// is set from msg, not hardcoded.
```

With:
```go
// [MOD-PERM-MSG-7-2-4] Create Root Permission overlap checks.
// Spec v4 draft 13: find all active permissions (not revoked, not slashed,
// not repaid) for (schema_id, ECOSYSTEM, corporation). Unlike other overlap
// checks, validator_perm_id is not checked because ECOSYSTEM permissions
// always have validator_perm_id = NULL.
```

- [ ] **Step 3: Run the overlap tests**

Run: `go test ./x/perm/keeper/ -run TestCreateRootPermission_OverlapChecks -v -count=1`

Expected: PASS. The overlap tests already construct perms with `Type: ECOSYSTEM` in their setup, so they exercise the same filter predicate and should continue to pass.

- [ ] **Step 4: Commit**

```bash
git add x/perm/keeper/msg_server.go
git commit --no-gpg-sign -s -m "fix(perm): overlap check uses literal ECOSYSTEM per spec [MOD-PERM-MSG-7-2-4]"
```

#### Task 1.1.4 — Remove `permission_type` and `vs_operator` from proto

**Files:**
- Modify: [proto/verana/perm/v1/tx.proto:153-177](proto/verana/perm/v1/tx.proto#L153-L177)

- [ ] **Step 1: Replace the `MsgCreateRootPermission` message definition**

Current (lines 153-177):
```proto
message MsgCreateRootPermission {
  option (cosmos.msg.v1.signer) = "operator";
  option (amino.name) = "verana/x/perm/MsgCreateRootPermission";

  string corporation = 1 [(cosmos_proto.scalar) = "cosmos.AddressString"];
  string operator = 2 [(cosmos_proto.scalar) = "cosmos.AddressString"];
  uint64 schema_id = 3;
  string did = 4;
  google.protobuf.Timestamp effective_from = 5 [
    (gogoproto.stdtime) = true,
    (gogoproto.nullable) = true
  ];
  google.protobuf.Timestamp effective_until = 6 [
    (gogoproto.stdtime) = true,
    (gogoproto.nullable) = true
  ];
  uint64 validation_fees = 7;
  uint64 issuance_fees = 8;
  uint64 verification_fees = 9;
  // [MOD-PERM-MSG-7-1] permission_type mandatory per spec v4 draft 13:
  // one of ISSUER, VERIFIER, ISSUER_GRANTOR, VERIFIER_GRANTOR.
  PermissionType permission_type = 10;
  // [MOD-PERM-MSG-7-1] vs_operator mandatory per spec v4 draft 13.
  string vs_operator = 11 [(cosmos_proto.scalar) = "cosmos.AddressString"];
}
```

Replace with:
```proto
message MsgCreateRootPermission {
  option (cosmos.msg.v1.signer) = "operator";
  option (amino.name) = "verana/x/perm/MsgCreateRootPermission";

  // [MOD-PERM-MSG-7-1] Spec v4 draft 13 parameters.
  // perm.type is hardcoded to ECOSYSTEM by the handler per [MOD-PERM-MSG-7-3];
  // vs_operator is not set on root permissions.
  string corporation = 1 [(cosmos_proto.scalar) = "cosmos.AddressString"];
  string operator = 2 [(cosmos_proto.scalar) = "cosmos.AddressString"];
  uint64 schema_id = 3;
  string did = 4;
  google.protobuf.Timestamp effective_from = 5 [
    (gogoproto.stdtime) = true,
    (gogoproto.nullable) = true
  ];
  google.protobuf.Timestamp effective_until = 6 [
    (gogoproto.stdtime) = true,
    (gogoproto.nullable) = true
  ];
  uint64 validation_fees = 7;
  uint64 issuance_fees = 8;
  uint64 verification_fees = 9;
}
```

Field numbers 10 and 11 are retired. Do not reassign them.

- [ ] **Step 2: Regenerate protos**

Run: `make proto-gen`

Expected: updates `x/perm/types/tx.pb.go` and `api/verana/perm/v1/tx.pulsar.go`. No errors.

- [ ] **Step 3: Build the project to catch any lingering references**

Run: `go build ./...`

Expected: builds cleanly. If any code still references `msg.PermissionType` or `msg.VsOperator` on `MsgCreateRootPermission`, fix those references to use the hardcoded ECOSYSTEM literal or remove them.

- [ ] **Step 4: Run the full perm package tests**

Run: `go test ./x/perm/... -count=1`

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add proto/verana/perm/v1/tx.proto x/perm/types/tx.pb.go api/verana/perm/v1/tx.pulsar.go
git commit --no-gpg-sign -s -m "fix(perm): remove permission_type and vs_operator from MsgCreateRootPermission proto (spec [MOD-PERM-MSG-7-1])"
```

#### Task 1.1.5 — Delete the bug-repro file

**Files:**
- Delete: [x/perm/keeper/create_root_perm_spec_bug_repro_test.go](x/perm/keeper/create_root_perm_spec_bug_repro_test.go)

- [ ] **Step 1: Delete the file**

```bash
git rm x/perm/keeper/create_root_perm_spec_bug_repro_test.go
```

This file was written to demonstrate the bug by asserting that the server accepts spec-invalid input. Now that the fields are gone entirely, the file will not compile (it references `types.MsgCreateRootPermission.PermissionType` and `.VsOperator` which no longer exist).

- [ ] **Step 2: Build to confirm**

Run: `go build ./...`

Expected: builds cleanly.

- [ ] **Step 3: Commit**

```bash
git commit --no-gpg-sign -s -m "test(perm): remove obsolete spec-bug-repro test file"
```

#### Task 1.1.6 — Fix event emission to not reference removed fields

**Files:**
- Check: [x/perm/keeper/msg_server.go:590-603](x/perm/keeper/msg_server.go#L590-L603)

- [ ] **Step 1: Inspect the event emission block**

Read lines 590-603. If any `sdk.NewAttribute` call references `msg.PermissionType` or `msg.VsOperator` directly, remove those attribute lines. The current event attributes per grep are: `RootPermissionID, SchemaID, Corporation, Operator, EffectiveFrom, EffectiveUntil, ValidationFees, IssuanceFees, VerificationFees, Timestamp`. If those are all present and none reference the removed fields, no change is needed.

- [ ] **Step 2: Run tests**

Run: `go test ./x/perm/... -count=1`

Expected: PASS. If an event-emission test exists and asserts on a removed attribute, it will now fail; in that case remove the assertion.

- [ ] **Step 3: Commit only if changes were made**

```bash
git add x/perm/keeper/msg_server.go
git commit --no-gpg-sign -s -m "fix(perm): drop removed fields from CreateRootPermission event"
```

---

### Phase 1.2 — `MsgRenewPermissionVP` revert

Spec [MOD-PERM-MSG-2-1] lists exactly three parameters: `corporation, operator, id`. No `permission_type`.

#### Task 1.2.1 — Remove the handler guard that references `permission_type`

**Files:**
- Modify: [x/perm/keeper/msg_server.go:117-120](x/perm/keeper/msg_server.go#L117-L120)

- [ ] **Step 1: Delete the guard block**

Current (lines 117-120):
```go
// [MOD-PERM-MSG-2-2] spec v4 draft 13: permission_type must match existing.
if msg.PermissionType != types.PermissionType_UNSPECIFIED && applicantPerm.Type != msg.PermissionType {
    return nil, fmt.Errorf("permission_type mismatch: existing %s, requested %s", applicantPerm.Type, msg.PermissionType)
}
```

Delete these 4 lines entirely.

- [ ] **Step 2: Run the renew tests**

Run: `go test ./x/perm/keeper/ -run TestRenewPermissionVP -v -count=1`

Expected: tests that passed before this change continue to pass. If a test explicitly asserts a `permission_type mismatch` error, it needs to be removed in Task 1.2.3.

- [ ] **Step 3: Commit**

```bash
git add x/perm/keeper/msg_server.go
git commit --no-gpg-sign -s -m "fix(perm): remove non-spec permission_type guard from RenewPermissionVP"
```

#### Task 1.2.2 — Remove `permission_type` from `MsgRenewPermissionVP` proto

**Files:**
- Modify: [proto/verana/perm/v1/tx.proto:98-108](proto/verana/perm/v1/tx.proto#L98-L108) (approximate; verify before editing)

- [ ] **Step 1: Locate the message and remove the field**

Read `proto/verana/perm/v1/tx.proto` around line 98 to find the exact `MsgRenewPermissionVP` definition. Remove the `PermissionType permission_type = 4;` line and its preceding spec-comment. Leave fields 1-3 (corporation, operator, id) intact.

- [ ] **Step 2: Regenerate protos**

Run: `make proto-gen`

Expected: updates pb.go and pulsar.go. No errors.

- [ ] **Step 3: Build and run tests**

Run: `go build ./... && go test ./x/perm/... -count=1`

Expected: builds cleanly, all tests PASS.

- [ ] **Step 4: Commit**

```bash
git add proto/verana/perm/v1/tx.proto x/perm/types/tx.pb.go api/verana/perm/v1/tx.pulsar.go
git commit --no-gpg-sign -s -m "fix(perm): remove permission_type from MsgRenewPermissionVP proto (spec [MOD-PERM-MSG-2-1])"
```

#### Task 1.2.3 — Clean up renew tests that assert the removed behavior

**Files:**
- Modify: [x/perm/keeper/msg_server_test.go:799-873](x/perm/keeper/msg_server_test.go#L799-L873) (`TestRenewPermissionVP_ValidateBasic`)
- Check: any other `TestRenewPermissionVP*` that mentions `PermissionType`

- [ ] **Step 1: Grep for any renew test case that sets PermissionType or asserts on the mismatch error**

Run: `grep -n "PermissionType\|permission_type mismatch" x/perm/keeper/msg_server_test.go | grep -i renew`

For each match: if the case sets `PermissionType: ...` in a `MsgRenewPermissionVP{}` literal, remove that line. If the case asserts the "permission_type mismatch" error, remove the case.

- [ ] **Step 2: Run the tests**

Run: `go test ./x/perm/keeper/ -run TestRenewPermissionVP -v -count=1`

Expected: all PASS.

- [ ] **Step 3: Commit**

```bash
git add x/perm/keeper/msg_server_test.go
git commit --no-gpg-sign -s -m "test(perm): drop non-spec PermissionType assertions from RenewPermissionVP tests"
```

---

### Phase 1.3 — Cleanup and audit correction

#### Task 1.3.1 — Correct `final_audit.md`

**Files:**
- Modify: [final_audit.md](final_audit.md)

- [ ] **Step 1: Replace PERM-HIGH-1, PERM-HIGH-2, PERM-HIGH-3 with correction notes**

Find the three blocks (lines near 152-175 based on earlier grep). Replace each block in-place with a `RETRACTED` entry that cites the correct spec section and links to the fix commits. Example for PERM-HIGH-2:

```markdown
### PERM-HIGH-2 — MOD-PERM-MSG-7: permission_type hardcoded to ECOSYSTEM — RETRACTED 2026-04-24

**Retraction reason:** Spec v4 draft 13 [MOD-PERM-MSG-7-3] explicitly states `perm.type: ECOSYSTEM` (hardcoded). Earlier finding misread the spec. PR #280 implemented the incorrect finding; reverted in this plan.

**Original (incorrect) text:**
- File: x/perm/keeper/msg_server.go:706
- Spec says (claimed): Mandatory: permission_type (ISSUER|VERIFIER|ISSUER_GRANTOR|VERIFIER_GRANTOR)
- Actual spec [MOD-PERM-MSG-7-3]: perm.type: ECOSYSTEM (hardcoded)
```

Apply the same pattern to PERM-HIGH-1 and PERM-HIGH-3.

- [ ] **Step 2: Commit**

```bash
git add final_audit.md
git commit --no-gpg-sign -s -m "docs(audit): retract PERM-HIGH-1/2/3 findings that misread spec v4 draft 13"
```

#### Task 1.3.2 — Re-audit the rest of PR #280

**Files:**
- Read-only audit. No file changes in this task.

- [ ] **Step 1: Dispatch an audit agent over PR #280's other PERM changes**

Use the audit agent at [.claude-agents/audit.md](.claude-agents/audit.md). The prompt should be:

"Re-audit PR #280 on verana-labs/verana against VPR spec v4-draft13 (`/tmp/vpr-spec.md`, also at https://github.com/verana-labs/verifiable-trust-vpr-spec/blob/main/spec.md). For each PERM module change in PR #280, quote the spec verbatim with section anchor, compare to the implementation, and flag any other misreads. Known already-found misreads are MSG-7 permission_type, MSG-7 vs_operator, MSG-2 permission_type. Scope: all PERM Msg handlers touched by PR #280. Report each finding as VERIFIED | FALSE POSITIVE | NEEDS FURTHER REVIEW with spec citations."

- [ ] **Step 2: Review the audit report**

If any new real bugs are found, open a follow-up task list. If none are found, record that in a commit:

```bash
echo "PR #280 re-audit complete $(date -u +%Y-%m-%dT%H:%M:%SZ): no additional spec violations found beyond the three already reverted in this plan." >> docs/superpowers/audit-log.md
git add docs/superpowers/audit-log.md
git commit --no-gpg-sign -s -m "docs(audit): record PR #280 re-audit completion"
```

#### Task 1.3.3 — Full test run + build gate

**Files:**
- None (verification only)

- [ ] **Step 1: Run full test suite**

Run: `go test ./... -count=1`

Expected: all PASS. Any failures are either (a) test files still asserting the reverted behavior, in which case fix them, or (b) unrelated breakage, in which case stop and investigate before proceeding.

- [ ] **Step 2: Build binaries**

Run: `make build`

Expected: `veranad` binary produced, no errors.

- [ ] **Step 3: Tag an intermediate commit**

No changes expected here, just a checkpoint tag:

```bash
git tag -a stage1-perm-revert-complete -m "Stage 1 of perm spec v4 revert: build and tests green"
```

---

### Phase 1.4 — Devnet redeploy and Mohammad-flow smoke test

#### Task 1.4.1 — Cut release, mirror to S3, redeploy devnet

**Files:**
- Release workflow only; no code changes.

- [ ] **Step 1: Open the PR and merge to main**

```bash
git push -u origin <branch-name>
gh pr create --title "fix(perm): revert spec-violating changes to CreateRootPermission and RenewPermissionVP" --body "$(cat <<'EOF'
## Summary
Reverts three changes introduced by PR #280 that violated VPR spec v4 draft 13:
1. Remove `permission_type` field from `MsgCreateRootPermission` (spec [MOD-PERM-MSG-7-1] does not include it)
2. Remove `vs_operator` field from `MsgCreateRootPermission` (spec [MOD-PERM-MSG-7-1] does not include it)
3. Remove `permission_type` field from `MsgRenewPermissionVP` (spec [MOD-PERM-MSG-2-1] does not include it)
Restores `perm.type: ECOSYSTEM` hardcode per [MOD-PERM-MSG-7-3] and fixes the overlap-check filter to use literal ECOSYSTEM per [MOD-PERM-MSG-7-2-4].

Triggered by devnet integrator report: `create-root-perm` on devnet was storing `type: UNSPECIFIED`, blocking downstream `self-create-perm` and VP flows.

## Test plan
- [x] `TestCreateRootPermission` asserts `perm.Type == ECOSYSTEM` literal
- [x] `go test ./x/perm/... -count=1` passes
- [x] `go build ./...` passes
- [ ] Devnet redeploy + Mohammad end-to-end flow passes (post-merge)
EOF
)"
```

Merge once CI is green.

- [ ] **Step 2: Create a new prerelease tag on verana-labs/verana**

```bash
gh release create v0.10.1-dev.11 --repo verana-labs/verana --prerelease --generate-notes --target main
```

- [ ] **Step 3: Run the devnet full reset from the verana-deploy playbook**

Follow [docs/01-deployment.md "Devnet Full Reset from GitHub Release"](https://github.com/verana-labs/verana-deploy/blob/fix/deploy-devnet-issue/docs/01-deployment.md#devnet-full-reset-from-github-release-vms-key-preserving) with `VERANA_TAG=v0.10.1-dev.11`. This is the same seven-step sequence executed on 2026-04-23; the procedure is validated.

Expected outcomes:
- Step 1 (health check): green
- Step 2 (copy binaries): `s3://utc-public-bucket/vna-devnet-1/binaries/` shows `v0.10.1-dev.11`
- Step 3 (genesis rebuild): fresh genesis uploaded
- Step 4 (full VM redeploy, no promote): node1 producing blocks
- Gate check: height > 0 and increasing, network = `vna-devnet-1`
- Step 5 (promote): validator count becomes 3
- Step 6 (persistent peers): refreshed
- Step 7 (health check): green

#### Task 1.4.2 — Execute the Mohammad regression flow against devnet

**Files:**
- None (operational verification)

- [ ] **Step 1: Run the Mohammad end-to-end flow manually**

Using the dev1 wallet and corporation `verana1afk9zr2hn2jsac63h4hm60vl9z3e5u69gndzf7c99cqge3vzwjzsh3z8fv`:

```bash
# 1. Create TR (may already exist from prior runs; skip if so)
# 2. Create credential schema with issuer_onboarding_mode=ECOSYSTEM_VALIDATION_PROCESS
# 3. Create root permission (no --permission-type flag needed after fix)
EFFECTIVE_FROM=$(date -u -v+60S +%Y-%m-%dT%H:%M:%SZ)
veranad tx perm create-root-perm <schema_id> <did> 0 0 0 \
  --corporation "verana1afk9zr2hn2jsac63h4hm60vl9z3e5u69gndzf7c99cqge3vzwjzsh3z8fv" \
  --effective-from "$EFFECTIVE_FROM" \
  --effective-until "2035-12-31T23:59:59Z" \
  --from dev1 \
  --node https://rpc.devnet.verana.network/ \
  --chain-id vna-devnet-1 \
  --keyring-backend test \
  --fees 750000uvna \
  --gas auto \
  --gas-adjustment 1.5 \
  -y --broadcast-mode sync
```

- [ ] **Step 2: Query the root perm and assert type == ECOSYSTEM**

```bash
sleep 10  # wait for inclusion
veranad query perm list-perm --node https://rpc.devnet.verana.network/ -o json | jq '.permissions[] | select(.type == "ECOSYSTEM") | {id, type, schema_id, did}'
```

Expected: at least one entry with `"type": "ECOSYSTEM"`. Record its `id`.

- [ ] **Step 3: Run self-create-perm referencing the ECOSYSTEM root**

```bash
sleep 90  # wait past effective_from
veranad tx perm self-create-perm issuer <ecosystem_root_perm_id> <applicant_did> \
  --corporation <applicant_corp> \
  --effective-from <future> \
  --effective-until <far-future> \
  --from <applicant_operator_key> \
  --node https://rpc.devnet.verana.network/ \
  --chain-id vna-devnet-1 \
  --keyring-backend test \
  --fees 750000uvna \
  --gas auto --gas-adjustment 1.5 \
  -y --broadcast-mode sync
```

Expected: tx lands successfully. Before the fix this returned `validator_perm_id must reference an ECOSYSTEM permission`.

- [ ] **Step 4: Reply to Mohammad with the PR link, devnet redeploy run ID, and the working command sequence**

Stage 1 is complete when Mohammad confirms the flow works end-to-end.

---

## Stage 2 — Test hardening

Stage 2 ships after Stage 1 is deployed and Mohammad's flow is verified. Each phase is independently mergeable.

### Phase 2.1 — ValidateBasic test suite for PERM

Currently only `TestRenewPermissionVP_ValidateBasic` exists in the repo. PERM has 11 messages with `ValidateBasic` implementations. This phase adds dedicated boundary tests for the four highest-risk messages.

#### Task 2.1.1 — `TestMsgCreateRootPermission_ValidateBasic`

**Files:**
- Create: `x/perm/types/msg_create_root_permission_test.go`

- [ ] **Step 1: Write the full test file**

```go
package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana/x/perm/types"
)

// Tests MsgCreateRootPermission.ValidateBasic per spec [MOD-PERM-MSG-7-1] and
// [MOD-PERM-MSG-7-2-1] basic checks. Every mandatory field gets a rejection test.
func TestMsgCreateRootPermission_ValidateBasic(t *testing.T) {
	validAddr := sdk.AccAddress([]byte("test_address________")).String()
	validDid := "did:example:123456789abcdefghi"

	valid := func() *types.MsgCreateRootPermission {
		return &types.MsgCreateRootPermission{
			Corporation:      validAddr,
			Operator:         validAddr,
			SchemaId:         1,
			Did:              validDid,
			ValidationFees:   0,
			IssuanceFees:     0,
			VerificationFees: 0,
		}
	}

	tests := []struct {
		name    string
		mutate  func(m *types.MsgCreateRootPermission)
		wantErr string
	}{
		{"valid baseline", func(m *types.MsgCreateRootPermission) {}, ""},
		{"empty corporation", func(m *types.MsgCreateRootPermission) { m.Corporation = "" }, "invalid corporation address"},
		{"invalid corporation bech32", func(m *types.MsgCreateRootPermission) { m.Corporation = "not-bech32" }, "invalid corporation address"},
		{"empty operator", func(m *types.MsgCreateRootPermission) { m.Operator = "" }, "invalid operator address"},
		{"invalid operator bech32", func(m *types.MsgCreateRootPermission) { m.Operator = "cosmos1garbage" }, "invalid operator address"},
		{"schema_id = 0", func(m *types.MsgCreateRootPermission) { m.SchemaId = 0 }, "schema ID cannot be 0"},
		{"empty did", func(m *types.MsgCreateRootPermission) { m.Did = "" }, "DID is required"},
		{"malformed did", func(m *types.MsgCreateRootPermission) { m.Did = "not-a-did" }, "invalid DID format"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := valid()
			tc.mutate(m)
			err := m.ValidateBasic()
			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
			}
		})
	}
}
```

- [ ] **Step 2: Run it**

Run: `go test ./x/perm/types/ -run TestMsgCreateRootPermission_ValidateBasic -v -count=1`

Expected: all 8 cases PASS. If "valid baseline" fails, a mandatory field is not being set in the `valid()` helper; fix it. If a rejection case PASSES but shouldn't, a check is missing in `ValidateBasic`; add it, then re-run.

- [ ] **Step 3: Commit**

```bash
git add x/perm/types/msg_create_root_permission_test.go
git commit --no-gpg-sign -s -m "test(perm): add ValidateBasic boundary tests for MsgCreateRootPermission"
```

#### Task 2.1.2 — `TestMsgRenewPermissionVP_ValidateBasic`

**Files:**
- Create: `x/perm/types/msg_renew_permission_vp_test.go`

- [ ] **Step 1: Write the test file**

```go
package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana/x/perm/types"
)

// Tests MsgRenewPermissionVP.ValidateBasic per spec [MOD-PERM-MSG-2-1] and
// [MOD-PERM-MSG-2-2-1] basic checks. Spec parameters are only:
// corporation, operator, id.
func TestMsgRenewPermissionVP_ValidateBasic(t *testing.T) {
	validAddr := sdk.AccAddress([]byte("test_address________")).String()

	valid := func() *types.MsgRenewPermissionVP {
		return &types.MsgRenewPermissionVP{
			Corporation: validAddr,
			Operator:    validAddr,
			Id:          1,
		}
	}

	tests := []struct {
		name    string
		mutate  func(m *types.MsgRenewPermissionVP)
		wantErr string
	}{
		{"valid baseline", func(m *types.MsgRenewPermissionVP) {}, ""},
		{"empty corporation", func(m *types.MsgRenewPermissionVP) { m.Corporation = "" }, "invalid corporation address"},
		{"invalid corporation bech32", func(m *types.MsgRenewPermissionVP) { m.Corporation = "not-bech32" }, "invalid corporation address"},
		{"empty operator", func(m *types.MsgRenewPermissionVP) { m.Operator = "" }, "invalid operator address"},
		{"id = 0", func(m *types.MsgRenewPermissionVP) { m.Id = 0 }, "perm ID cannot be 0"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := valid()
			tc.mutate(m)
			err := m.ValidateBasic()
			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
			}
		})
	}
}
```

- [ ] **Step 2: Run**

Run: `go test ./x/perm/types/ -run TestMsgRenewPermissionVP_ValidateBasic -v -count=1`

Expected: all 5 cases PASS.

- [ ] **Step 3: Commit**

```bash
git add x/perm/types/msg_renew_permission_vp_test.go
git commit --no-gpg-sign -s -m "test(perm): add ValidateBasic boundary tests for MsgRenewPermissionVP"
```

#### Task 2.1.3 — `TestMsgStartPermissionVP_ValidateBasic`

**Files:**
- Create: `x/perm/types/msg_start_permission_vp_test.go`

- [ ] **Step 1: Write the test file**

```go
package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana/x/perm/types"
)

// Tests MsgStartPermissionVP.ValidateBasic per spec [MOD-PERM-MSG-1-1] and
// [MOD-PERM-MSG-1-2-1]. Allowed `type` values per spec: ISSUER_GRANTOR,
// VERIFIER_GRANTOR, ISSUER, VERIFIER, HOLDER. NOT ECOSYSTEM, NOT UNSPECIFIED.
func TestMsgStartPermissionVP_ValidateBasic(t *testing.T) {
	validAddr := sdk.AccAddress([]byte("test_address________")).String()
	validDid := "did:example:123456789abcdefghi"

	valid := func() *types.MsgStartPermissionVP {
		return &types.MsgStartPermissionVP{
			Corporation:     validAddr,
			Operator:        validAddr,
			Type:            types.PermissionType_ISSUER,
			ValidatorPermId: 1,
			Did:             validDid,
		}
	}

	tests := []struct {
		name    string
		mutate  func(m *types.MsgStartPermissionVP)
		wantErr string
	}{
		{"valid ISSUER", func(m *types.MsgStartPermissionVP) {}, ""},
		{"valid VERIFIER", func(m *types.MsgStartPermissionVP) { m.Type = types.PermissionType_VERIFIER }, ""},
		{"valid ISSUER_GRANTOR", func(m *types.MsgStartPermissionVP) { m.Type = types.PermissionType_ISSUER_GRANTOR }, ""},
		{"valid VERIFIER_GRANTOR", func(m *types.MsgStartPermissionVP) { m.Type = types.PermissionType_VERIFIER_GRANTOR }, ""},
		{"valid HOLDER", func(m *types.MsgStartPermissionVP) { m.Type = types.PermissionType_HOLDER }, ""},
		{"type UNSPECIFIED rejected", func(m *types.MsgStartPermissionVP) { m.Type = types.PermissionType_UNSPECIFIED }, "invalid permission type"},
		{"type ECOSYSTEM rejected", func(m *types.MsgStartPermissionVP) { m.Type = types.PermissionType_ECOSYSTEM }, "invalid permission type"},
		{"validator_perm_id = 0", func(m *types.MsgStartPermissionVP) { m.ValidatorPermId = 0 }, "validator perm ID cannot be 0"},
		{"empty did", func(m *types.MsgStartPermissionVP) { m.Did = "" }, "DID is required"},
		{"malformed did", func(m *types.MsgStartPermissionVP) { m.Did = "garbage" }, "invalid DID format"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := valid()
			tc.mutate(m)
			err := m.ValidateBasic()
			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
			}
		})
	}
}
```

- [ ] **Step 2: Run**

Run: `go test ./x/perm/types/ -run TestMsgStartPermissionVP_ValidateBasic -v -count=1`

Expected: all 10 cases PASS. If "type ECOSYSTEM rejected" or "type UNSPECIFIED rejected" does not fail, the enum whitelist in `ValidateBasic` (at [types.go:12-74](x/perm/types/types.go#L12-L74)) is not enforcing the spec restriction; tighten it until both cases pass.

- [ ] **Step 3: Commit**

```bash
git add x/perm/types/msg_start_permission_vp_test.go x/perm/types/types.go
git commit --no-gpg-sign -s -m "test(perm): add ValidateBasic boundary tests for MsgStartPermissionVP"
```

#### Task 2.1.4 — `TestMsgSelfCreatePermission_ValidateBasic`

**Files:**
- Create: `x/perm/types/msg_self_create_permission_test.go`

- [ ] **Step 1: Write the test file**

```go
package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana/x/perm/types"
)

// Tests MsgSelfCreatePermission.ValidateBasic per spec [MOD-PERM-MSG-14-1] and
// [MOD-PERM-MSG-14-2-1]. Allowed `type` values per spec: ISSUER or VERIFIER ONLY.
func TestMsgSelfCreatePermission_ValidateBasic(t *testing.T) {
	validAddr := sdk.AccAddress([]byte("test_address________")).String()
	validDid := "did:example:123456789abcdefghi"

	valid := func() *types.MsgSelfCreatePermission {
		return &types.MsgSelfCreatePermission{
			Corporation:     validAddr,
			Operator:        validAddr,
			Type:            types.PermissionType_ISSUER,
			ValidatorPermId: 1,
			Did:             validDid,
		}
	}

	tests := []struct {
		name    string
		mutate  func(m *types.MsgSelfCreatePermission)
		wantErr string
	}{
		{"valid ISSUER", func(m *types.MsgSelfCreatePermission) {}, ""},
		{"valid VERIFIER", func(m *types.MsgSelfCreatePermission) { m.Type = types.PermissionType_VERIFIER }, ""},
		{"type UNSPECIFIED rejected", func(m *types.MsgSelfCreatePermission) { m.Type = types.PermissionType_UNSPECIFIED }, "type must be ISSUER or VERIFIER"},
		{"type ECOSYSTEM rejected", func(m *types.MsgSelfCreatePermission) { m.Type = types.PermissionType_ECOSYSTEM }, "type must be ISSUER or VERIFIER"},
		{"type ISSUER_GRANTOR rejected", func(m *types.MsgSelfCreatePermission) { m.Type = types.PermissionType_ISSUER_GRANTOR }, "type must be ISSUER or VERIFIER"},
		{"type VERIFIER_GRANTOR rejected", func(m *types.MsgSelfCreatePermission) { m.Type = types.PermissionType_VERIFIER_GRANTOR }, "type must be ISSUER or VERIFIER"},
		{"type HOLDER rejected", func(m *types.MsgSelfCreatePermission) { m.Type = types.PermissionType_HOLDER }, "type must be ISSUER or VERIFIER"},
		{"validator_perm_id = 0", func(m *types.MsgSelfCreatePermission) { m.ValidatorPermId = 0 }, "validator perm ID cannot be 0"},
		{"empty did", func(m *types.MsgSelfCreatePermission) { m.Did = "" }, "DID is required"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := valid()
			tc.mutate(m)
			err := m.ValidateBasic()
			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
			}
		})
	}
}
```

- [ ] **Step 2: Run**

Run: `go test ./x/perm/types/ -run TestMsgSelfCreatePermission_ValidateBasic -v -count=1`

Expected: all 9 cases PASS. Any rejection-case failure means `ValidateBasic` at [types.go:340-392](x/perm/types/types.go#L340-L392) is not enforcing the ISSUER/VERIFIER-only restriction; tighten and re-run.

- [ ] **Step 3: Commit**

```bash
git add x/perm/types/msg_self_create_permission_test.go x/perm/types/types.go
git commit --no-gpg-sign -s -m "test(perm): add ValidateBasic boundary tests for MsgSelfCreatePermission"
```

---

### Phase 2.2 — CLI-boundary integration test

The current test suite constructs `MsgXxx` struct literals in Go and calls the handler directly. This bypasses the `autocli` flag bindings entirely, which is exactly how Mohammad's bug escaped. This phase adds a CLI-level test that builds and submits the transaction through the same path the `veranad` binary uses.

#### Task 2.2.1 — Minimal CLI-boundary harness

**Files:**
- Create: `x/perm/client/cli/cli_integration_test.go`

- [ ] **Step 1: Write the harness**

```go
package cli_test

import (
	"context"
	"testing"

	"cosmossdk.io/core/address"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/testutil"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana/x/perm/module"
)

// buildPermTxCommand returns the full tx command tree for the perm module
// exactly as it is wired into the veranad binary. Tests invoke this with
// argv slices, simulating what a user types at the shell.
func buildPermTxCommand(t *testing.T) *cobra.Command {
	t.Helper()
	cfg := module.AppModule{}.AutoCLIOptions()
	cmd, err := cfg.Tx.BuildModuleCommand(module.AppModule{}.Name())
	require.NoError(t, err)
	return cmd
}

// execute runs the given argv through the perm tx command tree and returns
// the command's output and any error. stdin is closed.
func execute(t *testing.T, clientCtx client.Context, argv []string) (string, error) {
	t.Helper()
	cmd := buildPermTxCommand(t)
	out, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, argv)
	return out.String(), err
}
```

- [ ] **Step 2: Build**

Run: `go build ./x/perm/client/cli/...`

Expected: builds cleanly. If `AutoCLIOptions` returns a different shape in our SDK version, adjust imports accordingly.

- [ ] **Step 3: Commit**

```bash
git add x/perm/client/cli/cli_integration_test.go
git commit --no-gpg-sign -s -m "test(perm): add CLI-boundary test harness"
```

#### Task 2.2.2 — `TestCLI_CreateRootPermission_Mohammad_Regression`

**Files:**
- Modify: `x/perm/client/cli/cli_integration_test.go`

- [ ] **Step 1: Append the Mohammad regression test**

Append to the harness file:

```go
// TestCLI_CreateRootPermission_Mohammad_Regression reproduces the devnet bug
// reported by Mohammad on 2026-04-23. Before the Stage 1 fix, running
// `veranad tx perm create-root-perm` without a --permission-type flag
// produced a Msg with PermissionType=UNSPECIFIED and the chain stored it.
// After the fix, the field is gone; the command must still succeed, and the
// resulting Msg's declared permission type is implicit (ECOSYSTEM, set by
// the handler per [MOD-PERM-MSG-7-3]).
//
// This test asserts the CLI does not expose a --permission-type or
// --vs-operator flag on create-root-perm, because the spec does not include
// those fields.
func TestCLI_CreateRootPermission_Mohammad_Regression(t *testing.T) {
	cmd := buildPermTxCommand(t)
	require.NotNil(t, cmd)

	sub, _, err := cmd.Find([]string{"create-root-perm"})
	require.NoError(t, err, "create-root-perm subcommand must exist")

	flagSet := sub.Flags()
	require.Nil(t, flagSet.Lookup("permission-type"),
		"spec [MOD-PERM-MSG-7-1] does not define permission_type; CLI flag must not exist")
	require.Nil(t, flagSet.Lookup("vs-operator"),
		"spec [MOD-PERM-MSG-7-1] does not define vs_operator; CLI flag must not exist")

	// The Use line must only list the spec-defined positional args.
	expected := "create-root-perm [schema-id] [did] [validation-fees] [issuance-fees] [verification-fees]"
	require.Equal(t, expected, sub.Use,
		"create-root-perm Use string must match spec [MOD-PERM-MSG-7-1] parameters")
}
```

- [ ] **Step 2: Run it**

Run: `go test ./x/perm/client/cli/ -run TestCLI_CreateRootPermission_Mohammad_Regression -v -count=1`

Expected: PASS after Stage 1. Before Stage 1 it would have failed on the `Nil` assertions because the fields were present.

- [ ] **Step 3: Commit**

```bash
git add x/perm/client/cli/cli_integration_test.go
git commit --no-gpg-sign -s -m "test(perm): lock in Mohammad devnet regression at CLI boundary"
```

---

### Phase 2.3 — Static analyzer: "mandatory per spec" without ValidateBasic check

This is the cheapest durable fix. A tool that walks all `proto/**/*.proto` files, finds every field annotated `mandatory per spec`, and asserts the corresponding `ValidateBasic` method references the field. If any annotated field has no matching check, CI fails.

#### Task 2.3.1 — Write the analyzer

**Files:**
- Create: `tools/speccoverage/main.go`

- [ ] **Step 1: Write the analyzer**

```go
// Command speccoverage walks proto files for fields annotated with
// "mandatory per spec" comments and verifies that the corresponding
// ValidateBasic in x/{module}/types/types.go references the field name
// (as proof that a check exists). Exits non-zero if any annotated field
// has no check.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	annotationRE = regexp.MustCompile(`(?i)mandatory per spec`)
	fieldRE      = regexp.MustCompile(`^\s*(?:repeated\s+)?\S+\s+(\w+)\s*=\s*\d+`)
	messageRE    = regexp.MustCompile(`^\s*message\s+(\w+)\s*\{`)
)

type annotatedField struct {
	module, message, field, protoFile string
}

func main() {
	root := flag.String("root", ".", "repository root")
	flag.Parse()

	var findings []annotatedField

	err := filepath.WalkDir(filepath.Join(*root, "proto"), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".proto") {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		var module string
		parts := strings.Split(path, string(os.PathSeparator))
		for i, p := range parts {
			if p == "proto" && i+2 < len(parts) {
				module = parts[i+2]
				break
			}
		}

		var currentMsg string
		var pendingAnnotation bool
		for scanner.Scan() {
			line := scanner.Text()
			if m := messageRE.FindStringSubmatch(line); m != nil {
				currentMsg = m[1]
				continue
			}
			if annotationRE.MatchString(line) {
				pendingAnnotation = true
				continue
			}
			if pendingAnnotation {
				if m := fieldRE.FindStringSubmatch(line); m != nil {
					findings = append(findings, annotatedField{
						module: module, message: currentMsg, field: m[1], protoFile: path,
					})
					pendingAnnotation = false
				}
			}
		}
		return scanner.Err()
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	var failures int
	for _, af := range findings {
		typesPath := filepath.Join(*root, "x", af.module, "types", "types.go")
		body, err := os.ReadFile(typesPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cannot read %s: %v\n", typesPath, err)
			failures++
			continue
		}
		// Crude check: find the ValidateBasic for this message, scan its body
		// for a reference to the field name (snake -> camel).
		fieldCamel := snakeToCamel(af.field)
		methodHeader := "(msg *" + af.message + ") ValidateBasic()"
		idx := strings.Index(string(body), methodHeader)
		if idx == -1 {
			fmt.Fprintf(os.Stderr, "FAIL: %s.%s.%s is mandatory per spec but %s has no ValidateBasic for %s\n",
				af.module, af.message, af.field, typesPath, af.message)
			failures++
			continue
		}
		tail := string(body[idx:])
		end := strings.Index(tail, "\n}\n")
		if end == -1 {
			end = len(tail)
		}
		methodBody := tail[:end]
		if !strings.Contains(methodBody, "msg."+fieldCamel) {
			fmt.Fprintf(os.Stderr, "FAIL: %s.%s.%s is mandatory per spec but its ValidateBasic does not reference msg.%s\n",
				af.module, af.message, af.field, fieldCamel)
			failures++
		}
	}

	if failures > 0 {
		fmt.Fprintf(os.Stderr, "%d spec-coverage failures\n", failures)
		os.Exit(1)
	}
	fmt.Printf("spec-coverage OK: %d annotated fields, all checked\n", len(findings))
}

func snakeToCamel(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if len(p) == 0 {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, "")
}
```

- [ ] **Step 2: Build it**

Run: `go build -o /tmp/speccoverage ./tools/speccoverage/`

Expected: builds cleanly.

- [ ] **Step 3: Run against the current repo**

Run: `/tmp/speccoverage -root=.`

Expected: reports `spec-coverage OK: N annotated fields, all checked` (after Stage 1 removed the non-spec annotations). If any FAIL lines appear, those are real coverage gaps to address.

- [ ] **Step 4: Commit**

```bash
git add tools/speccoverage/main.go
git commit --no-gpg-sign -s -m "tools(speccoverage): fail CI when mandatory-per-spec proto fields lack ValidateBasic coverage"
```

#### Task 2.3.2 — Wire the analyzer into CI

**Files:**
- Modify: `.github/workflows/test.yml` (or whichever workflow runs go test on PRs)

- [ ] **Step 1: Read the existing workflow**

Run: `cat .github/workflows/test.yml 2>/dev/null || ls .github/workflows/`

Identify the workflow that runs on pull_request and exercises `go test`. Add a step before the test step that runs the analyzer.

- [ ] **Step 2: Add the step**

In the test workflow job, after the Go setup step and before the `go test` step, insert:

```yaml
      - name: Spec-coverage check
        run: |
          go run ./tools/speccoverage -root=.
```

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/<file>.yml
git commit --no-gpg-sign -s -m "ci: gate PRs on tools/speccoverage (no mandatory-spec field without ValidateBasic)"
```

---

### Phase 2.4 — Update the test-generator skill

The `.claude-agents/test-suite.md` skill currently generates test cases from "MUST abort" bullets in the spec. It does not systematically generate cases for the proto-3 zero-value of each mandatory field. This phase extends the skill so auto-generated test suites cover the class of bug Mohammad found.

#### Task 2.4.1 — Add a "field-omission" generation rule

**Files:**
- Modify: [.claude-agents/test-suite.md](.claude-agents/test-suite.md)

- [ ] **Step 1: Add a new subsection after "1a: Spec-derived tests"**

Insert before "### 1b: Happy path tests":

```markdown
### 1a-bis: Field-omission tests (from proto annotations)

For EACH proto field on the handler's Msg, read the field's comment annotation in `proto/verana/{module}/v1/tx.proto`. If the annotation includes `mandatory per spec`, generate two rejection cases:

ABORT-OMIT-{field}: construct the Msg WITHOUT setting `{field}` (proto3 zero value) and call `ValidateBasic()`. Expect error containing `{field}` or `required` or `invalid`.

For every enum-typed mandatory field, additionally generate:

ABORT-UNSPECIFIED-{field}: construct the Msg with `{field}: TypeXxx_UNSPECIFIED` and call `ValidateBasic()`. Expect error.

For every enum-typed field where the spec restricts to a subset of values (e.g. MsgSelfCreatePermission type must be ISSUER or VERIFIER), additionally generate one rejection case per non-allowed enum value.

These rules exist because Mohammad's devnet bug (2026-04-23) shipped in a PR whose test suite exclusively generated happy-path cases that set every field — no case omitted a newly-added field, so proto3 zero-value storage escaped detection.
```

- [ ] **Step 2: Commit**

```bash
git add .claude-agents/test-suite.md
git commit --no-gpg-sign -s -m "docs(agents): require field-omission + UNSPECIFIED-enum generation in test-suite skill"
```

---

## Self-review checklist (run before executing this plan)

**Spec coverage:** Every Stage 1 task cites the exact spec section ([MOD-PERM-MSG-7-1], [MOD-PERM-MSG-7-3], [MOD-PERM-MSG-7-2-4], [MOD-PERM-MSG-2-1]) that the code change enforces. Stage 2 tests each carry a doc comment quoting the spec anchor. ✓

**Placeholder scan:** No "TBD", "TODO", "fill in details", "similar to above". Every code block is complete. ✓

**Type consistency:** Test names in Stage 2 (`TestMsgCreateRootPermission_ValidateBasic`, `TestCLI_CreateRootPermission_Mohammad_Regression`) match the message types they test. No function is referenced before it is defined by a prior task. ✓

**Ordering:** Stage 1 phases are serial (1.1 → 1.2 → 1.3 → 1.4). Within 1.1, tests are flipped before handler is changed (TDD), then proto is removed after handler no longer depends on it (compile order). Stage 2 phases are independent and can be done in parallel after Stage 1 ships. ✓

**Devnet blast radius:** Stage 1 involves a destructive devnet redeploy (Task 1.4.1). The procedure is the same validated seven-step sequence from 2026-04-23; validator keys are preserved from S3. Acceptable. ✓

---

## Commit hygiene

**Strict rules (per user + [CLAUDE.md](CLAUDE.md)):**
- Every commit message is exactly **1 line**, format `{type}({module}): {short description}`.
- **Never** include `Co-Authored-By` lines. No exceptions.
- Types used in this plan: `fix`, `test`, `tools`, `ci`, `docs`, `chore`.
- Every task ends in a commit; no task bundles unrelated changes.
