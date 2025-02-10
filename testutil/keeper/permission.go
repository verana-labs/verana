package keeper

import (
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
	"github.com/stretchr/testify/require"

	cstypes "github.com/verana-labs/verana-blockchain/x/credentialschema/types"
	"github.com/verana-labs/verana-blockchain/x/permission/keeper"
	"github.com/verana-labs/verana-blockchain/x/permission/types"
)

func PermissionKeeper(t testing.TB) (keeper.Keeper, *MockCredentialSchemaKeeper, sdk.Context) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	credentialSchemaKeeper := NewMockCredentialSchemaKeeper()

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority.String(),
		credentialSchemaKeeper,
		NewMockTrustRegistryKeeper(),
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	if err := k.SetParams(ctx, types.DefaultParams()); err != nil {
		panic(err)
	}

	return k, credentialSchemaKeeper, ctx
}

type MockCredentialSchemaKeeper struct {
	credentialSchemas map[uint64]cstypes.CredentialSchema
}

func NewMockCredentialSchemaKeeper() *MockCredentialSchemaKeeper {
	return &MockCredentialSchemaKeeper{
		credentialSchemas: make(map[uint64]cstypes.CredentialSchema),
	}
}

func (k *MockCredentialSchemaKeeper) GetCredentialSchemaById(ctx sdk.Context, id uint64) (cstypes.CredentialSchema, error) {
	if cs, ok := k.credentialSchemas[id]; ok {
		return cs, nil
	}
	return cstypes.CredentialSchema{}, cstypes.ErrCredentialSchemaNotFound
}

func (k *MockCredentialSchemaKeeper) CreateMockCredentialSchema(id uint64, issuerPermMode, verifierPermMode cstypes.CredentialSchemaPermManagementMode) {
	k.credentialSchemas[id] = cstypes.CredentialSchema{
		Id:                         id,
		IssuerPermManagementMode:   issuerPermMode,
		VerifierPermManagementMode: verifierPermMode,
	}
}
