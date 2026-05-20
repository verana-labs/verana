# MOD-GF Governance Framework Module — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create a new `x/gf` Cosmos SDK module that owns `GovernanceFrameworkVersion` (GFV) and `GovernanceFrameworkDocument` (GFD) storage and exposes the four spec methods (MOD-GF-MSG-1, MSG-2, QRY-1, QRY-2). MOD-GF is polymorphic over an `Ecosystem` OR a `Corporation` as the GF owner.

**Architecture:** Mirror the existing `x/tr` module structure since it already implements equivalent GFV/GFD logic. The new module accepts `EcosystemKeeper` and `CorporationKeeper` interfaces for subject-existence validation (per MOD-GF-MSG-1-2-1 / MSG-2-2-1) and a `DelegationKeeper` for AUTHZ-CHECK-1. **All three keepers are required at construction time** (no nil, no setters) — this avoids the brittleness of post-build wiring. Until issue #305 lands (TR→EC rename), an adapter inside `x/gf/keeper` lets the existing `x/tr` keeper serve as the `EcosystemKeeper`. Until issue #303 lands (MOD-CO), a stub `CorporationKeeper` returns "not found" for all calls (causing corporation-targeted GF calls to abort cleanly with `ErrSubjectNotFound`). Both replacements live in `x/gf/keeper/` (NOT in app/), so the dependency direction is GF → TR (temporary, removed in #305) rather than app → GF + app → TR (which would create awkward wiring).

**Tech Stack:** Go 1.23.8, Cosmos SDK v0.50.x with `cosmossdk.io/collections`, gogo-proto, testify/require.

---

## Pre-flight

- [ ] **Branch.** Already on `feat/mod-gf-module` (created off `origin/main`). Confirm with `git branch --show-current`.

- [ ] **Sanity check baseline.**

  Run: `go build ./... && go vet ./...`
  Expected: exit 0.

- [ ] **Spec snapshot.** Re-read MOD-GF section of `/tmp/vpr-spec/spec.md` (lines 2455–2606) before starting. The spec is authoritative — every basic check listed there must have a corresponding validation in the keeper.

---

## File Structure

### Created files

```
proto/verana/gf/v1/types.proto         # GovernanceFrameworkVersion, GovernanceFrameworkDocument
proto/verana/gf/v1/params.proto        # Params
proto/verana/gf/v1/tx.proto            # MsgAddGovernanceFrameworkDocument, MsgIncreaseActiveGovernanceFrameworkVersion, MsgUpdateParams
proto/verana/gf/v1/query.proto         # GetGovernanceFrameworkVersion, ListGovernanceFrameworkVersions, Params
proto/verana/gf/v1/genesis.proto       # GenesisState
proto/verana/gf/module/v1/module.proto # Module config

x/gf/types/keys.go            # ModuleName, store prefixes
x/gf/types/codec.go           # Amino + interface registration
x/gf/types/errors.go          # Module-specific errors
x/gf/types/events.go          # Event type / attribute constants
x/gf/types/expected_keepers.go # DelegationKeeper, EcosystemKeeper, CorporationKeeper interfaces
x/gf/types/genesis.go         # GenesisState validation
x/gf/types/params.go          # DefaultParams, Validate
x/gf/types/types.go           # IsValidBCP47, IsValidDigestSRI helpers (copied from x/tr/types)
x/gf/types/msgs.go            # ValidateBasic for each msg

x/gf/keeper/keeper.go         # Keeper struct, NewKeeper, collections, counter helper
x/gf/keeper/genesis.go        # InitGenesis, ExportGenesis
x/gf/keeper/params.go         # GetParams, SetParams
x/gf/keeper/msg_server.go     # msgServer{} router
x/gf/keeper/msg_update_params.go    # UpdateParams handler
x/gf/keeper/msg_add_gf_document.go  # MOD-GF-MSG-1 (validation + execution)
x/gf/keeper/msg_increase_active_gfv.go # MOD-GF-MSG-2 (validation + execution)
x/gf/keeper/query.go          # querier{} struct
x/gf/keeper/query_params.go   # Params query
x/gf/keeper/query_gfv.go      # GetGovernanceFrameworkVersion, ListGovernanceFrameworkVersions

x/gf/keeper/adapters.go       # Interim TR-as-Ecosystem adapter + stub CorporationKeeper (removed/replaced in #303/#305)

x/gf/module/module.go         # AppModule, AppModuleBasic, depinject In/Out
x/gf/module/autocli.go        # AutoCLI bindings
x/gf/module/genesis.go        # InitGenesis/ExportGenesis wrappers

x/gf/keeper/msg_add_gf_document_test.go
x/gf/keeper/msg_increase_active_gfv_test.go
x/gf/keeper/query_gfv_test.go
x/gf/keeper/keeper_test.go
x/gf/types/genesis_test.go
x/gf/types/params_test.go

testutil/keeper/gf.go        # Test keeper constructor
```

### Modified files

```
app/app.go         # Wire x/gf module + keeper into the app
app/app_config.go  # Register x/gf in module manager + permissions
```

### Files NOT touched in this PR

- `x/tr/keeper/gfv.go`, `x/tr/keeper/gfd.go` — these stay in place. They are deleted in issue #305 (TR→EC rename), which migrates EC's GF creation to delegate to MOD-GF.
- No existing module's behavior changes; MOD-GF is purely additive.

---

## Phase 1 — Proto Schema

## Task 1: Create proto/verana/gf/v1/types.proto

**File:** `proto/verana/gf/v1/types.proto`

- [ ] **Step 1.1: Write the file.**

```protobuf
syntax = "proto3";
package verana.gf.v1;

import "amino/amino.proto";
import "cosmos_proto/cosmos.proto";
import "gogoproto/gogo.proto";
import "google/protobuf/timestamp.proto";

option go_package = "github.com/verana-labs/verana/x/gf/types";

// GovernanceFrameworkVersion represents a versioned governance framework owned
// by either an Ecosystem (ecosystem_id set, corporation empty) or a Corporation
// (corporation set, ecosystem_id zero). Exactly one of the two owner fields is set.
message GovernanceFrameworkVersion {
  uint64 id = 1;
  uint64 ecosystem_id = 2;
  string corporation = 3 [(cosmos_proto.scalar) = "cosmos.AddressString"];
  google.protobuf.Timestamp created = 4 [
    (gogoproto.stdtime) = true,
    (gogoproto.nullable) = false,
    (amino.dont_omitempty) = true
  ];
  int32 version = 5;
  google.protobuf.Timestamp active_since = 6 [
    (gogoproto.stdtime) = true,
    (gogoproto.nullable) = false,
    (amino.dont_omitempty) = true
  ];
}

// GovernanceFrameworkDocument is a single (language, url, digest_sri) tuple
// attached to a GovernanceFrameworkVersion.
message GovernanceFrameworkDocument {
  uint64 id = 1;
  uint64 gfv_id = 2;
  google.protobuf.Timestamp created = 3 [
    (gogoproto.stdtime) = true,
    (gogoproto.nullable) = false,
    (amino.dont_omitempty) = true
  ];
  string language = 4;
  string url = 5;
  string digest_sri = 6;
}

// GovernanceFrameworkVersionWithDocs is the response shape for the query layer,
// returning a version with its nested documents.
message GovernanceFrameworkVersionWithDocs {
  uint64 id = 1;
  uint64 ecosystem_id = 2;
  string corporation = 3 [(cosmos_proto.scalar) = "cosmos.AddressString"];
  google.protobuf.Timestamp created = 4 [
    (gogoproto.stdtime) = true,
    (gogoproto.nullable) = false,
    (amino.dont_omitempty) = true
  ];
  int32 version = 5;
  google.protobuf.Timestamp active_since = 6 [
    (gogoproto.stdtime) = true,
    (gogoproto.nullable) = false,
    (amino.dont_omitempty) = true
  ];
  repeated GovernanceFrameworkDocument documents = 7 [
    (gogoproto.nullable) = false,
    (amino.dont_omitempty) = true
  ];
}
```

- [ ] **Step 1.2: Commit.**

```bash
git add proto/verana/gf/v1/types.proto
git commit -m "feat(gf): add types.proto for GFV/GFD entities"
```

---

## Task 2: Create proto/verana/gf/v1/params.proto

**File:** `proto/verana/gf/v1/params.proto`

- [ ] **Step 2.1: Write the file.**

```protobuf
syntax = "proto3";
package verana.gf.v1;

import "amino/amino.proto";
import "gogoproto/gogo.proto";

option go_package = "github.com/verana-labs/verana/x/gf/types";

// Params defines the parameters for the x/gf module.
// The module has no parameters at this time; the message is kept for forward compat
// so that gov-proposed parameter changes can land without proto migrations.
message Params {
  option (amino.name) = "verana/x/gf/Params";
  option (gogoproto.equal) = true;
}
```

- [ ] **Step 2.2: Commit.**

```bash
git add proto/verana/gf/v1/params.proto
git commit -m "feat(gf): add params.proto (empty params, forward-compat)"
```

---

## Task 3: Create proto/verana/gf/v1/tx.proto

**File:** `proto/verana/gf/v1/tx.proto`

- [ ] **Step 3.1: Write the file.**

```protobuf
syntax = "proto3";
package verana.gf.v1;

import "amino/amino.proto";
import "cosmos/msg/v1/msg.proto";
import "cosmos_proto/cosmos.proto";
import "gogoproto/gogo.proto";
import "verana/gf/v1/params.proto";

option go_package = "github.com/verana-labs/verana/x/gf/types";

// Msg defines the Msg service.
service Msg {
  option (cosmos.msg.v1.service) = true;

  // UpdateParams defines a (governance) operation for updating the module
  // parameters. The authority defaults to the x/gov module account.
  rpc UpdateParams(MsgUpdateParams) returns (MsgUpdateParamsResponse);

  // [MOD-GF-MSG-1] Add Governance Framework Document
  rpc AddGovernanceFrameworkDocument(MsgAddGovernanceFrameworkDocument) returns (MsgAddGovernanceFrameworkDocumentResponse);

  // [MOD-GF-MSG-2] Increase Active Governance Framework Version
  rpc IncreaseActiveGovernanceFrameworkVersion(MsgIncreaseActiveGovernanceFrameworkVersion) returns (MsgIncreaseActiveGovernanceFrameworkVersionResponse);
}

// MsgUpdateParams is the Msg/UpdateParams request type.
message MsgUpdateParams {
  option (cosmos.msg.v1.signer) = "authority";
  option (amino.name) = "verana/x/gf/MsgUpdateParams";

  // authority is the address that controls the module (defaults to x/gov).
  string authority = 1 [(cosmos_proto.scalar) = "cosmos.AddressString"];

  // params defines the module parameters to update.
  // NOTE: All parameters must be supplied.
  Params params = 2 [
    (gogoproto.nullable) = false,
    (amino.dont_omitempty) = true
  ];
}

message MsgUpdateParamsResponse {}

// [MOD-GF-MSG-1] Adds (or replaces) a GovernanceFrameworkDocument for a draft
// version owned by an Ecosystem or a Corporation.
message MsgAddGovernanceFrameworkDocument {
  option (cosmos.msg.v1.signer) = "corporation";
  option (cosmos.msg.v1.signer) = "operator";
  option (amino.name) = "verana/x/gf/MsgAddGovernanceFrameworkDocument";

  // corporation is the signing corporation (group_policy_address).
  string corporation = 1 [(cosmos_proto.scalar) = "cosmos.AddressString"];
  // operator is the account authorized by corporation to run this Msg.
  string operator = 2 [(cosmos_proto.scalar) = "cosmos.AddressString"];
  // ecosystem_id is optional. If set, the GF target is an Ecosystem owned by
  // `corporation`. If zero, the GF target is the signing Corporation's own CGF.
  uint64 ecosystem_id = 3;
  // doc_language is a BCP 47 language tag.
  string doc_language = 4;
  // doc_url is the URL where the document is published.
  string doc_url = 5;
  // doc_digest_sri is the SRI digest of the document.
  string doc_digest_sri = 6;
  // version is the target governance framework version.
  int32 version = 7;
}

message MsgAddGovernanceFrameworkDocumentResponse {}

// [MOD-GF-MSG-2] Activates the next governance framework version for an
// Ecosystem or Corporation owned by the signing corporation.
message MsgIncreaseActiveGovernanceFrameworkVersion {
  option (cosmos.msg.v1.signer) = "corporation";
  option (cosmos.msg.v1.signer) = "operator";
  option (amino.name) = "verana/x/gf/MsgIncreaseActiveGovernanceFrameworkVersion";

  // corporation is the signing corporation (group_policy_address).
  string corporation = 1 [(cosmos_proto.scalar) = "cosmos.AddressString"];
  // operator is the account authorized by corporation to run this Msg.
  string operator = 2 [(cosmos_proto.scalar) = "cosmos.AddressString"];
  // ecosystem_id is optional. If set, target is the Ecosystem; otherwise the
  // signing Corporation's own CGF.
  uint64 ecosystem_id = 3;
}

message MsgIncreaseActiveGovernanceFrameworkVersionResponse {}
```

