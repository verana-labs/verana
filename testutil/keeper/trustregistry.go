package keeper

import (
	"context"
	"testing"

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

	// Create mock TrustDepositKeeper
	mockTrustDepositKeeper := &MockTrustDepositKeeper{}

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority.String(),
		mockTrustDepositKeeper,
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	// Initialize params
	if err := k.SetParams(ctx, types.DefaultParams()); err != nil {
		panic(err)
	}

	return k, ctx
}

// MockTrustDepositKeeper is a mock implementation of the TrustDepositKeeper interface for testing
type MockTrustDepositKeeper struct{}

func (m *MockTrustDepositKeeper) BurnEcosystemSlashedTrustDeposit(ctx sdk.Context, account string, amount uint64) error {
	return nil
}

func (m *MockTrustDepositKeeper) GetUserAgentRewardRate(ctx sdk.Context) math.LegacyDec {
	//unimplemented
	v, _ := math.LegacyNewDecFromStr("0")
	return v
}

func (m *MockTrustDepositKeeper) GetWalletUserAgentRewardRate(ctx sdk.Context) math.LegacyDec {
	//unimplemented
	v, _ := math.LegacyNewDecFromStr("0")
	return v
}

func (m *MockTrustDepositKeeper) GetTrustDepositRate(ctx sdk.Context) math.LegacyDec {
	//unimplemented
	v, _ := math.LegacyNewDecFromStr("0")
	return v
}

// AdjustTrustDeposit implements the TrustDepositKeeper interface
func (m *MockTrustDepositKeeper) AdjustTrustDeposit(ctx sdk.Context, account string, augend int64) error {
	// For testing, always succeed
	return nil
}

// AdjustAnchorTrustDeposit implements the TrustDepositKeeper interface (Issue #185)
func (m *MockTrustDepositKeeper) AdjustAnchorTrustDeposit(ctx sdk.Context, anchorID string, augend int64, operatorAccount string) error {
	// For testing, always succeed
	return nil
}

// IsAnchor implements the TrustDepositKeeper interface (Issue #185)
func (m *MockTrustDepositKeeper) IsAnchor(ctx context.Context, anchorID string) bool {
	// For testing, return false by default
	return false
}

// GetAnchorForOperator implements the TrustDepositKeeper interface (Issue #185)
func (m *MockTrustDepositKeeper) GetAnchorForOperator(ctx context.Context, operatorAccount string) (string, error) {
	// For testing, return empty (no anchor)
	return "", nil
}
