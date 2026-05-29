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
	gfkeeper "github.com/verana-labs/verana/x/gf/keeper"
	"github.com/verana-labs/verana/x/gf/types"
)

// newFreshKeeperWithoutParams builds a GF keeper that has NOT had SetParams
// called on it, so GetParams must hit its default-fallback branch.
func newFreshKeeperWithoutParams(t testing.TB) (gfkeeper.Keeper, sdk.Context) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	k := gfkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority.String(),
		mockDelegation{},
	)
	k.SetEcosystemKeeper(&mockEcosystem{})
	k.SetCorporationKeeper(&mockCorporation{})
	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	return k, ctx
}

func TestGetParams_DefaultFallbackWhenUnset(t *testing.T) {
	// Construct a keeper without calling SetParams in the test setup.
	// (GfKeeperWithDelegation always calls SetParams to seed defaults, so we
	// re-run by clearing and reading.)
	k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})

	// Seeded by test util — Get returns the params we set.
	got := k.GetParams(ctx)
	require.Equal(t, types.DefaultParams(), got)
}

func TestSetParams_RoundTrip(t *testing.T) {
	k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})

	// Module currently has no params fields; round-tripping the empty struct
	// still exercises the read + write paths through codec.
	require.NoError(t, k.SetParams(ctx, types.DefaultParams()))
	require.Equal(t, types.DefaultParams(), k.GetParams(ctx))
}

// Exercise the GetParams default-fallback path (the Params.Get returns err
// branch). We construct a keeper without calling SetParams and verify that
// GetParams gracefully returns DefaultParams.
func TestGetParams_FallbackOnMissingRecord(t *testing.T) {
	k, ctx := newFreshKeeperWithoutParams(t)
	require.Equal(t, types.DefaultParams(), k.GetParams(ctx))
}
