# Verana Test Harness Journey Suite

## 1. Scope

### Modules implemented — write journeys now
| Module | Spec prefix | Status |
|--------|-------------|--------|
| MOD-CO | Corporation | PR #319 merged |
| MOD-GF | Governance Framework | PR #318 merged |
| MOD-EC | Ecosystem (renamed from TR) | PR #322 this branch |
| MOD-DE | Delegation (partial — authz bootstrap only) | already wired |

### Modules planned — plan only, no journeys yet
| Module | Spec prefix |
|--------|-------------|
| MOD-CS | Credential Schema |
| MOD-PP | Permission |
| MOD-TD | Trust Deposit |
| MOD-DI | Digest |
| MOD-XR | Exchange Rate |

---

## 2. Data Models (proto-derived)

### Corporation (x/co)
```
id               uint64
policy_address   string   // group_policy_address; signs all delegable msgs
did              string   // unique per-Corporation
created          Timestamp
modified         Timestamp
language         string   // BCP-47, max 17 chars
active_version   uint32   // starts at 1
```

### GovernanceFrameworkVersion (x/gf)
```
id               uint64
ecosystem_id     uint64   // set when subject is Ecosystem; null when subject is Corporation
corporation_id   uint64   // set when subject is Corporation; null when subject is Ecosystem
created          Timestamp
version          uint32
active_since     Timestamp  // null until IncreaseActiveGFV promotes this version
```

### GovernanceFrameworkDocument (x/gf)
```
id               uint64
gfv_id           uint64
created          Timestamp
language         string   // BCP-47
url              string
digest_sri       string
```

### Ecosystem (x/ec)
```
id               uint64
did              string
corporation_id   uint64
created          Timestamp
modified         Timestamp
archived         bool     // false at creation
language         string   // BCP-47, max 17 chars
active_version   uint32   // starts at 1
```

### OperatorAuthorization (x/de)
```
id               uint64
corporation_id   uint64
operator         string   // grantee account
msg_types        []string
expiration       Timestamp (optional)
spend_limit      []Coin (optional)
remaining_spend  []Coin
period           Duration (optional)
```

---

## 3. Account Fixtures

All accounts are derived from deterministic mnemonics so runs are reproducible.
Prefix: `verana` (address encoder).

### Corp A — primary test corporation (used by J001–J006)
| Role | Name in harness | Purpose |
|------|----------------|---------|
| Group admin | `co_admin` | Creates the corporation, submits group proposals |
| Group member 1 | `co_member1` | Votes on group proposals |
| Group member 2 | `co_member2` | Votes on group proposals |
| Operator | `co_operator` | Executes delegable messages on behalf of Corp A |

Decision policy: ThresholdDecisionPolicy, threshold = 2 of 3 members (weight 1 each), voting_period = 30s.

### Corp B — secondary corporation (used by J101–J102, existing journeys)
| Role | Name in harness | Purpose |
|------|----------------|---------|
| Group admin | `ec_admin` | Creates Corp B |
| Group member 1 | `ec_member1` | Votes |
| Group member 2 | `ec_member2` | Votes |
| Operator | `ec_operator` | Executes delegable messages on behalf of Corp B |

### Corp C — adversary (used for cross-corp negative tests)
| Role | Name in harness |
|------|----------------|
| Group admin | `corp_c_admin` |
| Operator | `corp_c_operator` |

Corp C is created at the start of J003 (the first journey that needs a DID conflict test). Corp C creation uses `MsgCreateCorporation` signed by `corp_c_admin` with did="did:verana:co:corp-c". Corp C does NOT need a bootstrap operator authz for the purposes of the journeys below (it is used only as an adversary whose DID or ecosystem id is referenced in negative tests).

### System accounts
| Name | Address | Role |
|------|---------|------|
| cooluser | verana16mzeyu9l6kua2cdg9x0jk5g6e7h0kk8q6uadu4 | Funded faucet in genesis |

---

## 4. Balance Requirements

All funded accounts must hold **at least 50 VNA (50_000_000 uvna)** before any journey starts.

Per-transaction fee: ~750_000 uvna (gas auto, 1.5x adjustment). Each account must have enough for all transactions it signs in a journey. Accounts using feegrant need 0 uvna for message fees but still need the base tx fee from their own balance if feegrant does not cover it — default setup grants feegrant so alice_operator pays 0.

Corp policy addresses must hold **≥ 5 VNA** to:
- Cover feegrant spend for the operator
- Survive multi-step journeys without draining

---

## 5. Transaction Dependency Graph