- [ ] **Step 3.2: Commit.**

```bash
git add proto/verana/gf/v1/tx.proto
git commit -m "feat(gf): add tx.proto with MSG-1, MSG-2 and UpdateParams"
```

---

## Task 4: Create proto/verana/gf/v1/query.proto and genesis.proto

**Files:** `proto/verana/gf/v1/query.proto`, `proto/verana/gf/v1/genesis.proto`

- [ ] **Step 4.1: Write query.proto.**

```protobuf
syntax = "proto3";
package verana.gf.v1;

import "amino/amino.proto";
import "cosmos_proto/cosmos.proto";
import "gogoproto/gogo.proto";
import "google/api/annotations.proto";
import "verana/gf/v1/params.proto";
import "verana/gf/v1/types.proto";

option go_package = "github.com/verana-labs/verana/x/gf/types";

// Query defines the Query service.
service Query {
  // Params returns the total set of module parameters.
  rpc Params(QueryParamsRequest) returns (QueryParamsResponse) {
    option (google.api.http).get = "/verana/gf/v1/params";
  }

  // [MOD-GF-QRY-1] Get Governance Framework Version
  rpc GetGovernanceFrameworkVersion(QueryGetGovernanceFrameworkVersionRequest) returns (QueryGetGovernanceFrameworkVersionResponse) {
    option (google.api.http).get = "/gf/v1/get";
  }

  // [MOD-GF-QRY-2] List Governance Framework Versions
  rpc ListGovernanceFrameworkVersions(QueryListGovernanceFrameworkVersionsRequest) returns (QueryListGovernanceFrameworkVersionsResponse) {
    option (google.api.http).get = "/gf/v1/list";
  }
}

message QueryParamsRequest {}

message QueryParamsResponse {
  Params params = 1 [(gogoproto.nullable) = false, (amino.dont_omitempty) = true];
}

message QueryGetGovernanceFrameworkVersionRequest {
  uint64 id = 1;
  string preferred_language = 2;
}

message QueryGetGovernanceFrameworkVersionResponse {
  GovernanceFrameworkVersionWithDocs version = 1 [(gogoproto.nullable) = false, (amino.dont_omitempty) = true];
}

message QueryListGovernanceFrameworkVersionsRequest {
  uint64 ecosystem_id = 1;
  string corporation = 2 [(cosmos_proto.scalar) = "cosmos.AddressString"];
  bool active_only = 3;
  string preferred_language = 4;
  uint32 response_max_size = 5;
}

message QueryListGovernanceFrameworkVersionsResponse {
  repeated GovernanceFrameworkVersionWithDocs versions = 1 [(gogoproto.nullable) = false, (amino.dont_omitempty) = true];
}
```

- [ ] **Step 4.2: Write genesis.proto.**

```protobuf
syntax = "proto3";
package verana.gf.v1;

import "amino/amino.proto";
import "gogoproto/gogo.proto";
import "verana/gf/v1/params.proto";
import "verana/gf/v1/types.proto";

option go_package = "github.com/verana-labs/verana/x/gf/types";

// GenesisState defines the gf module's genesis state.
message GenesisState {
  Params params = 1 [(gogoproto.nullable) = false, (amino.dont_omitempty) = true];
  repeated GovernanceFrameworkVersion versions = 2 [(gogoproto.nullable) = false, (amino.dont_omitempty) = true];
  repeated GovernanceFrameworkDocument documents = 3 [(gogoproto.nullable) = false, (amino.dont_omitempty) = true];
}
```

- [ ] **Step 4.3: Commit.**

```bash
git add proto/verana/gf/v1/query.proto proto/verana/gf/v1/genesis.proto
git commit -m "feat(gf): add query.proto and genesis.proto"
```

---

## Task 5: Create proto/verana/gf/module/v1/module.proto and regenerate proto code

**File:** `proto/verana/gf/module/v1/module.proto`

- [ ] **Step 5.1: Write module.proto. Use `proto/verana/tr/module/v1/module.proto` as the template — replace `tr` with `gf` and update the `go_import`.**

Open `proto/verana/tr/module/v1/module.proto`, copy contents to the new path, then in the new file replace:
- `verana.tr.module.v1` → `verana.gf.module.v1`
- `go_import = "tr/module"` → `go_import = "gf/module"`

- [ ] **Step 5.2: Regenerate proto code.**

Run: `make proto-gen`
Expected: no errors. Creates `x/gf/types/*.pb.go`, `x/gf/module/module.pb.go`, ts-proto outputs.

- [ ] **Step 5.3: Verify generation.**

Run: `ls x/gf/types/*.pb.go`
Expected: `types.pb.go`, `params.pb.go`, `tx.pb.go`, `query.pb.go`, `query.pb.gw.go`, `genesis.pb.go`.

- [ ] **Step 5.4: Commit.**

```bash
git add proto/verana/gf/module/v1/module.proto x/gf/types/*.pb.go x/gf/types/*.pb.gw.go x/gf/module/module.pb.go
git commit -m "feat(gf): add module.proto and regen pb.go for module"
```

---

## Phase 2 — Types Package

## Task 6: Create x/gf/types/keys.go

**File:** `x/gf/types/keys.go`

- [ ] **Step 6.1: Write the file.**

```go
package types

import "cosmossdk.io/collections"

const (
	// ModuleName defines the module name.
	ModuleName = "gf"

	// StoreKey defines the primary module store key.
	StoreKey = ModuleName

	// GovModuleName duplicates the x/gov module name to avoid a dependency.
	// MUST be synced if upstream renames it.
	GovModuleName = "gov"
)

var (
	ParamsKey                      = collections.NewPrefix(1)
	GovernanceFrameworkVersionKey  = collections.NewPrefix(2)
	GovernanceFrameworkDocumentKey = collections.NewPrefix(3)
	CounterKey                     = collections.NewPrefix(4)
	// Secondary indexes for O(1) lookups by (owner, version).
	GFVersionByEcosystemKey   = collections.NewPrefix(5) // (ecosystem_id, version) -> gfv_id
	GFVersionByCorporationKey = collections.NewPrefix(6) // (corporation, version)  -> gfv_id
)
```

- [ ] **Step 6.2: Commit.**

```bash
git add x/gf/types/keys.go
git commit -m "feat(gf): add types/keys.go"
```

---

## Task 7: Create x/gf/types/errors.go, events.go, params.go

**Files:** `x/gf/types/errors.go`, `x/gf/types/events.go`, `x/gf/types/params.go`

- [ ] **Step 7.1: Write errors.go.**

```go
package types

import "cosmossdk.io/errors"

var (
	ErrInvalidSigner = errors.Register(ModuleName, 1100, "expected gov account as only signer")

	ErrInvalidSubject       = errors.Register(ModuleName, 1101, "invalid GF subject: must be either ecosystem_id or corporation")
	ErrSubjectNotFound      = errors.Register(ModuleName, 1102, "GF subject not found")
	ErrSubjectNotControlled = errors.Register(ModuleName, 1103, "signing corporation is not the controller of the target subject")
	ErrInvalidVersion       = errors.Register(ModuleName, 1104, "invalid governance framework version")
	ErrInvalidLanguage      = errors.Register(ModuleName, 1105, "invalid BCP 47 language tag")
	ErrInvalidURL           = errors.Register(ModuleName, 1106, "invalid URL")
	ErrInvalidDigestSRI     = errors.Register(ModuleName, 1107, "invalid digest_sri")
	ErrNoActivatableVersion = errors.Register(ModuleName, 1108, "no governance framework version available to activate")
	ErrMissingDefaultLang   = errors.Register(ModuleName, 1109, "no document found for the default language of this version")
)
```

- [ ] **Step 7.2: Write events.go.**

```go
package types

const (
	EventTypeAddGFDocument    = "add_gf_document"
	EventTypeIncreaseGFActive = "increase_active_gf_version"

	AttributeKeyCorporation = "corporation"
	AttributeKeyEcosystemID = "ecosystem_id"
	AttributeKeyGFVersionID = "gfv_id"
	AttributeKeyGFDocID     = "gfd_id"
	AttributeKeyVersion     = "version"
	AttributeKeyLanguage    = "language"
)
```

- [ ] **Step 7.3: Write params.go.**

```go
package types

func NewParams() Params {
	return Params{}
}

func DefaultParams() Params {
	return NewParams()
}

// Validate returns nil; the module has no params today.
func (p Params) Validate() error { return nil }
```

- [ ] **Step 7.4: Commit.**

```bash
git add x/gf/types/errors.go x/gf/types/events.go x/gf/types/params.go
git commit -m "feat(gf): add errors, events, and params types"
```

---

## Task 8: Create x/gf/types/types.go and msgs.go

**Files:** `x/gf/types/types.go`, `x/gf/types/msgs.go`

- [ ] **Step 8.1: Copy BCP 47 + URL + digest helpers from `x/tr/types/types.go` into `x/gf/types/types.go`. The implementation is identical — the helpers don't depend on TR specifics.**

```go
package types

import (
	"net/url"
	"regexp"
	"strings"
)

// IsValidBCP47 returns true if s looks like a valid BCP 47 language tag.
// (Mirrors x/tr/types.IsValidBCP47.)
func IsValidBCP47(s string) bool {
	if s == "" {
		return false
	}
	if len(s) > 17 {
		return false
	}
	// Simple per-subtag check: alpha-only or alphanum, 1-8 chars, separated by '-'.
	for i, part := range strings.Split(s, "-") {
		if len(part) < 1 || len(part) > 8 {
			return false
		}
		if i == 0 {
			for _, r := range part {
				if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')) {
					return false
				}
			}
			continue
		}
		for _, r := range part {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
				return false
			}
		}
	}
	return true
}

func IsValidURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	return u.Scheme != "" && u.Host != ""
}

var digestSRIRe = regexp.MustCompile(`^(sha256|sha384|sha512)-[A-Za-z0-9+/=]+$`)

func IsValidDigestSRI(s string) bool { return digestSRIRe.MatchString(s) }
```

