package keeper_test

import (
	"context"
	"testing"
	"time"

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
	groupkeeper "github.com/cosmos/cosmos-sdk/x/group/keeper"
	"github.com/stretchr/testify/require"

	cotypes "github.com/verana-labs/verana/x/co/types"

	"github.com/verana-labs/verana/x/co/keeper"
	gfkeeper "github.com/verana-labs/verana/x/gf/keeper"
	gftypes "github.com/verana-labs/verana/x/gf/types"
)

// minimal stub keepers for building a real gfkeeper.Keeper instance.
type adapterStubDel struct{}

func (adapterStubDel) CheckOperatorAuthorization(_ context.Context, _, _, _ string, _ time.Time) error {
	return nil
}

type adapterStubEco struct{}

func (adapterStubEco) GetEcosystemView(_ context.Context, _ uint64) (gftypes.EcosystemView, bool) {
	return gftypes.EcosystemView{}, false
}
func (adapterStubEco) SetEcosystemActiveVersion(_ context.Context, _ uint64, _ uint32) error {
	return nil
}

func newGFKeeperForAdapter(t *testing.T) (gfkeeper.Keeper, sdk.Context) {
	t.Helper()
	storeKey := storetypes.NewKVStoreKey(gftypes.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	gfk := gfkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority,
		adapterStubDel{},
	)
	gfk.SetEcosystemKeeper(adapterStubEco{})
	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	return gfk, ctx
}

// TestGroupKeeperAdapter_ImplementsInterface verifies the SDK group keeper
// adapter satisfies the MOD-CO interface. We don't run real x/group methods
// here (they need the full SDK store wiring); the contract test is that
// depinject can resolve types.GroupKeeper from the SDK keeper.
func TestGroupKeeperAdapter_ImplementsInterface(t *testing.T) {
	var sdkK groupkeeper.Keeper
	var iface cotypes.GroupKeeper = keeper.NewGroupKeeperAdapter(sdkK)
	require.NotNil(t, iface)
}

func TestGFKeeperAdapter_PassThrough(t *testing.T) {
	gfk, ctx := newGFKeeperForAdapter(t)
	a := keeper.NewGFKeeperAdapter(gfk)

	// Seed an initial GF version for corporation_id=42.
	require.NoError(t, a.CreateInitialGFVersionForCorporation(ctx, 42, "en", "https://x.example/c.pdf", "sha256-aGVsbG8="))

	versions, err := a.ListVersionsByCorporation(ctx, 42, 1, false, "")
	require.NoError(t, err)
	require.Len(t, versions, 1)
	require.Equal(t, uint32(1), versions[0].Version)
	require.Equal(t, uint64(42), versions[0].CorporationId)

	versions, err = a.ListVersionsByCorporation(ctx, 42, 99, true, "")
	require.NoError(t, err)
	require.Empty(t, versions)

	_, err = a.ListVersionsByCorporation(ctx, 0, 1, false, "")
	require.Error(t, err)
}
