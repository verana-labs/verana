package types

import "cosmossdk.io/collections"

const (
	// ModuleName defines the module name
	ModuleName = "dd"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_diddirectory"
)

var (
	ParamsKey       = []byte("p_diddirectory")
	DIDDirectoryKey = collections.NewPrefix(1)
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}
