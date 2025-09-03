package types

import "cosmossdk.io/collections"

const (
	// ModuleName defines the module name
	ModuleName = "perm"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_permission"
)

const (
	BondDenom = "uvna"
)

var (
	ParamsKey            = []byte("p_permission")
	PermissionKey        = collections.NewPrefix(0)
	PermissionCounterKey = collections.NewPrefix(1)
	PermissionSessionKey = collections.NewPrefix(2)
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}