```
[funded accounts]
       │
       ▼
  J001: CreateCorporation (Corp A)
       │
       ▼
  J002: Bootstrap OperatorAuthorization (Corp A → co_operator)
       │
       ├──▶ J003: UpdateCorporation
       │
       ├──▶ J004: Corp CGF – AddGFDocument (v2, no ecosystem_id)
       │         │
       │         └──▶ J005: Corp CGF – IncreaseActiveGFV (v1→v2)
       │
       ├──▶ J006: CO + GF Queries (GetCorporation, ListCorporations, GetGFV, ListGFVs)
       │
       ├──▶ J020: CreateEcosystem
       │         │
       │         ├──▶ J021: EC GF – AddGFDocument (v2, ecosystem_id set)
       │         │         │
       │         │         └──▶ J022: EC GF – IncreaseActiveGFV (v1→v2)
       │         │
       │         ├──▶ J023: UpdateEcosystem (DID rotation)
       │         │
       │         ├──▶ J024: ArchiveEcosystem + Unarchive
       │         │
       │         └──▶ J025: EC Queries (GetEcosystem, ListEcosystems)
       │
       └──▶ J007: RevokeOperatorAuthorization (cleanup)

[Corp B accounts funded]
       │
       ▼
  J101: Corp B setup (EC group + authz) ── REWRITE (TR→EC)
       │
       ▼
  J102: EC + GF authz operations ────────── REWRITE (TR→EC)
```

Planned (not implementing now):
```
  J201: CS authz setup
  J202: CS authz operations
  J301: Perm authz setup
  J302–J310: Perm operations (skipped, rewrite pending)
  J401: TD reclaim yield
  J501: DI store digest
  J601–J603: XR exchange rates
```

---

## 6. Journey Catalog

---

### J001 — CreateCorporation (Corp A)

**Spec**: MOD-CO-MSG-1  
**Signer**: co_admin (any funded account — no corporation required)  
**Prerequisites**: co_admin, co_member1, co_member2, co_operator accounts funded (≥ 50 VNA each)  
**No AUTHZ-CHECK** (MOD-CO-MSG-1 is explicitly exempt per AUTHZ-CHECK-5)

#### Inputs
| Field | Value |
|-------|-------|
| signer | co_admin |
| members | [{co_admin, weight:1}, {co_member1, weight:1}, {co_member2, weight:1}] |
| decision_policy | ThresholdDecisionPolicy{threshold:2, voting_period:30s} |
| did | "did:verana:co:corp-a" |
| language | "en" |
| doc_url | "https://corp-a.example.com/cgf-v1.pdf" |
| doc_digest_sri | "sha256-aGVsbG8=" |

#### Expected chain state
- Corporation entry: `co.id = auto, co.policy_address = group_policy_addr, co.did = "did:verana:co:corp-a", co.language = "en", co.active_version = 1, co.created = co.modified = block_time`
- GovernanceFrameworkVersion: `gfv.version = 1, gfv.corporation_id = co.id, gfv.ecosystem_id = null, gfv.active_since = block_time`
- GovernanceFrameworkDocument: `gfd.gfv_id = gfv.id, gfd.language = "en", gfd.url = doc_url`

#### Verification
- Query GetCorporation by co.id → assert all fields
- Query GetGFV by gfv.id → assert version=1, active_since set
- Confirm `create_corporation` event emitted with correct attributes

#### Saves to J001.json
`corp_a_id, corp_a_policy_addr, corp_a_group_id, co_admin_addr, co_member1_addr, co_member2_addr, co_operator_addr`

---

### J002 — Bootstrap OperatorAuthorization (Corp A → co_operator)

**Spec**: MOD-DE-MSG-3  
**Mechanism**: Group proposal (co_admin submits, co_admin + co_member1 vote YES, MsgExec runs with group policy as signer)  
**Prerequisites**: J001 complete; co_operator funded (≥ 5 VNA); corp_a_policy_addr funded (≥ 5 VNA for feegrant)

#### Why group proposal
On first authorization, no OperatorAuthorization exists yet. The corporation executes GrantOperatorAuthorization via x/group MsgExec. When x/group runs a proposal, the inner message signers are replaced by the group_policy_address (the executor). The implementation must allow the group_policy_address to pass AUTHZ-CHECK when it acts as its own operator — this is the bootstrap path used in existing journeys (lib.GrantOperatorAuthorizationViaGroup). Verify at implementation time by reading x/de/keeper/msg_grant_operator_authorization.go to confirm the self-authorization check.

