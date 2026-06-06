package keeper_test

import (
	"context"
	"testing"
	"time"

	"cosmossdk.io/core/address"
	storetypes "cosmossdk.io/store/types"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	cotypes "github.com/verana-labs/verana/x/co/types"
	"github.com/verana-labs/verana/x/xr/keeper"
	module "github.com/verana-labs/verana/x/xr/module"
	"github.com/verana-labs/verana/x/xr/types"
)

// MockDelegationKeeper is a mock implementation of the DelegationKeeper interface for testing.
type MockDelegationKeeper struct {
	ErrToReturn error
}

func (m *MockDelegationKeeper) CheckOperatorAuthorization(_ context.Context, _, _, _ string, _ time.Time) error {
	return m.ErrToReturn
}

// mockCorpKeeper backs AUTHZ-CHECK-5 in MOD-XR tests. It resolves any signing
// account by default (permissive); add an address to unregistered to exercise
// the ErrCorporationNotRegistered abort path.
type mockCorpKeeper struct {
	unregistered map[string]bool
}

func newMockCorpKeeper() *mockCorpKeeper {
	return &mockCorpKeeper{unregistered: map[string]bool{}}
}

func (m *mockCorpKeeper) ResolveCorporationByPolicyAddress(_ context.Context, addr string) (types.CorporationView, error) {
	if m.unregistered[addr] {
		return types.CorporationView{}, cotypes.ErrCorporationNotRegistered
	}
	return types.CorporationView{Id: 1, PolicyAddress: addr}, nil
}

type fixture struct {
	ctx              context.Context
	keeper           keeper.Keeper
	addressCodec     address.Codec
	delegationKeeper *MockDelegationKeeper
	corpKeeper       *mockCorpKeeper
}

func initFixture(t *testing.T) *fixture {
	t.Helper()

	encCfg := moduletestutil.MakeTestEncodingConfig(module.AppModule{})
	addressCodec := addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	storeService := runtime.NewKVStoreService(storeKey)
	ctx := testutil.DefaultContextWithDB(t, storeKey, storetypes.NewTransientStoreKey("transient_test")).Ctx

	authority := authtypes.NewModuleAddress(types.GovModuleName)
	mockDelegationKeeper := &MockDelegationKeeper{}
	corpKeeper := newMockCorpKeeper()

	k := keeper.NewKeeper(
		storeService,
		encCfg.Codec,
		addressCodec,
		authority,
		mockDelegationKeeper,
		corpKeeper,
	)

	// Initialize params
	if err := k.Params.Set(ctx, types.DefaultParams()); err != nil {
		t.Fatalf("failed to set params: %v", err)
	}

	return &fixture{
		ctx:              ctx,
		keeper:           k,
		addressCodec:     addressCodec,
		delegationKeeper: mockDelegationKeeper,
		corpKeeper:       corpKeeper,
	}
}