- [ ] **Step 8.2: Write msgs.go (ValidateBasic for the three Msg types).**

```go
package types

import (
	"cosmossdk.io/errors"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// ValidateBasic on MsgUpdateParams: authority must be present.
func (m *MsgUpdateParams) ValidateBasic() error {
	if m.Authority == "" {
		return errors.Wrap(sdkerrors.ErrInvalidAddress, "authority is required")
	}
	return m.Params.Validate()
}

// ValidateBasic on MsgAddGovernanceFrameworkDocument.
func (m *MsgAddGovernanceFrameworkDocument) ValidateBasic() error {
	if m.Corporation == "" {
		return errors.Wrap(sdkerrors.ErrInvalidAddress, "corporation is required")
	}
	if m.Operator == "" {
		return errors.Wrap(sdkerrors.ErrInvalidAddress, "operator is required")
	}
	if m.DocLanguage == "" {
		return errors.Wrap(ErrInvalidLanguage, "doc_language is required")
	}
	if !IsValidBCP47(m.DocLanguage) {
		return errors.Wrap(ErrInvalidLanguage, m.DocLanguage)
	}
	if m.DocUrl == "" {
		return errors.Wrap(ErrInvalidURL, "doc_url is required")
	}
	if !IsValidURL(m.DocUrl) {
		return errors.Wrap(ErrInvalidURL, m.DocUrl)
	}
	if m.DocDigestSri == "" {
		return errors.Wrap(ErrInvalidDigestSRI, "doc_digest_sri is required")
	}
	if !IsValidDigestSRI(m.DocDigestSri) {
		return errors.Wrap(ErrInvalidDigestSRI, m.DocDigestSri)
	}
	if m.Version < 1 {
		return errors.Wrap(ErrInvalidVersion, "version must be >= 1")
	}
	return nil
}

// ValidateBasic on MsgIncreaseActiveGovernanceFrameworkVersion.
func (m *MsgIncreaseActiveGovernanceFrameworkVersion) ValidateBasic() error {
	if m.Corporation == "" {
		return errors.Wrap(sdkerrors.ErrInvalidAddress, "corporation is required")
	}
	if m.Operator == "" {
		return errors.Wrap(sdkerrors.ErrInvalidAddress, "operator is required")
	}
	return nil
}
```

- [ ] **Step 8.3: Commit.**

```bash
git add x/gf/types/types.go x/gf/types/msgs.go
git commit -m "feat(gf): add validation helpers and ValidateBasic for Msgs"
```

---

## Task 9: Create x/gf/types/codec.go and genesis.go

**Files:** `x/gf/types/codec.go`, `x/gf/types/genesis.go`

- [ ] **Step 9.1: Write codec.go (interface + amino registration).**

```go
package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUpdateParams{},
		&MsgAddGovernanceFrameworkDocument{},
		&MsgIncreaseActiveGovernanceFrameworkVersion{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgUpdateParams{}, "verana/x/gf/MsgUpdateParams", nil)
	cdc.RegisterConcrete(&MsgAddGovernanceFrameworkDocument{}, "verana/x/gf/MsgAddGovernanceFrameworkDocument", nil)
	cdc.RegisterConcrete(&MsgIncreaseActiveGovernanceFrameworkVersion{}, "verana/x/gf/MsgIncreaseActiveGovernanceFrameworkVersion", nil)
}
```

- [ ] **Step 9.2: Write genesis.go.**

```go
package types

func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:    DefaultParams(),
		Versions:  nil,
		Documents: nil,
	}
}

// Validate performs basic genesis state validation.
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}
	// GFV: exactly one of ecosystem_id (>0) and corporation (non-empty) must be set.
	for _, gfv := range gs.Versions {
		hasEco := gfv.EcosystemId > 0
		hasCorp := gfv.Corporation != ""
		if hasEco == hasCorp {
			return ErrInvalidSubject
		}
		if gfv.Version < 1 {
			return ErrInvalidVersion
		}
	}
	// GFD: gfv_id must reference a GFV in this genesis.
	versionIDs := map[uint64]struct{}{}
	for _, gfv := range gs.Versions {
		versionIDs[gfv.Id] = struct{}{}
	}
	for _, gfd := range gs.Documents {
		if _, ok := versionIDs[gfd.GfvId]; !ok {
			return ErrInvalidVersion
		}
		if !IsValidBCP47(gfd.Language) {
			return ErrInvalidLanguage
		}
	}
	return nil
}
```

- [ ] **Step 9.3: Commit.**

```bash
git add x/gf/types/codec.go x/gf/types/genesis.go
git commit -m "feat(gf): add codec registration and genesis validation"
```

---

## Task 10: Create x/gf/types/expected_keepers.go

**File:** `x/gf/types/expected_keepers.go`

- [ ] **Step 10.1: Write the file.**

```go
package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DelegationKeeper is the minimum surface MOD-GF needs from x/de for AUTHZ-CHECK-1.
// Mirrors x/tr/types.DelegationKeeper.
type DelegationKeeper interface {
	CheckOperatorAuthorization(ctx sdk.Context, corporation string, operator string, msgType string) error
}

// EcosystemView is the read shape MOD-GF needs to validate ecosystem subjects.
// `Corporation` is the controlling corporation (group_policy_address).
// `Language` is the ecosystem's primary language.
// `ActiveVersion` is the ecosystem's current active GF version.
type EcosystemView struct {
	Id            uint64
	Corporation   string
	Language      string
	ActiveVersion int32
}

// EcosystemKeeper is the minimum surface MOD-GF needs for ecosystem-targeted GF ops.
// Until issue #305 (TR→EC rename) lands, the x/tr keeper provides this via an adapter.
type EcosystemKeeper interface {
	GetEcosystemView(ctx context.Context, ecosystemID uint64) (EcosystemView, bool)
	SetEcosystemActiveVersion(ctx context.Context, ecosystemID uint64, newVersion int32) error
}

// CorporationView is the read shape MOD-GF needs to validate corporation subjects.
// ActiveVersion uses int32 to match EcosystemView and the underlying spec
// (`active_version (int)`), preventing silent overflow on cast in resolveSubject.
type CorporationView struct {
	GroupPolicyAddress string
	Language           string
	ActiveVersion      int32
}

// CorporationKeeper is the minimum surface MOD-GF needs for corporation-targeted GF ops.
// Until issue #303 (MOD-CO) lands, a stub keeper returns (zero, false) for all calls.
// The implementation MUST also bump `corp.modified` when active_version is updated
// (per MOD-GF-MSG-2-3 step "Set subject.modified to current timestamp").
type CorporationKeeper interface {
	GetCorporationView(ctx context.Context, groupPolicyAddress string) (CorporationView, bool)
	SetCorporationActiveVersion(ctx context.Context, groupPolicyAddress string, newVersion int32) error
}
```

- [ ] **Step 10.2: Commit.**

```bash
git add x/gf/types/expected_keepers.go
git commit -m "feat(gf): declare DelegationKeeper, EcosystemKeeper, CorporationKeeper interfaces"
```

---

## Phase 3 — Keeper

## Task 11: Create x/gf/keeper/keeper.go

**File:** `x/gf/keeper/keeper.go`

- [ ] **Step 11.1: Write the file.**

```go
package keeper

import (
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana/x/gf/types"
)

type Keeper struct {
	cdc          codec.BinaryCodec
	storeService store.KVStoreService
	logger       log.Logger
	authority    string

	Schema                   collections.Schema
	Params                   collections.Item[types.Params]
	GFVersion                collections.Map[uint64, types.GovernanceFrameworkVersion]
	GFDocument               collections.Map[uint64, types.GovernanceFrameworkDocument]
	GFVersionByEcosystem     collections.Map[collections.Pair[uint64, int32], uint64]
	GFVersionByCorporation   collections.Map[collections.Pair[string, int32], uint64]
	Counter                  collections.Map[string, uint64]

	delegationKeeper  types.DelegationKeeper
	ecosystemKeeper   types.EcosystemKeeper
	corporationKeeper types.CorporationKeeper
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	logger log.Logger,
	authority string,
	delegationKeeper types.DelegationKeeper,
	ecosystemKeeper types.EcosystemKeeper,
	corporationKeeper types.CorporationKeeper,
) Keeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}

	sb := collections.NewSchemaBuilder(storeService)
	k := Keeper{
		cdc:          cdc,
		storeService: storeService,
		authority:    authority,
		logger:       logger,
		Params:       collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		GFVersion:    collections.NewMap(sb, types.GovernanceFrameworkVersionKey, "gf_version", collections.Uint64Key, codec.CollValue[types.GovernanceFrameworkVersion](cdc)),
		GFDocument:   collections.NewMap(sb, types.GovernanceFrameworkDocumentKey, "gf_document", collections.Uint64Key, codec.CollValue[types.GovernanceFrameworkDocument](cdc)),
		GFVersionByEcosystem:   collections.NewMap(sb, types.GFVersionByEcosystemKey, "gf_version_by_ecosystem", collections.PairKeyCodec(collections.Uint64Key, collections.Int32Key), collections.Uint64Value),
		GFVersionByCorporation: collections.NewMap(sb, types.GFVersionByCorporationKey, "gf_version_by_corporation", collections.PairKeyCodec(collections.StringKey, collections.Int32Key), collections.Uint64Value),
		Counter:                collections.NewMap(sb, types.CounterKey, "counter", collections.StringKey, collections.Uint64Value),

		delegationKeeper:  delegationKeeper,
		ecosystemKeeper:   ecosystemKeeper,
		corporationKeeper: corporationKeeper,
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema
	return k
}

func (k Keeper) GetAuthority() string { return k.authority }

func (k Keeper) Logger() log.Logger {
	return k.logger.With("module", "x/"+types.ModuleName)
}

func (k Keeper) GetNextID(ctx sdk.Context, entityType string) (uint64, error) {
	current, err := k.Counter.Get(ctx, entityType)
	if err != nil {
		current = 0
	}
	next := current + 1
	if err := k.Counter.Set(ctx, entityType, next); err != nil {
		return 0, fmt.Errorf("failed to set counter: %w", err)
	}
	return next, nil
}
```

- [ ] **Step 11.2: Build.**

  Run: `go build ./x/gf/...`
  Expected: exit 0.

- [ ] **Step 11.3: Commit.**

```bash
git add x/gf/keeper/keeper.go
git commit -m "feat(gf): add keeper with collections and counter helper"
```

---

## Task 12: Create x/gf/keeper/params.go, msg_server.go, msg_update_params.go

**Files:** `x/gf/keeper/params.go`, `x/gf/keeper/msg_server.go`, `x/gf/keeper/msg_update_params.go`

- [ ] **Step 12.1: Write params.go.**

```go
package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana/x/gf/types"
)

func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	p, err := k.Params.Get(ctx)
	if err != nil {
		return types.DefaultParams()
	}
	return p
}

func (k Keeper) SetParams(ctx sdk.Context, p types.Params) error {
	return k.Params.Set(ctx, p)
}
```

- [ ] **Step 12.2: Write msg_server.go.**

```go
package keeper

import "github.com/verana-labs/verana/x/gf/types"

type msgServer struct {
	Keeper
}

func NewMsgServerImpl(k Keeper) types.MsgServer {
	return &msgServer{Keeper: k}
}

var _ types.MsgServer = msgServer{}
```