#### Inputs
| Field | Value |
|-------|-------|
| corporation | corp_a_policy_addr (from J001) |
| operator | corp_a_policy_addr (self — bootstrap) |
| grantee | co_operator_addr |
| msg_types | [MsgUpdateCorporation, MsgAddGFD, MsgIncreaseActiveGFV, MsgCreateEcosystem, MsgUpdateEcosystem, MsgArchiveEcosystem, MsgGrantOperatorAuthorization, MsgRevokeOperatorAuthorization] |
| with_feegrant | true |
| expiration | (unset — no expiry) |

#### Expected chain state
- OperatorAuthorization: `authz.corporation_id = corp_a_id, authz.operator = co_operator_addr, authz.msg_types = above`
- FeeGrant: `fg.grantor_corporation_id = corp_a_id, fg.grantee = co_operator_addr`

#### Verification
- Query ListOperatorAuthorizations(corporation_id=corp_a_id) → find co_operator entry
- Confirm feegrant by querying x/feegrant

#### Saves to J002.json
`authz_id`

---

### J003 — UpdateCorporation

**Spec**: MOD-CO-MSG-2  
**Signer**: corporation = corp_a_policy_addr, operator = co_operator  
**Prerequisites**: J001, J002  
**AUTHZ-CHECK**: Must pass for (corp_a, co_operator, MsgUpdateCorporation)

#### Test steps

**Step 1 — Happy path: DID rotation**
| Field | Value |
|-------|-------|
| corporation | corp_a_policy_addr |
| operator | co_operator_addr |
| did | "did:verana:co:corp-a-rotated" |

Expected: co.did updated, co.modified bumped, `update_corporation` event emitted.

**Step 2 — No-op: same DID**
Send UpdateCorporation with did = "did:verana:co:corp-a-rotated" (already set).  
Expected: succeeds (no error). Spec MOD-CO-MSG-2-2-1 allows rotating to the current value. Whether `co.modified` is bumped is implementation-defined — verify actual behavior and assert accordingly.

**Step 3 — DID conflict: another corp already owns the target DID**
Precondition: Corp C exists with did = "did:verana:co:corp-c".  
Send UpdateCorporation(corp_a, co_operator, did="did:verana:co:corp-c").  
Expected: abort with ErrDIDOwnershipConflict.

**Step 4 — Wrong operator (unauthorized)**
Send UpdateCorporation(corp_a, corp_c_operator_addr, ...).  
Expected: abort with authorization error (AUTHZ-CHECK-1 fails).

#### Verification
- After Step 1: GetCorporation → assert did = "did:verana:co:corp-a-rotated", modified > created
- After Step 2: GetCorporation → assert modified unchanged

---

### J004 — Corp CGF: AddGovernanceFrameworkDocument (version 2, corporation subject)

**Spec**: MOD-GF-MSG-1 (ecosystem_id = null → subject is Corporation)  
**Signer**: corporation = corp_a_policy_addr, operator = co_operator  
**Prerequisites**: J001, J002  
**AUTHZ-CHECK**: Must pass for (corp_a, co_operator, MsgAddGFD)

#### Key spec rule
`version` MUST be greater than `subject.active_version` (currently 1). Must be exactly `max_existing_version + 1` or refer to an already-existing GFV. First call: version = 2.

#### Test steps

**Step 1 — Happy path: add v2 document**
| Field | Value |
|-------|-------|
| corporation | corp_a_policy_addr |
| operator | co_operator_addr |
| ecosystem_id | (not set) |
| doc_language | "en" |
| doc_url | "https://corp-a.example.com/cgf-v2.pdf" |
| doc_digest_sri | "sha256-dmVyYW5h" |
| version | 2 |

Expected: GFV(v2, corp_id=corp_a_id, ecosystem_id=null) created; GFD created for it; `add_gf_document` event emitted.

**Step 2 — Replace same-language doc for same version**
Send again with doc_url = "https://corp-a.example.com/cgf-v2-updated.pdf", version = 2.  
Expected: existing GFD for (gfv_id, language="en") is UPDATED in place (no new GFD created).

**Step 3 — Add second language for same version**
Send version=2, doc_language="fr", doc_url = "https://corp-a.example.com/cgf-v2-fr.pdf".  
Expected: second GFD created under the same GFV(v2).

**Step 4 — Attempt to modify active version (version=1)**
Send version = 1 (active version).  
Expected: abort — spec MOD-GF-MSG-1-2-1 states "version MUST be greater than subject.active_version".

