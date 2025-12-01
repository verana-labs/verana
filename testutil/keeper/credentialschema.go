package keeper

import (
	"context"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana/x/cs/keeper"
	"github.com/verana-labs/verana/x/cs/types"
	trtypes "github.com/verana-labs/verana/x/tr/types"
)

// MockBankKeeper is a mock implementation of types.BankKeeper
type MockBankKeeper struct {
	bankBalances map[string]sdk.Coins
}

func (k *MockBankKeeper) SendCoins(ctx context.Context, from, to sdk.AccAddress, amt sdk.Coins) error {
	// For testing purposes, just return nil (success)
	return nil
}

func (k *MockBankKeeper) HasBalance(ctx context.Context, addr sdk.AccAddress, amt sdk.Coin) bool {
	// For testing purposes, just return nil (success)
	return true
}

func (k *MockBankKeeper) BurnCoins(ctx context.Context, name string, amt sdk.Coins) error {
	// For testing purposes, just return nil (success)
	return nil
}

func (k *MockBankKeeper) SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	// For testing purposes, just return nil (success)
	return nil
}

func (k *MockBankKeeper) SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error {
	// For testing purposes, just return nil (success)
	return nil
}

func (k *MockBankKeeper) SpendableCoins(ctx context.Context, address sdk.AccAddress) sdk.Coins {
	// For testing purposes, return empty coins
	return sdk.Coins{}
}

func (k *MockBankKeeper) GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	// For testing purposes, return zero coin
	return sdk.NewCoin(denom, math.ZeroInt())
}

func (k *MockBankKeeper) GetAllBalances(ctx context.Context, addr sdk.AccAddress) sdk.Coins {
	// For testing purposes, return empty coins
	return sdk.Coins{}
}

func NewMockBankKeeper() *MockBankKeeper {
	return &MockBankKeeper{
		bankBalances: make(map[string]sdk.Coins),
	}
}

// Implement required methods from types.BankKeeper interface
func (k *MockBankKeeper) SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	return nil
}

// MockTrustRegistryKeeper is a mock implementation of types.TrustRegistryKeeper
type MockTrustRegistryKeeper struct {
	trustRegistries map[uint64]trtypes.TrustRegistry
}

func (k *MockTrustRegistryKeeper) GetTrustUnitPrice(ctx sdk.Context) uint64 {
	return 1
}

func NewMockTrustRegistryKeeper() *MockTrustRegistryKeeper {
	return &MockTrustRegistryKeeper{
		trustRegistries: make(map[uint64]trtypes.TrustRegistry),
	}
}

func (k *MockTrustRegistryKeeper) GetTrustRegistry(ctx sdk.Context, id uint64) (trtypes.TrustRegistry, error) {
	if tr, ok := k.trustRegistries[id]; ok {
		return tr, nil
	}
	return trtypes.TrustRegistry{}, trtypes.ErrTrustRegistryNotFound
}

func (k *MockTrustRegistryKeeper) CreateMockTrustRegistry(creator string, did string) uint64 {
	id := uint64(len(k.trustRegistries) + 1)
	k.trustRegistries[id] = trtypes.TrustRegistry{
		Id:            id,
		Did:           did,
		Controller:    creator,
		ActiveVersion: 1,
		Language:      "en",
	}
	return id
}

func CredentialschemaKeeper(t testing.TB) (keeper.Keeper, *MockTrustRegistryKeeper, sdk.Context) { // Changed return types
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	// Create mock keepers
	bankKeeper := NewMockBankKeeper()
	trustRegistryKeeper := NewMockTrustRegistryKeeper()
	mockTrustDepositKeeper := &MockTrustDepositKeeper{}
	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority.String(),
		bankKeeper,
		trustRegistryKeeper,
		mockTrustDepositKeeper,
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	// Initialize params
	if err := k.SetParams(ctx, types.DefaultParams()); err != nil {
		panic(err)
	}

	return k, trustRegistryKeeper, ctx // Return the mock keeper
}