- [ ] **Step 12.3: Write msg_update_params.go.**

```go
package keeper

import (
	"context"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/verana-labs/verana/x/gf/types"
)

func (ms msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if ms.GetAuthority() != msg.Authority {
		return nil, errors.Wrapf(sdkerrors.ErrUnauthorized, "expected %s got %s", ms.GetAuthority(), msg.Authority)
	}
	if err := msg.Params.Validate(); err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := ms.SetParams(ctx, msg.Params); err != nil {
		return nil, err
	}
	return &types.MsgUpdateParamsResponse{}, nil
}
```

- [ ] **Step 12.4: Build.**

  Run: `go build ./x/gf/...`
  Expected: exit 0.

- [ ] **Step 12.5: Commit.**

```bash
git add x/gf/keeper/params.go x/gf/keeper/msg_server.go x/gf/keeper/msg_update_params.go
git commit -m "feat(gf): add params keeper and UpdateParams handler"
```

---

## Task 13: Implement MOD-GF-MSG-1 AddGovernanceFrameworkDocument

**File:** `x/gf/keeper/msg_add_gf_document.go`

This handler maps directly to MOD-GF-MSG-1-2-1 (basic checks) and MOD-GF-MSG-1-3 (execution). Read those spec sections (lines 2475–2518) before writing the code.

- [ ] **Step 13.1: Write the handler.**

```go
package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"
	cerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana/x/gf/types"
)

// subjectKind is an internal tag for which owner the request targets.
type subjectKind int

const (
	subjectEcosystem subjectKind = iota
	subjectCorporation
)

// resolvedSubject holds the validated owner of a GF operation.
type resolvedSubject struct {
	kind          subjectKind
	ecosystemID   uint64
	corporation   string
	language      string
	activeVersion int32
}

// resolveSubject implements the spec's "Define subject as ..." block for MOD-GF-MSG-1-2-1
// and MOD-GF-MSG-2-2-1.
func (k Keeper) resolveSubject(ctx context.Context, signingCorp string, ecosystemID uint64) (resolvedSubject, error) {
	if ecosystemID != 0 {
		eco, ok := k.ecosystemKeeper.GetEcosystemView(ctx, ecosystemID)
		if !ok {
			return resolvedSubject{}, cerrors.Wrapf(types.ErrSubjectNotFound, "ecosystem %d", ecosystemID)
		}
		if eco.Corporation != signingCorp {
			return resolvedSubject{}, types.ErrSubjectNotControlled
		}
		return resolvedSubject{
			kind:          subjectEcosystem,
			ecosystemID:   eco.Id,
			language:      eco.Language,
			activeVersion: eco.ActiveVersion,
		}, nil
	}

	corp, ok := k.corporationKeeper.GetCorporationView(ctx, signingCorp)
	if !ok {
		return resolvedSubject{}, cerrors.Wrapf(types.ErrSubjectNotFound, "corporation %s", signingCorp)
	}
	return resolvedSubject{
		kind:          subjectCorporation,
		corporation:   corp.GroupPolicyAddress,
		language:      corp.Language,
		activeVersion: corp.ActiveVersion,
	}, nil
}

// maxVersionFor returns the highest known GFV.version for the subject, or 0 if none.
// Also returns whether a GFV with `targetVersion` already exists.
func (k Keeper) maxVersionFor(ctx context.Context, sub resolvedSubject, targetVersion int32) (maxV int32, hasTarget bool, gfvID uint64, err error) {
	switch sub.kind {
	case subjectEcosystem:
		iter, e := k.GFVersionByEcosystem.Iterate(ctx, collections.NewPrefixedPairRange[uint64, int32](sub.ecosystemID))
		if e != nil {
			err = e
			return
		}
		defer iter.Close()
		for ; iter.Valid(); iter.Next() {
			key, e := iter.Key()
			if e != nil {
				err = e
				return
			}
			v := key.K2()
			if v > maxV {
				maxV = v
			}
			if v == targetVersion {
				hasTarget = true
				id, e := iter.Value()
				if e != nil {
					err = e
					return
				}
				gfvID = id
			}
		}
	case subjectCorporation:
		iter, e := k.GFVersionByCorporation.Iterate(ctx, collections.NewPrefixedPairRange[string, int32](sub.corporation))
		if e != nil {
			err = e
			return
		}
		defer iter.Close()
		for ; iter.Valid(); iter.Next() {
			key, e := iter.Key()
			if e != nil {
				err = e
				return
			}
			v := key.K2()
			if v > maxV {
				maxV = v
			}
			if v == targetVersion {
				hasTarget = true
				id, e := iter.Value()
				if e != nil {
					err = e
					return
				}
				gfvID = id
			}
		}
	}
	return
}

// AddGovernanceFrameworkDocument implements MOD-GF-MSG-1.
func (ms msgServer) AddGovernanceFrameworkDocument(goCtx context.Context, msg *types.MsgAddGovernanceFrameworkDocument) (*types.MsgAddGovernanceFrameworkDocumentResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	// AUTHZ-CHECK-1
	if err := ms.delegationKeeper.CheckOperatorAuthorization(ctx, msg.Corporation, msg.Operator, sdk.MsgTypeURL(msg)); err != nil {
		return nil, err
	}

	// Resolve subject (Ecosystem or Corporation).
	sub, err := ms.resolveSubject(ctx, msg.Corporation, msg.EcosystemId)
	if err != nil {
		return nil, err
	}

	// Version checks per MOD-GF-MSG-1-2-1.
	maxV, hasTarget, existingGfvID, err := ms.maxVersionFor(ctx, sub, msg.Version)
	if err != nil {
		return nil, err
	}
	if !hasTarget && msg.Version != maxV+1 {
		return nil, cerrors.Wrapf(types.ErrInvalidVersion, "version must be %d", maxV+1)
	}
	if msg.Version <= sub.activeVersion {
		return nil, cerrors.Wrapf(types.ErrInvalidVersion, "must be greater than active_version %d", sub.activeVersion)
	}

	// Execute per MOD-GF-MSG-1-3.
	var gfv types.GovernanceFrameworkVersion
	if hasTarget {
		gfv, err = ms.GFVersion.Get(ctx, existingGfvID)
		if err != nil {
			return nil, fmt.Errorf("fetch gfv %d: %w", existingGfvID, err)
		}
	} else {
		nextID, err := ms.GetNextID(ctx, "gfv")
		if err != nil {
			return nil, err
		}
		gfv = types.GovernanceFrameworkVersion{
			Id:      nextID,
			Created: ctx.BlockTime(),
			Version: msg.Version,
		}
		if sub.kind == subjectEcosystem {
			gfv.EcosystemId = sub.ecosystemID
		} else {
			gfv.Corporation = sub.corporation
		}
		if err := ms.GFVersion.Set(ctx, gfv.Id, gfv); err != nil {
			return nil, fmt.Errorf("persist gfv: %w", err)
		}
		// Maintain secondary index.
		if sub.kind == subjectEcosystem {
			if err := ms.GFVersionByEcosystem.Set(ctx, collections.Join(sub.ecosystemID, msg.Version), gfv.Id); err != nil {
				return nil, fmt.Errorf("persist gfv eco index: %w", err)
			}
		} else {
			if err := ms.GFVersionByCorporation.Set(ctx, collections.Join(sub.corporation, msg.Version), gfv.Id); err != nil {
				return nil, fmt.Errorf("persist gfv corp index: %w", err)
			}
		}
	}

	// Upsert the document for (gfv, language).
	var existingGFD types.GovernanceFrameworkDocument
	hasExisting := false
	if err := ms.GFDocument.Walk(ctx, nil, func(_ uint64, doc types.GovernanceFrameworkDocument) (bool, error) {
		if doc.GfvId == gfv.Id && doc.Language == msg.DocLanguage {
			existingGFD = doc
			hasExisting = true
			return true, nil
		}
		return false, nil
	}); err != nil {
		return nil, fmt.Errorf("walk gfd: %w", err)
	}

	var gfd types.GovernanceFrameworkDocument
	if hasExisting {
		gfd = existingGFD
		gfd.Url = msg.DocUrl
		gfd.DigestSri = msg.DocDigestSri
	} else {
		nextID, err := ms.GetNextID(ctx, "gfd")
		if err != nil {
			return nil, err
		}
		gfd = types.GovernanceFrameworkDocument{
			Id:        nextID,
			GfvId:     gfv.Id,
			Created:   ctx.BlockTime(),
			Language:  msg.DocLanguage,
			Url:       msg.DocUrl,
			DigestSri: msg.DocDigestSri,
		}
	}
	if err := ms.GFDocument.Set(ctx, gfd.Id, gfd); err != nil {
		return nil, fmt.Errorf("persist gfd: %w", err)
	}

	// Emit event.
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeAddGFDocument,
		sdk.NewAttribute(types.AttributeKeyCorporation, msg.Corporation),
		sdk.NewAttribute(types.AttributeKeyEcosystemID, fmt.Sprintf("%d", msg.EcosystemId)),
		sdk.NewAttribute(types.AttributeKeyGFVersionID, fmt.Sprintf("%d", gfv.Id)),
		sdk.NewAttribute(types.AttributeKeyGFDocID, fmt.Sprintf("%d", gfd.Id)),
		sdk.NewAttribute(types.AttributeKeyVersion, fmt.Sprintf("%d", msg.Version)),
		sdk.NewAttribute(types.AttributeKeyLanguage, msg.DocLanguage),
	))

	return &types.MsgAddGovernanceFrameworkDocumentResponse{}, nil
}
```

- [ ] **Step 13.2: Build.**

  Run: `go build ./x/gf/...`
  Expected: exit 0.

- [ ] **Step 13.3: Commit.**

```bash
git add x/gf/keeper/msg_add_gf_document.go
git commit -m "feat(gf): implement MOD-GF-MSG-1 AddGovernanceFrameworkDocument"
```

---

## Task 14: Implement MOD-GF-MSG-2 IncreaseActiveGovernanceFrameworkVersion

**File:** `x/gf/keeper/msg_increase_active_gfv.go`

Maps to MOD-GF-MSG-2-2-1 (basic checks) and MOD-GF-MSG-2-3 (execution) in the spec.

- [ ] **Step 14.1: Write the handler.**

