package co_test

import (
	"encoding/json"
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
	groupkeeper "github.com/cosmos/cosmos-sdk/x/group/keeper"
	"github.com/stretchr/testify/require"

	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"

	modulev1 "github.com/verana-labs/verana/api/verana/co/module/v1"
	co "github.com/verana-labs/verana/x/co/module"
	"github.com/verana-labs/verana/x/co/types"
	dekeeper "github.com/verana-labs/verana/x/de/keeper"
	detypes "github.com/verana-labs/verana/x/de/types"
	gfkeeper "github.com/verana-labs/verana/x/gf/keeper"
	gftypes "github.com/verana-labs/verana/x/gf/types"
)

func TestAppModuleBasic_TrivialMethods(t *testing.T) {
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	b := co.NewAppModuleBasic(cdc)
	require.Equal(t, types.ModuleName, b.Name())
	// DefaultGenesis must round-trip through ValidateGenesis.
	gs := b.DefaultGenesis(cdc)
	require.NoError(t, b.ValidateGenesis(cdc, nil, gs))
	// Bad json fails ValidateGenesis.
	require.Error(t, b.ValidateGenesis(cdc, nil, json.RawMessage(`{"params": "not-an-object"}`)))
}

func TestAppModule_InitExportGenesis(t *testing.T) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

	// Build a GF keeper too (its concrete is required by the InitGenesis chain
	// indirectly — but only the CO keeper runs here).
	gfStoreKey := storetypes.NewKVStoreKey(gftypes.StoreKey)
	gfStateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	gfStateStore.MountStoreWithDB(gfStoreKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, gfStateStore.LoadLatestVersion())

	// Build a DE keeper too: MOD-CO wires it post-construction via the #308
	// cycle break (in.DeKeeper.SetCorporationKeeper), so it must be a real keeper.
	deStoreKey := storetypes.NewKVStoreKey(detypes.StoreKey)
	deStateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	deStateStore.MountStoreWithDB(deStoreKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, deStateStore.LoadLatestVersion())
	addrCodec := addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())

	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	in := co.ModuleInputs{
		StoreService:     runtime.NewKVStoreService(storeKey),
		Cdc:              cdc,
		Config:           &modulev1.Module{},
		Logger:           log.NewNopLogger(),
		DelegationKeeper: stubDelegation{},
		GroupKeeper:      groupkeeper.Keeper{},
		GFKeeper:         gfkeeper.NewKeeper(cdc, runtime.NewKVStoreService(gfStoreKey), log.NewNopLogger(), authority, stubGFDelegation{}),
		DeKeeper:         dekeeper.NewKeeper(runtime.NewKVStoreService(deStoreKey), cdc, addrCodec, authtypes.NewModuleAddress(govtypes.ModuleName)),
	}
	out := co.ProvideModule(in)
	require.NotNil(t, out.Module)
	require.Equal(t, authority, out.CoKeeper.GetAuthority())

	mod := co.NewAppModule(cdc, out.CoKeeper)
	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	mod.InitGenesis(ctx, cdc, mod.DefaultGenesis(cdc))
	require.JSONEq(t, string(mod.DefaultGenesis(cdc)), string(mod.ExportGenesis(ctx, cdc)))

	require.Equal(t, uint64(1), mod.ConsensusVersion())
	require.NoError(t, mod.BeginBlock(nil))
	require.NoError(t, mod.EndBlock(nil))
}

func TestProvideModule_CustomAuthority(t *testing.T) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	gfStoreKey := storetypes.NewKVStoreKey(gftypes.StoreKey)
	deStoreKey := storetypes.NewKVStoreKey(detypes.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(gfStoreKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(deStoreKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	gfk := gfkeeper.NewKeeper(cdc, runtime.NewKVStoreService(gfStoreKey), log.NewNopLogger(), authority, stubGFDelegation{})
	addrCodec := addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	dek := dekeeper.NewKeeper(runtime.NewKVStoreService(deStoreKey), cdc, addrCodec, authtypes.NewModuleAddress(govtypes.ModuleName))

	custom := authtypes.NewModuleAddress("custom").String()
	out := co.ProvideModule(co.ModuleInputs{
		StoreService:     runtime.NewKVStoreService(storeKey),
		Cdc:              cdc,
		Config:           &modulev1.Module{Authority: custom},
		Logger:           log.NewNopLogger(),
		DelegationKeeper: stubDelegation{},
		GroupKeeper:      groupkeeper.Keeper{},
		GFKeeper:         gfk,
		DeKeeper:         dek,
	})
	require.Equal(t, custom, out.CoKeeper.GetAuthority(), "explicit Authority must override gov default")
}