**Step 5 — Attempt to skip a version (version=3 when max=2)**
Send version = 3 (no existing GFV at v2+1=3, and 3 ≠ max+1=3... wait: if v2 already exists, max is 2, so v3 = max+1 is allowed). Instead test version=4 when only v2 exists: expected abort since 4 ≠ 2+1 and no GFV at v4 exists.

**Step 5 — Wrong ecosystem_id (not owned by corp)**
Send ecosystem_id = <ecosystem owned by Corp C>.  
Expected: abort with authorization error.

#### Verification
- GetGFV(gfv_id) → assert version=2, active_since=null (not yet promoted)
- ListGFVs(corporation_id=corp_a_id) → 2 entries (v1, v2)

#### Saves to J004.json
`corp_a_gfv2_id`

---

### J005 — Corp CGF: IncreaseActiveGovernanceFrameworkVersion (v1 → v2)

**Spec**: MOD-GF-MSG-2 (ecosystem_id = null → subject is Corporation)  
**Signer**: corporation = corp_a_policy_addr, operator = co_operator  
**Prerequisites**: J001, J002, J004 (GFV v2 with at least one GFD in subject.language must exist)  
**AUTHZ-CHECK**: Must pass for (corp_a, co_operator, MsgIncreaseActiveGFV)

#### Key spec rules
- GFV at `active_version + 1` must exist.
- A GFD for `gfv.id` with `language = subject.language` ("en") MUST exist. If no default-language document exists for the next version, abort.

#### Test steps

**Step 1 — Happy path: promote v2**
| Field | Value |
|-------|-------|
| corporation | corp_a_policy_addr |
| operator | co_operator_addr |
| ecosystem_id | (not set) |

Expected: corp.active_version bumped 1→2; corp.modified updated; gfv(v2).active_since set to block_time; `increase_active_gfv` event emitted.

**Step 2 — Attempt to promote again (no v3 exists)**
Send IncreaseActiveGFV again immediately.  
Expected: abort — no GFV at active_version+1 (3) exists.

**Step 3 — Missing default-language document**
Precondition: add GFV v3 with only "fr" document (not "en").  
Send IncreaseActiveGFV.  
Expected: abort — no GFD for gfv(v3) with language = "en" exists.

#### Verification
- GetCorporation → assert active_version = 2, modified > prior
- GetGFV(gfv2_id) → assert active_since is set

---

### J006 — CO + GF Queries

**Spec**: MOD-CO-QRY-1, MOD-CO-QRY-2, MOD-GF-QRY-1, MOD-GF-QRY-2  
**Prerequisites**: J001, J004, J005 (need both corporations and multiple versions)

#### MOD-CO-QRY-1: GetCorporation
- `GetCorporation(id=corp_a_id, active_gf_only=false, preferred_language="")` → returns all GFVs and all GFDs
- `GetCorporation(id=corp_a_id, active_gf_only=true, preferred_language="en")` → returns only active version GFV + one GFD per version
- `GetCorporation(id=99999)` → not found error

#### MOD-CO-QRY-2: ListCorporations
- `ListCorporations(modified_after=epoch)` → all corporations, ordered by modified desc
- `ListCorporations(response_max_size=1)` → returns at most 1
- `ListCorporations(response_max_size=2000)` → error (max 1024)

#### MOD-GF-QRY-1: GetGFV
- `GetGFV(id=corp_a_gfv2_id, preferred_language="en")` → returns GFV with one GFD
- `GetGFV(id=corp_a_gfv2_id, preferred_language="fr")` → returns GFV with French GFD
- `GetGFV(id=99999)` → not found error

#### MOD-GF-QRY-2: ListGFVs
- `ListGFVs(corporation_id=corp_a_id)` → both GFVs, ordered by version asc
- `ListGFVs(corporation_id=corp_a_id, active_only=true)` → only GFV at active_version (2)
- `ListGFVs(ecosystem_id=ec_alpha_id)` → ecosystem's GFVs, ordered by version asc
- `ListGFVs(ecosystem_id=X, corporation_id=Y)` → error (both set — exactly one must be set)
- `ListGFVs(ecosystem_id=0, corporation_id=0)` → error (neither set — exactly one must be set)

---

### J007 — RevokeOperatorAuthorization

**Spec**: MOD-DE-MSG-4  
**Signer**: corporation = corp_a_policy_addr, operator = co_operator  
**Prerequisites**: J001, J002  
**AUTHZ-CHECK**: Must pass for (corp_a, co_operator, MsgRevokeOperatorAuthorization)  
**Note**: Run LAST in Corp A sequence — revocation removes co_operator's ability to act.

