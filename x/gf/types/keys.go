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
	GFVersionByCorporationKey = collections.NewPrefix(6) // (corporation,  version) -> gfv_id
)
