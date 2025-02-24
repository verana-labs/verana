package keeper

import (
	"context"
	"testing"

	"cosmossdk.io/log"
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

	"github.com/verana-labs/verana-blockchain/x/credentialschema/keeper"
	"github.com/verana-labs/verana-blockchain/x/credentialschema/types"
	trtypes "github.com/verana-labs/verana-blockchain/x/trustregistry/types"
)

// MockBankKeeper is a mock implementation of types.BankKeeper
type MockBankKeeper struct {
	bankBalances map[string]sdk.Coins
}

func (k *MockBankKeeper) BurnCoins(ctx context.Context, name string, amt sdk.Coins) error {
	//TODO implement me
	panic("implement me")
}

func (k *MockBankKeeper) SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	//TODO implement me
	panic("implement me")
}

func (k *MockBankKeeper) SendCoinsFromModuleToModule(ctx sdk.Context, senderModule, recipientModule string, amt sdk.Coins) error {
	//TODO implement me
	panic("implement me")
}

func (k *MockBankKeeper) SpendableCoins(ctx context.Context, address sdk.AccAddress) sdk.Coins {
	//TODO implement me
	panic("implement me")
}

func (k *MockBankKeeper) GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	//TODO implement me
	panic("implement me")
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

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority.String(),
		bankKeeper,
		trustRegistryKeeper,
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	// Initialize params
	if err := k.SetParams(ctx, types.DefaultParams()); err != nil {
		panic(err)
	}

	return k, trustRegistryKeeper, ctx // Return the mock keeper
}
