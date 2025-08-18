package types

import "cosmossdk.io/collections"

const (
	// ModuleName defines the module name
	ModuleName = "tr"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// GovModuleName duplicates the gov module's name to avoid a dependency with x/gov.
	// It should be synced with the gov module's name if it is ever changed.
	// See: https://github.com/cosmos/cosmos-sdk/blob/v0.52.0-beta.2/x/gov/types/keys.go#L9
	GovModuleName = "gov"
)

// ParamsKey is the prefix to retrieve all Params
var (
	ParamsKey                      = collections.NewPrefix("p_tr")
	TrustRegistryKey               = collections.NewPrefix(1) // Primary Trust Registry storage - using ID as key
	TrustRegistryDIDIndex          = collections.NewPrefix(2) // Index for DID lookups
	GovernanceFrameworkVersionKey  = collections.NewPrefix(3)
	GovernanceFrameworkDocumentKey = collections.NewPrefix(4)
	CounterKey                     = collections.NewPrefix(5)
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}
