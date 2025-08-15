package keeper

//func TrustdepositKeeper(t testing.TB) (keeper.Keeper, sdk.Context) {
//	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
//
//	db := dbm.NewMemDB()
//	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
//	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
//	require.NoError(t, stateStore.LoadLatestVersion())
//
//	registry := codectypes.NewInterfaceRegistry()
//	cdc := codec.NewProtoCodec(registry)
//	authority := authtypes.NewModuleAddress(govtypes.ModuleName)
//	bankKeeper := NewMockBankKeeper()
//
//	k := keeper.NewKeeper(
//		cdc,
//		runtime.NewKVStoreService(storeKey),
//		log.NewNopLogger(),
//		authority.String(),
//		bankKeeper,
//	)
//
//	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
//
//	// Initialize params
//	if err := k.SetParams(ctx, types.DefaultParams()); err != nil {
//		panic(err)
//	}
//
//	return k, ctx
//}