```go
package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"
	cerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana/x/gf/types"
)

// IncreaseActiveGovernanceFrameworkVersion implements MOD-GF-MSG-2.
func (ms msgServer) IncreaseActiveGovernanceFrameworkVersion(goCtx context.Context, msg *types.MsgIncreaseActiveGovernanceFrameworkVersion) (*types.MsgIncreaseActiveGovernanceFrameworkVersionResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	// AUTHZ-CHECK-1
	if err := ms.delegationKeeper.CheckOperatorAuthorization(ctx, msg.Corporation, msg.Operator, sdk.MsgTypeURL(msg)); err != nil {
		return nil, err
	}

	sub, err := ms.resolveSubject(ctx, msg.Corporation, msg.EcosystemId)
	if err != nil {
		return nil, err
	}

	nextVersion := sub.activeVersion + 1
	// Lookup next GFV via secondary index.
	var gfvID uint64
	switch sub.kind {
	case subjectEcosystem:
		gfvID, err = ms.GFVersionByEcosystem.Get(ctx, collections.Join(sub.ecosystemID, nextVersion))
	case subjectCorporation:
		gfvID, err = ms.GFVersionByCorporation.Get(ctx, collections.Join(sub.corporation, nextVersion))
	}
	if err != nil {
		return nil, cerrors.Wrapf(types.ErrNoActivatableVersion, "no GFV for next version %d", nextVersion)
	}
	gfv, err := ms.GFVersion.Get(ctx, gfvID)
	if err != nil {
		return nil, fmt.Errorf("fetch gfv %d: %w", gfvID, err)
	}

	// Spec MOD-GF-MSG-2-2-1: a document for subject.language MUST exist on this version.
	hasDefaultLang := false
	if err := ms.GFDocument.Walk(ctx, nil, func(_ uint64, doc types.GovernanceFrameworkDocument) (bool, error) {
		if doc.GfvId == gfv.Id && doc.Language == sub.language {
			hasDefaultLang = true
			return true, nil
		}
		return false, nil
	}); err != nil {
		return nil, fmt.Errorf("walk gfd: %w", err)
	}
	if !hasDefaultLang {
		return nil, types.ErrMissingDefaultLang
	}

	// Execute MOD-GF-MSG-2-3.
	now := ctx.BlockTime()
	gfv.ActiveSince = now
	if err := ms.GFVersion.Set(ctx, gfv.Id, gfv); err != nil {
		return nil, fmt.Errorf("persist gfv: %w", err)
	}

	switch sub.kind {
	case subjectEcosystem:
		if err := ms.ecosystemKeeper.SetEcosystemActiveVersion(ctx, sub.ecosystemID, nextVersion); err != nil {
			return nil, fmt.Errorf("update ecosystem active version: %w", err)
		}
	case subjectCorporation:
		if err := ms.corporationKeeper.SetCorporationActiveVersion(ctx, sub.corporation, nextVersion); err != nil {
			return nil, fmt.Errorf("update corporation active version: %w", err)
		}
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeIncreaseGFActive,
		sdk.NewAttribute(types.AttributeKeyCorporation, msg.Corporation),
		sdk.NewAttribute(types.AttributeKeyEcosystemID, fmt.Sprintf("%d", msg.EcosystemId)),
		sdk.NewAttribute(types.AttributeKeyGFVersionID, fmt.Sprintf("%d", gfv.Id)),
		sdk.NewAttribute(types.AttributeKeyVersion, fmt.Sprintf("%d", nextVersion)),
	))

	return &types.MsgIncreaseActiveGovernanceFrameworkVersionResponse{}, nil
}
```

- [ ] **Step 14.2: Build.**

  Run: `go build ./x/gf/...`
  Expected: exit 0.

- [ ] **Step 14.3: Commit.**

```bash
git add x/gf/keeper/msg_increase_active_gfv.go
git commit -m "feat(gf): implement MOD-GF-MSG-2 IncreaseActiveGovernanceFrameworkVersion"
```

---

## Task 15: Implement queries (QRY-1 and QRY-2)

**Files:** `x/gf/keeper/query.go`, `x/gf/keeper/query_params.go`, `x/gf/keeper/query_gfv.go`

- [ ] **Step 15.1: Write query.go.**

```go
package keeper

import "github.com/verana-labs/verana/x/gf/types"

type querier struct {
	Keeper
}

func NewQueryServerImpl(k Keeper) types.QueryServer {
	return &querier{Keeper: k}
}

var _ types.QueryServer = querier{}
```

- [ ] **Step 15.2: Write query_params.go.**

```go
package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana/x/gf/types"
)

func (q querier) Params(goCtx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	return &types.QueryParamsResponse{Params: q.GetParams(sdk.UnwrapSDKContext(goCtx))}, nil
}
```

- [ ] **Step 15.3: Write query_gfv.go.**

```go
package keeper

import (
	"context"
	"sort"

	"cosmossdk.io/collections"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/verana-labs/verana/x/gf/types"
)

func (q querier) GetGovernanceFrameworkVersion(goCtx context.Context, req *types.QueryGetGovernanceFrameworkVersionRequest) (*types.QueryGetGovernanceFrameworkVersionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	gfv, err := q.GFVersion.Get(goCtx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "gfv %d not found", req.Id)
	}
	docs, err := q.collectDocs(goCtx, gfv.Id, req.PreferredLanguage)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "collect docs: %v", err)
	}
	return &types.QueryGetGovernanceFrameworkVersionResponse{
		Version: types.GovernanceFrameworkVersionWithDocs{
			Id:          gfv.Id,
			EcosystemId: gfv.EcosystemId,
			Corporation: gfv.Corporation,
			Created:     gfv.Created,
			Version:     gfv.Version,
			ActiveSince: gfv.ActiveSince,
			Documents:   docs,
		},
	}, nil
}

func (q querier) ListGovernanceFrameworkVersions(goCtx context.Context, req *types.QueryListGovernanceFrameworkVersionsRequest) (*types.QueryListGovernanceFrameworkVersionsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	hasEco := req.EcosystemId > 0
	hasCorp := req.Corporation != ""
	if hasEco == hasCorp {
		return nil, status.Error(codes.InvalidArgument, "exactly one of ecosystem_id or corporation must be set")
	}
	if req.ResponseMaxSize == 0 {
		req.ResponseMaxSize = 64
	}
	if req.ResponseMaxSize > 1024 {
		return nil, status.Error(codes.InvalidArgument, "response_max_size must be <= 1024")
	}

	// Collect matching gfv_ids via secondary index.
	var gfvIDs []uint64
	if hasEco {
		iter, err := q.GFVersionByEcosystem.Iterate(goCtx, collections.NewPrefixedPairRange[uint64, int32](req.EcosystemId))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "iterate: %v", err)
		}
		defer iter.Close()
		for ; iter.Valid(); iter.Next() {
			id, err := iter.Value()
			if err != nil {
				return nil, status.Errorf(codes.Internal, "iter value: %v", err)
			}
			gfvIDs = append(gfvIDs, id)
		}
	} else {
		iter, err := q.GFVersionByCorporation.Iterate(goCtx, collections.NewPrefixedPairRange[string, int32](req.Corporation))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "iterate: %v", err)
		}
		defer iter.Close()
		for ; iter.Valid(); iter.Next() {
			id, err := iter.Value()
			if err != nil {
				return nil, status.Errorf(codes.Internal, "iter value: %v", err)
			}
			gfvIDs = append(gfvIDs, id)
		}
	}

	versions := make([]types.GovernanceFrameworkVersionWithDocs, 0, len(gfvIDs))
	for _, id := range gfvIDs {
		gfv, err := q.GFVersion.Get(goCtx, id)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "fetch gfv %d: %v", id, err)
		}
		if req.ActiveOnly && gfv.ActiveSince.IsZero() {
			continue
		}
		docs, err := q.collectDocs(goCtx, gfv.Id, req.PreferredLanguage)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "collect docs: %v", err)
		}
		versions = append(versions, types.GovernanceFrameworkVersionWithDocs{
			Id:          gfv.Id,
			EcosystemId: gfv.EcosystemId,
			Corporation: gfv.Corporation,
			Created:     gfv.Created,
			Version:     gfv.Version,
			ActiveSince: gfv.ActiveSince,
			Documents:   docs,
		})
	}
	// Spec MOD-GF-QRY-2-3: order by ascending version.
	sort.Slice(versions, func(i, j int) bool { return versions[i].Version < versions[j].Version })
	if uint32(len(versions)) > req.ResponseMaxSize {
		versions = versions[:req.ResponseMaxSize]
	}
	return &types.QueryListGovernanceFrameworkVersionsResponse{Versions: versions}, nil
}

func (q querier) collectDocs(ctx context.Context, gfvID uint64, preferredLang string) ([]types.GovernanceFrameworkDocument, error) {
	var out []types.GovernanceFrameworkDocument
	var preferred *types.GovernanceFrameworkDocument
	if err := q.GFDocument.Walk(ctx, nil, func(_ uint64, d types.GovernanceFrameworkDocument) (bool, error) {
		if d.GfvId != gfvID {
			return false, nil
		}
		if preferredLang != "" {
			if d.Language == preferredLang && preferred == nil {
				cp := d
				preferred = &cp
			}
			return false, nil
		}
		out = append(out, d)
		return false, nil
	}); err != nil {
		return nil, err
	}
	if preferredLang != "" {
		if preferred != nil {
			return []types.GovernanceFrameworkDocument{*preferred}, nil
		}
		// Fall back to all docs if preferred language not present (spec QRY-1-3 says "preferring").
		_ = q.GFDocument.Walk(ctx, nil, func(_ uint64, d types.GovernanceFrameworkDocument) (bool, error) {
			if d.GfvId == gfvID {
				out = append(out, d)
			}
			return false, nil
		})
		return out, nil
	}
	return out, nil
}
```

- [ ] **Step 15.4: Build.**

  Run: `go build ./x/gf/...`
  Expected: exit 0.

- [ ] **Step 15.5: Commit.**

```bash
git add x/gf/keeper/query.go x/gf/keeper/query_params.go x/gf/keeper/query_gfv.go
git commit -m "feat(gf): implement MOD-GF-QRY-1 and QRY-2 queries"
```

---

## Task 16: Genesis init/export in keeper

**File:** `x/gf/keeper/genesis.go`

- [ ] **Step 16.1: Write the file.**

```go
package keeper

import (
	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana/x/gf/types"
)

func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) error {
	if err := k.SetParams(ctx, gs.Params); err != nil {
		return err
	}
	var maxGFV, maxGFD uint64
	for _, gfv := range gs.Versions {
		if err := k.GFVersion.Set(ctx, gfv.Id, gfv); err != nil {
			return err
		}
		if gfv.EcosystemId > 0 {
			if err := k.GFVersionByEcosystem.Set(ctx, collections.Join(gfv.EcosystemId, gfv.Version), gfv.Id); err != nil {
				return err
			}
		} else {
			if err := k.GFVersionByCorporation.Set(ctx, collections.Join(gfv.Corporation, gfv.Version), gfv.Id); err != nil {
				return err
			}
		}
		if gfv.Id > maxGFV {
			maxGFV = gfv.Id
		}
	}
	for _, gfd := range gs.Documents {
		if err := k.GFDocument.Set(ctx, gfd.Id, gfd); err != nil {
			return err
		}
		if gfd.Id > maxGFD {
			maxGFD = gfd.Id
		}
	}
	if err := k.Counter.Set(ctx, "gfv", maxGFV); err != nil {
		return err
	}
	if err := k.Counter.Set(ctx, "gfd", maxGFD); err != nil {
		return err
	}
	return nil
}

func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	gs := &types.GenesisState{
		Params: k.GetParams(ctx),
	}
	_ = k.GFVersion.Walk(ctx, nil, func(_ uint64, gfv types.GovernanceFrameworkVersion) (bool, error) {
		gs.Versions = append(gs.Versions, gfv)
		return false, nil
	})
	_ = k.GFDocument.Walk(ctx, nil, func(_ uint64, gfd types.GovernanceFrameworkDocument) (bool, error) {
		gs.Documents = append(gs.Documents, gfd)
		return false, nil
	})
	return gs
}
```

- [ ] **Step 16.2: Build.**

  Run: `go build ./x/gf/...`
  Expected: exit 0.

- [ ] **Step 16.3: Commit.**

```bash
git add x/gf/keeper/genesis.go
git commit -m "feat(gf): add InitGenesis and ExportGenesis"
```

---

