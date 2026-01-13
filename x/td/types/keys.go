package types

import "cosmossdk.io/collections"

const (
	// ModuleName defines the module name
	ModuleName = "td"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_trustdeposit"

	RouterKey             = ModuleName
	YieldIntermediatePool = "yield_intermediate_pool"
)

var (
	ParamsKey       = []byte("p_trustdeposit")
	TrustDepositKey = collections.NewPrefix(1)
	DustKey         = collections.NewPrefix(2)

	// Anchor-based POC keys
	AnchorKey            = collections.NewPrefix(3) // anchor_id -> Anchor
	VerifiableServiceKey = collections.NewPrefix(4) // operator_account -> VerifiableService
	OperatorAllowanceKey = collections.NewPrefix(5) // (anchor_id, operator) -> OperatorAllowance
)

const (
	BondDenom = "uvna"
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}