#### Inputs
| Field | Value |
|-------|-------|
| corporation | corp_a_policy_addr |
| operator | co_operator_addr |
| grantee | co_operator_addr |

#### Expected chain state
- OperatorAuthorization for (corp_a_id, co_operator_addr) deleted
- FeeGrant for (corp_a_id, co_operator_addr) revoked

#### Verification
- ListOperatorAuthorizations(corporation_id=corp_a_id) → no entry for co_operator
- Subsequent delegable message from co_operator → AUTHZ-CHECK-1 fails

---

### J020 — CreateEcosystem

**Spec**: MOD-EC-MSG-1  
**Signer**: corporation = corp_a_policy_addr, operator = co_operator  
**Prerequisites**: J001, J002  
**AUTHZ-CHECK**: Must pass for (corp_a, co_operator, MsgCreateEcosystem)

#### Test steps

**Step 1 — Happy path**
| Field | Value |
|-------|-------|
| corporation | corp_a_policy_addr |
| operator | co_operator_addr |
| did | "did:verana:ec:alpha" |
| language | "en" |
| doc_url | "https://corp-a.example.com/ec-gf-v1.pdf" |
| doc_digest_sri | "sha256-ZWNhbHBoYQ==" |

Expected:
- Ecosystem: `ec.id=auto, ec.did="did:verana:ec:alpha", ec.corporation_id=corp_a_id, ec.archived=false, ec.active_version=1, ec.created=ec.modified=block_time`
- GFV: `gfv.ecosystem_id=ec.id, gfv.corporation_id=null, gfv.version=1, gfv.active_since=block_time`
- GFD: `gfd.gfv_id=gfv.id, gfd.language="en", gfd.url=doc_url`
- Event `create_ecosystem` with ecosystem_id, corporation_id, did, language attributes

**Step 2 — Same DID, same corp (allowed)**
Send CreateEcosystem with same did="did:verana:ec:alpha", same corp_a.  
Expected: succeeds, new ec.id allocated (DID uniqueness is per-corporation-id, not globally unique per DID).

**Step 3 — Same DID, different corp (conflict)**
Send CreateEcosystem with did="did:verana:ec:alpha", corporation=corp_c_policy_addr.  
Expected: abort with ErrDIDOwnershipConflict (MOD-EC-MSG-1-2-1: DID already exists for a different corp_id).

**Step 4 — Missing mandatory field**
Send CreateEcosystem with did="" (empty).  
Expected: abort (ValidateBasic fails).

#### Saves to J020.json
`ec_alpha_id, ec_alpha_gfv1_id`

---

### J021 — Ecosystem GF: AddGovernanceFrameworkDocument (version 2, ecosystem subject)

**Spec**: MOD-GF-MSG-1 (ecosystem_id set → subject is Ecosystem)  
**Signer**: corporation = corp_a_policy_addr, operator = co_operator  
**Prerequisites**: J001, J002, J020  
**AUTHZ-CHECK**: Must pass for (corp_a, co_operator, MsgAddGFD)

#### Key spec rules (same as J004 but subject is ecosystem)
- `ecosystem_id` must be set and owned by corp_a
- `version` must be > ec.active_version (currently 1); first new version = 2
- GFV with version = max+1 is auto-created if it does not yet exist

#### Test steps

**Step 1 — Happy path: add v2 document for ecosystem**
| Field | Value |
|-------|-------|
| corporation | corp_a_policy_addr |
| operator | co_operator_addr |
| ecosystem_id | ec_alpha_id (from J020) |
| doc_language | "en" |
| doc_url | "https://corp-a.example.com/ec-gf-v2.pdf" |
| doc_digest_sri | "sha256-ZWNnZnYy" |
| version | 2 |

Expected: GFV(v2, ecosystem_id=ec_alpha_id, corporation_id=null) created; GFD created.

**Step 2 — ecosystem_id owned by different corp**
Send with ecosystem_id = <ecosystem owned by Corp C>.  
Expected: abort (subject.corporation_id ≠ co.id).

**Step 3 — version ≤ active_version**
Send version = 1 (active version).  
Expected: abort (spec: version MUST be greater than subject.active_version).

**Step 4 — skip a version (gap in version sequence)**
After GFV(v2) exists for ec_alpha, send version = 4 (skipping v3).  
Expected: abort — spec: version must equal existing GFV version OR exactly max_existing_version + 1. v4 ≠ v2+1=3 and no GFV(v4) exists.

#### Saves to J021.json
`ec_alpha_gfv2_id`

---

### J022 — Ecosystem GF: IncreaseActiveGovernanceFrameworkVersion (v1 → v2)

