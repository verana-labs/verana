package keeper

import (
	"context"
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

	cotypes "github.com/verana-labs/verana/x/co/types"
	"github.com/verana-labs/verana/x/td/keeper"
	"github.com/verana-labs/verana/x/td/types"
)

// MockTDCorporationKeeper backs AUTHZ-CHECK-5 in MOD-TD tests. It resolves any
// signing account by default (permissive) so pre-existing positive-path tests
// pass; add an address to Unregistered to exercise the
// ErrCorporationNotRegistered abort path.
type MockTDCorporationKeeper struct {
	Unregistered map[string]bool
	ids          map[string]uint64
	nextID       uint64
}

func NewMockTDCorporationKeeper() *MockTDCorporationKeeper {
	return &MockTDCorporationKeeper{Unregistered: map[string]bool{}, ids: map[string]uint64{}, nextID: 1}
}

// IDFor returns the stable corporation_id assigned to addr (assigning a fresh
// id on first use). Trust deposits are keyed by corporation_id, so tests use
// this to look up records by the same id the keeper resolves internally.
func (m *MockTDCorporationKeeper) IDFor(addr string) uint64 {
	if m.ids == nil {
		m.ids = map[string]uint64{}
	}
	if id, ok := m.ids[addr]; ok {
		return id
	}
	if m.nextID == 0 {
		m.nextID = 1
	}
	id := m.nextID
	m.ids[addr] = id
	m.nextID++
	return id
}

func (m *MockTDCorporationKeeper) ResolveCorporationByPolicyAddress(_ context.Context, addr string) (types.CorporationView, error) {
	if m.Unregistered[addr] {
		return types.CorporationView{}, cotypes.ErrCorporationNotRegistered
	}
	return types.CorporationView{Id: m.IDFor(addr), PolicyAddress: addr}, nil
}

// MockMintKeeper is a mock implementation of types.MintKeeper
type MockMintKeeper struct{}

func NewMockMintKeeper() types.MintKeeper {
	return &MockMintKeeper{}
}

func (k *MockMintKeeper) GetBlocksPerYear(ctx context.Context) (uint64, error) {
	// Return a default value for testing (6311520 blocks per year for 5 second blocks)
	return 6311520, nil
}

func TrustdepositKeeper(t testing.TB) (keeper.Keeper, sdk.Context) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)
	bankKeeper := NewMockBankKeeper()
	mintKeeper := NewMockMintKeeper()
	delegationKeeper := &MockDelegationKeeper{}

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority.String(),
		bankKeeper,
		mintKeeper,
		delegationKeeper,
		NewMockTDCorporationKeeper(),
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	// Initialize params
	if err := k.SetParams(ctx, types.DefaultParams()); err != nil {
		panic(err)
	}

	return k, ctx
}

// TrustdepositKeeperWithDelegation creates a keeper with a MockDelegationKeeper for AUTHZ testing.
// The returned MockDelegationKeeper can be configured to return errors via ErrToReturn.
func TrustdepositKeeperWithDelegation(t testing.TB) (keeper.Keeper, sdk.Context, *MockDelegationKeeper) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)
	bankKeeper := NewMockBankKeeper()
	mintKeeper := NewMockMintKeeper()
	delegationKeeper := &MockDelegationKeeper{}

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority.String(),
		bankKeeper,
		mintKeeper,
		delegationKeeper,
		NewMockTDCorporationKeeper(),
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	if err := k.SetParams(ctx, types.DefaultParams()); err != nil {
		panic(err)
	}

	return k, ctx, delegationKeeper
}

// TrustdepositKeeperWithCorp creates a keeper exposing the AUTHZ-CHECK-5
// CorporationKeeper mock so a test can mark a signing account unregistered and
// assert the ErrCorporationNotRegistered abort path.
func TrustdepositKeeperWithCorp(t testing.TB) (keeper.Keeper, sdk.Context, *MockDelegationKeeper, *MockTDCorporationKeeper) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)
	delegationKeeper := &MockDelegationKeeper{}
	coKeeper := NewMockTDCorporationKeeper()

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority.String(),
		NewMockBankKeeper(),
		NewMockMintKeeper(),
		delegationKeeper,
		coKeeper,
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	if err := k.SetParams(ctx, types.DefaultParams()); err != nil {
		panic(err)
	}

	return k, ctx, delegationKeeper, coKeeper
}