## Phase 4 — Module Wiring

## Task 17: Create x/gf/module files

**Files:** `x/gf/module/module.go`, `x/gf/module/autocli.go`, `x/gf/module/genesis.go`

Use `x/tr/module/module.go` as the template — copy it to `x/gf/module/module.go` and replace `tr`/`trustregistry` with `gf`/`gf`. Adjust imports.

- [ ] **Step 17.1: Write module.go.**

Open `x/tr/module/module.go`, copy contents, replace all module-name occurrences (`tr` → `gf`, `trustregistrymodulekeeper` → `gfmodulekeeper`), and remove TR-specific fields if any. The resulting file:
- Imports `github.com/verana-labs/verana/x/gf/keeper` and `github.com/verana-labs/verana/x/gf/types`.
- Declares `AppModule`, `AppModuleBasic` with the standard methods.
- Implements `RegisterServices` (registers Msg + Query servers).

- [ ] **Step 17.2: Write autocli.go.**

Use `x/tr/module/autocli.go` as a template. The new file declares CLI for:

Tx:
- `MsgAddGovernanceFrameworkDocument` — exposes `--corporation`, `--operator`, `--ecosystem-id`, `--doc-language`, `--doc-url`, `--doc-digest-sri`, `--version`
- `MsgIncreaseActiveGovernanceFrameworkVersion` — exposes `--corporation`, `--operator`, `--ecosystem-id`
- `MsgUpdateParams` — `Skip: true` (governance only)

Query:
- `GetGovernanceFrameworkVersion` — positional `id`, optional `--preferred-language`
- `ListGovernanceFrameworkVersions` — `--ecosystem-id`, `--corporation`, `--active-only`, `--preferred-language`, `--response-max-size`
- `Params` — no args

- [ ] **Step 17.3: Write genesis.go (module-level wrapper).**

```go
package module

import (
	"encoding/json"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana/x/gf/keeper"
	"github.com/verana-labs/verana/x/gf/types"
)

// InitGenesis decodes raw bytes into GenesisState and applies it.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, cdc codec.JSONCodec, data json.RawMessage) {
	var gs types.GenesisState
	if err := cdc.UnmarshalJSON(data, &gs); err != nil {
		panic(fmt.Errorf("unmarshal gf genesis: %w", err))
	}
	if err := k.InitGenesis(ctx, gs); err != nil {
		panic(err)
	}
}

// ExportGenesis returns the module's exported genesis as raw JSON.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper, cdc codec.JSONCodec) json.RawMessage {
	gs := k.ExportGenesis(ctx)
	bz, err := cdc.MarshalJSON(gs)
	if err != nil {
		panic(err)
	}
	return bz
}
```

- [ ] **Step 17.4: Build.**

  Run: `go build ./x/gf/...`
  Expected: exit 0.

- [ ] **Step 17.5: Commit.**

```bash
git add x/gf/module/
git commit -m "feat(gf): add module.go, autocli.go, and genesis wrapper"
```

---

## Task 18: Adapters live in x/gf/keeper (interim, removed in #305/#303)

Place the adapter and stub in the gf package so they're constructed alongside the keeper, not in `app/`. This avoids the post-build wiring brittleness flagged in the plan audit.

**File:** `x/gf/keeper/adapters.go`

- [ ] **Step 18.1: Write the adapter.**

```go
package keeper

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	gftypes "github.com/verana-labs/verana/x/gf/types"
	trkeeper "github.com/verana-labs/verana/x/tr/keeper"
)

// TRAsEcosystemKeeper wraps the existing x/tr keeper to satisfy
// gftypes.EcosystemKeeper. Removed when issue #305 (TR→EC rename) lands —
// the renamed EC keeper will implement gftypes.EcosystemKeeper directly.
type TRAsEcosystemKeeper struct {
	k trkeeper.Keeper
}

func NewTRAsEcosystemKeeper(k trkeeper.Keeper) gftypes.EcosystemKeeper {
	return TRAsEcosystemKeeper{k: k}
}

func (a TRAsEcosystemKeeper) GetEcosystemView(ctx context.Context, id uint64) (gftypes.EcosystemView, bool) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	tr, err := a.k.GetTrustRegistry(sdkCtx, id)
	if err != nil {
		return gftypes.EcosystemView{}, false
	}
	return gftypes.EcosystemView{
		Id:            tr.Id,
		Corporation:   tr.Corporation,
		Language:      tr.Language,
		ActiveVersion: tr.ActiveVersion,
	}, true
}

func (a TRAsEcosystemKeeper) SetEcosystemActiveVersion(ctx context.Context, id uint64, newVersion int32) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	tr, err := a.k.GetTrustRegistry(sdkCtx, id)
	if err != nil {
		return fmt.Errorf("get trust registry %d: %w", id, err)
	}
	tr.ActiveVersion = newVersion
	tr.Modified = sdkCtx.BlockTime()
	return a.k.TrustRegistry.Set(sdkCtx, tr.Id, tr)
}

// StubCorporationKeeper returns (zero, false) for all lookups. Replaced when
// issue #303 (MOD-CO) lands; until then corporation-targeted MOD-GF calls
// abort cleanly with gftypes.ErrSubjectNotFound (because GetCorporationView
// returns ok=false).
type StubCorporationKeeper struct{}

func NewStubCorporationKeeper() gftypes.CorporationKeeper {
	return StubCorporationKeeper{}
}

func (StubCorporationKeeper) GetCorporationView(ctx context.Context, _ string) (gftypes.CorporationView, bool) {
	return gftypes.CorporationView{}, false
}

func (StubCorporationKeeper) SetCorporationActiveVersion(ctx context.Context, _ string, _ int32) error {
	return errors.New("corporation keeper not wired yet (MOD-CO pending in issue #303)")
}
```

- [ ] **Step 18.2: Build.**

  Run: `go build ./x/gf/...`
  Expected: exit 0.

- [ ] **Step 18.3: Commit.**

```bash
git add x/gf/keeper/adapters.go
git commit -m "feat(gf): add TR-as-Ecosystem adapter and stub Corporation keeper"
```

---

## Task 19: Wire x/gf into app/app.go and app/app_config.go

**Files:** `app/app.go`, `app/app_config.go`

All three required keepers are passed at construction time. No post-build setters, no nil interfaces.

- [ ] **Step 19.1: Add imports to `app/app.go`.**

Locate the existing module side-effect import block and add:
```go
_ "github.com/verana-labs/verana/x/gf/module" // import for side-effects
```

Locate the existing module keeper alias block and add:
```go
gfmodulekeeper "github.com/verana-labs/verana/x/gf/keeper"
```

- [ ] **Step 19.2: Add field to `App` struct.**

Below the `TrustregistryKeeper` field, add:
```go
GfKeeper gfmodulekeeper.Keeper
```

- [ ] **Step 19.3: Choose wiring strategy based on existing depinject pattern.**

Inspect `app/app.go` around the `appBuilder.Build(...)` call to confirm whether keepers are constructed via depinject providers (interface auto-matching) or explicitly. The existing x/tr keeper wiring should be the reference.

