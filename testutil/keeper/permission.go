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

	cstypes "github.com/verana-labs/verana-blockchain/x/cs/types"
	"github.com/verana-labs/verana-blockchain/x/perm/keeper"
	"github.com/verana-labs/verana-blockchain/x/perm/types"
)

func PermissionKeeper(t testing.TB) (keeper.Keeper, *MockCredentialSchemaKeeper, *MockTrustRegistryKeeper, sdk.Context) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	// Create mock keepers
	csKeeper := NewMockCredentialSchemaKeeper()
	trkKeeper := NewMockTrustRegistryKeeper()
	bankKeeper := NewMockBankKeeper()
	mockTrustDepositKeeper := &MockTrustDepositKeeper{}
	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority.String(),
		csKeeper,
		trkKeeper,
		mockTrustDepositKeeper,
		bankKeeper,
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	// Initialize params
	if err := k.SetParams(ctx, types.DefaultParams()); err != nil {
		panic(err)
	}

	return k, csKeeper, trkKeeper, ctx
}

type MockCredentialSchemaKeeper struct {
	credentialSchemas map[uint64]cstypes.CredentialSchema
}

func NewMockCredentialSchemaKeeper() *MockCredentialSchemaKeeper {
	return &MockCredentialSchemaKeeper{
		credentialSchemas: make(map[uint64]cstypes.CredentialSchema),
	}
}

func (k *MockCredentialSchemaKeeper) UpdateMockCredentialSchema(id uint64, trId uint64, issuerPermMode, verifierPermMode cstypes.CredentialSchemaPermManagementMode) {
	k.credentialSchemas[id] = cstypes.CredentialSchema{
		Id:                         id,
		TrId:                       trId,
		IssuerPermManagementMode:   issuerPermMode,
		VerifierPermManagementMode: verifierPermMode,
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
