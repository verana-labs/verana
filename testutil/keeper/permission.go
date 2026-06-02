package keeper

import (
	"context"
	"hash/fnv"
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

	cstypes "github.com/verana-labs/verana/x/cs/types"
	ectypes "github.com/verana-labs/verana/x/ec/types"
	"github.com/verana-labs/verana/x/pp/keeper"
	"github.com/verana-labs/verana/x/pp/types"
)

func PermissionKeeper(t testing.TB) (keeper.Keeper, *MockCredentialSchemaKeeper, *MockPermEcosystemKeeper, *MockPermCorporationKeeper, sdk.Context, *MockDelegationKeeper) {
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
	ekKeeper := NewMockPermEcosystemKeeper()
	coKeeper := NewMockPermCorporationKeeper(ekKeeper)
	bankKeeper := NewMockBankKeeper()
	mockTrustDepositKeeper := &MockTrustDepositKeeper{}
	mockDelegationKeeper := &MockDelegationKeeper{}
	mockDigestKeeper := &MockDigestKeeper{}
	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority.String(),
		csKeeper,
		ekKeeper,
		coKeeper,
		mockTrustDepositKeeper,
		bankKeeper,
		mockDelegationKeeper,
		mockDigestKeeper,
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	// Initialize params
	if err := k.SetParams(ctx, types.DefaultParams()); err != nil {
		panic(err)
	}

	return k, csKeeper, ekKeeper, coKeeper, ctx, mockDelegationKeeper
}

// MockPermEcosystemKeeper is a mock implementation of x/pp types.EcosystemKeeper.
// Replaces the legacy MockTrustRegistryKeeper post-MOD-EC rename: stores
// Ecosystem rows keyed by id and a per-mock map of policy_address →
// CorporationView so the linked MockPermCorporationKeeper can resolve signers
// back to a corp.id matching ec.CorporationId.
type MockPermEcosystemKeeper struct {
	ecosystems              map[uint64]ectypes.Ecosystem
	trustUnitPrice          uint64
	corporationByPolicyAddr map[string]types.CorporationView
}

func NewMockPermEcosystemKeeper() *MockPermEcosystemKeeper {
	return &MockPermEcosystemKeeper{
		ecosystems:              make(map[uint64]ectypes.Ecosystem),
		trustUnitPrice:          1,
		corporationByPolicyAddr: make(map[string]types.CorporationView),
	}
}

func (k *MockPermEcosystemKeeper) GetTrustUnitPrice(ctx sdk.Context) uint64 {
	return k.trustUnitPrice
}

func (k *MockPermEcosystemKeeper) GetEcosystem(ctx context.Context, id uint64) (ectypes.Ecosystem, error) {
	if ec, ok := k.ecosystems[id]; ok {
		return ec, nil
	}
	return ectypes.Ecosystem{}, ectypes.ErrEcosystemNotFound
}

