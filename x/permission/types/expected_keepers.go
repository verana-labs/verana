package types

import (
	"context"

	"cosmossdk.io/math"

	credentialschematypes "github.com/verana-labs/verana-blockchain/x/credentialschema/types"
	trustregistrytypes "github.com/verana-labs/verana-blockchain/x/trustregistry/types"

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

// TrustRegistryKeeper defines the expected trust registry keeper
type TrustRegistryKeeper interface {
	GetTrustRegistry(ctx sdk.Context, id uint64) (trustregistrytypes.TrustRegistry, error)
	GetTrustUnitPrice(ctx sdk.Context) uint64
}

// TrustDepositKeeper defines the expected interface for the Trust Deposit module.
type TrustDepositKeeper interface {
	AdjustTrustDeposit(ctx sdk.Context, account string, augend int64) error
	GetTrustDepositRate(ctx sdk.Context) math.LegacyDec
	GetUserAgentRewardRate(ctx sdk.Context) math.LegacyDec
	GetWalletUserAgentRewardRate(ctx sdk.Context) math.LegacyDec
	BurnEcosystemSlashedTrustDeposit(ctx sdk.Context, account string, amount uint64) error
}
