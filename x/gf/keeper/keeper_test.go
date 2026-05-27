package keeper_test

import (
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

	keepertest "github.com/verana-labs/verana/testutil/keeper"
	"github.com/verana-labs/verana/x/gf/keeper"
	"github.com/verana-labs/verana/x/gf/types"
)

func TestNewKeeper_PanicOnInvalidAuthority(t *testing.T) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	require.Panics(t, func() {
		keeper.NewKeeper(
			cdc,
			runtime.NewKVStoreService(storeKey),
			log.NewNopLogger(),
			"not-a-bech32-address",
			mockDelegation{},
		)
	}, "NewKeeper must panic when authority is not a valid bech32 address")
}

func TestGetAuthority(t *testing.T) {
	k, _ := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
	require.Equal(t, authtypes.NewModuleAddress(govtypes.ModuleName).String(), k.GetAuthority())
}

func TestLogger(t *testing.T) {
	k, _ := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
	require.NotNil(t, k.Logger())
}

func TestGetNextID(t *testing.T) {
	k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})

	// First call on an empty counter starts at 1.
	id1, err := k.GetNextID(ctx, "gfv")
	require.NoError(t, err)
	require.Equal(t, uint64(1), id1)

	// Subsequent calls increment monotonically.
	id2, err := k.GetNextID(ctx, "gfv")
	require.NoError(t, err)
	require.Equal(t, uint64(2), id2)

	// Counters are per-entity-type — a different key starts fresh.
	idA, err := k.GetNextID(ctx, "gfd")
	require.NoError(t, err)
	require.Equal(t, uint64(1), idA)
}

// Ensure the constructor doesn't accidentally throw when the same context
// is reused for SetParams (smoke check, complementary to the test util).
func TestNewKeeper_SetParamsRoundTrip(t *testing.T) {
	k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
	require.NoError(t, k.SetParams(ctx, types.DefaultParams()))
	_ = sdk.UnwrapSDKContext(ctx)
	_ = cmtproto.Header{} // ensure cmtproto import is exercised (kept for explicitness alongside other tests)
}
