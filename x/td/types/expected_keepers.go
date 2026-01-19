package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	groupkeeper "github.com/cosmos/cosmos-sdk/x/group/keeper"
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
	SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
	GetAllBalances(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	BurnCoins(ctx context.Context, name string, amt sdk.Coins) error
}

// ParamSubspace defines the expected Subspace interface for parameters.
type ParamSubspace interface {
	Get(context.Context, []byte, interface{})
	Set(context.Context, []byte, interface{})
}

// MintKeeper defines the expected interface for the Mint module.
// Mint keeper has Params collections.Item[types.Params] where Params has BlocksPerYear field
type MintKeeper interface {
	// GetBlocksPerYear returns the blocks_per_year parameter from mint module
	GetBlocksPerYear(ctx context.Context) (uint64, error)
}

// GroupKeeper is an alias for the group keeper
// Used for auto-executing group proposals in EndBlocker.
type GroupKeeper = groupkeeper.Keeper
