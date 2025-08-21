package types

import "cosmossdk.io/collections"

const (
	// ModuleName defines the module name
	ModuleName = "td"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_trustdeposit"

	RouterKey = ModuleName
)

var (
	ParamsKey       = []byte("p_trustdeposit")
	TrustDepositKey = collections.NewPrefix(1)
)

const (
	BondDenom = "uvna"
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}