**Spec**: MOD-GF-MSG-2 (ecosystem_id set → subject is Ecosystem)  
**Signer**: corporation = corp_a_policy_addr, operator = co_operator  
**Prerequisites**: J001, J002, J020, J021  
**AUTHZ-CHECK**: Must pass for (corp_a, co_operator, MsgIncreaseActiveGFV)

#### Test steps

**Step 1 — Happy path**
| Field | Value |
|-------|-------|
| corporation | corp_a_policy_addr |
| operator | co_operator_addr |
| ecosystem_id | ec_alpha_id |

Expected: ec.active_version bumped 1→2; ec.modified updated; gfv(v2).active_since set.

**Step 2 — No next version exists**
Send again after Step 1 (no v3 GFV).  
Expected: abort (no GFV at active_version+1).

**Step 3 — Next version exists but missing default-language GFD**
Setup: call MsgAddGFD(ecosystem_id=ec_alpha_id, version=3, doc_language="fr", doc_url=..., doc_digest_sri=...) to create GFV(v3) with only a French document.  
Send IncreaseActiveGFV(ecosystem_id=ec_alpha_id).  
Expected: abort — spec MOD-GF-MSG-2-2-1: "Find a GovernanceFrameworkDocument gfd for gfd.gfv_id = gfv.id and gfd.language = subject.language. If no document is found, transaction MUST abort." ec.language = "en", no "en" GFD exists for v3.

#### Verification
- GetEcosystem(ec_alpha_id) → assert active_version=2, modified > created
- GetGFV(ec_alpha_gfv2_id) → assert active_since set

---

### J023 — UpdateEcosystem (DID rotation)

**Spec**: MOD-EC-MSG-2  
**Signer**: corporation = corp_a_policy_addr, operator = co_operator  
**Prerequisites**: J001, J002, J020  
**AUTHZ-CHECK**: Must pass for (corp_a, co_operator, MsgUpdateEcosystem)

#### Test steps

**Step 1 — Happy path: rotate DID**
| Field | Value |
|-------|-------|
| corporation | corp_a_policy_addr |
| operator | co_operator_addr |
| id | ec_alpha_id |
| did | "did:verana:ec:alpha-v2" |

Expected: ec.did updated, ec.modified bumped, `update_ecosystem` event emitted.

**Step 2 — No-op: same DID**
Send with did = "did:verana:ec:alpha-v2" (already set).  
Expected: succeeds, ec.modified NOT bumped, no update event emitted.

**Step 3 — DID conflict with different corp**
Precondition: Corp B owns Ecosystem B with did="did:verana:ec:beta".  
Send UpdateEcosystem(corp_a, co_operator, id=ec_alpha_id, did="did:verana:ec:beta").  
Expected: abort ErrDIDOwnershipConflict.

**Step 4 — Ecosystem not owned by this corp**
Send with id = <ecosystem owned by Corp B>.  
Expected: abort ErrUnauthorizedOperator (ec.corporation_id ≠ co.id).

**Step 5 — Wrong operator**
Send with operator = corp_c_operator.  
Expected: abort (AUTHZ-CHECK-1 fails).

---

### J024 — ArchiveEcosystem and Unarchive

**Spec**: MOD-EC-MSG-3  
**Signer**: corporation = corp_a_policy_addr, operator = co_operator  
**Prerequisites**: J001, J002, J020  
**AUTHZ-CHECK**: Must pass for (corp_a, co_operator, MsgArchiveEcosystem)

#### Test steps

**Step 1 — Happy path: archive**
| Field | Value |
|-------|-------|
| corporation | corp_a_policy_addr |
| operator | co_operator_addr |
| id | ec_alpha_id |
| archive | true |

Expected: ec.archived=true, ec.modified bumped, `archive_ecosystem` event with archive_status="archived".

**Step 2 — Idempotency abort: archive already archived**
Send archive=true again.  
Expected: abort ErrAlreadyInTargetArchiveState.

**Step 3 — Happy path: unarchive**
Send archive=false.  
Expected: ec.archived=false, ec.modified bumped, `archive_ecosystem` event with archive_status="unarchived".

**Step 4 — Idempotency abort: unarchive already unarchived**
Send archive=false again.  
Expected: abort ErrAlreadyInTargetArchiveState.

**Step 5 — Wrong corporation**
Send with corporation = corp_b_policy_addr.  
Expected: abort ErrUnauthorizedOperator.

---

### J025 — EC Queries

**Spec**: MOD-EC-QRY-1, MOD-EC-QRY-2  
**Prerequisites**: J020, J021, J022

