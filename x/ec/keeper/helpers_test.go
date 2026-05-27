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
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana/x/ec/keeper"
	"github.com/verana-labs/verana/x/ec/types"
	gftypes "github.com/verana-labs/verana/x/gf/types"
)

// stubDelegationKeeper / stubCorporationKeeper / stubGFKeeper provide the
// minimal expected_keepers surface so params-level tests can construct an
// x/ec Keeper without depending on the live x/de, x/co and x/gf modules.
type stubDelegationKeeper struct{}

func (stubDelegationKeeper) CheckOperatorAuthorization(_ context.Context, _, _, _ string, _ time.Time) error {
	return nil
}

type stubCorporationKeeper struct{}

func (stubCorporationKeeper) ResolveByPolicyAddress(_ context.Context, _ string) (types.CorporationView, bool) {
	return types.CorporationView{}, false
}

func (stubCorporationKeeper) GetByID(_ context.Context, _ uint64) (types.CorporationView, bool) {
	return types.CorporationView{}, false
}

type stubGFKeeper struct{}

func (stubGFKeeper) CreateInitialGFVersionForEcosystem(_ context.Context, _ uint64, _, _, _ string) error {
	return nil
}

func (stubGFKeeper) ListVersionsByEcosystem(_ context.Context, _ uint64, _ uint32, _ bool, _ string) ([]gftypes.GovernanceFrameworkVersionWithDocs, error) {
	return nil, nil
}

// setupMsgServer wires a minimal in-memory x/ec Keeper for params /
// msg_update_params tests. The keeper does not need real cross-module
// dependencies for params flows, so stub keepers are used.
func setupMsgServer(t testing.TB) (keeper.Keeper, types.MsgServer, sdk.Context) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority,
		stubDelegationKeeper{},
		stubCorporationKeeper{},
		stubGFKeeper{},
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	return k, keeper.NewMsgServerImpl(k), ctx
}
