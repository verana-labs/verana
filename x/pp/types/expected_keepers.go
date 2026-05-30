package types

import (
	"context"
	"time"

	"cosmossdk.io/math"

	credentialschematypes "github.com/verana-labs/verana/x/cs/types"
	ectypes "github.com/verana-labs/verana/x/ec/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AccountKeeper defines the expected interface for the Account module.
type AccountKeeper interface {
	GetAccount(context.Context, sdk.AccAddress) sdk.AccountI // only used for simulation
	// Methods imported from account should be defined here
}

// BankKeeper defines the expected interface for the Bank module.
type BankKeeper interface {
	SpendableCoins(context.Context, sdk.AccAddress) sdk.Coins
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoins(ctx context.Context, from, to sdk.AccAddress, amt sdk.Coins) error
	HasBalance(ctx context.Context, addr sdk.AccAddress, amt sdk.Coin) bool
}

// ParamSubspace defines the expected Subspace interface for parameters.
type ParamSubspace interface {
	Get(context.Context, []byte, interface{})
	Set(context.Context, []byte, interface{})
}

type CredentialSchemaKeeper interface {
	GetCredentialSchemaById(ctx sdk.Context, id uint64) (credentialschematypes.CredentialSchema, error)
}

// EcosystemKeeper defines the expected ecosystem keeper.
// Replaces the legacy TrustRegistryKeeper post-MOD-EC rename: x/pp needs to
// read the Ecosystem row (ec.CorporationId) to authorize CredentialSchema
// owners, and still needs trust-unit pricing for fee math.
type EcosystemKeeper interface {
	GetEcosystem(ctx context.Context, id uint64) (ectypes.Ecosystem, error)
	GetTrustUnitPrice(ctx sdk.Context) uint64
}

// CorporationView is the read shape MOD-PERM needs about a Corporation
// subject for AUTHZ-CHECK-5: turn the signing `corporation` policy_address
// into the uint64 co.Id used to validate ec.CorporationId ownership.
type CorporationView struct {
	Id            uint64
	PolicyAddress string
}

// CorporationKeeper backs AUTHZ-CHECK-5 for MOD-PP messages and the
// corporation_id <-> policy_address resolution the Participant entity needs:
// participants persist corporation_id (uint64), but fund-flows (trust deposit,
// feegrant, slashing) operate on the Corporation policy_address account.
type CorporationKeeper interface {
	ResolveByPolicyAddress(ctx context.Context, policyAddress string) (CorporationView, bool)
	ResolveByID(ctx context.Context, id uint64) (CorporationView, bool)
}

// TrustDepositKeeper defines the expected interface for the Trust Deposit module.
type TrustDepositKeeper interface {
	AdjustTrustDeposit(ctx sdk.Context, account string, augend int64, reason string) error
	AdjustTrustDepositOnBehalf(ctx sdk.Context, account string, funder sdk.AccAddress, amount int64) error
	GetTrustDepositRate(ctx sdk.Context) math.LegacyDec
	GetUserAgentRewardRate(ctx sdk.Context) math.LegacyDec
	GetWalletUserAgentRewardRate(ctx sdk.Context) math.LegacyDec
	BurnEcosystemSlashedTrustDeposit(ctx sdk.Context, account string, amount uint64) error
}

// DigestKeeper defines the expected interface for the Digest (DI) module.
// Used by [MOD-PERM-MSG-10] to persist credential digests discovered during
// permission-session creation. Called keeper-to-keeper (no signer check) per
// spec [MOD-DI-MSG-1] header: "This method can be called directly by Create
// or Update Participant Session module with no checks."
type DigestKeeper interface {
	StoreDigestModuleCall(ctx context.Context, authority, digest, digestAlgorithm string) error
}

// DelegationKeeper defines the expected interface for the Delegation Engine (DE) module.
// Used to perform [AUTHZ-CHECK] for (authority, operator) pairs.
type DelegationKeeper interface {
	CheckOperatorAuthorization(ctx context.Context, authority string, operator string, msgTypeURL string, now time.Time) error
	// CheckVSOperatorAuthorization checks if a VS operator is authorized to act on behalf of the authority.
	// [AUTHZ-CHECK-3] A VSOperatorAuthorization entry must exist where authority and vs_operator match.
	CheckVSOperatorAuthorization(ctx context.Context, authority string, vsOperator string) error
	// VSOA CRUD methods (storage in DE, orchestration in Perm)
	AddPermToVSOA(ctx context.Context, authority, vsOperator string, permID uint64) error
	RemovePermFromVSOA(ctx context.Context, authority, vsOperator string, permID uint64) ([]uint64, error)
	GetVSOAPermissions(ctx context.Context, authority, vsOperator string) ([]uint64, error)
	HasOperatorAuthorization(ctx context.Context, authority, operator string) (bool, error)
	// Fee grant methods
	GrantFeeAllowance(ctx context.Context, authority string, grantee string, msgTypes []string, expiration *time.Time, spendLimit sdk.Coins, period *time.Duration) error
	RevokeFeeAllowance(ctx context.Context, authority string, grantee string) error
}