#### MOD-EC-QRY-1: GetEcosystem
- `GetEcosystem(id=ec_alpha_id, active_gf_only=false, preferred_language="")` → returns all GFVs and GFDs
- `GetEcosystem(id=ec_alpha_id, active_gf_only=true, preferred_language="en")` → only active GFV (v2) with one GFD
- `GetEcosystem(id=99999)` → not found error

#### MOD-EC-QRY-2: ListEcosystems
- `ListEcosystems(corporation_id=corp_a_id)` → all corp A ecosystems, id ASC (no modified_after)
- `ListEcosystems(modified_after=t_before_update)` → ec_alpha (updated by J023) appears before ec created earlier, ordered by modified DESC. Use ec_alpha's modified timestamp from J023 (UpdateEcosystem bumps modified) to produce distinct timestamps for ordering assertion.
- `ListEcosystems(corporation_id=corp_a_id, modified_after=t0)` → combined filter
- `ListEcosystems(response_max_size=1)` → at most 1 result
- `ListEcosystems(response_max_size=2000)` → error (> 1024)

---

### J101 — Corp B Setup (REWRITE: TR→EC rename)

**Status**: Exists, BROKEN — imports reference old TR types.  
**Spec**: MOD-CO-MSG-1, MOD-DE-MSG-3 (bootstrap)  
**Accounts**: ec_admin, ec_member1, ec_member2, ec_operator (Corp B)

This journey is the same structure as J001+J002 but uses Corp B accounts. After rewriting:
- All `trtypes.*` imports → `ectypes.*`
- All `verana-labs/verana/x/tr` paths → `verana-labs/verana/x/ec`
- All `TrustRegistry` field names → `Ecosystem` field names
- Event type constants: `EventTypeCreateTrustRegistry` → `EventTypeCreateEcosystem`
- Attribute key constants updated accordingly
- lib helper calls: any `CreateTrustRegistry` → `CreateEcosystem`

**Steps** (same as before, new type names):
1. Create/recover ec_admin, ec_member1, ec_member2, ec_operator from mnemonics
2. Fund all accounts (50 VNA each) from cooluser
3. CreateCorporation (Corp B) via co_admin using new `cotypes.MsgCreateCorporation`
4. Fund Corp B policy_address (5 VNA)
5. Bootstrap OperatorAuthorization via group proposal for ec_operator
6. Save: corp_b_id, corp_b_policy_addr, corp_b_group_id, ec_operator_addr

---

### J102 — EC + GF Authz Operations (REWRITE: TR→EC rename)

**Status**: Exists, BROKEN — imports reference old TR types.  
**Spec**: MOD-EC-MSG-1/2/3, MOD-GF-MSG-1/2  
**Accounts**: Corp B (from J101)  
**Pattern**: For each operation: (a) attempt without auth → expect failure, (b) grant auth, (c) attempt with auth → expect success

#### Operations tested (each with fail-then-pass)
1. CreateEcosystem (MsgCreateEcosystem)
2. AddGFDocument for ecosystem (MsgAddGFD with ecosystem_id set)
3. IncreaseActiveGFVersion for ecosystem (MsgIncreaseActiveGFV with ecosystem_id set)
4. UpdateEcosystem (MsgUpdateEcosystem)
5. ArchiveEcosystem (MsgArchiveEcosystem) — includes archive+unarchive+idempotency

#### Rename changes required
- `trtypes.MsgCreateTrustRegistry` → `ectypes.MsgCreateEcosystem`
- `trtypes.MsgUpdateTrustRegistry` → `ectypes.MsgUpdateEcosystem`
- `trtypes.MsgArchiveTrustRegistry` → `ectypes.MsgArchiveEcosystem`
- All TR event type strings → EC event type strings
- All TR attribute key constants → EC attribute key constants
- Import path `verana-labs/verana/x/tr/types` → `verana-labs/verana/x/ec/types`
- lib helper function calls renamed accordingly
- Response field: `TrustRegistryId` → `EcosystemId`

---

## 7. Planned Journeys (implement after remaining modules land)

### J201 — CS Authz Setup
Spec: similar to J101 but for Credential Schema module.

### J202 — CS Authz Operations
Spec: MOD-CS-MSG-1 through MOD-CS-MSG-7 + queries.

### J301 — Perm Authz Setup
Spec: MOD-PP-MSG-1 through MOD-PP-MSG-15.  
Blocked: J302-J310 currently skipped pending AUTHZ-CHECK-5 rewrite.

