package types

import "cosmossdk.io/collections"

const (
	// ModuleName defines the module name
	ModuleName = "de"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// GovModuleName duplicates the gov module's name to avoid a dependency with x/gov.
	// It should be synced with the gov module's name if it is ever changed.
	// See: https://github.com/cosmos/cosmos-sdk/blob/v0.52.0-beta.2/x/gov/types/keys.go#L9
	GovModuleName = "gov"
)

var (
	// ParamsKey is the prefix to retrieve all Params
	ParamsKey = collections.NewPrefix("p_de")

	// OperatorAuthorizationKey is the prefix for OperatorAuthorization storage,
	// keyed by its own uint64 id (spec v4-rc2).
	OperatorAuthorizationKey = collections.NewPrefix("oa_de")
	// OperatorAuthorizationByCorpOpKey is the secondary index
	// (corporation_id, operator) -> OperatorAuthorization.id.
	OperatorAuthorizationByCorpOpKey = collections.NewPrefix("oa_corpop_de")
	// OperatorAuthorizationSeqKey backs the OperatorAuthorization id counter.
	OperatorAuthorizationSeqKey = collections.NewPrefix("oa_seq_de")

	// FeeGrantKey is the prefix for FeeGrant storage, keyed by the composite
	// (grantor_corporation_id, grantee).
	FeeGrantKey = collections.NewPrefix("fg_de")

	// VSOperatorAuthorizationKey is the prefix for VSOperatorAuthorization
	// storage, keyed by its own uint64 id (spec v4-rc2).
	VSOperatorAuthorizationKey = collections.NewPrefix("vsoa_de")
	// VSOAByCorpOpKey is the secondary index
	// (corporation_id, vs_operator) -> VSOperatorAuthorization.id.
	VSOAByCorpOpKey = collections.NewPrefix("vsoa_corpop_de")
	// VSOAByParticipantKey is the tertiary index participant_id -> vsoa.id, used
	// for global participant_id uniqueness and MSG-6 / MSG-9 lookups.
	VSOAByParticipantKey = collections.NewPrefix("vsoa_part_de")
	// VSOASeqKey backs the VSOperatorAuthorization id counter.
	VSOASeqKey = collections.NewPrefix("vsoa_seq_de")
)
