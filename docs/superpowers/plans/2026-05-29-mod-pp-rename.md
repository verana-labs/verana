# MOD-PP Rename (x/perm → x/pp) Implementation Plan

> **For agentic workers:** This is a large mechanical rename. Verification gates (`make proto-gen`, `go build ./...`, `go test ./...`) replace per-step unit TDD. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Rename the `x/perm` Permission module to `x/pp` Participant module per spec v4-rc2, including proto package, messages, enums, entity fields, REST paths, Go keeper, app wiring, cross-module callers, ts-proto, and testharness.

**Architecture:** Mirror the proven TR→EC rename (#322). Generated code (`api/verana/**`, `ts-proto/src/codec/verana/**`) is committed, so regenerate after proto edits. No data migration — fresh testnet on this version bump.

**Tech Stack:** Cosmos SDK (Go), protobuf/buf, pulsar codegen, ts-proto, Go testharness.

**Base branch:** `feat/mod-pp-rename` off `origin/main` (1b328a1 — has MOD-CO #319, MOD-GF #318, TR→EC #322).

---

## Spec Audit (line-by-line vs https://verana-labs.github.io/verifiable-trust-vpr-spec/ MOD-PP)

Confirmed against the live spec Participant data model + the #307 issue. ✓ = spec and plan agree.

### Entity: Permission → Participant
| Spec field | Type | Mand. | Current proto (Permission) | Action |
|---|---|---|---|---|
| `id` | uint64 | ✓ key | `id` (1) | keep |
| `schema_id` | uint64 | ✓ | `schema_id` (2) | keep |
| `role` | ParticipantRole | ✓ | `type` PermissionType (3) | **rename type→role**, enum→ParticipantRole |
| `did` | string | **✓ mandatory** | `did` (4) optional | keep field, **make validation mandatory** |
| `corporation_id` | **uint64** | ✓ | `corporation` AddressString (5) | **rename+retype → corporation_id uint64** (reserve old #5, new number) |
| `vs_operator` | account | ✓ | `vs_operator` (37) | keep |
| `created/adjusted/slashed/repaid` | ts | mixed | (6/8/10/12) | keep |
| `effective_from/until/modified` | ts | mixed | (14/15/16) | keep |
| `validation_fees/issuance_fees/verification_fees` | num | ✓ | (17/18/19) | keep |
| `deposit/slashed_deposit/repaid_deposit` | num | ✓ | (20/21/22) | keep |
| `revoked` | ts | opt | (23) | keep |
| `validator_participant_id` | uint64 | opt | `validator_perm_id` (26) | **rename** |
| `op_state` | OnboardingState | ✓ | `vp_state` ValidationState (27) | **rename + enum rename** |
| `op_exp` | ts | opt | `vp_exp` (28) | **rename** |
| `op_last_state_change` | ts | ✓ | `vp_last_state_change` (29) | **rename** |
| `op_validator_deposit` | num | opt | `vp_validator_deposit` (30) | **rename** |
| `op_current_fees` | num | ✓ | `vp_current_fees` (31) | **rename** |
| `op_current_deposit` | num | ✓ | `vp_current_deposit` (32) | **rename** |
| `op_summary_digest` | str | opt | `vp_summary_digest` (33) | **rename** |
| `issuance_fee_discount/verification_fee_discount` | num | ✓ | (35/36) | keep |

### Enums
- `PermissionType` → **`ParticipantRole`**; values unchanged (UNSPECIFIED=0, ISSUER, VERIFIER, ISSUER_GRANTOR, VERIFIER_GRANTOR, ECOSYSTEM, HOLDER). ✓
- `ValidationState` → **`OnboardingState`**; values: keep `ONBOARDING_STATE_UNSPECIFIED=0`, PENDING, VALIDATED, TERMINATED. ✓ (spec lists PENDING/VALIDATED/TERMINATED; proto needs the 0 sentinel)

### ParticipantSession (was PermissionSession)
Spec: `id` uuid, `corporation_id` uint64, `vs_operator` account, `created`, `modified`, `session_records[]`.
Current PermissionSession: id(1), corporation(2), vs_operator(3), **agent_perm_id(4)**, session_records(5), created(6), modified(7).
**Actions:** rename corporation→corporation_id (uint64); **drop session-level `agent_perm_id`** (moves into record as `agent_participant_id`); renumber to spec order.

### ParticipantSessionRecord (was PermissionSessionRecord)
Spec: `id` uint64 (key, NEW), `created`, `issuer_participant_id` opt, `verifier_participant_id` opt, `wallet_agent_participant_id` opt, `agent_participant_id` opt.
Current: created(1), issuer_perm_id(2), verifier_perm_id(3), wallet_agent_perm_id(4).
**Actions:** add `id` uint64; rename *_perm_id → *_participant_id; add `agent_participant_id`.

### Messages (rename map + amino names)
| Old | New | Old amino | New amino |
|---|---|---|---|
| MsgStartPermissionVP | MsgStartParticipantOP | verana/x/perm/MsgStartPermissionVP | verana/x/pp/MsgStartParticipantOP |
| MsgRenewPermissionVP | MsgRenewParticipantOP | verana/x/perm/MsgRenewPermissionVP | verana/x/pp/MsgRenewParticipantOP |
| MsgSetPermissionVPToValidated | MsgSetParticipantOPToValidated | verana/x/perm/MsgSetPermVPValidated | verana/x/pp/MsgSetPartOPValidated |
| MsgCancelPermissionVPLastRequest | MsgCancelParticipantOPLastRequest | verana/x/perm/MsgCancelPermVPLastReq | verana/x/pp/MsgCancelPartOPLastReq |
| MsgCreateRootPermission | MsgCreateRootParticipant | verana/x/perm/MsgCreateRootPermission | verana/x/pp/MsgCreateRootParticipant |
| MsgAdjustPermission | MsgSetParticipantEffectiveUntil | verana/x/perm/MsgAdjustPermission | verana/x/pp/MsgSetPartEffectiveUntil |
| MsgRevokePermission | MsgRevokeParticipant | verana/x/perm/MsgRevokePermission | verana/x/pp/MsgRevokeParticipant |
| MsgCreateOrUpdatePermissionSession | MsgCreateOrUpdateParticipantSession | verana/x/perm/MsgCreateOrUpdatePermSess | verana/x/pp/MsgCreateOrUpdatePartSess |
| MsgSlashPermissionTrustDeposit | MsgSlashParticipantTrustDeposit | verana/x/perm/MsgSlashPermTD | verana/x/pp/MsgSlashParticipantTD |
| MsgRepayPermissionSlashedTrustDeposit | MsgRepayParticipantSlashedTrustDeposit | verana/x/perm/MsgRepayPermSlashedTD | verana/x/pp/MsgRepayPartSlashedTD |
| MsgSelfCreatePermission | MsgSelfCreateParticipant | verana/x/perm/MsgSelfCreatePermission | verana/x/pp/MsgSelfCreateParticipant |
| MsgUpdateParams | MsgUpdateParams (keep) | verana/x/perm/MsgUpdateParams | verana/x/pp/MsgUpdateParams |

`MsgTriggerResolver` — NOT present on main (PR #288 unmerged). Skip; do not invent.

### REST paths: `/verana/perm/v1/*` → `/verana/pp/v1/*` (7 paths). ✓

### Key audit decisions (deviations / clarifications)
1. **VSOA fields kept.** `vs_operator_authz_*` (proto 38-42) remain on Participant in #307. The issue defers their removal/move to ParticipantAuthorizationRecord to #309. Removing now would break `vsoa_feegrant.go` (out of scope). **Keep them, renamed module only.**
2. **Field numbers:** keep existing numbers for pure name-only renames (type→role, vp_*→op_*, validator_perm_id→validator_participant_id) — functionally identical, lower regen risk. Use reserve-old + new-number ONLY where wire type changes: `corporation`(str)→`corporation_id`(uint64) on Participant and ParticipantSession. Testnet reset ⇒ no migration concern. Proto field numbers are not part of the published spec, so this does not affect spec compliance.
3. **`agent_perm_id` relocation** (session→record) is a structural change but is explicitly defined in issue scope item 3 and the spec; included here.
4. **OnboardingState 0 sentinel** kept as `ONBOARDING_STATE_UNSPECIFIED` (spec omits it; proto requires a 0 default).

---

## Rename reference tables (mechanical)

### Go identifiers
| Old | New |
|---|---|
| `Permission` (struct) | `Participant` |
| `PermissionSession` | `ParticipantSession` |
| `PermissionSessionRecord` | `ParticipantSessionRecord` |
| `PermissionType` | `ParticipantRole` |
| `ValidationState` | `OnboardingState` |
| `ValidatorPermId` | `ValidatorParticipantId` |
| `VpState/VpExp/VpLastStateChange/VpValidatorDeposit/VpCurrentFees/VpCurrentDeposit/VpSummaryDigest` | `OpState/OpExp/OpLastStateChange/OpValidatorDeposit/OpCurrentFees/OpCurrentDeposit/OpSummaryDigest` |
| `Corporation` (Participant/Session field) | `CorporationId` (uint64) |
| `Type` (Participant field) | `Role` |
| import `x/perm` | `x/pp` |
| ModuleName `"perm"` | `"pp"` |

### Go file renames (git mv inside x/pp)
- `keeper/start_perm_vp.go` → `keeper/start_participant_op.go`
- `keeper/perm_validated.go` → `keeper/participant_validated.go`
- `types/msg_start_permission_vp_test.go` → `types/msg_start_participant_op_test.go`
- `types/msg_renew_permission_vp_test.go` → `types/msg_renew_participant_op_test.go`
- `types/msg_self_create_permission_test.go` → `types/msg_self_create_participant_test.go`
- `types/msg_create_root_permission_test.go` → `types/msg_create_root_participant_test.go`
- `keeper/csps.go`, `keeper/vsoa_feegrant.go` — keep names (acronyms still valid)

### Cross-module / app touchpoints
- `x/de/types/types.go:31-41,57` — 11 type-URL map entries + `MsgCreateOrUpdatePermissionSessionTypeURL` const → `/verana.pp.v1.MsgCreateOrUpdateParticipantSession`
- `x/de/keeper/msg_server_test.go` — type-URL strings in tests
- `app/app.go:10,115,214,272-274,355` — import, keeper alias `permissionmodulekeeper`, field `PermissionKeeper`, getter, depinject
- `app/app_config.go:57,69,120,154,182,210,370-371` — module aliases, ModuleName in genesis/begin/end/accPerms order, module config
- `app/upgrades/types/types.go:5` — keeper import + `GetPermissionKeeper`
- `testutil/keeper/permission.go` → `participant.go` — imports + mock names

### ts-proto touchpoints
- `ts-proto/src/codec/verana/perm/` → regenerated to `verana/pp/` via `make proto-ts`
- `ts-proto/src/amino-converter/perm.ts` → `pp.ts` (11 converters, aminoType strings per table)
- `ts-proto/src/signing.ts` — imports, `veranaTypeUrls`, `veranaRegistryTypes`, `createVeranaAminoTypes` (type-URL + message-name updates)
- `ts-proto/test/src/journeys/perm*.ts` + `deGrantPermOperatorAuthorization.ts` — message/type-URL/field updates (out of CI hard-gate; update for compile)

### testharness touchpoints
- `testharness/journeys/journey30x_perm_*.go` (302-310) — type-URL strings, `permtypes` usage, field reads (`VpState`→`OpState`, `Type`→`Role`)
- `testharness/lib/{transactions,queries,helpers}.go` — `permtypes` import + message constructors + field reads

---

## Execution Phases (each ends at a verification gate)

### Phase 1 — Proto layer
- [ ] **1.1** `git mv proto/verana/perm proto/verana/pp`
- [ ] **1.2** In every `proto/verana/pp/**/*.proto` + `*.swagger.json`: `verana.perm.v1`→`verana.pp.v1`, `verana.perm.module`→`verana.pp.module`, `go_package ".../x/perm/types"`→`".../x/pp/types"`, REST `/verana/perm/v1/`→`/verana/pp/v1/`.
- [ ] **1.3** `types.proto`: rename messages `Permission`→`Participant`, `PermissionSession`→`ParticipantSession`, `PermissionSessionRecord`→`ParticipantSessionRecord`; enums per table; fields per Spec Audit table (type→role, vp_*→op_*, validator_perm_id→validator_participant_id, corporation→corporation_id uint64 with `reserved 5; reserved "corporation";` + new number; session corporation→corporation_id; drop session agent_perm_id; add record `id` + `agent_participant_id` + rename record *_perm_id). Keep `vs_operator_authz_*`.
- [ ] **1.4** `tx.proto`: rename 11 Msgs + responses + rpc names per table; update amino names per table; rename in-message fields (`type`→`role`, `validator_perm_id`→`validator_participant_id`, `vp_summary_digest`→`op_summary_digest`, CSPS `*_perm_id`→`*_participant_id`).
- [ ] **1.5** `query.proto`: rename Query rpcs/messages (ListPermissions→ListParticipants, GetPermission→GetParticipant, sessions, FindPermissionsWithDID→FindParticipantsWithDID, FindBeneficiaries keep), repeated `permissions`→`participants` fields, REST paths.
- [ ] **1.6** `genesis.proto`: `permissions`→`participants`, `permission_sessions`→`participant_sessions`, `next_permission_id`→`next_participant_id`.
- [ ] **1.7** `params.proto`: amino name → `verana/x/pp/Params`.
- [ ] **Gate:** `cd proto && buf lint` passes.

### Phase 2 — Regenerate codegen
- [ ] **2.1** `git rm -r api/verana/perm` then `make proto-gen` (regenerates `api/verana/pp/**` pulsar + grpc).
- [ ] **2.2** Verify `api/verana/pp/` exists, `api/verana/perm/` gone.
- [ ] **Gate:** `ls api/verana/pp/v1/tx.pulsar.go` exists.

### Phase 3 — Go module (x/perm → x/pp)
- [ ] **3.1** `git mv x/perm x/pp`.
- [ ] **3.2** Repo-wide import path `github.com/verana-labs/verana/x/perm` → `.../x/pp` (Go files only).
- [ ] **3.3** `x/pp/types/keys.go`: `ModuleName="perm"`→`"pp"`, `MemStoreKey="mem_permission"`→`"mem_pp"`, `ParamsKey "p_permission"`→`"p_pp"`.
- [ ] **3.4** Go identifier renames per table across `x/pp/**` (struct types, enum types, field names, `types.ModuleName` usages). Includes `module/module.go` package import `api/verana/perm/module`→`api/verana/pp/module`.
- [ ] **3.5** `git mv` the 6 Go files per file-rename table; update their package-internal references.
- [ ] **3.6** `keys.go` store-prefix Go var names may stay (PermissionKey etc.) but rename to ParticipantKey/ParticipantCounterKey/ParticipantSessionKey for clarity; the byte prefixes are arbitrary (testnet reset).
- [ ] **3.7** `events.go`: attribute key strings `validator_perm_id`→`validator_participant_id`, `vp_summary_digest`→`op_summary_digest`, `vp_exp`→`op_exp`; emit `corporation_id`, `role`.
- [ ] **3.8** Keeper logic: where Participant is created, set `CorporationId` (resolve via existing AUTHZ-CHECK-5 corp resolution already in place — `msg.Corporation` account → `co.id`; if resolver not yet wired, persist `CorporationId` from resolved id). `did` validation mandatory in `types.go` ValidateBasic for create/start messages. CSPS keeper: populate `ParticipantSessionRecord.Id` (increment) and `AgentParticipantId` from msg.
- [ ] **3.9** `module/autocli.go`: command names (`start-perm-vp`→`start-participant-op`, etc.), positional/flag proto-field names per renames.
- [ ] **Gate:** `go build ./x/pp/...` compiles.

### Phase 4 — App wiring + cross-module
- [ ] **4.1** `x/de/types/types.go`: 11 type-URL entries + const → `/verana.pp.v1.*` new message names.
- [ ] **4.2** `x/de/keeper/msg_server_test.go`: type-URL strings.
- [ ] **4.3** `app/app.go`, `app/app_config.go`, `app/upgrades/types/types.go`: aliases/keeper/field/getter/ModuleName/module-config (keep Go alias readable, e.g. `ppmodulekeeper`).
- [ ] **4.4** `testutil/keeper/permission.go`→`participant.go`: imports + mock type names (`MockTrustRegistry`-style perm mocks if any).
- [ ] **4.5** Any `x/td` comment/string referencing perm message names.
- [ ] **Gate:** `go build ./...` compiles.

### Phase 5 — ts-proto
- [ ] **5.1** `git rm -r ts-proto/src/codec/verana/perm` then `make proto-ts` (regenerates `verana/pp/`).
- [ ] **5.2** `git mv ts-proto/src/amino-converter/perm.ts pp.ts`; update imports (`../codec/verana/pp/v1/...`), exported converter names, aminoType strings per table, field names.
- [ ] **5.3** `ts-proto/src/signing.ts`: imports from `./codec/verana/pp/v1/tx` + `./amino-converter/pp`, `veranaTypeUrls` (new keys + `/verana.pp.v1.*`), `veranaRegistryTypes`, `createVeranaAminoTypes`.
- [ ] **5.4** `ts-proto/src/index.ts` + `helpers/aminoConverters.ts` if they reference perm.
- [ ] **5.5** Update `ts-proto/test/src/journeys/perm*.ts` + `deGrantPermOperatorAuthorization.ts` imports/type-URLs/fields for compile.
- [ ] **Gate:** `cd ts-proto && npm run build` (tsc) passes; `npm run smoke`.

### Phase 6 — testharness
- [ ] **6.1** `testharness/lib/{transactions,queries,helpers}.go`: `permtypes` import path, message constructors → new names, field reads (`VpState`→`OpState`, `Type`→`Role`, `Corporation`→`CorporationId`).
- [ ] **6.2** `testharness/journeys/journey30x_perm_*.go`: type-URL strings + `permtypes.*` + field reads.
- [ ] **Gate:** `go build ./testharness/...`.

### Phase 7 — Full verification
- [ ] **7.1** `make proto-gen && make proto-ts` clean (no diff drift).
- [ ] **7.2** `go build ./...`.
- [ ] **7.3** `go test ./x/pp/... ./x/de/... ./app/...` (then `go test ./...`).
- [ ] **7.4** Acceptance grep: zero refs to `MsgStartPermissionVP`, `vp_state`, `ValidationState`, `PermissionType`, `validator_perm_id`, `verana.perm.v1`, `x/perm` anywhere.
- [ ] **7.5** `cd ts-proto && npm run build && npm run smoke`.
- [ ] **7.6** Commit per phase or as one `refactor(pp)!:` commit (max 2-line message, no Co-Authored-By).

---

## Acceptance criteria (from #307)
- [ ] `make proto-gen` succeeds
- [ ] `go test ./...` passes
- [ ] No refs to `MsgStartPermissionVP`, `vp_state`, `ValidationState`, `PermissionType`, `validator_perm_id`
- [ ] `Participant.did` mandatory in validation
- [ ] `Participant.corporation_id` is uint64
- [ ] `ParticipantSessionRecord.id` generated + persisted
- [ ] `x/pp` genesis round-trip test passes
- [ ] CLI under `/pp/v1/*`
- [ ] testharness journeys updated + building
- [ ] Events carry `corporation_id` + participant `id`

## Out of scope (deferred)
- VSOA field removal/move to ParticipantAuthorizationRecord (#309-#313)
- AUTHZ-CHECK-5 enforcement changes (#308) beyond persisting corporation_id
- Method shape changes beyond renames + the session/record restructure defined above
