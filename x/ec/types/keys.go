package types

import "cosmossdk.io/collections"

const (
	// ModuleName is the module name (and store key).
	ModuleName = "ec"

	// StoreKey defines the primary module store key.
	StoreKey = ModuleName
)

var (
	ParamsKey             = collections.NewPrefix(1)
	EcosystemKey          = collections.NewPrefix(2) // id → Ecosystem
	EcosystemByDIDCorpKey = collections.NewPrefix(3) // (did, corporation_id) → id (per-Ecosystem consistency-invariant index)
	CounterKey            = collections.NewPrefix(4)
)
