# MOD-CO Corporation Module — Implementation Plan

> Sequence position **2 of 13** in the v4-rc2 spec rebase. Branched off `feat/mod-gf-module` (PR #318, in review). Rebases onto main after #318 merges.

**Goal:** Implement the `x/co` Corporation module per spec v4-rc2. Adds the canonical `Corporation` entity (id + policy_address + did), the atomic MOD-CO-MSG-1 flow that wraps `MsgCreateGroupWithPolicy` and seeds the v1 GF, MOD-CO-MSG-2 (DID rotation), gov-only MOD-CO-MSG-3 (params), and three queries. Hooks into MOD-GF in two directions:

- **MOD-CO satisfies `gftypes.CorporationKeeper`** (the resolver + lookup + setter MOD-GF needs from MOD-CO)
- **MOD-CO consumes a new `cotypes.GFKeeper`** (the v1-GF creation + per-corp listing MOD-CO needs from MOD-GF)

**Side effects on MOD-GF** (in same PR, since dropping the stub provider is required for depinject to wire MOD-CO as the real CorporationKeeper):

- Remove `ProvideCorporationKeeper` from `x/gf/module/module.go` (MOD-CO provides it now)
- Remove `StubCorporationKeeper` struct from `x/gf/keeper/adapters.go` (no longer used)
- Add public method `CreateInitialGFVersionForCorporation` to `x/gf/keeper` (called by MOD-CO-MSG-1 to seed v1 GF)
- Add public method `ListVersionsByCorporation` to `x/gf/keeper` (called by MOD-CO-QRY-1 and QRY-2 to enrich responses with nested GFV+GFD)

**Tech Stack:** Go 1.23.8, Cosmos SDK v0.50.x, `cosmossdk.io/collections`, `x/group` from cosmos-sdk, gogo-proto, testify/require. Same patterns as MOD-GF.

---

## Pre-flight

- [ ] **Branch.** Off `feat/mod-gf-module` HEAD:
  ```
  git checkout feat/mod-gf-module && git pull
  git checkout -b feat/mod-co-module
  ```
- [ ] **Baseline.** `go build ./... && go test ./... -count=1` — expect clean.
- [ ] **Spec snapshot.** Re-read `/tmp/vpr-spec/spec.md` §1354–1370 (Corporation entity), §1869–1876 (AUTHZ-CHECK-5), §1988–2196 (MOD-CO module: MSG-1/2/3 + QRY-1/2/3).

---

## File structure

### Created

```
proto/verana/co/v1/types.proto         # Corporation entity + Owner type if needed
proto/verana/co/v1/params.proto        # empty Params (forward-compat)
proto/verana/co/v1/tx.proto            # MsgCreateCorporation, MsgUpdateCorporation, MsgUpdateParams
proto/verana/co/v1/query.proto         # GetCorporation, ListCorporations, Params
proto/verana/co/v1/genesis.proto       # GenesisState
proto/verana/co/module/v1/module.proto # depinject Module config

x/co/types/keys.go                     # ModuleName, store prefixes
x/co/types/errors.go                   # ErrCorporationNotRegistered, ErrDIDExists, etc.
x/co/types/events.go                   # EventType + AttributeKey constants
x/co/types/codec.go                    # RegisterInterfaces + RegisterLegacyAminoCodec
x/co/types/genesis.go                  # DefaultGenesis + Validate
x/co/types/params.go                   # NewParams, DefaultParams, Validate
x/co/types/msgs.go                     # ValidateBasic for MsgCreate/Update/UpdateParams
x/co/types/expected_keepers.go         # DelegationKeeper, GroupKeeper, GFKeeper interfaces

x/co/keeper/keeper.go                  # Keeper struct + NewKeeper + GetAuthority/Logger/GetNextID
x/co/keeper/params.go                  # GetParams, SetParams
x/co/keeper/corporation.go             # Native lookups: GetCorporation, GetByPolicyAddress, GetByDID, SetActiveVersion
x/co/keeper/corporation_keeper.go      # Methods that satisfy gftypes.CorporationKeeper (ResolveByPolicyAddress, GetByID, SetActiveVersion)
x/co/keeper/msg_server.go              # msgServer router
x/co/keeper/msg_update_params.go       # UpdateParams (gov-only)
x/co/keeper/msg_create_corporation.go  # MOD-CO-MSG-1 atomic execution
x/co/keeper/msg_update_corporation.go  # MOD-CO-MSG-2 DID rotation
x/co/keeper/query.go                   # querier router
x/co/keeper/query_params.go            # Params query
x/co/keeper/query_corporation.go       # GetCorporation + ListCorporations

x/co/keeper/genesis.go                 # InitGenesis + ExportGenesis (Corporation entries only; GFV/GFD owned by x/gf)

x/co/module/module.go                  # AppModule + depinject Provide: ProvideModule + ProvideCorporationKeeper
x/co/module/autocli.go                 # AutoCLI bindings

testutil/keeper/co.go                  # CoKeeperWithMocks(t, delegation, gfKeeper, groupKeeper) constructor

# Tests (mirror MOD-GF coverage discipline — every handler + query + helper + keeper method, edge cases)
x/co/keeper/keeper_test.go
x/co/keeper/params_test.go
x/co/keeper/query_params_test.go
x/co/keeper/msg_update_params_test.go
x/co/keeper/msg_create_corporation_test.go
x/co/keeper/msg_update_corporation_test.go
x/co/keeper/query_corporation_test.go
x/co/keeper/genesis_test.go
x/co/keeper/corporation_keeper_test.go   # tests the gftypes.CorporationKeeper interface impl
x/co/types/types_test.go                 # if any helper functions added
x/co/types/msgs_test.go                  # ValidateBasic exhaustive
x/co/types/codec_test.go
x/co/types/genesis_test.go
x/co/module/genesis_test.go
x/co/module/helpers_test.go              # local stub keepers for module-level tests
```

### Modified

```
app/app.go                            # wire CoKeeper into depinject inject list
app/app_config.go                     # add x/co to module side-effect imports + genesisModuleOrder + module config list
x/gf/module/module.go                 # REMOVE ProvideCorporationKeeper provider + the function; REMOVE CorporationKeeper field from ModuleInputs; pass nil to NewKeeper (signature changes); ADD ProvideGFKeeper for MOD-CO to consume
x/gf/keeper/adapters.go               # REMOVE StubCorporationKeeper struct + NewStubCorporationKeeper function
x/gf/keeper/keeper.go                 # DROP corporationKeeper param from NewKeeper; ADD SetCorporationKeeper(*Keeper) pointer-receiver setter; ADD public CreateInitialGFVersionForCorporation + ListVersionsByCorporation methods
x/gf/keeper/msg_add_gf_document.go    # ADD defensive nil-check on k.corporationKeeper in resolveSubject (returns ErrSubjectNotFound if nil)
x/gf/keeper/query_gfv.go              # ADD defensive nil-check on q.corporationKeeper in active_only subject lookup
x/gf/types/expected_keepers.go        # documentation comment updated to drop "until MOD-CO lands" caveat
testutil/keeper/gf.go                 # constructor updated to call SetCorporationKeeper internally so existing tests keep working
```

### NOT touched in this PR

- `x/tr/` — TR→EC rename is sequence #305, separate PR
- `x/gf/keeper/adapters.go TRAsEcosystemKeeper` — still needed until #305 lands
- `x/perm/` — sequence #307
- Any other module's Msg signatures

---

## Phase 1 — Proto Schema

### Task 1: types.proto (Corporation entity)

```protobuf
syntax = "proto3";
package verana.co.v1;

import "amino/amino.proto";
import "cosmos_proto/cosmos.proto";
import "gogoproto/gogo.proto";
import "google/protobuf/timestamp.proto";

option go_package = "github.com/verana-labs/verana/x/co/types";

// Corporation is the canonical VPR-level entity for a legal/organizational
// entity acting in the registry. Identified by its `id` (uint64); anchored
// on-chain by a `policy_address` account that signs on its behalf.
message Corporation {
  uint64 id = 1;
  string policy_address = 2 [(cosmos_proto.scalar) = "cosmos.AddressString"];
  string did = 3;
  google.protobuf.Timestamp created = 4 [
    (gogoproto.stdtime) = true,
    (gogoproto.nullable) = false,
    (amino.dont_omitempty) = true
  ];
  google.protobuf.Timestamp modified = 5 [
    (gogoproto.stdtime) = true,
    (gogoproto.nullable) = false,
    (amino.dont_omitempty) = true
  ];
  string language = 6;
  uint32 active_version = 7;
}
```

### Task 2: params.proto, tx.proto, query.proto, genesis.proto

All follow the same shape as MOD-GF. Notable details:

**tx.proto:**

```protobuf
// MsgCreateCorporation — MOD-CO-MSG-1. Signer is the submitting account.
message MsgCreateCorporation {
  option (cosmos.msg.v1.signer) = "signer";
  option (amino.name) = "verana/x/co/MsgCreateCorporation";

  string signer = 1 [(cosmos_proto.scalar) = "cosmos.AddressString"];
  repeated cosmos.group.v1.MemberRequest members = 2 [(gogoproto.nullable) = false];
  string group_metadata = 3;
  string group_policy_metadata = 4;
  google.protobuf.Any decision_policy = 5 [(cosmos_proto.accepts_interface) = "cosmos.group.v1.DecisionPolicy"];
  string did = 6;
  string language = 7;
  string doc_url = 8;
  string doc_digest_sri = 9;
}

message MsgCreateCorporationResponse {
  uint64 corporation_id = 1;
  string policy_address = 2 [(cosmos_proto.scalar) = "cosmos.AddressString"];
}

// MsgUpdateCorporation — MOD-CO-MSG-2.
message MsgUpdateCorporation {
  option (cosmos.msg.v1.signer) = "corporation";
  option (cosmos.msg.v1.signer) = "operator";
  option (amino.name) = "verana/x/co/MsgUpdateCorporation";

  string corporation = 1 [(cosmos_proto.scalar) = "cosmos.AddressString"];
  string operator = 2 [(cosmos_proto.scalar) = "cosmos.AddressString"];
  string did = 3;
}
```

**query.proto:**

```protobuf
message QueryGetCorporationRequest {
  uint64 corporation_id = 1;
  bool active_gf_only = 2;
  string preferred_language = 3;
}

message QueryGetCorporationResponse {
  CorporationWithGF corporation = 1 [(gogoproto.nullable) = false];
}

message CorporationWithGF {
  uint64 id = 1;
  string policy_address = 2 [(cosmos_proto.scalar) = "cosmos.AddressString"];
  string did = 3;
  google.protobuf.Timestamp created = 4 [(gogoproto.stdtime) = true, (gogoproto.nullable) = false];
  google.protobuf.Timestamp modified = 5 [(gogoproto.stdtime) = true, (gogoproto.nullable) = false];
  string language = 6;
  uint32 active_version = 7;
  // Imported type from x/gf to nest GF data
  repeated verana.gf.v1.GovernanceFrameworkVersionWithDocs versions = 8 [(gogoproto.nullable) = false];
}

message QueryListCorporationsRequest {
  google.protobuf.Timestamp modified_after = 1 [(gogoproto.stdtime) = true];
  bool active_gf_only = 2;
  string preferred_language = 3;
  uint32 response_max_size = 4;
}

message QueryListCorporationsResponse {
  repeated CorporationWithGF corporations = 1 [(gogoproto.nullable) = false];
}
```

> **Note**: `CorporationWithGF.versions` imports `verana.gf.v1.GovernanceFrameworkVersionWithDocs`. Cross-package proto imports work — but I need to verify the buf/proto-gen config picks it up. Fallback: define a local `co.v1.GFVersionWithDocs` that mirrors the gf shape. Decide during Task 4 (regen).

### Task 3: module/v1/module.proto + regen

Standard pattern (mirror `proto/verana/gf/module/v1/module.proto`). Then `make proto-gen`. Expect new files in `x/co/types/*.pb.go`, `api/verana/co/v1/*.pulsar.go`, `ts-proto/src/codec/verana/co/`.

---

## Phase 2 — Types Package

### Task 4: keys.go, errors.go, events.go, params.go

Trivial — follow MOD-GF patterns. Key store prefixes:

```go
var (
    ParamsKey                  = collections.NewPrefix(1)
    CorporationKey             = collections.NewPrefix(2) // id → Corporation
    CorporationByPolicyAddrKey = collections.NewPrefix(3) // policy_address → id (reverse index for AUTHZ-CHECK-5)
    CorporationByDIDKey        = collections.NewPrefix(4) // did → id (for uniqueness check)
    CounterKey                 = collections.NewPrefix(5)
)
```

Errors:
```go
ErrCorporationNotRegistered = errors.Register(ModuleName, 1100, "signing account is not the policy_address of a registered Corporation")
ErrDIDAlreadyExists         = errors.Register(ModuleName, 1101, "DID is already registered by another Corporation")
ErrPolicyAddressAlreadyBound = errors.Register(ModuleName, 1102, "policy_address is already bound to an existing Corporation")
ErrInvalidLanguage          = errors.Register(ModuleName, 1103, "invalid BCP 47 language tag")
ErrInvalidURL               = errors.Register(ModuleName, 1104, "invalid URL")
ErrInvalidDigestSRI         = errors.Register(ModuleName, 1105, "invalid digest_sri")
ErrInvalidDecisionPolicy    = errors.Register(ModuleName, 1106, "invalid decision_policy")
ErrInvalidMembers           = errors.Register(ModuleName, 1107, "invalid members list")
```

### Task 5: msgs.go (ValidateBasic) — three Msgs, full branch coverage

Same pattern as MOD-GF. For `MsgCreateCorporation` exhaustive checks: signer present, members non-empty + valid weights, decision_policy non-nil + correct type, did valid syntax, language BCP47, url valid, sri valid. Empty-field rejection for every required field.

### Task 6: expected_keepers.go (3 interfaces)

```go
// DelegationKeeper — for AUTHZ-CHECK-1 on MOD-CO-MSG-2. Signature matches x/de
// exactly so depinject autowires the DE keeper.
type DelegationKeeper interface {
    CheckOperatorAuthorization(ctx context.Context, corporation, operator, msgTypeURL string, now time.Time) error
}

// GroupKeeper — minimum surface to call MsgCreateGroupWithPolicy. We don't import
// the full x/group keeper type — just the Msg-server side we need.
type GroupKeeper interface {
    CreateGroupWithPolicy(ctx context.Context, req *group.MsgCreateGroupWithPolicy) (*group.MsgCreateGroupWithPolicyResponse, error)
}

// GFKeeper — what MOD-CO needs FROM MOD-GF.
type GFKeeper interface {
    // Called by MOD-CO-MSG-1 step 3 to seed the v1 GF for a fresh corporation.
    CreateInitialGFVersionForCorporation(ctx context.Context, corpID uint64, language, docURL, docDigestSRI string) error
    // Called by MOD-CO-QRY-1 and QRY-2 to nest GF data into Corporation responses.
    ListVersionsByCorporation(ctx context.Context, corpID uint64, activeVersion int32, activeOnly bool, preferredLang string) ([]gftypes.GovernanceFrameworkVersionWithDocs, error)
}
```

The `GroupKeeper` is the x/group module's keeper interface — we use it directly. For depinject wiring, the x/group keeper concrete type satisfies this interface (the `CreateGroupWithPolicy` Msg server method matches the signature shape).

> Actually let me verify during Task 8: the cleanest call may be via the Msg server router, not the keeper directly. If keeper doesn't expose it, fall back to `MsgServiceRouter`. Track decision during implementation.

---

## Phase 3 — Keeper

### Task 7: keeper.go (struct + constructor + collections + counter)

```go
type Keeper struct {
    cdc          codec.BinaryCodec
    storeService store.KVStoreService
    logger       log.Logger
    authority    string

    Schema                      collections.Schema
    Params                      collections.Item[types.Params]
    Corporation                 collections.Map[uint64, types.Corporation]
    CorporationByPolicyAddress  collections.Map[string, uint64]
    CorporationByDID            collections.Map[string, uint64]
    Counter                     collections.Map[string, uint64]

    delegationKeeper types.DelegationKeeper
    groupKeeper      types.GroupKeeper
    gfKeeper         types.GFKeeper
}

func NewKeeper(
    cdc codec.BinaryCodec,
    storeService store.KVStoreService,
    logger log.Logger,
    authority string,
    delegationKeeper types.DelegationKeeper,
    groupKeeper types.GroupKeeper,
    gfKeeper types.GFKeeper,
) Keeper { ... }
```

Panics on invalid authority bech32. Schema build follows MOD-GF pattern.

### Task 8: msg_create_corporation.go (MOD-CO-MSG-1 atomic execution)

The biggest method in the module. Mirrors spec MOD-CO-MSG-1-3 step-by-step:

```go
func (ms msgServer) CreateCorporation(goCtx context.Context, msg *types.MsgCreateCorporation) (*types.MsgCreateCorporationResponse, error) {
    if err := msg.ValidateBasic(); err != nil {
        return nil, err
    }
    ctx := sdk.UnwrapSDKContext(goCtx)

    // Per-Corporation DID uniqueness invariant (MOD-CO-MSG-1-2-1).
    if _, err := ms.CorporationByDID.Get(ctx, msg.Did); err == nil {
        return nil, cerrors.Wrapf(types.ErrDIDAlreadyExists, "did %s already registered", msg.Did)
    }

    // Atomic x/group creation. group_policy_as_admin = true is hardcoded per spec.
    resp, err := ms.groupKeeper.CreateGroupWithPolicy(ctx, &group.MsgCreateGroupWithPolicy{
        Admin:               msg.Signer,
        Members:             msg.Members,
        GroupMetadata:       msg.GroupMetadata,
        GroupPolicyMetadata: msg.GroupPolicyMetadata,
        DecisionPolicy:      msg.DecisionPolicy,
        GroupPolicyAsAdmin:  true,
    })
    if err != nil {
        return nil, fmt.Errorf("create group with policy: %w", err)
    }
    policyAddr := resp.GroupPolicyAddress

    // Inverse of AUTHZ-CHECK-5: no Corporation must already exist for this policy_address.
    // (Practically impossible because group_policy_address is freshly minted, but a defensive
    //  check costs nothing and documents the invariant.)
    if _, err := ms.CorporationByPolicyAddress.Get(ctx, policyAddr); err == nil {
        return nil, cerrors.Wrapf(types.ErrPolicyAddressAlreadyBound, "policy_address %s already bound", policyAddr)
    }

    // Allocate Corporation id from counter.
    id, err := ms.GetNextID(ctx, "corporation")
    if err != nil {
        return nil, fmt.Errorf("alloc corporation id: %w", err)
    }
    now := ctx.BlockTime()
    co := types.Corporation{
        Id:            id,
        PolicyAddress: policyAddr,
        Did:           msg.Did,
        Created:       now,
        Modified:      now,
        Language:      msg.Language,
        ActiveVersion: 1,
    }
    if err := ms.Corporation.Set(ctx, co.Id, co); err != nil {
        return nil, fmt.Errorf("persist corporation: %w", err)
    }
    if err := ms.CorporationByPolicyAddress.Set(ctx, policyAddr, co.Id); err != nil {
        return nil, fmt.Errorf("set policy_address index: %w", err)
    }
    if err := ms.CorporationByDID.Set(ctx, msg.Did, co.Id); err != nil {
        return nil, fmt.Errorf("set DID index: %w", err)
    }

    // Delegate v1 GF creation to MOD-GF (spec MOD-CO-MSG-1-3 step 3 — v4-rc2: gfv.active_since = now).
    if err := ms.gfKeeper.CreateInitialGFVersionForCorporation(ctx, co.Id, msg.Language, msg.DocUrl, msg.DocDigestSri); err != nil {
        return nil, fmt.Errorf("seed v1 gf: %w", err)
    }

    ctx.EventManager().EmitEvent(sdk.NewEvent(
        types.EventTypeCreateCorporation,
        sdk.NewAttribute(types.AttributeKeyCorporationID, fmt.Sprintf("%d", co.Id)),
        sdk.NewAttribute(types.AttributeKeyPolicyAddress, policyAddr),
        sdk.NewAttribute(types.AttributeKeyDID, msg.Did),
    ))

    return &types.MsgCreateCorporationResponse{
        CorporationId: co.Id,
        PolicyAddress: policyAddr,
    }, nil
}
```

### Task 9: msg_update_corporation.go (MOD-CO-MSG-2)

Standard: ValidateBasic → AUTHZ-CHECK-1 → look up `co` by signing `corporation` account (the policy_address) → check new DID uniqueness across OTHER corps → update co.did + co.modified → update CorporationByDID index (remove old entry, set new).

Important edge case: rotating to the same value `co.did` already holds is a no-op and MUST be allowed (spec MOD-CO-MSG-2-2-1). Handle by checking `if existing.Id == co.Id { allow }` in the uniqueness check.

### Task 10: corporation_keeper.go (satisfy gftypes.CorporationKeeper)

```go
func (k Keeper) ResolveByPolicyAddress(ctx context.Context, policyAddr string) (gftypes.CorporationView, bool) {
    id, err := k.CorporationByPolicyAddress.Get(ctx, policyAddr)
    if err != nil {
        return gftypes.CorporationView{}, false
    }
    co, err := k.Corporation.Get(ctx, id)
    if err != nil {
        return gftypes.CorporationView{}, false
    }
    return gftypes.CorporationView{
        Id:            co.Id,
        PolicyAddress: co.PolicyAddress,
        Language:      co.Language,
        ActiveVersion: int32(co.ActiveVersion),
    }, true
}

func (k Keeper) GetByID(ctx context.Context, id uint64) (gftypes.CorporationView, bool) {
    co, err := k.Corporation.Get(ctx, id)
    if err != nil {
        return gftypes.CorporationView{}, false
    }
    return gftypes.CorporationView{
        Id:            co.Id,
        PolicyAddress: co.PolicyAddress,
        Language:      co.Language,
        ActiveVersion: int32(co.ActiveVersion),
    }, true
}

// SetActiveVersion is called by MOD-GF-MSG-2 when activating the next GF version.
// Spec MOD-GF-MSG-2-3: update subject.active_version + subject.modified.
func (k Keeper) SetActiveVersion(ctx context.Context, corpID uint64, newVersion int32) error {
    co, err := k.Corporation.Get(ctx, corpID)
    if err != nil {
        return fmt.Errorf("corporation %d not found: %w", corpID, err)
    }
    co.ActiveVersion = uint32(newVersion)
    sdkCtx := sdk.UnwrapSDKContext(ctx)
    co.Modified = sdkCtx.BlockTime()
    return k.Corporation.Set(ctx, corpID, co)
}
```

Note the `int32` ↔ `uint32` cast: spec uses signed int for active_version; our entity uses uint32 (chain's existing convention from x/tr). Safe cast because versions only grow positive.

### Task 11: query_corporation.go (GetCorporation + ListCorporations)

GetCorporation: lookup by id, fetch nested GF data via `gfKeeper.ListVersionsByCorporation` (filtered by active_only + preferred_language).

ListCorporations: iterate the Corporation map, filter by `modified_after`, sort by `modified` descending (spec convention: most-recent first), cap at `response_max_size` (default 64, max 1024), enrich each with GF data via the same GFKeeper call.

Implementation note for ordering: collect matching Corporation entries into a slice during iteration, then `sort.Slice(corps, func(i, j int) bool { return corps[i].Modified.After(corps[j].Modified) })` before applying the size cap.

### Task 12: params.go, msg_server.go, msg_update_params.go, query.go, query_params.go

Trivial — identical patterns to MOD-GF.

### Task 13: genesis.go (Init/Export round-trip)

InitGenesis: persist corporations + restore both reverse indices + restore counter. ExportGenesis: dump corporations only (GFV/GFD belong to x/gf).

---

## Phase 4 — Module wiring + MOD-GF stub removal

### Task 14: module.go + autocli.go

Standard depinject pattern. Notable: register TWO providers:

```go
func init() {
    appmodule.Register(
        &modulev1.Module{},
        appmodule.Provide(ProvideModule),
        appmodule.Provide(ProvideCorporationKeeper),  // satisfies gftypes.CorporationKeeper
    )
}

// ProvideCorporationKeeper wraps the MOD-CO keeper as gftypes.CorporationKeeper
// so MOD-GF can autowire it. This REPLACES the StubCorporationKeeper that
// MOD-GF was providing during the foundation phase.
func ProvideCorporationKeeper(k keeper.Keeper) gftypes.CorporationKeeper {
    return k
}
```

The provider just returns the keeper itself (because MOD-CO keeper already satisfies the interface via Task 10's methods).

### Task 15: REMOVE MOD-GF stubs

In `x/gf/module/module.go`:
- Delete `appmodule.Provide(ProvideCorporationKeeper)` line from init()
- Delete the `ProvideCorporationKeeper` function

In `x/gf/keeper/adapters.go`:
- Delete `StubCorporationKeeper` struct
- Delete `NewStubCorporationKeeper` function
- Keep `TRAsEcosystemKeeper` (still needed until #305)

In `x/gf/types/expected_keepers.go`:
- Update the comment on `CorporationKeeper` to drop "Until issue #303 lands, a stub..." caveat

### Task 16: ADD public methods to MOD-GF keeper

In `x/gf/keeper/keeper.go` (or a new `x/gf/keeper/corp_integration.go`):

```go
// CreateInitialGFVersionForCorporation seeds v1 GFV+GFD for a freshly-created
// corporation. Called by MOD-CO-MSG-1 step 3. Per spec MOD-CO-MSG-1-3 (v4-rc2):
//
//   gfv.id              = auto-incremented
//   gfv.ecosystem_id    = 0
//   gfv.corporation_id  = corpID
//   gfv.created         = block_time
//   gfv.version         = 1
//   gfv.active_since    = block_time
//
//   gfd.id              = auto-incremented
//   gfd.gfv_id          = gfv.id
//   gfd.created         = block_time
//   gfd.language, url, digest_sri = caller-provided
//
// The (corpID, version=1) entry is also registered in GFVersionByCorporation
// secondary index.
func (k Keeper) CreateInitialGFVersionForCorporation(ctx context.Context, corpID uint64, language, docURL, docDigestSRI string) error {
    sdkCtx := sdk.UnwrapSDKContext(ctx)
    now := sdkCtx.BlockTime()

    gfvID, err := k.GetNextID(sdkCtx, "gfv")
    if err != nil {
        return err
    }
    gfv := types.GovernanceFrameworkVersion{
        Id:            gfvID,
        CorporationId: corpID,
        Created:       now,
        Version:       1,
        ActiveSince:   now,
    }
    if err := k.GFVersion.Set(ctx, gfv.Id, gfv); err != nil {
        return err
    }
    if err := k.GFVersionByCorporation.Set(ctx, collections.Join(corpID, int32(1)), gfv.Id); err != nil {
        return err
    }

    gfdID, err := k.GetNextID(sdkCtx, "gfd")
    if err != nil {
        return err
    }
    gfd := types.GovernanceFrameworkDocument{
        Id:        gfdID,
        GfvId:     gfv.Id,
        Created:   now,
        Language:  language,
        Url:       docURL,
        DigestSri: docDigestSRI,
    }
    return k.GFDocument.Set(ctx, gfd.Id, gfd)
}

// ListVersionsByCorporation returns GFV+GFD entries for a corporation, with
// the same filter semantics as MOD-GF-QRY-2. Caller passes the corp's current
// active_version so this method doesn't need to call back into the
// CorporationKeeper (avoids cycle at call time).
func (k Keeper) ListVersionsByCorporation(ctx context.Context, corpID uint64, activeVersion int32, activeOnly bool, preferredLang string) ([]types.GovernanceFrameworkVersionWithDocs, error) {
    iter, err := k.GFVersionByCorporation.Iterate(ctx, collections.NewPrefixedPairRange[uint64, int32](corpID))
    if err != nil {
        return nil, err
    }
    defer iter.Close()

    var out []types.GovernanceFrameworkVersionWithDocs
    for ; iter.Valid(); iter.Next() {
        id, err := iter.Value()
        if err != nil {
            return nil, err
        }
        gfv, err := k.GFVersion.Get(ctx, id)
        if err != nil {
            return nil, err
        }
        if activeOnly && gfv.Version != activeVersion {
            continue
        }
        docs, err := k.collectDocsForGFV(ctx, gfv.Id, preferredLang)
        if err != nil {
            return nil, err
        }
        out = append(out, types.GovernanceFrameworkVersionWithDocs{
            Id:            gfv.Id,
            CorporationId: gfv.CorporationId,
            Created:       gfv.Created,
            Version:       gfv.Version,
            ActiveSince:   gfv.ActiveSince,
            Documents:     docs,
        })
    }
    sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
    return out, nil
}
```

The `collectDocsForGFV` helper is extracted from the existing `query_gfv.go collectDocs` so both the query layer and this method share the same impl.

### Task 17: app.go + app_config.go wiring

`app/app.go`:
- Add side-effect import: `_ "github.com/verana-labs/verana/x/co/module"`
- Add keeper alias: `comodulekeeper "github.com/verana-labs/verana/x/co/keeper"`
- Add struct field: `CoKeeper comodulekeeper.Keeper` (placed near GfKeeper)
- Add to depinject.Inject list: `&app.CoKeeper,`

`app/app_config.go`:
- Add side-effect import: `_ "github.com/verana-labs/verana/x/co/module"`
- Add type import: `comoduletypes "github.com/verana-labs/verana/x/co/types"`
- Add `comoduletypes.ModuleName` to `genesisModuleOrder`, `beginBlockers`, `endBlockers`
- Add module config entry mirroring the others

For depinject auto-wiring: MOD-GF's ModuleInputs declares `CorporationKeeper gftypes.CorporationKeeper`. MOD-CO's `ProvideCorporationKeeper(k keeper.Keeper) gftypes.CorporationKeeper` is the supplier. depinject resolves the chain: MOD-CO keeper built first → wrapped as gftypes.CorporationKeeper → fed into MOD-GF's ModuleInputs.

Similarly MOD-CO declares `GFKeeper cotypes.GFKeeper`. MOD-GF's keeper directly satisfies that interface (Task 16). For depinject, we need a provider in `x/gf/module/module.go`:

```go
// ProvideGFKeeper supplies the MOD-GF keeper as the GFKeeper interface needed
// by MOD-CO and (later) MOD-EC.
func ProvideGFKeeper(k keeper.Keeper) cotypes.GFKeeper {
    return k
}
```

And register it in MOD-GF's `init()`:
```go
appmodule.Provide(ProvideGFKeeper),
```

This creates a depinject cycle at the **interface** level (MOD-CO needs gftypes.CorporationKeeper, MOD-GF needs cotypes.GFKeeper) but **not** at the keeper-construction level (each keeper is constructed once, then wrapped as the other's expected interface). depinject handles this kind of mutual dependency cleanly because the providers return interfaces, not concrete types — the actual construction order is: x/co keeper → x/gf keeper → wrap both via providers → inject.

Actually wait — both keepers need EACH OTHER at construction time. That's a real cycle. Let me think.

**The actual issue:** MOD-CO's NewKeeper takes `gfKeeper types.GFKeeper`. MOD-GF's NewKeeper takes `corporationKeeper types.CorporationKeeper`. Neither can be constructed before the other.

**Solution:** Use a **lazy keeper** pattern. One of them gets a "deferred" reference set after both are constructed. Cleanest: MOD-CO holds a setter for GFKeeper post-construction (and same for MOD-GF holding a setter for CorporationKeeper).

But this is exactly the post-build-setter pattern I rejected in MOD-GF's audit because it's brittle. Need to reconsider.

**Alternative solution**: Use a shim — a type that holds a pointer-to-pointer, fillable by depinject AFTER both keepers exist. Cosmos SDK uses this for things like the IBC keeper.

Looking at how x/perm handles its dependency on x/tr + x/td + x/cs + x/de... they all use depinject inputs and there are no cycles because the dependency graph is a DAG.

**Real solution**: break the cycle by making one direction lazy. MOD-CO's GFKeeper field is set via depinject directly (no cycle here — MOD-GF keeper is built first, then MOD-CO needs it). For MOD-GF's CorporationKeeper, we either:

a) Use the post-build setter (which I rejected before)
b) Have MOD-GF accept a "lazy" wrapper that resolves at call time, like `func() CorporationKeeper`

Looking at this more carefully: MOD-CO is built AFTER MOD-GF (MOD-GF is sequence 1, MOD-CO is sequence 2). So MOD-CO needs GFKeeper at construction → MOD-GF must already exist. ✓ That's a DAG.

But MOD-GF's NewKeeper takes a CorporationKeeper. We currently pass StubCorporationKeeper. If we want to swap in MOD-CO's keeper, MOD-GF needs to be constructed AFTER MOD-CO — but then MOD-CO needs GFKeeper at construction, which depends on MOD-GF.

**That's the cycle.** Real and unavoidable with the current design.

**Pragmatic fix**: Use a post-construction setter on MOD-GF for the CorporationKeeper. This is what I'd been avoiding but it's the cleanest cycle-breaker. Apply only to the CorporationKeeper field (not for the others, which are DAG-safe).

So the wiring becomes:
1. MOD-GF keeper built with EcosystemKeeper + DelegationKeeper + CorporationKeeper=nil (or a placeholder)
2. MOD-CO keeper built with GFKeeper = MOD-GF keeper
3. Post-build hook: `gfKeeper.SetCorporationKeeper(coKeeper)`

The post-build hook can live in app.go after `appBuilder.Build(...)` returns. This is exactly the pattern I rejected for MOD-GF's interim adapter — but here it's unavoidable due to the cycle, so we accept it.

Need to add `SetCorporationKeeper` method to MOD-GF's keeper. Make it a pointer-receiver to avoid silent copy bugs.

Let me note this as an architectural decision in the plan:

**Architectural decision (Task 17) — cleaner cycle-break:**

Re-analyzed: MOD-GF is built FIRST (sequence 1), MOD-CO is built SECOND (sequence 2). So MOD-CO can take `gfKeeper` as a normal depinject input — no cycle in that direction. The cycle is only MOD-GF → MOD-CO. Resolution:

- **MOD-CO's NewKeeper** takes `gfKeeper` as a normal depinject input (clean DAG).
- **MOD-GF's NewKeeper signature CHANGES**: drop the `corporationKeeper` parameter entirely (since MOD-CO doesn't exist yet at MOD-GF construction time). Add a pointer-receiver setter:
  ```go
  func (k *Keeper) SetCorporationKeeper(ck types.CorporationKeeper) {
      k.corporationKeeper = ck
  }
  ```
- **app.go post-build**: after `depinject.Inject(...)` returns, call `app.GfKeeper.SetCorporationKeeper(app.CoKeeper)`. Go auto-takes the address of the addressable struct field.
- **MOD-GF's call sites** that use `corporationKeeper` (resolveSubject, query_gfv.go) gain a defensive nil-check returning ErrSubjectNotFound if the setter was forgotten.
- **MOD-GF's ModuleInputs** drops the `CorporationKeeper` field — depinject doesn't need to resolve it.
- **MOD-GF's testutil** (testutil/keeper/gf.go) constructor accepts a CorporationKeeper arg and calls SetCorporationKeeper internally — existing tests keep working unchanged.
- **MOD-GF's existing tests** use mockCorporation through the testutil and continue to work without modification.

This eliminates the dual-provider depinject conflict cleanly. MOD-CO is the sole provider of CorporationKeeper (via its own depinject Provide if needed for other modules, or just via the setter call directly).

Actually wait, let me reconsider. There's a simpler way: don't have MOD-CO satisfy gftypes.CorporationKeeper directly. Instead, have an adapter package or a third "wiring" package that gets both keepers and exposes the right interface to each.

Or simplest: depinject providers can use depinject.LazyAware. Let me check.

Actually depinject doesn't have lazy. The simplest pattern is the setter. Let me go with the setter approach and document it clearly.

I'll proceed with the setter.

---

## Phase 5 — Tests

### Task 18-24: Match MOD-GF test coverage discipline

Mirror the MOD-GF test set exactly: every handler has a positive + failure-mode test for every spec precondition, every helper has direct unit tests, every error path is exercised. Target: 80%+ keeper coverage, 100% of hand-written non-defensive code.

Specific test scenarios required by spec:

**MsgCreateCorporation:**
- Happy path: returns id + policy_address, persists Corporation, both reverse indices populated, gfKeeper.CreateInitialGFVersionForCorporation called with correct args
- Duplicate DID → ErrDIDAlreadyExists
- x/group MsgCreateGroupWithPolicy failure → propagated
- gfKeeper.CreateInitialGFVersionForCorporation failure → propagated (transaction rolls back — verify Corporation is NOT persisted in this case)
- ValidateBasic failures: empty signer, no members, member with zero weight, member with bad address, nil decision_policy, empty did, invalid BCP47 language, invalid URL, invalid digest_sri

**MsgUpdateCorporation:**
- Happy path: did rotated, modified updated, CorporationByDID index swapped
- Rotating to SAME value: no-op, allowed
- Rotating to DID already held by another Corporation → ErrDIDAlreadyExists
- AUTHZ-CHECK-1 failure → propagated
- Signing account not registered as a Corporation → ErrCorporationNotRegistered

**GetCorporation query:**
- Returns nested GF data via gfKeeper.ListVersionsByCorporation
- active_only filter passes through
- preferred_language filter passes through
- Unknown id → NotFound

**ListCorporations query:**
- modified_after filter
- response_max_size cap (1-1024, default 64)
- Empty result

**CorporationKeeper interface impl:**
- ResolveByPolicyAddress: known + unknown
- GetByID: known + unknown
- SetActiveVersion: happy + unknown id

**Genesis round-trip:** Import + export preserves Corporations + reverse indices + counter

### Task 25: app smoke test (`make build && veranad init`)

Expect clean genesis with `co` module section.

---

## Phase 6 — Verification + Draft PR

### Task 26: Full repo regression

`go build ./...` + `go test ./... -count=1` — clean.

### Task 27: Push + draft PR

```
git push -u origin feat/mod-co-module
gh pr create --draft --repo verana-labs/verana \
  --title "feat(co)!: implement Corporation module (MOD-CO) per spec v4-rc2" \
  --body "..."  # Closes #303
```

Marked DRAFT because stacked on top of #318 (which is still in review). Once #318 merges, rebase onto main and mark ready for review.

---

## Risk register

| Risk | Mitigation |
|---|---|
| Cyclic MOD-CO ↔ MOD-GF keeper dependency at construction time | Post-build setter on MOD-GF.SetCorporationKeeper, called in app.go after appBuilder.Build returns. Documented in adapters.go. |
| MOD-GF handlers panic on nil CorporationKeeper if app.go forgets the setter | Defensive nil-check in `resolveSubject` and the new public methods; returns ErrSubjectNotFound with clear message |
| x/group MsgCreateGroupWithPolicy not directly callable via keeper | Use the x/group MsgServer or the Msg-service router. Verify pattern from existing x/cs or x/td calls. |
| Proto cross-package import `verana.gf.v1.GovernanceFrameworkVersionWithDocs` in MOD-CO's query response | Defined locally in co.v1 if buf complains. Worst case, generate Go struct directly without proto-level nesting. |
| PR #318 changes in review affect MOD-CO | Rebase aggressively. Stacked PRs require rebases on each upstream change. |
| Rebasing onto main after #318 merges | Standard `git rebase --onto origin/main feat/mod-gf-module` flow; document in PR description |

---

## Self-review checklist (run before draft PR)

- [ ] `make proto-gen` succeeds; new `.pb.go` files include `co.v1.*`
- [ ] `go build ./...` clean
- [ ] `go test ./... -count=1` clean, no regressions across any module
- [ ] `make build && veranad init` produces valid genesis with `co` section
- [ ] MOD-GF tests still pass (no broken stub references)
- [ ] App.go has the post-build setter call: `app.GfKeeper.SetCorporationKeeper(app.CoKeeper)`
- [ ] No `Co-Authored-By` lines in any commit
- [ ] Plan doc kept local (NOT staged for commit)
- [ ] PR opened as DRAFT
- [ ] PR description explicitly states "depends on #318; will rebase after #318 merges"
