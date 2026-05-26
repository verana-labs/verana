package keeper_test

import (
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana/testutil/keeper"
	"github.com/verana-labs/verana/x/co/keeper"
	"github.com/verana-labs/verana/x/co/types"
)

func TestNewKeeper_PanicsOnInvalidAuthority(t *testing.T) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	require.Panics(t, func() {
		keeper.NewKeeper(
			cdc,
			runtime.NewKVStoreService(storeKey),
			log.NewNopLogger(),
			"not-a-bech32",
			&mockDelegation{},
			&mockGroup{},
			&mockGF{},
		)
	}, "NewKeeper must panic when authority is not bech32")
}

func TestGetAuthority(t *testing.T) {
	k, _ := keepertest.CoKeeper(t, &mockDelegation{}, &mockGroup{}, &mockGF{})
	require.Equal(t, authtypes.NewModuleAddress(govtypes.ModuleName).String(), k.GetAuthority())
}

func TestLogger_HasModuleScope(t *testing.T) {
	k, _ := keepertest.CoKeeper(t, &mockDelegation{}, &mockGroup{}, &mockGF{})
	require.NotNil(t, k.Logger())
}

func TestGetNextID_SequentialAndPerEntity(t *testing.T) {
	k, ctx := keepertest.CoKeeper(t, &mockDelegation{}, &mockGroup{}, &mockGF{})
	for i := uint64(1); i <= 3; i++ {
		got, err := k.GetNextID(ctx, "co")
		require.NoError(t, err)
		require.Equal(t, i, got)
	}
	// Counters are namespaced by entity name.
	other, err := k.GetNextID(ctx, "other")
	require.NoError(t, err)
	require.Equal(t, uint64(1), other)
}