**If depinject auto-matches `DelegationKeeper`:**
- `app.GfKeeper` will be constructed by depinject with `DelegationKeeper` already wired.
- `EcosystemKeeper` and `CorporationKeeper` will NOT be auto-matched (they're MOD-GF-specific interfaces).
- Solution: provide a `depinject.Supply(...)` wrapper that supplies pre-constructed adapter instances BEFORE depinject runs the GF module's provider. This requires re-ordering keeper construction so that:
  1. TR keeper is built first.
  2. `NewTRAsEcosystemKeeper(app.TrustregistryKeeper)` and `NewStubCorporationKeeper()` are constructed.
  3. Both are supplied to depinject before GF runs.

**If keepers are constructed manually (not via depinject):**
- Just construct GF after TR:
  ```go
  app.GfKeeper = gfmodulekeeper.NewKeeper(
      appCodec,
      runtime.NewKVStoreService(keys[gftypes.StoreKey]),
      logger.With("module", "x/gf"),
      authtypes.NewModuleAddress(govtypes.ModuleName).String(),
      app.DeKeeper,
      gfmodulekeeper.NewTRAsEcosystemKeeper(app.TrustregistryKeeper),
      gfmodulekeeper.NewStubCorporationKeeper(),
  )
  ```

Pick whichever matches the existing chain's pattern. Both eliminate the nil-keeper risk.

- [ ] **Step 19.4: Update app_config.go.**

In the modules list, add an entry for `gf` (mirror the TR entry):
```go
{
    Name: gftypes.ModuleName,
    Config: appconfig.WrapAny(&gfmoduletypes.Module{}),
},
```

Add imports at the top:
```go
gftypes "github.com/verana-labs/verana/x/gf/types"
gfmoduletypes "github.com/verana-labs/verana/api/verana/gf/module/v1"
```

No special permissions (mint/burn/etc) are required.

- [ ] **Step 19.5: Build the entire repo.**

  Run: `go build ./...`
  Expected: exit 0.

- [ ] **Step 19.6: Run vet.**

  Run: `go vet ./...`
  Expected: exit 0.

- [ ] **Step 19.7: Commit.**

```bash
git add app/app.go app/app_config.go
git commit -m "feat(gf): wire x/gf module into app with EcosystemKeeper adapter"
```

---

## Task 20: Add testutil/keeper/gf.go

**File:** `testutil/keeper/gf.go`

Use `testutil/keeper/trustregistry.go` as the template.

- [ ] **Step 20.1: Write the file.**

```go
package keeper

import (
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana/x/gf/keeper"
	"github.com/verana-labs/verana/x/gf/types"
)

// GfKeeperWithDelegation returns a GF keeper wired with a mock delegation keeper
// (so tests can simulate AUTHZ-CHECK-1 outcomes) and stub Eco/Corp keepers
// (configurable per test).
func GfKeeperWithDelegation(
	t testing.TB,
	del types.DelegationKeeper,
	eco types.EcosystemKeeper,
	corp types.CorporationKeeper,
) (keeper.Keeper, sdk.Context) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority.String(),
		del,
		eco,
		corp,
	)
	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	if err := k.SetParams(ctx, types.DefaultParams()); err != nil {
		panic(err)
	}
	return k, ctx
}
```

- [ ] **Step 20.2: Build.**

  Run: `go build ./testutil/...`
  Expected: exit 0.

- [ ] **Step 20.3: Commit.**

```bash
git add testutil/keeper/gf.go
git commit -m "test(gf): add GfKeeperWithDelegation test util"
```

---

## Phase 5 — Tests

## Task 21: Unit tests for MOD-GF-MSG-1 AddGovernanceFrameworkDocument

**File:** `x/gf/keeper/msg_add_gf_document_test.go`

Each `t.Run` is its own scenario with its own keeper instance. Use mock Eco/Corp keepers to simulate subject lookups.

- [ ] **Step 21.1: Write the test file.**

```go
package keeper_test

import (
	"context"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana/testutil/keeper"
	"github.com/verana-labs/verana/x/gf/keeper"
	"github.com/verana-labs/verana/x/gf/types"
)

const (
	testCorp     = "cosmos1corp00000000000000000000000000000abc"
	testOperator = "cosmos1op0000000000000000000000000000000abc"
	testEcoCorp  = "cosmos1ecocorp0000000000000000000000000abc"
	testDigest   = "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26"
	testURL      = "https://example.com/gf-v1.html"
)

// mockDelegation implements types.DelegationKeeper.
type mockDelegation struct{ err error }

func (m mockDelegation) CheckOperatorAuthorization(sdk.Context, string, string, string) error {
	return m.err
}

// mockEcosystem implements types.EcosystemKeeper.
type mockEcosystem struct {
	view  types.EcosystemView
	found bool
	setFn func(uint64, int32) error
}

func (m *mockEcosystem) GetEcosystemView(ctx context.Context, id uint64) (types.EcosystemView, bool) {
	return m.view, m.found
}
func (m *mockEcosystem) SetEcosystemActiveVersion(ctx context.Context, id uint64, v int32) error {
	if m.setFn != nil {
		return m.setFn(id, v)
	}
	return nil
}

// mockCorporation implements types.CorporationKeeper.
type mockCorporation struct {
	view  types.CorporationView
	found bool
	setFn func(string, uint32) error
}

func (m *mockCorporation) GetCorporationView(ctx context.Context, _ string) (types.CorporationView, bool) {
	return m.view, m.found
}
func (m *mockCorporation) SetCorporationActiveVersion(ctx context.Context, _ string, _ uint32) error {
	if m.setFn != nil {
		return m.setFn("", 0)
	}
	return nil
}

func TestAddGovernanceFrameworkDocument(t *testing.T) {
	t.Run("MOD-GF-MSG-1: happy path adds GFV+GFD to Corporation subject", func(t *testing.T) {
		corp := &mockCorporation{
			view:  types.CorporationView{GroupPolicyAddress: testCorp, Language: "en", ActiveVersion: 0},
			found: true,
		}
		eco := &mockEcosystem{}
		k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, eco, corp)
		ms := keeper.NewMsgServerImpl(k)

		now := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
		ctx = ctx.WithBlockTime(now)

		_, err := ms.AddGovernanceFrameworkDocument(ctx, &types.MsgAddGovernanceFrameworkDocument{
			Corporation:  testCorp,
			Operator:     testOperator,
			EcosystemId:  0,
			DocLanguage:  "en",
			DocUrl:       testURL,
			DocDigestSri: testDigest,
			Version:      1,
		})
		require.NoError(t, err)

		// Spec MOD-GF-MSG-1-3 — exactly one GFV with corporation set.
		var gfvCount int
		_ = k.GFVersion.Walk(ctx, nil, func(_ uint64, gfv types.GovernanceFrameworkVersion) (bool, error) {
			gfvCount++
			require.Equal(t, testCorp, gfv.Corporation)
			require.Zero(t, gfv.EcosystemId)
			require.Equal(t, int32(1), gfv.Version)
			require.True(t, gfv.ActiveSince.IsZero())
			return false, nil
		})
		require.Equal(t, 1, gfvCount)
	})

	t.Run("MOD-GF-MSG-1-2-1: AUTHZ-CHECK-1 failure aborts", func(t *testing.T) {
		corp := &mockCorporation{view: types.CorporationView{GroupPolicyAddress: testCorp}, found: true}
		eco := &mockEcosystem{}
		k, ctx := keepertest.GfKeeperWithDelegation(t,
			mockDelegation{err: sdk.ErrInvalidRequest.Wrap("unauthorized")},
			eco, corp,
		)
		ms := keeper.NewMsgServerImpl(k)
		_, err := ms.AddGovernanceFrameworkDocument(ctx, validMsg(testCorp, testOperator, 0, 1))
		require.Error(t, err)
	})

	t.Run("MOD-GF-MSG-1-2-1: ecosystem_id set but ecosystem not controlled by signer aborts", func(t *testing.T) {
		eco := &mockEcosystem{
			view:  types.EcosystemView{Id: 1, Corporation: "other_corp", Language: "en"},
			found: true,
		}
		corp := &mockCorporation{}
		k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, eco, corp)
		ms := keeper.NewMsgServerImpl(k)
		_, err := ms.AddGovernanceFrameworkDocument(ctx, validMsg(testCorp, testOperator, 1, 1))
		require.ErrorIs(t, err, types.ErrSubjectNotControlled)
	})

	t.Run("MOD-GF-MSG-1-2-1: version must be > active_version", func(t *testing.T) {
		corp := &mockCorporation{
			view:  types.CorporationView{GroupPolicyAddress: testCorp, Language: "en", ActiveVersion: 2},
			found: true,
		}
		eco := &mockEcosystem{}
		k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, eco, corp)
		ms := keeper.NewMsgServerImpl(k)
		_, err := ms.AddGovernanceFrameworkDocument(ctx, validMsg(testCorp, testOperator, 0, 2))
		require.ErrorIs(t, err, types.ErrInvalidVersion)
	})

	t.Run("MOD-GF-MSG-1-3: replaces existing GFD for same language", func(t *testing.T) {
		corp := &mockCorporation{
			view:  types.CorporationView{GroupPolicyAddress: testCorp, Language: "en", ActiveVersion: 0},
			found: true,
		}
		eco := &mockEcosystem{}
		k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, eco, corp)
		ms := keeper.NewMsgServerImpl(k)

		_, err := ms.AddGovernanceFrameworkDocument(ctx, validMsg(testCorp, testOperator, 0, 1))
		require.NoError(t, err)

		// Add a SECOND GFD for the same (gfv, language) — should update in place.
		updated := validMsg(testCorp, testOperator, 0, 1)
		updated.DocUrl = "https://example.com/gf-v1-updated.html"
		_, err = ms.AddGovernanceFrameworkDocument(ctx, updated)
		require.NoError(t, err)

		var gfdCount int
		_ = k.GFDocument.Walk(ctx, nil, func(_ uint64, d types.GovernanceFrameworkDocument) (bool, error) {
			gfdCount++
			require.Equal(t, "https://example.com/gf-v1-updated.html", d.Url)
			return false, nil
		})
		require.Equal(t, 1, gfdCount, "GFD count must be 1 — same-language doc must be replaced, not appended")
	})

	t.Run("MOD-GF-MSG-1-3: ecosystem subject creates GFV with ecosystem_id set", func(t *testing.T) {
		eco := &mockEcosystem{
			view:  types.EcosystemView{Id: 7, Corporation: testCorp, Language: "en"},
			found: true,
		}
		corp := &mockCorporation{}
		k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, eco, corp)
		ms := keeper.NewMsgServerImpl(k)

		_, err := ms.AddGovernanceFrameworkDocument(ctx, validMsg(testCorp, testOperator, 7, 1))
		require.NoError(t, err)

		_ = k.GFVersion.Walk(ctx, nil, func(_ uint64, gfv types.GovernanceFrameworkVersion) (bool, error) {
			require.Equal(t, uint64(7), gfv.EcosystemId)
			require.Equal(t, "", gfv.Corporation)
			return false, nil
		})
	})
}

func validMsg(corp, op string, ecoID uint64, version int32) *types.MsgAddGovernanceFrameworkDocument {
	return &types.MsgAddGovernanceFrameworkDocument{
		Corporation:  corp,
		Operator:     op,
		EcosystemId:  ecoID,
		DocLanguage:  "en",
		DocUrl:       testURL,
		DocDigestSri: testDigest,
		Version:      version,
	}
}
```

- [ ] **Step 21.2: Run tests.**

  Run: `go test ./x/gf/keeper/... -run TestAddGovernanceFrameworkDocument -v`
  Expected: PASS for all sub-tests.

- [ ] **Step 21.3: Commit.**

```bash
git add x/gf/keeper/msg_add_gf_document_test.go
git commit -m "test(gf): MOD-GF-MSG-1 happy + 4 precondition scenarios"
```

---

## Task 22: Unit tests for MOD-GF-MSG-2

**File:** `x/gf/keeper/msg_increase_active_gfv_test.go`

- [ ] **Step 22.1: Write the test file.**

```go
package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana/testutil/keeper"
	"github.com/verana-labs/verana/x/gf/keeper"
	"github.com/verana-labs/verana/x/gf/types"
)

func TestIncreaseActiveGovernanceFrameworkVersion(t *testing.T) {
	t.Run("MOD-GF-MSG-2: happy path activates next version for Corporation", func(t *testing.T) {
		corp := &mockCorporation{
			view:  types.CorporationView{GroupPolicyAddress: testCorp, Language: "en", ActiveVersion: 0},
			found: true,
		}
		var newActive uint32
		corp.setFn = func(_ string, v uint32) error { newActive = v; return nil }

		eco := &mockEcosystem{}
		k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, eco, corp)
		ms := keeper.NewMsgServerImpl(k)

		now := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
		ctx = ctx.WithBlockTime(now)

		// Setup: add a v1 GFV+GFD via MSG-1 (active_version stays 0).
		_, err := ms.AddGovernanceFrameworkDocument(ctx, validMsg(testCorp, testOperator, 0, 1))
		require.NoError(t, err)

		// Now bump active_version 0 → 1.
		_, err = ms.IncreaseActiveGovernanceFrameworkVersion(ctx, &types.MsgIncreaseActiveGovernanceFrameworkVersion{
			Corporation: testCorp,
			Operator:    testOperator,
			EcosystemId: 0,
		})
		require.NoError(t, err)
		require.Equal(t, uint32(1), newActive)

		// GFV active_since should now be set.
		_ = k.GFVersion.Walk(ctx, nil, func(_ uint64, gfv types.GovernanceFrameworkVersion) (bool, error) {
			require.False(t, gfv.ActiveSince.IsZero())
			require.Equal(t, now, gfv.ActiveSince)
			return false, nil
		})
	})

	t.Run("MOD-GF-MSG-2-2-1: aborts when no next-version GFV exists", func(t *testing.T) {
		corp := &mockCorporation{
			view:  types.CorporationView{GroupPolicyAddress: testCorp, Language: "en", ActiveVersion: 0},
			found: true,
		}
		eco := &mockEcosystem{}
		k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, eco, corp)
		ms := keeper.NewMsgServerImpl(k)

		// No MSG-1 first — nothing to activate.
		_, err := ms.IncreaseActiveGovernanceFrameworkVersion(ctx, &types.MsgIncreaseActiveGovernanceFrameworkVersion{
			Corporation: testCorp,
			Operator:    testOperator,
		})
		require.ErrorIs(t, err, types.ErrNoActivatableVersion)
	})

	t.Run("MOD-GF-MSG-2-2-1: aborts when default-language doc missing", func(t *testing.T) {
		corp := &mockCorporation{
			view:  types.CorporationView{GroupPolicyAddress: testCorp, Language: "en", ActiveVersion: 0},
			found: true,
		}
		eco := &mockEcosystem{}
		k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, eco, corp)
		ms := keeper.NewMsgServerImpl(k)

		// MSG-1 adds an "fr" doc — default language is "en".
		_, err := ms.AddGovernanceFrameworkDocument(ctx, &types.MsgAddGovernanceFrameworkDocument{
			Corporation:  testCorp,
			Operator:     testOperator,
			EcosystemId:  0,
			DocLanguage:  "fr",
			DocUrl:       testURL,
			DocDigestSri: testDigest,
			Version:      1,
		})
		require.NoError(t, err)

		_, err = ms.IncreaseActiveGovernanceFrameworkVersion(ctx, &types.MsgIncreaseActiveGovernanceFrameworkVersion{
			Corporation: testCorp,
			Operator:    testOperator,
		})
		require.ErrorIs(t, err, types.ErrMissingDefaultLang)
	})
}
```

- [ ] **Step 22.2: Run tests.**

  Run: `go test ./x/gf/keeper/... -run TestIncreaseActiveGovernanceFrameworkVersion -v`
  Expected: PASS.

- [ ] **Step 22.3: Commit.**

```bash
git add x/gf/keeper/msg_increase_active_gfv_test.go
git commit -m "test(gf): MOD-GF-MSG-2 happy + 2 precondition scenarios"
```

---

## Task 23: Query tests

**File:** `x/gf/keeper/query_gfv_test.go`

- [ ] **Step 23.1: Write the test file.**

```go
package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana/testutil/keeper"
	"github.com/verana-labs/verana/x/gf/keeper"
	"github.com/verana-labs/verana/x/gf/types"
)

func TestQueryGetGovernanceFrameworkVersion(t *testing.T) {
	t.Run("MOD-GF-QRY-1: returns GFV with docs, preferred language filter applied", func(t *testing.T) {
		corp := &mockCorporation{
			view:  types.CorporationView{GroupPolicyAddress: testCorp, Language: "en", ActiveVersion: 0},
			found: true,
		}
		k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, corp)
		ms := keeper.NewMsgServerImpl(k)
		// Setup: add 2 docs (en, fr) under v1.
		_, err := ms.AddGovernanceFrameworkDocument(ctx, validMsg(testCorp, testOperator, 0, 1))
		require.NoError(t, err)
		frMsg := validMsg(testCorp, testOperator, 0, 1)
		frMsg.DocLanguage = "fr"
		frMsg.DocUrl = "https://example.com/gf-v1-fr.html"
		_, err = ms.AddGovernanceFrameworkDocument(ctx, frMsg)
		require.NoError(t, err)

		qs := keeper.NewQueryServerImpl(k)
		// No filter — both docs.
		resp, err := qs.GetGovernanceFrameworkVersion(ctx, &types.QueryGetGovernanceFrameworkVersionRequest{Id: 1})
		require.NoError(t, err)
		require.Len(t, resp.Version.Documents, 2)

		// Preferred language "fr" — only the fr doc.
		respFR, err := qs.GetGovernanceFrameworkVersion(ctx, &types.QueryGetGovernanceFrameworkVersionRequest{Id: 1, PreferredLanguage: "fr"})
		require.NoError(t, err)
		require.Len(t, respFR.Version.Documents, 1)
		require.Equal(t, "fr", respFR.Version.Documents[0].Language)
	})
}

func TestQueryListGovernanceFrameworkVersions(t *testing.T) {
	t.Run("MOD-GF-QRY-2: exactly one of ecosystem_id/corporation must be set", func(t *testing.T) {
		k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
		qs := keeper.NewQueryServerImpl(k)
		_, err := qs.ListGovernanceFrameworkVersions(ctx, &types.QueryListGovernanceFrameworkVersionsRequest{})
		require.Error(t, err)
		_, err = qs.ListGovernanceFrameworkVersions(ctx, &types.QueryListGovernanceFrameworkVersionsRequest{
			EcosystemId: 1,
			Corporation: "x",
		})
		require.Error(t, err)
	})

	t.Run("MOD-GF-QRY-2-3: results ordered by ascending version, active_only respected", func(t *testing.T) {
		corp := &mockCorporation{
			view:  types.CorporationView{GroupPolicyAddress: testCorp, Language: "en", ActiveVersion: 0},
			found: true,
		}
		k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, corp)
		ms := keeper.NewMsgServerImpl(k)

		// Add v1 + activate, then add v2 (not activated).
		_, err := ms.AddGovernanceFrameworkDocument(ctx, validMsg(testCorp, testOperator, 0, 1))
		require.NoError(t, err)
		_, err = ms.IncreaseActiveGovernanceFrameworkVersion(ctx, &types.MsgIncreaseActiveGovernanceFrameworkVersion{
			Corporation: testCorp,
			Operator:    testOperator,
		})
		require.NoError(t, err)
		corp.view.ActiveVersion = 1 // simulate ecosystem-side bump
		_, err = ms.AddGovernanceFrameworkDocument(ctx, validMsg(testCorp, testOperator, 0, 2))
		require.NoError(t, err)

		qs := keeper.NewQueryServerImpl(k)
		all, err := qs.ListGovernanceFrameworkVersions(ctx, &types.QueryListGovernanceFrameworkVersionsRequest{Corporation: testCorp})
		require.NoError(t, err)
		require.Len(t, all.Versions, 2)
		require.Equal(t, int32(1), all.Versions[0].Version)
		require.Equal(t, int32(2), all.Versions[1].Version)

		activeOnly, err := qs.ListGovernanceFrameworkVersions(ctx, &types.QueryListGovernanceFrameworkVersionsRequest{
			Corporation: testCorp,
			ActiveOnly:  true,
		})
		require.NoError(t, err)
		require.Len(t, activeOnly.Versions, 1)
		require.Equal(t, int32(1), activeOnly.Versions[0].Version)
	})
}
```

- [ ] **Step 23.2: Run tests.**

  Run: `go test ./x/gf/keeper/... -run 'TestQuery' -v`
  Expected: PASS.

- [ ] **Step 23.3: Commit.**

```bash
git add x/gf/keeper/query_gfv_test.go
git commit -m "test(gf): MOD-GF-QRY-1 and QRY-2 happy paths"
```

---

## Task 24: Genesis round-trip test

**File:** `x/gf/types/genesis_test.go`

- [ ] **Step 24.1: Write the test.**

```go
package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana/x/gf/types"
)

func TestGenesisValidate(t *testing.T) {
	t.Run("default genesis is valid", func(t *testing.T) {
		require.NoError(t, types.DefaultGenesis().Validate())
	})

	t.Run("rejects GFV with both ecosystem_id and corporation set", func(t *testing.T) {
		gs := types.GenesisState{
			Params: types.DefaultParams(),
			Versions: []types.GovernanceFrameworkVersion{
				{Id: 1, EcosystemId: 1, Corporation: "x", Version: 1},
			},
		}
		require.ErrorIs(t, gs.Validate(), types.ErrInvalidSubject)
	})

	t.Run("rejects GFV with neither ecosystem_id nor corporation set", func(t *testing.T) {
		gs := types.GenesisState{
			Params: types.DefaultParams(),
			Versions: []types.GovernanceFrameworkVersion{
				{Id: 1, Version: 1},
			},
		}
		require.ErrorIs(t, gs.Validate(), types.ErrInvalidSubject)
	})

	t.Run("rejects GFD with dangling gfv_id", func(t *testing.T) {
		gs := types.GenesisState{
			Params: types.DefaultParams(),
			Versions: []types.GovernanceFrameworkVersion{
				{Id: 1, Corporation: "x", Version: 1},
			},
			Documents: []types.GovernanceFrameworkDocument{
				{Id: 1, GfvId: 999, Language: "en"},
			},
		}
		require.ErrorIs(t, gs.Validate(), types.ErrInvalidVersion)
	})
}
```

- [ ] **Step 24.2: Run.**

  Run: `go test ./x/gf/types/... -v`
  Expected: PASS.

- [ ] **Step 24.3: Commit.**

```bash
git add x/gf/types/genesis_test.go
git commit -m "test(gf): genesis validation cases"
```

---

## Phase 6 — Verification + PR

## Task 25: Full repo build, vet, and test pass

- [ ] **Step 25.1: Run a full build.**

  Run: `go build ./...`
  Expected: exit 0, no errors.

- [ ] **Step 25.2: Run vet.**

  Run: `go vet ./...`
  Expected: exit 0.

- [ ] **Step 25.3: Run all module tests.**

  Run: `go test ./x/gf/... ./testutil/...`
  Expected: PASS.

- [ ] **Step 25.4: Run the full test suite to ensure no regressions.**

  Run: `go test ./...`
  Expected: PASS. If any unrelated tests fail, capture output and stop — do not silently merge.

- [ ] **Step 25.5: Confirm app starts.**

  Run: `make build && ./build/veranad init test-node --chain-id verana-test --overwrite 2>&1 | tail -5`
  Expected: Genesis file generated, no panic from x/gf module init.

---

## Task 26: Create the pull request

- [ ] **Step 26.1: Push the branch.**

  Run: `git push -u origin feat/mod-gf-module`

- [ ] **Step 26.2: Open the PR.**

  Run:
  ```bash
  gh pr create --repo verana-labs/verana \
    --title "feat(gf)!: implement MOD-GF Governance Framework module per spec v4" \
    --body "$(cat <<'EOF'
## Summary

Implements the new MOD-GF module per spec v4. Closes #304.

- New `x/gf` Cosmos SDK module
- Entities: `GovernanceFrameworkVersion`, `GovernanceFrameworkDocument` (now owned by MOD-GF, polymorphic over Ecosystem | Corporation)
- Messages: `MsgAddGovernanceFrameworkDocument` (MOD-GF-MSG-1), `MsgIncreaseActiveGovernanceFrameworkVersion` (MOD-GF-MSG-2), `MsgUpdateParams`
- Queries: `GetGovernanceFrameworkVersion` (MOD-GF-QRY-1), `ListGovernanceFrameworkVersions` (MOD-GF-QRY-2)
- Interim `trEcosystemAdapter` lets x/tr serve as the EcosystemKeeper until issue #305 renames it. `stubCorporationKeeper` lets corporation-targeted GF calls compile until issue #303 (MOD-CO) lands.

## Test plan

- [x] Unit tests for MOD-GF-MSG-1 (happy + 4 precondition scenarios)
- [x] Unit tests for MOD-GF-MSG-2 (happy + 2 precondition scenarios)
- [x] Unit tests for queries (filtering, ordering, active_only)
- [x] Genesis validation cases
- [x] Full repo `go build ./...` and `go test ./...` pass
EOF
)"
  ```

- [ ] **Step 26.3: Move the linked issue to "In review" in the Verana project.**

  Run:
  ```bash
  PROJECT_ID="PVT_kwDOCqLJQs4BM4v6"
  STATUS_FIELD="PVTSSF_lADOCqLJQs4BM4v6zg8CnT0"
  IN_REVIEW="4cc61d42"
  ITEM_ID="PVTI_lADOCqLJQs4BM4v6zgtLRyI"
  gh project item-edit --id "$ITEM_ID" --field-id "$STATUS_FIELD" --project-id "$PROJECT_ID" --single-select-option-id "$IN_REVIEW"
  ```

---

## Self-Review Checklist

- [ ] Every spec section in MOD-GF (lines 2455–2606 of spec.md) has a corresponding task or check.
- [ ] All field names match between proto and Go: `EcosystemId` (not `EcosystemID`), `GfvId` (not `GFVID`), `DocLanguage` (not `Language` on Msg side but `Language` on entity side — both intentional, mirroring spec).
- [ ] Counter keys `"gfv"` and `"gfd"` consistent across keeper, genesis, and test setups.
- [ ] No placeholders, TODOs, or "implement later" comments in code blocks.
- [ ] Tests cover positive + negative paths for each Msg.
- [ ] Tests use isolated keepers (one keeper per `t.Run`).
- [ ] Adapter pattern is documented as interim (removed in #305 / #303).

---

## Out of scope

- AUTHZ-CHECK-5 (sequence issue 5; #308)
- TR → EC rename + EC's MSG-1 delegating to MOD-GF (sequence issue 3; #305)
- MOD-CO consuming MOD-GF (sequence issue 2; #303)
- Testharness integration journeys (sequence issue 13; #316)

These all land in subsequent PRs without touching the contract of MOD-GF.