### J401 — TD Reclaim Yield
Spec: MOD-TD-MSG-4 (ReclaimTrustDepositYield).

### J501 — DI Store Digest
Spec: MOD-DI-MSG-1 (StoreDigest).

### J601-J603 — XR Exchange Rate
Spec: MOD-XR-MSG-1/2/3 + queries. Uses x/gov governance proposals.

---

## 8. lib/ Changes Required

### lib/transactions.go
- Rename `CreateTrustRegistry()` → already named `CreateEcosystem()` (confirm import path)
- Confirm all internal type references use `ectypes` not `trtypes`
- Add: `UpdateCorporation()`, helper for query wrappers if missing

### lib/helpers.go
- Rename helper wrappers: `CreateEcosystemWithAuthority()`, `UpdateEcosystemWithAuthority()`, `ArchiveEcosystemWithAuthority()` — confirm these use updated type names
- Add: `CreateCorporationWithAuthority()` — wraps MsgCreateCorporation via group proposal (for J001 variant where corp_admin submits directly)
- Add: `UpdateCorporationWithAuthority()` — wraps MsgUpdateCorporation
- Add: `AddGFDWithAuthority()` for corporation subject (ecosystem_id = null)
- Add: `IncreaseActiveGFVWithAuthority()` for corporation subject
- Add: `RevokeOperatorAuthorizationViaGroup()` — for J007
- Update: `VerifyEcosystem()` — confirm uses `ectypes`

### lib/queries.go
- Add: `QueryCorporation()`, `QueryListCorporations()`
- Add: `QueryGFV()`, `QueryListGFVs()`
- Confirm: `QueryEcosystem()`, `QueryListEcosystems()` use `ectypes`

### lib/fixtures.go
- Extend JourneyResult struct: add fields for `CorporationId`, `CorporationPolicyAddr`, `GFV2Id`

### testharness/cmd/main.go
- Add case entries: 1, 2, 3, 4, 5, 6, 7 (new CO/GF/EC journeys)
- Confirm 101, 102 cases call the rewritten journey functions

---

## 9. File Layout After Implementation

```
testharness/
├── cmd/main.go                          (updated: add J001-J007 cases)
├── journey-suite.md                     (this file)
├── journeys/
│   ├── journey001_co_create_corp.go     (new)
│   ├── journey002_co_operator_authz.go  (new)
│   ├── journey003_co_update_corp.go     (new)
│   ├── journey004_gf_corp_add_gfd.go    (new)
│   ├── journey005_gf_corp_increase_gfv.go (new)
│   ├── journey006_co_gf_queries.go      (new)
│   ├── journey007_co_revoke_authz.go    (new)
│   ├── journey020_ec_create_ecosystem.go (new)
│   ├── journey021_ec_gf_add_gfd.go     (new)
│   ├── journey022_ec_gf_increase_gfv.go (new)
│   ├── journey023_ec_update.go          (new)
│   ├── journey024_ec_archive.go         (new)
│   ├── journey025_ec_queries.go         (new)
│   ├── journey101_ec_authz_setup.go     (rewrite: TR→EC)
│   ├── journey102_ec_authz_operations.go (rewrite: TR→EC)
│   └── ... (existing journeys 201-603 unchanged for now)
├── lib/
│   ├── client.go      (no change)
│   ├── fixtures.go    (extend JourneyResult)
│   ├── helpers.go     (add CO/GF helpers, fix EC rename)
│   ├── queries.go     (add CO/GF queries, confirm EC)
│   ├── transactions.go (fix EC imports, add CO helpers)
│   └── utils.go       (no change)
└── scripts/           (no change)
```

---

## 10. Implementation Order

1. Fix `lib/transactions.go` and `lib/helpers.go` (TR→EC rename + add CO/GF helpers)
2. Fix `lib/queries.go` (add CO/GF query wrappers)
3. Fix `lib/fixtures.go` (extend JourneyResult)
4. Rewrite `journey101_ec_authz_setup.go`
5. Rewrite `journey102_ec_authz_operations.go`
6. Add `journey001_co_create_corp.go`
7. Add `journey002_co_operator_authz.go`
8. Add `journey003_co_update_corp.go`
9. Add `journey004_gf_corp_add_gfd.go`
10. Add `journey005_gf_corp_increase_gfv.go`
11. Add `journey006_co_gf_queries.go`
12. Add `journey007_co_revoke_authz.go`
13. Add `journey020_ec_create_ecosystem.go` through `journey025_ec_queries.go`
14. Update `cmd/main.go`
15. `go build ./...` — verify clean compile
16. Run individual journeys against a local chain
