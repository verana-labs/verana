package types

import "cosmossdk.io/collections"

const (
	// ModuleName defines the module name.
	ModuleName = "co"

	// StoreKey defines the primary module store key.
	StoreKey = ModuleName

	// GovModuleName duplicates the x/gov module name to avoid a dependency.
	// MUST be synced if upstream renames it.
	GovModuleName = "gov"
)

var (
	ParamsKey                      = collections.NewPrefix(1)
	CorporationKey                 = collections.NewPrefix(2) // id → Corporation
	CorporationByPolicyAddressKey  = collections.NewPrefix(3) // policy_address → id (reverse index for AUTHZ-CHECK-5)
	CorporationByDIDKey            = collections.NewPrefix(4) // did → id (for uniqueness check)
	CounterKey                     = collections.NewPrefix(5)
)
