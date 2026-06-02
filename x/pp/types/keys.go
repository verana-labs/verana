package types

import "cosmossdk.io/collections"

const (
	// ModuleName defines the module name
	ModuleName = "pp"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_pp"
)

const (
	BondDenom = "uvna"
)

var (
	ParamsKey             = []byte("p_pp")
	ParticipantKey        = collections.NewPrefix(0)
	ParticipantCounterKey = collections.NewPrefix(1)
	ParticipantSessionKey = collections.NewPrefix(2)
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}
