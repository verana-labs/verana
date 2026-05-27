package keeper

import (
	"context"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
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

	"github.com/verana-labs/verana/x/cs/keeper"
	"github.com/verana-labs/verana/x/cs/types"
	ectypes "github.com/verana-labs/verana/x/ec/types"
)

// MockBankKeeper is a mock implementation of types.BankKeeper
type MockBankKeeper struct {
	bankBalances map[string]sdk.Coins
}

func (k *MockBankKeeper) SendCoins(ctx context.Context, from, to sdk.AccAddress, amt sdk.Coins) error {
	return nil
}

func (k *MockBankKeeper) HasBalance(ctx context.Context, addr sdk.AccAddress, amt sdk.Coin) bool {
	return true
}

func (k *MockBankKeeper) BurnCoins(ctx context.Context, name string, amt sdk.Coins) error {
	return nil
}

func (k *MockBankKeeper) SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	return nil
}

func (k *MockBankKeeper) SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error {
	return nil
}

func (k *MockBankKeeper) SpendableCoins(ctx context.Context, address sdk.AccAddress) sdk.Coins {
	return sdk.Coins{}
}

func (k *MockBankKeeper) GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	return sdk.NewCoin(denom, math.ZeroInt())
}

func (k *MockBankKeeper) GetAllBalances(ctx context.Context, addr sdk.AccAddress) sdk.Coins {
	return sdk.Coins{}
}

func NewMockBankKeeper() *MockBankKeeper {
	return &MockBankKeeper{
		bankBalances: make(map[string]sdk.Coins),
	}
}

func (k *MockBankKeeper) SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientAddr string, amt sdk.Coins) error {
	return nil
}

// MockEcosystemKeeper is a mock implementation of types.EcosystemKeeper.
// Replaces the legacy MockTrustRegistryKeeper post-MOD-EC rename: stores
// Ecosystem rows keyed by id (instead of TrustRegistry rows) so tests can
// exercise the cs.ecosystem_id → ec.CorporationId ownership check.
type MockEcosystemKeeper struct {
	ecosystems map[uint64]ectypes.Ecosystem
	// corporationByPolicyAddr is shared with MockCorporationKeeper so the
	// resolved co.Id matches ec.CorporationId for the same `creator`.
	corporationByPolicyAddr map[string]types.CorporationView
}

func NewMockEcosystemKeeper() *MockEcosystemKeeper {
	return &MockEcosystemKeeper{
		ecosystems:              make(map[uint64]ectypes.Ecosystem),
		corporationByPolicyAddr: make(map[string]types.CorporationView),
	}
}

func (k *MockEcosystemKeeper) GetEcosystem(ctx context.Context, id uint64) (ectypes.Ecosystem, error) {
	if ec, ok := k.ecosystems[id]; ok {
		return ec, nil
	}
	return ectypes.Ecosystem{}, ectypes.ErrEcosystemNotFound
}

// CreateMockEcosystem registers an Ecosystem owned by the given signing
// `corporation` (a bech32 policy_address). Also wires the policy_address →
// CorporationView mapping so the paired MockCorporationKeeper resolves the
// signer to a corp.id matching ec.CorporationId.
func (k *MockEcosystemKeeper) CreateMockEcosystem(corporation string, did string) uint64 {
	id := uint64(len(k.ecosystems) + 1)
	// Use the same numeric id for corp.id so callers don't need to thread a
	// separate corporation id through tests.
	corpID := id
	k.ecosystems[id] = ectypes.Ecosystem{
		Id:            id,
		Did:           did,
		CorporationId: corpID,
		ActiveVersion: 1,
		Language:      "en",
	}
	k.corporationByPolicyAddr[corporation] = types.CorporationView{
		Id:            corpID,
		PolicyAddress: corporation,
	}
	return id
}

// MockCorporationKeeper resolves a signing policy_address to a
// CorporationView; shares its backing map with MockEcosystemKeeper so
// the (ec.CorporationId == co.Id) ownership check passes for the same
// `corporation` that registered the Ecosystem.
//
// Unknown policy addresses get a *distinct* CorporationView with a high,
// stable id (math.MaxInt32) that will never match a CreateMockEcosystem
// row (those start at 1). This lets "wrong signer" tests exercise the
// strict ownership-mismatch path without each test having to pre-register
// every counterexample address.
type MockCorporationKeeper struct {
	corporationByPolicyAddr map[string]types.CorporationView
}

func NewMockCorporationKeeper(ekKeeper *MockEcosystemKeeper) *MockCorporationKeeper {
	return &MockCorporationKeeper{corporationByPolicyAddr: ekKeeper.corporationByPolicyAddr}
}

const mockUnregisteredCorpID uint64 = 1 << 31

func (k *MockCorporationKeeper) ResolveByPolicyAddress(ctx context.Context, policyAddress string) (types.CorporationView, bool) {
	if v, ok := k.corporationByPolicyAddr[policyAddress]; ok {
		return v, true
	}
	return types.CorporationView{Id: mockUnregisteredCorpID, PolicyAddress: policyAddress}, true
}

func CredentialschemaKeeper(t testing.TB) (keeper.Keeper, *MockEcosystemKeeper, sdk.Context) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	// Create mock keepers
	bankKeeper := NewMockBankKeeper()
	ecosystemKeeper := NewMockEcosystemKeeper()
	coKeeper := NewMockCorporationKeeper(ecosystemKeeper)
	mockDelegationKeeper := &MockDelegationKeeper{}
	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority.String(),
		bankKeeper,
		ecosystemKeeper,
		coKeeper,
		mockDelegationKeeper,
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	// Initialize params
	if err := k.SetParams(ctx, types.DefaultParams()); err != nil {
		panic(err)
	}

	return k, ecosystemKeeper, ctx
}
