package ecosystem_test

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
	"github.com/stretchr/testify/require"

	ecosystem "github.com/verana-labs/verana/x/ec/module"
	"github.com/verana-labs/verana/x/ec/keeper"
	"github.com/verana-labs/verana/x/ec/types"
	gftypes "github.com/verana-labs/verana/x/gf/types"
)

type genStub struct{}

func (genStub) CheckOperatorAuthorization(_ context.Context, _, _, _ string, _ time.Time) error {
	return nil
}
func (genStub) ResolveByPolicyAddress(_ context.Context, _ string) (types.CorporationView, bool) {
	return types.CorporationView{}, false
}
func (genStub) GetByID(_ context.Context, _ uint64) (types.CorporationView, bool) {
	return types.CorporationView{}, false
}
func (genStub) CreateInitialGFVersionForEcosystem(_ context.Context, _ uint64, _, _, _ string) error {
	return nil
}
func (genStub) ListVersionsByEcosystem(_ context.Context, _ uint64, _ uint32, _ bool, _ string) ([]gftypes.GovernanceFrameworkVersionWithDocs, error) {
	return nil, nil
}

func newKeeper(t *testing.T) (keeper.Keeper, sdk.Context) {
	t.Helper()
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	stub := genStub{}
	k := keeper.NewKeeper(cdc, runtime.NewKVStoreService(storeKey), log.NewNopLogger(), authority, stub, stub, stub)
	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	return k, ctx
}

// TestGenesis_RoundTrip pins that InitGenesis + ExportGenesis is lossless
// for the EC module post-rename: only Ecosystem rows + the "ec" counter
// (GFV/GFD live in x/gf and MUST NOT appear here).
func TestGenesis_RoundTrip(t *testing.T) {
	k, ctx := newKeeper(t)

	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	gs := &types.GenesisState{
		Params: types.DefaultParams(),
		Ecosystems: []types.Ecosystem{
			{Id: 1, Did: "did:example:1", CorporationId: 10, Created: now, Modified: now, Language: "en", ActiveVersion: 1},
			{Id: 2, Did: "did:example:2", CorporationId: 10, Created: now, Modified: now, Language: "en", ActiveVersion: 1, Archived: true},
		},
		Counters: []types.Counter{{EntityType: "ec", Value: 2}},
	}
	require.NoError(t, gs.Validate())

	ecosystem.InitGenesis(ctx, k, *gs)

	got := ecosystem.ExportGenesis(ctx, k)
	require.Equal(t, gs.Params, got.Params)
	require.Len(t, got.Ecosystems, 2)
	require.Equal(t, gs.Ecosystems[0], got.Ecosystems[0])
	require.Equal(t, gs.Ecosystems[1], got.Ecosystems[1])
	require.Len(t, got.Counters, 1)
	require.Equal(t, "ec", got.Counters[0].EntityType)
	require.Equal(t, uint64(2), got.Counters[0].Value)
}
