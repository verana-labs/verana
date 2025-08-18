package keeper_test

import (
	"context"
	"cosmossdk.io/core/address"
	"github.com/verana-labs/verana/x/cs/keeper"
)

type fixture struct {
	ctx          context.Context
	keeper       keeper.Keeper
	addressCodec address.Codec
}

//func initFixture(t *testing.T) *fixture {
//	t.Helper()
//
//	encCfg := moduletestutil.MakeTestEncodingConfig(module.AppModule{})
//	addressCodec := addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
//	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
//
//	storeService := runtime.NewKVStoreService(storeKey)
//	ctx := testutil.DefaultContextWithDB(t, storeKey, storetypes.NewTransientStoreKey("transient_test")).Ctx
//
//	authority := authtypes.NewModuleAddress(types.GovModuleName)
//
//	k := keeper.NewKeeper(
//		storeService,
//		encCfg.Codec,
//		addressCodec,
//		authority,
//	)
//
//	// Initialize params
//	if err := k.Params.Set(ctx, types.DefaultParams()); err != nil {
//		t.Fatalf("failed to set params: %v", err)
//	}
//
//	return &fixture{
//		ctx:          ctx,
//		keeper:       k,
//		addressCodec: addressCodec,
//	}
//}
