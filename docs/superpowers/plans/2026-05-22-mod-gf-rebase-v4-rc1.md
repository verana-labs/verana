# MOD-GF PR #318 — Rebase onto Spec v4-rc1

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Rebase the in-flight MOD-GF implementation on `feat/mod-gf-module` (PR #318) from the v4-rc0 address-based model to v4-rc1's `corporation_id` (uint64) FK model. **Path A**: keep the `GFV.active_since` field (the spec entity definition omits it but 4 execution sections still write to it — keeping the field aligns with the executable parts of the spec; the inconsistency is flagged separately for Fabrice to resolve).

**Architecture:** All changes are local to the GF module — no other modules are touched. Since MOD-CO (#303) and the TR→EC rename (#305) haven't landed yet, the rebase makes Corporation-targeted GF calls *no worse* than today (they still fail via the StubCorporationKeeper), and intentionally regresses Ecosystem-targeted GF calls to also fail (the TR adapter can no longer bridge `tr.Corporation` (string) → `EcosystemView.CorporationID` (uint64) without MOD-CO's resolver). This regression is acceptable: PR #318 has never been "fully wired" — it's the foundation, and full functionality unlocks once #303 and #305 land.

**Additional fix bundled in:** The current `active_only` query filter uses `gfv.ActiveSince.IsZero()` to identify the active version — this is **wrong per spec MOD-GF-QRY-2-1** (line 2610) which says "return only the entry corresponding to the subject's `active_version`". The rebase changes the filter to `gfv.version == subject.active_version`, fetched via the EcosystemKeeper/CorporationKeeper. (Independent of `active_since` field decisions.)

**Tech Stack:** Go 1.23.8, Cosmos SDK v0.50.x with `cosmossdk.io/collections`, gogo-proto, testify/require.

---

## Pre-flight

- [ ] **Branch.** Already on `feat/mod-gf-module` (PR #318). Pull latest first:
  ```
  git checkout feat/mod-gf-module
  git pull origin feat/mod-gf-module
  ```
- [ ] **Sanity check baseline.**
  ```
  go build ./... && go test ./x/gf/...
  ```
  Expected: clean build, all tests pass (this is the current pre-rebase state).
- [ ] **Spec snapshot.** Re-read `/tmp/vpr-spec/spec.md` lines 1383–1407 (GFV entity), 2480–2580 (MOD-GF-MSG-1/2 spec), and 2585–2620 (MOD-GF-QRY-1/2 spec) before starting.
- [ ] **Spec inconsistency flagged for Fabrice (post-rebase, NOT blocking):** The GFV entity definition at line 1383 omits `active_since`, but execution sections at lines 2061 (MOD-ES-MSG-1-3), 2276 (MOD-CO-MSG-1-3), 2523 (MOD-GF-MSG-1-3), and 2578 (MOD-GF-MSG-2-3) all write to `gfv.active_since`. We're treating the execution sections as authoritative (keep the field). Post a comment on the spec repo asking Fabrice to either restore `active_since` to the entity definition or update the four execution sections.

---

## File Structure — what changes, what doesn't

### Modified

```
proto/verana/gf/v1/types.proto         # GFV.corporation (string) → corporation_id (uint64); same for GFVWithDocs. KEEP active_since (Path A).
proto/verana/gf/v1/query.proto         # ListRequest.corporation (string) → corporation_id (uint64)

x/gf/types/expected_keepers.go         # EcosystemView.Corporation (string) → CorporationID (uint64); CorporationKeeper interface rename + new resolver method
x/gf/keeper/keeper.go                  # GFVersionByCorporation collection key type: Pair[string, int32] → Pair[uint64, int32]
x/gf/keeper/adapters.go                # TRAsEcosystemKeeper sets CorporationID=0 with caveat (TR.Corporation can't be resolved without MOD-CO); StubCorporationKeeper interface signature updated
x/gf/keeper/msg_add_gf_document.go     # resolveSubject uses corporation_id (uint64); subject lookups via uint64 indices. KEEP gfv (no ActiveSince write in MSG-1 — already zero-valued by default).
x/gf/keeper/msg_increase_active_gfv.go # uint64 corporation_id; KEEP gfv.ActiveSince = now write (Path A — matches MOD-GF-MSG-2-3 line 2578).
x/gf/keeper/query_gfv.go               # active_only: SPEC FIX — check gfv.version == subject.active_version (was wrongly using gfv.ActiveSince.IsZero())
x/gf/keeper/genesis.go                 # XOR validation uses corporation_id > 0; ActiveSince untouched

x/gf/keeper/msg_add_gf_document_test.go      # mocks: CorporationView changes; field assertions: gfv.Corporation → gfv.CorporationId. ActiveSince assertions on MSG-1 result kept (still zero-valued for new GFV).
x/gf/keeper/msg_increase_active_gfv_test.go  # mock signature update; ActiveSince assertion on MSG-2 result kept (still set to block time).
x/gf/keeper/query_gfv_test.go                # active_only test rewritten to assert version match via subject lookup (was wrongly relying on ActiveSince timing)
x/gf/types/genesis_test.go                   # XOR test uses corporation_id (uint64) not "x" string
```

### Unchanged

```
proto/verana/gf/v1/tx.proto              # Msg signers + params unchanged (signer is still `corporation` account = policy_address)
proto/verana/gf/v1/params.proto          # No change
proto/verana/gf/v1/genesis.proto         # No change (GenesisState is just a container)
proto/verana/gf/module/v1/module.proto   # No change

x/gf/types/keys.go                       # Collection prefixes unchanged
x/gf/types/errors.go                     # No change
x/gf/types/events.go                     # No change (event attribute keys are already corporation_id-compatible)
x/gf/types/params.go                     # No change
x/gf/types/types.go                      # No change (BCP47/URL/SRI helpers)
x/gf/types/msgs.go                       # No change (ValidateBasic on Msg shape — unchanged)
x/gf/types/codec.go                      # No change
x/gf/module/module.go                    # No change (still wires the three keepers via depinject)
x/gf/module/autocli.go                   # No change (CLI fields are still `corporation` for signer + `ecosystem-id` for option)
x/gf/keeper/query.go                     # No change
x/gf/keeper/query_params.go              # No change
x/gf/keeper/params.go, msg_server.go, msg_update_params.go  # No change
testutil/keeper/gf.go                    # No change (constructor signature unchanged)
```

---

## Task 1: Proto schema changes + regen

**Files:** `proto/verana/gf/v1/types.proto`, `proto/verana/gf/v1/query.proto`

- [ ] **Step 1.1: Update `types.proto` — FK rename only, keep `active_since`.**

  In `GovernanceFrameworkVersion`: change field 3 from
  ```
    string corporation = 3 [(cosmos_proto.scalar) = "cosmos.AddressString"];
  ```
  to
  ```
    uint64 corporation_id = 3;
  ```
  
  **Keep** `active_since` (field 6) — Path A. The execution sections of the spec still write to it, so we maintain it.
  
  Same in `GovernanceFrameworkVersionWithDocs`: change `corporation` → `corporation_id`, keep `active_since`.

- [ ] **Step 1.2: Update `query.proto`.**

  In `QueryListGovernanceFrameworkVersionsRequest`: change field 2 from
  ```
    string corporation = 2 [(cosmos_proto.scalar) = "cosmos.AddressString"];
  ```
  to
  ```
    uint64 corporation_id = 2;
  ```

- [ ] **Step 1.3: Regenerate.**

  Run: `make proto-gen`
  Expected: no errors. New `x/gf/types/*.pb.go` reflects the schema changes.

- [ ] **Step 1.4: Sanity build (will break, that's expected).**

  Run: `go build ./x/gf/... 2>&1 | head -20`
  Expected: compile errors in keeper/adapters/tests referencing old field names — these are exactly what we'll fix in subsequent tasks.

---

## Task 2: Update `x/gf/types/expected_keepers.go`

**File:** `x/gf/types/expected_keepers.go`

- [ ] **Step 2.1: Update `EcosystemView`.**

  Replace
  ```go
  type EcosystemView struct {
      Id            uint64
      Corporation   string
      Language      string
      ActiveVersion int32
  }
  ```
  with
  ```go
  type EcosystemView struct {
      Id            uint64
      CorporationID uint64   // FK to Corporation.id (was: bech32 address)
      Language      string
      ActiveVersion int32
  }
  ```

- [ ] **Step 2.2: Replace `CorporationKeeper` interface.**

  Remove the current `GetCorporationView` and `SetCorporationActiveVersion` methods. Replace with:
  ```go
  // CorporationKeeper is the minimum surface MOD-GF needs for corporation-targeted GF ops
  // AND for AUTHZ-CHECK-5 resolution. Until issue #303 (MOD-CO) lands, a stub returns
  // (zero, false) so all corporation-targeted GF calls abort with ErrSubjectNotFound.
  type CorporationKeeper interface {
      // ResolveByPolicyAddress is AUTHZ-CHECK-5: given the signing corporation account
      // (= policy_address), return the registered Corporation (its id, language, active_version).
      ResolveByPolicyAddress(ctx context.Context, policyAddress string) (CorporationView, bool)

      // GetByID returns the Corporation by its id (used by query layer + subject validation
      // when the ecosystem references corporation_id and we need its active_version/language).
      GetByID(ctx context.Context, corporationID uint64) (CorporationView, bool)

      // SetActiveVersion bumps the active GF version on the Corporation and updates modified.
      SetActiveVersion(ctx context.Context, corporationID uint64, newVersion int32) error
  }

  // CorporationView is the read shape MOD-GF needs about a Corporation subject.
  type CorporationView struct {
      Id             uint64
      PolicyAddress  string
      Language       string
      ActiveVersion  int32
  }
  ```

  Drop the old `GroupPolicyAddress` field — replaced with `PolicyAddress` (same purpose, name aligned with spec terminology).

- [ ] **Step 2.3: Build (only types package).**

  Run: `go build ./x/gf/types/...`
  Expected: exit 0.

---

## Task 3: Update `x/gf/keeper/keeper.go`

**File:** `x/gf/keeper/keeper.go`

- [ ] **Step 3.1: Re-key `GFVersionByCorporation`.**

  Replace
  ```go
  GFVersionByCorporation collections.Map[collections.Pair[string, int32], uint64]
  ```
  with
  ```go
  GFVersionByCorporation collections.Map[collections.Pair[uint64, int32], uint64]
  ```

  And in `NewKeeper`, change the collection construction from
  ```go
  GFVersionByCorporation: collections.NewMap(sb, types.GFVersionByCorporationKey, "gf_version_by_corporation", collections.PairKeyCodec(collections.StringKey, collections.Int32Key), collections.Uint64Value),
  ```
  to
  ```go
  GFVersionByCorporation: collections.NewMap(sb, types.GFVersionByCorporationKey, "gf_version_by_corporation", collections.PairKeyCodec(collections.Uint64Key, collections.Int32Key), collections.Uint64Value),
  ```

- [ ] **Step 3.2: Build the package.**

  Run: `go build ./x/gf/keeper/...`
  Expected: still broken (handlers + adapters reference old field/method names) — that's fixed in subsequent tasks.

---

## Task 4: Update `x/gf/keeper/adapters.go`

**File:** `x/gf/keeper/adapters.go`

- [ ] **Step 4.1: Update `TRAsEcosystemKeeper.GetEcosystemView`.**

  Replace the body so it produces an `EcosystemView` with `CorporationID = 0` and a clear comment that this is intentionally non-functional until MOD-CO + TR→EC land:
  ```go
  func (a TRAsEcosystemKeeper) GetEcosystemView(ctx context.Context, id uint64) (gftypes.EcosystemView, bool) {
      sdkCtx := sdk.UnwrapSDKContext(ctx)
      tr, err := a.k.GetTrustRegistry(sdkCtx, id)
      if err != nil {
          return gftypes.EcosystemView{}, false
      }
      // INTERIM: TR stores `corporation` as a bech32 address; MOD-GF needs corporation_id (uint64).
      // Without MOD-CO (#303) to resolve address → id, we set CorporationID = 0. Any
      // ecosystem-targeted GF subject check will fail the "subject.CorporationID == co.id"
      // comparison. This adapter is removed entirely when issue #305 lands; until then
      // EC-targeted MOD-GF calls are non-functional. Corporation-targeted calls are
      // gated by the StubCorporationKeeper below and also non-functional.
      return gftypes.EcosystemView{
          Id:            tr.Id,
          CorporationID: 0,
          Language:      tr.Language,
          ActiveVersion: tr.ActiveVersion,
      }, true
  }
  ```

  `SetEcosystemActiveVersion` body is unchanged (it updates `tr.ActiveVersion` and `tr.Modified` by id — the FK type didn't matter for the setter).

- [ ] **Step 4.2: Update `StubCorporationKeeper` to the new interface.**

  Replace the two old methods with three new ones:
  ```go
  func (StubCorporationKeeper) ResolveByPolicyAddress(ctx context.Context, _ string) (gftypes.CorporationView, bool) {
      return gftypes.CorporationView{}, false
  }

  func (StubCorporationKeeper) GetByID(ctx context.Context, _ uint64) (gftypes.CorporationView, bool) {
      return gftypes.CorporationView{}, false
  }

  func (StubCorporationKeeper) SetActiveVersion(ctx context.Context, _ uint64, _ int32) error {
      return errors.New("corporation keeper not wired yet (MOD-CO pending in issue #303)")
  }
  ```

- [ ] **Step 4.3: Build.**

  Run: `go build ./x/gf/keeper/... 2>&1 | head -20`
  Expected: handlers still fail to compile (next tasks fix them).

---

## Task 5: Update `x/gf/keeper/msg_add_gf_document.go`

**File:** `x/gf/keeper/msg_add_gf_document.go`

- [ ] **Step 5.1: Update `resolveSubject` to use uint64 throughout.**

  Replace
  ```go
  type resolvedSubject struct {
      kind          subjectKind
      ecosystemID   uint64
      corporation   string
      language      string
      activeVersion int32
  }
  ```
  with
  ```go
  type resolvedSubject struct {
      kind          subjectKind
      ecosystemID   uint64
      corporationID uint64
      language      string
      activeVersion int32
  }
  ```

  Update the function body:
  ```go
  func (k Keeper) resolveSubject(ctx context.Context, signingCorp string, ecosystemID uint64) (resolvedSubject, error) {
      // First, resolve the signing account to a Corporation (AUTHZ-CHECK-5 surface).
      // The stub returns (zero, false) until #303 lands, so all corp-targeted calls fail here.
      coView, ok := k.corporationKeeper.ResolveByPolicyAddress(ctx, signingCorp)
      if !ok {
          return resolvedSubject{}, cerrors.Wrapf(types.ErrSubjectNotFound, "no Corporation registered for signing account %s", signingCorp)
      }

      if ecosystemID != 0 {
          eco, ok := k.ecosystemKeeper.GetEcosystemView(ctx, ecosystemID)
          if !ok {
              return resolvedSubject{}, cerrors.Wrapf(types.ErrSubjectNotFound, "ecosystem %d", ecosystemID)
          }
          if eco.CorporationID != coView.Id {
              return resolvedSubject{}, types.ErrSubjectNotControlled
          }
          return resolvedSubject{
              kind:          subjectEcosystem,
              ecosystemID:   eco.Id,
              language:      eco.Language,
              activeVersion: eco.ActiveVersion,
          }, nil
      }

      return resolvedSubject{
          kind:          subjectCorporation,
          corporationID: coView.Id,
          language:      coView.Language,
          activeVersion: coView.ActiveVersion,
      }, nil
  }
  ```

- [ ] **Step 5.2: Update `maxVersionFor` for corporation path to use uint64.**

  In the `subjectCorporation` case of the switch, change the iterator:
  ```go
  case subjectCorporation:
      iter, e := k.GFVersionByCorporation.Iterate(ctx, collections.NewPrefixedPairRange[uint64, int32](sub.corporationID))
  ```
  (only the type param and the field name change.)

- [ ] **Step 5.3: Update GFV write paths to use uint64 + drop ActiveSince.**

  In the "create new GFV" branch:
  ```go
  gfv = types.GovernanceFrameworkVersion{
      Id:      nextID,
      Created: ctx.BlockTime(),
      Version: msg.Version,
  }
  if sub.kind == subjectEcosystem {
      gfv.EcosystemId = sub.ecosystemID
  } else {
      gfv.CorporationId = sub.corporationID    // CHANGED: uint64 field + uint64 value
  }
  ```

  And the secondary index update:
  ```go
  if sub.kind == subjectEcosystem {
      if err := ms.GFVersionByEcosystem.Set(ctx, collections.Join(sub.ecosystemID, msg.Version), gfv.Id); err != nil {
          return nil, fmt.Errorf("persist gfv eco index: %w", err)
      }
  } else {
      if err := ms.GFVersionByCorporation.Set(ctx, collections.Join(sub.corporationID, msg.Version), gfv.Id); err != nil {
          return nil, fmt.Errorf("persist gfv corp index: %w", err)
      }
  }
  ```

  Update the event emission:
  ```go
  ctx.EventManager().EmitEvent(sdk.NewEvent(
      types.EventTypeAddGFDocument,
      sdk.NewAttribute(types.AttributeKeyCorporation, msg.Corporation),       // policy_address of signer
      sdk.NewAttribute(types.AttributeKeyEcosystemID, fmt.Sprintf("%d", msg.EcosystemId)),
      sdk.NewAttribute(types.AttributeKeyGFVersionID, fmt.Sprintf("%d", gfv.Id)),
      sdk.NewAttribute(types.AttributeKeyGFDocID, fmt.Sprintf("%d", gfd.Id)),
      sdk.NewAttribute(types.AttributeKeyVersion, fmt.Sprintf("%d", msg.Version)),
      sdk.NewAttribute(types.AttributeKeyLanguage, msg.DocLanguage),
  ))
  ```
  *(Note: also consider adding a `corporation_id` attribute when subject is a Corporation, derived from `sub.corporationID`. Optional polish.)*

- [ ] **Step 5.4: Build.**

  Run: `go build ./x/gf/keeper/msg_add_gf_document.go ./x/gf/keeper/msg_server.go ./x/gf/keeper/keeper.go ./x/gf/keeper/params.go ./x/gf/keeper/adapters.go 2>&1 | head -10`
  
  Expected: this file compiles; other handler / query files may still fail.

---

## Task 6: Update `x/gf/keeper/msg_increase_active_gfv.go`

**File:** `x/gf/keeper/msg_increase_active_gfv.go`

- [ ] **Step 6.1: Update the GFV index lookup.**

  In the switch:
  ```go
  case subjectCorporation:
      gfvID, err = ms.GFVersionByCorporation.Get(ctx, collections.Join(sub.corporationID, nextVersion))
  ```

- [ ] **Step 6.2: Keep `gfv.ActiveSince = now` write — Path A.**

  Spec MOD-GF-MSG-2-3 (line 2578) says "Set `gfv.active_since` to current timestamp." Keep the existing write and the subsequent `ms.GFVersion.Set(ctx, gfv.Id, gfv)`. The only change in this file is updating the corporation keeper call:
  ```go
  case subjectCorporation:
      if err := ms.corporationKeeper.SetActiveVersion(ctx, sub.corporationID, nextVersion); err != nil {
          return nil, fmt.Errorf("update corporation active version: %w", err)
      }
  ```
  (was `ms.corporationKeeper.SetCorporationActiveVersion(ctx, sub.corporation, nextVersion)` — old method name + string arg). The `gfv.ActiveSince = now` and the GFV persist stay exactly as they are today.

- [ ] **Step 6.3: Update event emission** (optional polish — same as MSG-1).

- [ ] **Step 6.4: Build.**

  Run: `go build ./x/gf/keeper/... 2>&1 | head -20`
  Expected: query file may still fail.

---

## Task 7: Update `x/gf/keeper/query_gfv.go`

**File:** `x/gf/keeper/query_gfv.go`

- [ ] **Step 7.1: Update the corporation iterator key type.**

  In `ListGovernanceFrameworkVersions`:
  ```go
  } else {
      iter, err := q.GFVersionByCorporation.Iterate(goCtx, collections.NewPrefixedPairRange[uint64, int32](req.CorporationId))
      ...
  }
  ```
  (note: `req.Corporation` → `req.CorporationId` after the proto regen.)

  Also the `hasCorp` check:
  ```go
  hasCorp := req.CorporationId != 0
  ```
  (was `req.Corporation != ""`.)

- [ ] **Step 7.2: Update `active_only` filter to use subject.active_version.**

  Since `gfv.ActiveSince` is gone, the old `if req.ActiveOnly && gfv.ActiveSince.IsZero()` filter doesn't work. New approach: look up the subject once, then filter for `gfv.version == subject.active_version`.

  Replace the per-GFV filter loop with this logic:
  ```go
  // Resolve subject.active_version once (only needed if ActiveOnly).
  var subjectActiveVersion int32
  if req.ActiveOnly {
      if hasEco {
          eco, ok := q.ecosystemKeeper.GetEcosystemView(goCtx, req.EcosystemId)
          if !ok {
              return nil, status.Errorf(codes.NotFound, "ecosystem %d not found", req.EcosystemId)
          }
          subjectActiveVersion = eco.ActiveVersion
      } else {
          coView, ok := q.corporationKeeper.GetByID(goCtx, req.CorporationId)
          if !ok {
              return nil, status.Errorf(codes.NotFound, "corporation %d not found", req.CorporationId)
          }
          subjectActiveVersion = coView.ActiveVersion
      }
  }

  versions := make([]types.GovernanceFrameworkVersionWithDocs, 0, len(gfvIDs))
  for _, id := range gfvIDs {
      gfv, err := q.GFVersion.Get(goCtx, id)
      if err != nil {
          return nil, status.Errorf(codes.Internal, "fetch gfv %d: %v", id, err)
      }
      if req.ActiveOnly && gfv.Version != subjectActiveVersion {
          continue
      }
      docs, err := q.collectDocs(goCtx, gfv.Id, req.PreferredLanguage)
      if err != nil {
          return nil, status.Errorf(codes.Internal, "collect docs: %v", err)
      }
      versions = append(versions, types.GovernanceFrameworkVersionWithDocs{
          Id:            gfv.Id,
          EcosystemId:   gfv.EcosystemId,
          CorporationId: gfv.CorporationId,
          Created:       gfv.Created,
          Version:       gfv.Version,
          ActiveSince:   gfv.ActiveSince,     // Path A — keep
          Documents:     docs,
      })
  }
  ```
  
  Same WithDocs shape update in `GetGovernanceFrameworkVersion` — keep `ActiveSince:` in the returned struct literal, only change `Corporation` → `CorporationId`.

- [ ] **Step 7.3: Build.**

  Run: `go build ./x/gf/... 2>&1 | head -10`
  Expected: package builds clean.

---

## Task 8: Update `x/gf/keeper/genesis.go`

**File:** `x/gf/keeper/genesis.go`

- [ ] **Step 8.1: Update `InitGenesis` corporation index path.**

  Replace
  ```go
  } else {
      if err := k.GFVersionByCorporation.Set(ctx, collections.Join(gfv.Corporation, gfv.Version), gfv.Id); err != nil {
          return err
      }
  }
  ```
  with
  ```go
  } else {
      if err := k.GFVersionByCorporation.Set(ctx, collections.Join(gfv.CorporationId, gfv.Version), gfv.Id); err != nil {
          return err
      }
  }
  ```

  And update the XOR check that determines which branch to take:
  ```go
  if gfv.EcosystemId > 0 {
      ... // ecosystem index
  } else {
      ... // corporation index (using gfv.CorporationId)
  }
  ```

- [ ] **Step 8.2: Build.**

  Run: `go build ./x/gf/...`
  Expected: clean.

---

## Task 9: Update `x/gf/types/genesis.go`

**File:** `x/gf/types/genesis.go`

- [ ] **Step 9.1: Update XOR validation.**

  Replace
  ```go
  hasEco := gfv.EcosystemId > 0
  hasCorp := gfv.Corporation != ""
  ```
  with
  ```go
  hasEco := gfv.EcosystemId > 0
  hasCorp := gfv.CorporationId > 0
  ```

---

## Task 10: Update tests

**Files:** `x/gf/keeper/msg_add_gf_document_test.go`, `x/gf/keeper/msg_increase_active_gfv_test.go`, `x/gf/keeper/query_gfv_test.go`, `x/gf/types/genesis_test.go`

- [ ] **Step 10.1: Update `mockCorporation` to new interface.**

  In `msg_add_gf_document_test.go`, replace the mock:
  ```go
  type mockCorporation struct {
      view  types.CorporationView
      found bool
      setFn func(uint64, int32) error
  }

  func (m *mockCorporation) ResolveByPolicyAddress(_ context.Context, _ string) (types.CorporationView, bool) {
      return m.view, m.found
  }
  func (m *mockCorporation) GetByID(_ context.Context, _ uint64) (types.CorporationView, bool) {
      return m.view, m.found
  }
  func (m *mockCorporation) SetActiveVersion(_ context.Context, id uint64, v int32) error {
      if m.setFn != nil {
          return m.setFn(id, v)
      }
      return nil
  }
  ```

  All test `corp := &mockCorporation{view: types.CorporationView{...}, found: true}` constructions need to pass:
  ```go
  view:  types.CorporationView{Id: 1, PolicyAddress: testCorp, Language: "en", ActiveVersion: 0},
  ```
  (Previously was `GroupPolicyAddress: testCorp`; now use `PolicyAddress` and add an `Id: 1`.)

- [ ] **Step 10.2: Update `mockEcosystem.view` constructions.**

  Replace
  ```go
  view: types.EcosystemView{Id: 1, Corporation: "other_corp", Language: "en"},
  ```
  with
  ```go
  view: types.EcosystemView{Id: 1, CorporationID: 999, Language: "en"},   // 999 = some non-matching corp id
  ```

  For the controlled case, set `CorporationID: 1` to match the test corporation's id.

- [ ] **Step 10.3: Update GFV field assertions — Path A keeps ActiveSince.**

  Replace `gfv.Corporation` (string) with `gfv.CorporationId` (uint64). KEEP `gfv.ActiveSince` assertions. Examples:
  ```go
  require.Equal(t, uint64(1), gfv.CorporationId)   // was: require.Equal(t, testCorp, gfv.Corporation)
  require.Zero(t, gfv.EcosystemId)
  require.True(t, gfv.ActiveSince.IsZero())        // keep — new GFV is still unactivated by default
  ```

- [ ] **Step 10.4: Update MSG-2 test — keep ActiveSince assertion, just fix mock signature.**

  In `msg_increase_active_gfv_test.go`, keep `require.False(t, gfv.ActiveSince.IsZero())` and `require.Equal(t, now, gfv.ActiveSince)` — MSG-2 still writes ActiveSince per Path A. Only change is the mock `setFn` signature:
  ```go
  corp.setFn = func(_ string, v uint32) error { newActive = v; return nil }
  ```
  becomes
  ```go
  corp.setFn = func(_ uint64, v int32) error { newActive = v; return nil }
  ```
  And `newActive` should be `int32`, not `uint32`. Adjust the corresponding `require.Equal(t, int32(1), newActive)`.

- [ ] **Step 10.5: Update query test for `active_only` — switch to subject-version semantics.**

  The current test expects `active_only` to filter by `ActiveSince != zero`. The new (correct) semantics filter by `gfv.version == subject.active_version`. After MSG-1 + MSG-2 (which activates v1), the mock corporation's `view.ActiveVersion` is bumped to 1, then v2 is added. The query with `active_only=true` should return only v1.

  Required test setup change: after MSG-2 activates v1, the test explicitly sets `corp.view.ActiveVersion = 1` (already present in the test). The query's active_only filter will now read this via `GetByID(1)` and filter to GFVs where `version == 1`. Existing assertions on `versions[0].Version == 1` still pass.

  Also update `Corporation: testCorp` in the query request → `CorporationId: 1`:
  ```go
  all, err := qs.ListGovernanceFrameworkVersions(ctx, &types.QueryListGovernanceFrameworkVersionsRequest{CorporationId: 1})
  ```

  The mock corporation's `GetByID` MUST return `view.ActiveVersion = 1` after the bump; this happens automatically since the test directly mutates `corp.view.ActiveVersion = 1` between MSG-2 and the v2 add (already in current test).

- [ ] **Step 10.6: Update genesis tests.**

  In `genesis_test.go`, replace `Corporation: "x"` with `CorporationId: 1` in GFV literals:
  ```go
  {Id: 1, EcosystemId: 1, CorporationId: 1, Version: 1},    // both set → ErrInvalidSubject
  {Id: 1, Version: 1},                                       // neither → ErrInvalidSubject
  {Id: 1, CorporationId: 1, Version: 1},                     // valid
  ```

- [ ] **Step 10.7: Run tests.**

  Run: `go test ./x/gf/... -v 2>&1 | tail -40`
  Expected: ALL tests pass.

---

## Task 11: Full repo build + test

- [ ] **Step 11.1: Full build.**

  Run: `go build ./...`
  Expected: exit 0.

- [ ] **Step 11.2: Full test suite.**

  Run: `go test ./... -count=1 2>&1 | tail -30`
  Expected: no FAIL, no panic. No regressions in any other module.

- [ ] **Step 11.3: Make build + init genesis smoke test.**

  Run:
  ```
  make build
  rm -rf /tmp/veranad-gf-rc1-smoke
  ./build/veranad init test-node --chain-id verana-test --home /tmp/veranad-gf-rc1-smoke 2>&1 | tail -3
  grep -A 5 '"gf"' /tmp/veranad-gf-rc1-smoke/config/genesis.json
  ```
  Expected: genesis contains `"gf": { "params": {}, "versions": [], "documents": [] }`, no panic.

---

## Task 12: Commit + push + PR update

- [ ] **Step 12.1: Single commit for the rebase.**

  Run:
  ```
  git add proto/verana/gf/v1/ x/gf/ api/verana/gf/ ts-proto/src/codec/verana/gf/
  git commit -m "feat(gf): rebase to spec v4-rc1 — corporation_id (uint64) FK, drop GFV.active_since"
  ```

  (This is a single coherent unit — schema + keeper + adapter + tests all move together.)

- [ ] **Step 12.2: Push to the existing PR branch.**

  Run: `git push origin feat/mod-gf-module`

- [ ] **Step 12.3: Update the PR description on GitHub** to call out the v4-rc1 alignment:

  Run:
  ```
  gh pr edit 318 --repo verana-labs/verana --body "$(cat <<'EOF'
  ## Summary

  Implements the new MOD-GF module per spec **v4-rc1**. Closes #304.

  - New `x/gf` Cosmos SDK module
  - Entities: `GovernanceFrameworkVersion`, `GovernanceFrameworkDocument` — now owned by MOD-GF, polymorphic over Ecosystem | Corporation via **`corporation_id` (uint64)** FK
  - GFV retains `active_since` field per spec execution sections MOD-ES-MSG-1-3 / MOD-CO-MSG-1-3 / MOD-GF-MSG-1-3 / MOD-GF-MSG-2-3 (the entity definition at spec line 1383 omits it but the execution sections still write to it — kept as-is until Fabrice resolves the inconsistency)
  - Messages: `MsgAddGovernanceFrameworkDocument` (MOD-GF-MSG-1), `MsgIncreaseActiveGovernanceFrameworkVersion` (MOD-GF-MSG-2), `MsgUpdateParams`
  - Queries: `GetGovernanceFrameworkVersion` (MOD-GF-QRY-1), `ListGovernanceFrameworkVersions` (MOD-GF-QRY-2) — **bug fix**: `active_only` filter was wrongly checking `ActiveSince.IsZero()`; now correctly uses subject.`active_version` per spec MOD-GF-QRY-2-1
  - Interim `TRAsEcosystemKeeper` lets the existing `x/tr` keeper serve as the `EcosystemKeeper` — sets `CorporationID=0` until issue #305 (TR→EC) and #303 (MOD-CO) land and the address-as-FK can be resolved properly. **Until then, both EC- and Corp-targeted GF calls are non-functional** (Corp-targeted gated by `StubCorporationKeeper`; EC-targeted gated by the placeholder `CorporationID=0`). This is intentional: PR #318 is the MOD-GF foundation; full functionality unlocks when #303 + #305 merge.
  - Depinject wiring: GF's `DelegationKeeper`, `EcosystemKeeper`, `CorporationKeeper` are all provided via the GF module's `init()`. No post-build setters.

  ## Test plan

  - [x] Unit tests for MOD-GF-MSG-1 — happy + 4 precondition scenarios (AUTHZ failure, ecosystem-not-controlled, version<=active_version, GFD upsert)
  - [x] Unit tests for MOD-GF-MSG-2 — happy + 2 precondition scenarios (no next version, missing default-lang doc)
  - [x] Unit tests for queries — preferred-language filter, ecosystem/corporation XOR (uint64), ascending-version order, `active_only` via subject.`active_version`
  - [x] Genesis validation cases — default valid, XOR violations rejected, dangling gfv_id rejected
  - [x] Full repo `go build ./...` and `go test ./...` pass — no regressions across any module
  - [x] `make build && veranad init` produces clean genesis with `gf` module section

  ## Spec rebase notes

  This PR was originally written against an earlier v4 draft where GFV used a bech32 `corporation` (string address) FK and carried `active_since`. Rebased onto v4-rc1 (post verana-labs/verifiable-trust-vpr-spec#132/#133/#134) which:

  - Changed `corporation` (address) → `corporation_id` (uint64) everywhere
  - Kept `GFV.active_since` field (spec inconsistency between entity definition and execution sections — flagged separately for Fabrice; execution sections currently treated as authoritative)
  - Added `MOD-DE-QRY-3` and `MOD-DE-QRY-4` (handled in #310, not this PR)
  - Reframed AUTHZ-CHECK-5 as a resolver (signing account → `co.id`) — exposed via the new `CorporationKeeper.ResolveByPolicyAddress` in this PR's expected_keepers
  EOF
  )"
  ```

- [ ] **Step 12.4: Re-request review** (optional, comment-only):

  Run:
  ```
  gh pr comment 318 --repo verana-labs/verana --body "Rebased onto spec v4-rc1. Key changes: \`GFV.corporation\` (address) → \`corporation_id\` (uint64); dropped \`GFV.active_since\`; \`active_only\` query filter now uses subject.\`active_version\`. Tests + smoke pass clean."
  ```

---

## Self-Review Checklist

- [ ] Every reference to `gfv.Corporation` (string) has become `gfv.CorporationId` (uint64) — search: `grep -r "gfv.Corporation\b" x/gf/`
- [ ] `gfv.ActiveSince` references KEPT — search confirms still present in: proto, GFVWithDocs construction in queries, MSG-2 write, MSG-1 test assertion (zero), MSG-2 test assertion (non-zero)
- [ ] `EcosystemView.Corporation` is gone; `EcosystemView.CorporationID` is used
- [ ] `CorporationView.GroupPolicyAddress` is gone; `CorporationView.PolicyAddress` is used
- [ ] `CorporationKeeper` has three methods: `ResolveByPolicyAddress`, `GetByID`, `SetActiveVersion`
- [ ] `GFVersionByCorporation` collection key is `Pair[uint64, int32]`
- [ ] Query `active_only` looks up subject via `GetByID` / `GetEcosystemView` and filters by `gfv.Version == subjectActiveVersion` (the OLD `ActiveSince.IsZero()` filter was wrong per spec MOD-GF-QRY-2-1)
- [ ] Adapter file has clear caveat comment about EC-targeted GF being non-functional until #303 + #305 land
- [ ] Single commit, push to existing PR branch, PR description updated

---

## What stays broken (intentionally) until #303 + #305 land

- Corporation-targeted MOD-GF-MSG-1/2 calls fail with `ErrSubjectNotFound` (StubCorporationKeeper)
- Ecosystem-targeted MOD-GF-MSG-1/2 calls fail with `ErrSubjectNotControlled` (TR adapter produces `CorporationID=0`, never matches resolved `co.id`)
- This was already true pre-rebase for corp-targeted; new for eco-targeted. Both unlock when #303 + #305 land.

Unit tests cover the happy paths with mock keepers, so PR is fully testable in isolation.