// CreateMockEcosystem registers an Ecosystem owned by the given signing
// `corporation` (a bech32 policy_address). Also wires the policy_address →
// CorporationView mapping so the paired MockPermCorporationKeeper resolves
// the signer to a corp.id matching ec.CorporationId.
func (k *MockPermEcosystemKeeper) CreateMockEcosystem(corporation string, did string) uint64 {
	id := uint64(len(k.ecosystems) + 1)
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

// RegisterCorp maps a policy_address to a Corporation id (idempotent). Lets
// tests that set Participant.corporation_id directly have it reversed back to
// an account by MockPermCorporationKeeper.ResolveByID for fund-flows. Ids start
// above the ecosystem-id range to avoid collisions.
func (k *MockPermEcosystemKeeper) RegisterCorp(addr string) uint64 {
	if v, ok := k.corporationByPolicyAddr[addr]; ok {
		return v.Id
	}
	id := uint64(100000 + len(k.corporationByPolicyAddr) + 1)
	k.corporationByPolicyAddr[addr] = types.CorporationView{Id: id, PolicyAddress: addr}
	return id
}

// MockPermCorporationKeeper resolves a signing policy_address to a
// CorporationView; shares its backing map with MockPermEcosystemKeeper so
// the (ec.CorporationId == co.Id) check passes for the same `corporation`
// that registered the Ecosystem. Falls back to always-valid
// CorporationView{Id: 1} when no mapping exists so legacy tests that don't
// register a corporation still pass through ownership checks.
type MockPermCorporationKeeper struct {
	corporationByPolicyAddr map[string]types.CorporationView
	policyAddrByID          map[uint64]string
}

func NewMockPermCorporationKeeper(ekKeeper *MockPermEcosystemKeeper) *MockPermCorporationKeeper {
	return &MockPermCorporationKeeper{
		corporationByPolicyAddr: ekKeeper.corporationByPolicyAddr,
		policyAddrByID:          make(map[uint64]string),
	}
}

func (k *MockPermCorporationKeeper) ResolveByPolicyAddress(ctx context.Context, policyAddress string) (types.CorporationView, bool) {
	if v, ok := k.corporationByPolicyAddr[policyAddress]; ok {
		k.policyAddrByID[v.Id] = policyAddress
		return v, true
	}
	// Distinct, deterministic sentinel id per unknown address — never matches a
	// CreateMockEcosystem row (which uses small ids starting at 1) thanks to the
	// high bit, yet stays distinct across addresses so overlap checks treat
	// different corporations as different. Lets "wrong signer" tests still see a
	// strict ownership mismatch.
	h := fnv.New64a()
	_, _ = h.Write([]byte(policyAddress))
	id := h.Sum64() | (uint64(1) << 62)
	v := types.CorporationView{Id: id, PolicyAddress: policyAddress}
	k.policyAddrByID[v.Id] = policyAddress
	return v, true
}

// ResolveByID reverses corporation_id → policy_address for fund-flow helpers.
func (k *MockPermCorporationKeeper) ResolveByID(ctx context.Context, id uint64) (types.CorporationView, bool) {
	for addr, v := range k.corporationByPolicyAddr {
		if v.Id == id {
			return types.CorporationView{Id: id, PolicyAddress: addr}, true
		}
	}
	if addr, ok := k.policyAddrByID[id]; ok {
		return types.CorporationView{Id: id, PolicyAddress: addr}, true
	}
	return types.CorporationView{}, false
}

type MockCredentialSchemaKeeper struct {
	credentialSchemas map[uint64]cstypes.CredentialSchema
}

func NewMockCredentialSchemaKeeper() *MockCredentialSchemaKeeper {
	return &MockCredentialSchemaKeeper{
		credentialSchemas: make(map[uint64]cstypes.CredentialSchema),
	}
}

func (k *MockCredentialSchemaKeeper) UpdateMockCredentialSchema(id uint64, ecosystemID uint64, issuerMode cstypes.IssuerOnboardingMode, verifierMode cstypes.VerifierOnboardingMode) {
	k.credentialSchemas[id] = cstypes.CredentialSchema{
		Id:                     id,
		EcosystemId:            ecosystemID,
		IssuerOnboardingMode:   issuerMode,
		VerifierOnboardingMode: verifierMode,
	}
}

func (k *MockCredentialSchemaKeeper) GetCredentialSchemaById(ctx sdk.Context, id uint64) (cstypes.CredentialSchema, error) {
	if cs, ok := k.credentialSchemas[id]; ok {
		return cs, nil
	}
	return cstypes.CredentialSchema{}, cstypes.ErrCredentialSchemaNotFound
}

func (k *MockCredentialSchemaKeeper) CreateMockCredentialSchema(id uint64, issuerMode cstypes.IssuerOnboardingMode, verifierMode cstypes.VerifierOnboardingMode) {
	k.credentialSchemas[id] = cstypes.CredentialSchema{
		Id:                     id,
		IssuerOnboardingMode:   issuerMode,
		VerifierOnboardingMode: verifierMode,
	}
}

func (k *MockCredentialSchemaKeeper) CreateMockCredentialSchemaFull(cs cstypes.CredentialSchema) {
	k.credentialSchemas[cs.Id] = cs
}

// MockDigestKeeper is a permissive mock of the DigestKeeper interface for
// perm module tests. It records each call so assertions can verify that
// perm invoked StoreDigestModuleCall during credential-issuance flows.
type MockDigestKeeper struct {
	Stored []MockDigestRecord
}

type MockDigestRecord struct {
	Authority       string
	Digest          string
	DigestAlgorithm string
}

func (m *MockDigestKeeper) StoreDigestModuleCall(_ context.Context, authority, digest, digestAlgorithm string) error {
	m.Stored = append(m.Stored, MockDigestRecord{Authority: authority, Digest: digest, DigestAlgorithm: digestAlgorithm})
	return nil
}
