package types

import (
	"context"
	"time"

	"cosmossdk.io/math"

	credentialschematypes "github.com/verana-labs/verana/x/cs/types"
	detypes "github.com/verana-labs/verana/x/de/types"
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

// CorporationView is the read shape MOD-PP needs about a Corporation
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
// Used by [MOD-PP-MSG-10] to persist credential digests discovered during
// participant-session creation. Called keeper-to-keeper (no signer check) per
// spec [MOD-DI-MSG-1] header: "This method can be called directly by Create
// or Update Participant Session module with no checks."
type DigestKeeper interface {
	StoreDigestModuleCall(ctx context.Context, authority, digest, digestAlgorithm string) error
}

// DelegationKeeper defines the expected interface for the Delegation Engine (DE)
// module per spec v4-rc2. The caller resolves the signing corporation account to
// its co.id (via AUTHZ-CHECK-5) before invoking the VSOA lifecycle / check
// methods, which take corporation_id (uint64).
type DelegationKeeper interface {
	// [AUTHZ-CHECK-1] operator-delegation check (corporation account + operator).
	CheckOperatorAuthorization(ctx context.Context, authority string, operator string, msgTypeURL string, now time.Time) error
	// [AUTHZ-CHECK-3] record-based VS operator authorization check on a participant.
	CheckVSOperatorAuthorizationOnParticipant(ctx context.Context, corporationID uint64, operator string, participantID uint64, msgType string) error
	// [AUTHZ-CHECK-4] record-based VS operator fee grant check.
	CheckVSOperatorFeeGrant(ctx context.Context, participantID uint64) error
	// [MOD-DE-MSG-5] grant a VS operator authorization record (module call).
	GrantVSOperatorAuthorization(ctx context.Context, corporationID uint64, vsOperator string, record detypes.ParticipantAuthorizationRecord) error
	// [MOD-DE-MSG-6] revoke a VS operator authorization record by participant id.
	RevokeVSOperatorAuthorization(ctx context.Context, participantID uint64) error
	// [MOD-DE-MSG-9] update a record's expiration by participant id.
	UpdateVSOperatorAuthorizationExpiration(ctx context.Context, participantID uint64, newExpiration time.Time) error
}
