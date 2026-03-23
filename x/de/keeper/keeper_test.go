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

	"github.com/verana-labs/verana/x/de/keeper"
	module "github.com/verana-labs/verana/x/de/module"
	"github.com/verana-labs/verana/x/de/types"
)

// MockPermKeeper implements types.PermKeeper for testing.
type MockPermKeeper struct {
	Permissions map[uint64]MockPermission
}

type MockPermission struct {
	Authority   string
	VsOperator  string
	WithFeegrant bool
	EffectiveUntil *time.Time
}

func (m *MockPermKeeper) GetPermissionForVSOA(ctx context.Context, permID uint64) (string, string, bool, *time.Time, error) {
	p, ok := m.Permissions[permID]
	if !ok {
		return "", "", false, nil, types.ErrPermissionNotFound
	}
	return p.Authority, p.VsOperator, p.WithFeegrant, p.EffectiveUntil, nil
}

type fixture struct {
	ctx          context.Context
	keeper       keeper.Keeper
	addressCodec address.Codec
	mockPerm     *MockPermKeeper
}

func initFixture(t *testing.T) *fixture {
	t.Helper()

	encCfg := moduletestutil.MakeTestEncodingConfig(module.AppModule{})
	addressCodec := addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	storeService := runtime.NewKVStoreService(storeKey)
	ctx := testutil.DefaultContextWithDB(t, storeKey, storetypes.NewTransientStoreKey("transient_test")).Ctx

	authority := authtypes.NewModuleAddress(types.GovModuleName)

	mockPerm := &MockPermKeeper{
		Permissions: make(map[uint64]MockPermission),
	}

	k := keeper.NewKeeper(
		storeService,
		encCfg.Codec,
		addressCodec,
		authority,
	)
	k.SetPermKeeper(mockPerm)

	// Initialize params
	if err := k.Params.Set(ctx, types.DefaultParams()); err != nil {
		t.Fatalf("failed to set params: %v", err)
	}

	return &fixture{
		ctx:          ctx,
		keeper:       k,
		addressCodec: addressCodec,
		mockPerm:     mockPerm,
	}
}
