package types

import (
	"cosmossdk.io/collections"
)

const (
	// ModuleName defines the module name
	ModuleName = "tr"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_trustregistry"
)

var (
	ParamsKey                      = collections.NewPrefix(0)
	TrustRegistryKey               = collections.NewPrefix(1) // Primary Trust Registry storage - using ID as key
	TrustRegistryDIDIndex          = collections.NewPrefix(2) // Index for DID lookups
	GovernanceFrameworkVersionKey  = collections.NewPrefix(3)
	GovernanceFrameworkDocumentKey = collections.NewPrefix(4)
	CounterKey                     = collections.NewPrefix(5)
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}
