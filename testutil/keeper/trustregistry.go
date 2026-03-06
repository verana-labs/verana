package keeper

import (
	"context"
	"testing"
	"time"

	"cosmossdk.io/math"

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

	"github.com/verana-labs/verana/x/tr/keeper"
	"github.com/verana-labs/verana/x/tr/types"
)

func TrustregistryKeeper(t testing.TB) (keeper.Keeper, sdk.Context) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	// Create mock keepers
	mockDelegationKeeper := &MockDelegationKeeper{}

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority.String(),
		mockDelegationKeeper,
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	// Initialize params
	if err := k.SetParams(ctx, types.DefaultParams()); err != nil {
		panic(err)
	}

	return k, ctx
}

// MockTrustDepositKeeper is a mock implementation of the TrustDepositKeeper interface for testing.
// Used by CS, DD, and PERM module test utilities (not by TR module itself).
type MockTrustDepositKeeper struct{}

func (m *MockTrustDepositKeeper) AdjustTrustDeposit(_ sdk.Context, _ string, _ int64) error {
	return nil
}

func (m *MockTrustDepositKeeper) GetTrustDepositRate(_ sdk.Context) math.LegacyDec {
	v, _ := math.LegacyNewDecFromStr("0")
	return v
}

func (m *MockTrustDepositKeeper) GetUserAgentRewardRate(_ sdk.Context) math.LegacyDec {
	v, _ := math.LegacyNewDecFromStr("0")
	return v
}

func (m *MockTrustDepositKeeper) GetWalletUserAgentRewardRate(_ sdk.Context) math.LegacyDec {
	v, _ := math.LegacyNewDecFromStr("0")
	return v
}

func (m *MockTrustDepositKeeper) BurnEcosystemSlashedTrustDeposit(_ sdk.Context, _ string, _ uint64) error {
	return nil
}

// MockDelegationKeeper is a mock implementation of the DelegationKeeper interface for testing.
// By default it allows all operator authorizations (no-op check).
type MockDelegationKeeper struct{}

func (m *MockDelegationKeeper) CheckOperatorAuthorization(_ context.Context, _, _, _ string, _ time.Time) error {
	return nil
}
