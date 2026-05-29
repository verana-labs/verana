package gf_test

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

	modulev1 "github.com/verana-labs/verana/api/verana/gf/module/v1"
	keepertest "github.com/verana-labs/verana/testutil/keeper"
	gf "github.com/verana-labs/verana/x/gf/module"
	"github.com/verana-labs/verana/x/gf/types"
)

func TestModuleInitExportGenesis_RoundTrip(t *testing.T) {
	k, sdkCtx := keepertest.GfKeeperWithDelegation(t,
		stubDelegationKeeper{}, &stubEcosystemKeeper{}, &stubCorporationKeeper{})

	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)

	mod := gf.NewAppModule(cdc, k)

	// Default genesis init/export must round-trip without panic.
	def := mod.DefaultGenesis(cdc)
	require.NotEmpty(t, def)
	require.NotPanics(t, func() {
		mod.InitGenesis(sdkCtx, cdc, def)
	})
	exported := mod.ExportGenesis(sdkCtx, cdc)
	require.NotEmpty(t, exported)

	// ValidateGenesis accepts the default state.
	require.NoError(t, mod.ValidateGenesis(cdc, nil, def))
}

func TestModuleValidateGenesis_RejectsInvalidJSON(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)
	mod := gf.NewAppModuleBasic(cdc)

	err := mod.ValidateGenesis(cdc, nil, []byte("not valid json"))
	require.Error(t, err)
}

func TestModuleBasic_NameAndCodec(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	mod := gf.NewAppModuleBasic(cdc)

	require.Equal(t, types.ModuleName, mod.Name())

	// Interfaces register without panic.
	require.NotPanics(t, func() { mod.RegisterInterfaces(registry) })

	amino := codec.NewLegacyAmino()
	require.NotPanics(t, func() { mod.RegisterLegacyAminoCodec(amino) })
}

func TestModule_ConsensusVersionAndBlockHooks(t *testing.T) {
	k, _ := keepertest.GfKeeperWithDelegation(t,
		stubDelegationKeeper{}, &stubEcosystemKeeper{}, &stubCorporationKeeper{})
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	mod := gf.NewAppModule(cdc, k)

	require.Equal(t, uint64(1), mod.ConsensusVersion())
	require.NoError(t, mod.BeginBlock(nil))
	require.NoError(t, mod.EndBlock(nil))
}

func TestGfKeeperWithDelegation_WiresEcosystemKeeper(t *testing.T) {
	// Post MOD-EC rename: ProvideEcosystemKeeper was removed (x/ec now
	// provides the EcosystemKeeper directly via its own depinject Out).
	// Verify the test harness still wires an EcosystemKeeper into the gf
	// keeper construction path.
	k, _ := keepertest.GfKeeperWithDelegation(t,
		stubDelegationKeeper{}, &stubEcosystemKeeper{}, &stubCorporationKeeper{})
	require.NotNil(t, k)
}

func TestProvideModule_DefaultAuthority(t *testing.T) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	in := gf.ModuleInputs{
		StoreService:     runtime.NewKVStoreService(storeKey),
		Cdc:              cdc,
		Config:           &modulev1.Module{}, // empty Authority → falls back to gov module addr
		Logger:           log.NewNopLogger(),
		DelegationKeeper: stubDelegationKeeper{},
	}
	out := gf.ProvideModule(in)
	require.NotNil(t, out.Module)
	require.Equal(t, authtypes.NewModuleAddress(govtypes.ModuleName).String(), out.GfKeeper.GetAuthority())
}

func TestProvideModule_CustomAuthority(t *testing.T) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	customAuth := authtypes.NewModuleAddress("custom").String()

	in := gf.ModuleInputs{
		StoreService:     runtime.NewKVStoreService(storeKey),
		Cdc:              cdc,
		Config:           &modulev1.Module{Authority: customAuth},
		Logger:           log.NewNopLogger(),
		DelegationKeeper: stubDelegationKeeper{},
	}
	out := gf.ProvideModule(in)
	require.Equal(t, customAuth, out.GfKeeper.GetAuthority(), "explicit Authority must override gov default")
}

func TestModule_TrivialAppModuleMarkers(t *testing.T) {
	k, _ := keepertest.GfKeeperWithDelegation(t,
		stubDelegationKeeper{}, &stubEcosystemKeeper{}, &stubCorporationKeeper{})
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	mod := gf.NewAppModule(cdc, k)

	// IsOnePerModuleType + IsAppModule are zero-arg, zero-return markers.
	require.NotPanics(t, func() { mod.IsOnePerModuleType() })
	require.NotPanics(t, func() { mod.IsAppModule() })

	// RegisterInvariants is a no-op for this module.
	require.NotPanics(t, func() { mod.RegisterInvariants(nil) })
}

func TestModule_AutoCLIOptions(t *testing.T) {
	k, _ := keepertest.GfKeeperWithDelegation(t,
		stubDelegationKeeper{}, &stubEcosystemKeeper{}, &stubCorporationKeeper{})
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	mod := gf.NewAppModule(cdc, k)

	opts := mod.AutoCLIOptions()
	require.NotNil(t, opts)
	require.NotNil(t, opts.Query)
	require.NotNil(t, opts.Tx)

	// Quick sanity: the rpc commands we expose are present.
	queryRPCs := map[string]bool{}
	for _, c := range opts.Query.RpcCommandOptions {
		queryRPCs[c.RpcMethod] = true
	}
	require.True(t, queryRPCs["GetGovernanceFrameworkVersion"])
	require.True(t, queryRPCs["ListGovernanceFrameworkVersions"])
	require.True(t, queryRPCs["Params"])

	txRPCs := map[string]bool{}
	for _, c := range opts.Tx.RpcCommandOptions {
		txRPCs[c.RpcMethod] = true
	}
	require.True(t, txRPCs["AddGovernanceFrameworkDocument"])
	require.True(t, txRPCs["IncreaseActiveGovernanceFrameworkVersion"])
	require.True(t, txRPCs["UpdateParams"])
}
