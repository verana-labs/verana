package keeper_test

import (
	"context"
	"testing"
	"time"

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
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana/testutil/keeper"
	cstypes "github.com/verana-labs/verana/x/cs/types"
	ectypes "github.com/verana-labs/verana/x/ec/types"
	"github.com/verana-labs/verana/x/pp/keeper"
	"github.com/verana-labs/verana/x/pp/types"
)

// TrackingBankKeeper tracks all balance changes for verification
type TrackingBankKeeper struct {
	Balances          map[string]sdk.Coins
	ModuleBalances    map[string]sdk.Coins
	TransferLog       []TransferRecord
	ModuleTransferLog []ModuleTransferRecord
}

type TransferRecord struct {
	From   string
	To     string
	Amount sdk.Coins
}

type ModuleTransferRecord struct {
	From       string
	ToModule   string
	Amount     sdk.Coins
	IsToModule bool
}

func NewTrackingBankKeeper() *TrackingBankKeeper {
	return &TrackingBankKeeper{
		Balances:          make(map[string]sdk.Coins),
		ModuleBalances:    make(map[string]sdk.Coins),
		TransferLog:       make([]TransferRecord, 0),
		ModuleTransferLog: make([]ModuleTransferRecord, 0),
	}
}

func (k *TrackingBankKeeper) SetBalance(addr string, coins sdk.Coins) {
	k.Balances[addr] = coins
}

func (k *TrackingBankKeeper) SendCoins(ctx context.Context, from, to sdk.AccAddress, amt sdk.Coins) error {
	fromAddr := from.String()
	toAddr := to.String()

	// Deduct from sender
	if existing, ok := k.Balances[fromAddr]; ok {
		k.Balances[fromAddr] = existing.Sub(amt...)
	} else {
		k.Balances[fromAddr] = sdk.Coins{}.Sub(amt...)
	}

	// Add to receiver
	if existing, ok := k.Balances[toAddr]; ok {
		k.Balances[toAddr] = existing.Add(amt...)
	} else {
		k.Balances[toAddr] = amt
	}

	k.TransferLog = append(k.TransferLog, TransferRecord{From: fromAddr, To: toAddr, Amount: amt})
	return nil
}

func (k *TrackingBankKeeper) HasBalance(ctx context.Context, addr sdk.AccAddress, amt sdk.Coin) bool {
	if existing, ok := k.Balances[addr.String()]; ok {
		return existing.AmountOf(amt.Denom).GTE(amt.Amount)
	}
	return false
}

func (k *TrackingBankKeeper) BurnCoins(ctx context.Context, name string, amt sdk.Coins) error {
	return nil
}

func (k *TrackingBankKeeper) SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	return nil
}

func (k *TrackingBankKeeper) SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error {
	return nil
}

func (k *TrackingBankKeeper) SpendableCoins(ctx context.Context, address sdk.AccAddress) sdk.Coins {
	if coins, ok := k.Balances[address.String()]; ok {
		return coins
	}
	return sdk.Coins{}
}

func (k *TrackingBankKeeper) GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	if coins, ok := k.Balances[addr.String()]; ok {
		return sdk.NewCoin(denom, coins.AmountOf(denom))
	}
	return sdk.NewCoin(denom, math.ZeroInt())
}

func (k *TrackingBankKeeper) GetAllBalances(ctx context.Context, addr sdk.AccAddress) sdk.Coins {
	if coins, ok := k.Balances[addr.String()]; ok {
		return coins
	}
	return sdk.Coins{}
}

func (k *TrackingBankKeeper) SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	fromAddr := senderAddr.String()

	// Deduct from sender
	if existing, ok := k.Balances[fromAddr]; ok {
		k.Balances[fromAddr] = existing.Sub(amt...)
	} else {
		k.Balances[fromAddr] = sdk.Coins{}.Sub(amt...)
	}

	// Add to module
	if existing, ok := k.ModuleBalances[recipientModule]; ok {
		k.ModuleBalances[recipientModule] = existing.Add(amt...)
	} else {
		k.ModuleBalances[recipientModule] = amt
	}

	k.ModuleTransferLog = append(k.ModuleTransferLog, ModuleTransferRecord{
		From:       fromAddr,
		ToModule:   recipientModule,
		Amount:     amt,
		IsToModule: true,
	})
	return nil
}

// GetTotalDeducted returns total amount deducted from an account
func (k *TrackingBankKeeper) GetTotalDeducted(addr string) sdk.Coins {
	total := sdk.Coins{}
	for _, record := range k.TransferLog {
		if record.From == addr {
			total = total.Add(record.Amount...)
		}
	}
	for _, record := range k.ModuleTransferLog {
		if record.From == addr {
			total = total.Add(record.Amount...)
		}
	}
	return total
}

// GetTotalReceived returns total amount received by an account
func (k *TrackingBankKeeper) GetTotalReceived(addr string) sdk.Coins {
	total := sdk.Coins{}
	for _, record := range k.TransferLog {
		if record.To == addr {
			total = total.Add(record.Amount...)
		}
	}
	return total
}

// TrackingTrustDepositKeeper tracks trust deposit changes
type TrackingTrustDepositKeeper struct {
	TrustDeposits       map[string]int64
	AdjustmentLog       []TrustDepositAdjustment
	OnBehalfLog         []TrustDepositOnBehalfAdjustment
	UserAgentRewardRate math.LegacyDec
	WalletUARewardRate  math.LegacyDec
	TrustDepositRate    math.LegacyDec
	bankKeeper          *TrackingBankKeeper // reference for on-behalf coin tracking
}

type TrustDepositAdjustment struct {
	Account string
	Amount  int64
}

type TrustDepositOnBehalfAdjustment struct {
	Account string
	Funder  sdk.AccAddress
	Amount  int64
}

func NewTrackingTrustDepositKeeper(uaRate, wuaRate, tdRate string, bankKeeper *TrackingBankKeeper) *TrackingTrustDepositKeeper {
	ua, _ := math.LegacyNewDecFromStr(uaRate)
	wua, _ := math.LegacyNewDecFromStr(wuaRate)
	td, _ := math.LegacyNewDecFromStr(tdRate)
	return &TrackingTrustDepositKeeper{
		TrustDeposits:       make(map[string]int64),
		AdjustmentLog:       make([]TrustDepositAdjustment, 0),
		OnBehalfLog:         make([]TrustDepositOnBehalfAdjustment, 0),
		UserAgentRewardRate: ua,
		WalletUARewardRate:  wua,
		TrustDepositRate:    td,
		bankKeeper:          bankKeeper,
	}
}

func (m *TrackingTrustDepositKeeper) BurnEcosystemSlashedTrustDeposit(ctx sdk.Context, account string, amount uint64) error {
	return nil
}

func (m *TrackingTrustDepositKeeper) GetUserAgentRewardRate(ctx sdk.Context) math.LegacyDec {
	return m.UserAgentRewardRate
}

func (m *TrackingTrustDepositKeeper) GetWalletUserAgentRewardRate(ctx sdk.Context) math.LegacyDec {
	return m.WalletUARewardRate
}

func (m *TrackingTrustDepositKeeper) GetTrustDepositRate(ctx sdk.Context) math.LegacyDec {
	return m.TrustDepositRate
}

func (m *TrackingTrustDepositKeeper) AdjustTrustDeposit(ctx sdk.Context, account string, augend int64, _ string) error {
	m.TrustDeposits[account] += augend
	m.AdjustmentLog = append(m.AdjustmentLog, TrustDepositAdjustment{Account: account, Amount: augend})
	return nil
}

func (m *TrackingTrustDepositKeeper) AdjustTrustDepositOnBehalf(ctx sdk.Context, account string, funder sdk.AccAddress, amount int64) error {
	// Mirror real implementation: deduct from funder via SendCoinsFromAccountToModule
	if m.bankKeeper != nil {
		err := m.bankKeeper.SendCoinsFromAccountToModule(ctx, funder, "td", sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, amount)))
		if err != nil {
			return err
		}
	}
	m.TrustDeposits[account] += amount
	m.OnBehalfLog = append(m.OnBehalfLog, TrustDepositOnBehalfAdjustment{Account: account, Funder: funder, Amount: amount})
	return nil
}

func (m *TrackingTrustDepositKeeper) GetTotalAdjustment(account string) int64 {
	total := int64(0)
	for _, adj := range m.AdjustmentLog {
		if adj.Account == account {
			total += adj.Amount
		}
	}
	for _, adj := range m.OnBehalfLog {
		if adj.Account == account {
			total += adj.Amount
		}
	}
	return total
}

// GetTotalOnBehalfFunded returns the total amount funded on behalf of an account (by third parties)
func (m *TrackingTrustDepositKeeper) GetTotalOnBehalfFunded(account string) int64 {
	total := int64(0)
	for _, adj := range m.OnBehalfLog {
		if adj.Account == account {
			total += adj.Amount
		}
	}
	return total
}

// TrackingCredentialSchemaKeeper for test setup
type TrackingCredentialSchemaKeeper struct {
	credentialSchemas map[uint64]cstypes.CredentialSchema
}

func NewTrackingCredentialSchemaKeeper() *TrackingCredentialSchemaKeeper {
	return &TrackingCredentialSchemaKeeper{
		credentialSchemas: make(map[uint64]cstypes.CredentialSchema),
	}
}

func (k *TrackingCredentialSchemaKeeper) UpdateMockCredentialSchema(id uint64, ecosystemID uint64, issuerMode cstypes.IssuerOnboardingMode, verifierMode cstypes.VerifierOnboardingMode) {
	k.credentialSchemas[id] = cstypes.CredentialSchema{
		Id:                     id,
		EcosystemId:            ecosystemID,
		IssuerOnboardingMode:   issuerMode,
		VerifierOnboardingMode: verifierMode,
	}
}

func (k *TrackingCredentialSchemaKeeper) GetCredentialSchemaById(ctx sdk.Context, id uint64) (cstypes.CredentialSchema, error) {
	if cs, ok := k.credentialSchemas[id]; ok {
		return cs, nil
	}
	return cstypes.CredentialSchema{}, cstypes.ErrCredentialSchemaNotFound
}

// TrackingEcosystemKeeper for test setup (replaces legacy TrackingTrustRegistryKeeper
// post-MOD-EC rename). Holds Ecosystem rows keyed by id and a per-mock map of
// policy_address → CorporationView so the linked TrackingCorporationKeeper can
// resolve signers back to a corp.id matching ec.CorporationId.
type TrackingEcosystemKeeper struct {
	ecosystems     map[uint64]ectypes.Ecosystem
	trustUnitPrice uint64
	// corporationByPolicyAddr is shared with TrackingCorporationKeeper so the
	// signer→corp.id resolution mirrors the on-chain co_keeper.
	corporationByPolicyAddr map[string]types.CorporationView
}

func NewTrackingEcosystemKeeper(trustUnitPrice uint64) *TrackingEcosystemKeeper {
	return &TrackingEcosystemKeeper{
		ecosystems:              make(map[uint64]ectypes.Ecosystem),
		trustUnitPrice:          trustUnitPrice,
		corporationByPolicyAddr: make(map[string]types.CorporationView),
	}
}

func (k *TrackingEcosystemKeeper) GetTrustUnitPrice(ctx sdk.Context) uint64 {
	return k.trustUnitPrice
}

func (k *TrackingEcosystemKeeper) GetEcosystem(ctx context.Context, id uint64) (ectypes.Ecosystem, error) {
	if ec, ok := k.ecosystems[id]; ok {
		return ec, nil
	}
	return ectypes.Ecosystem{}, ectypes.ErrEcosystemNotFound
}

// CreateMockEcosystem registers an Ecosystem owned by the given signing
// `creator` (a bech32 policy_address). Also wires the policy_address →
// CorporationView mapping so a TrackingCorporationKeeper built from this
// keeper resolves the signer to a corp.id matching ec.CorporationId.
func (k *TrackingEcosystemKeeper) CreateMockEcosystem(creator string, did string) uint64 {
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
	k.corporationByPolicyAddr[creator] = types.CorporationView{
		Id:            corpID,
		PolicyAddress: creator,
	}
	return id
}

// RegisterCorp maps a policy_address to a Corporation id (idempotent), so
// Participants whose corporation_id is set directly in tests can be reversed
// back to an account by TrackingCorporationKeeper.ResolveByID for fund-flows.
// Ids start above the ecosystem-id range to avoid collisions.
func (k *TrackingEcosystemKeeper) RegisterCorp(addr string) uint64 {
	if v, ok := k.corporationByPolicyAddr[addr]; ok {
		return v.Id
	}
	id := uint64(100000 + len(k.corporationByPolicyAddr) + 1)
	k.corporationByPolicyAddr[addr] = types.CorporationView{Id: id, PolicyAddress: addr}
	return id
}

// TrackingCorporationKeeper resolves a signing policy_address to a
// CorporationView; shares its backing map with TrackingEcosystemKeeper so
// the (ec.CorporationId == co.Id) check passes for the same `creator` that
// registered the Ecosystem.
type TrackingCorporationKeeper struct {
	corporationByPolicyAddr map[string]types.CorporationView
}

func NewTrackingCorporationKeeper(ekKeeper *TrackingEcosystemKeeper) *TrackingCorporationKeeper {
	return &TrackingCorporationKeeper{corporationByPolicyAddr: ekKeeper.corporationByPolicyAddr}
}

func (k *TrackingCorporationKeeper) ResolveByPolicyAddress(ctx context.Context, policyAddress string) (types.CorporationView, bool) {
	v, ok := k.corporationByPolicyAddr[policyAddress]
	return v, ok
}

func (k *TrackingCorporationKeeper) ResolveByID(ctx context.Context, id uint64) (types.CorporationView, bool) {
	for addr, v := range k.corporationByPolicyAddr {
		if v.Id == id {
			return types.CorporationView{Id: id, PolicyAddress: addr}, true
		}
	}
	return types.CorporationView{}, false
}

// setupTrackingMsgServer creates msg server with tracking mocks
func setupTrackingMsgServer(t testing.TB, uaRate, wuaRate, tdRate string, trustUnitPrice uint64) (
	keeper.Keeper,
	types.MsgServer,
	*TrackingCredentialSchemaKeeper,
	*TrackingEcosystemKeeper,
	*TrackingBankKeeper,
	*TrackingTrustDepositKeeper,
	sdk.Context,
) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	// Create tracking keepers
	csKeeper := NewTrackingCredentialSchemaKeeper()
	ekKeeper := NewTrackingEcosystemKeeper(trustUnitPrice)
	coKeeper := NewTrackingCorporationKeeper(ekKeeper)
	bankKeeper := NewTrackingBankKeeper()
	tdKeeper := NewTrackingTrustDepositKeeper(uaRate, wuaRate, tdRate, bankKeeper)

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority.String(),
		csKeeper,
		ekKeeper,
		coKeeper,
		tdKeeper,
		bankKeeper,
		&keepertest.MockDelegationKeeper{}, // permissive mock for CSPS tests
		&keepertest.MockDigestKeeper{},     // permissive mock for CSPS tests
	)

	// Set a specific block time for consistent testing
	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger()).WithBlockTime(blockTime)

	// Initialize params
	if err := k.SetParams(ctx, types.DefaultParams()); err != nil {
		panic(err)
	}

	return k, keeper.NewMsgServerImpl(k), csKeeper, ekKeeper, bankKeeper, tdKeeper, ctx

}

// TestAgentRewardsDistribution tests the complete fee distribution flow including:
// - Balance deductions from creator
// - Balance additions to beneficiaries, agents
// - Trust deposit updates for all parties
// - Verification that total deducted = total distributed
func TestAgentRewardsDistribution(t *testing.T) {
	// Configuration:
	// - user_agent_reward_rate = 10% (0.1)
	// - wallet_user_agent_reward_rate = 5% (0.05)
	// - trust_deposit_rate = 20% (0.2)
	// - trust_unit_price = 1 (for easy calculation)
	k, ms, csKeeper, trkKeeper, bankKeeper, tdKeeper, ctx := setupTrackingMsgServer(t,
		"0.1",  // user_agent_reward_rate
		"0.05", // wallet_user_agent_reward_rate
		"0.2",  // trust_deposit_rate
		1,      // trust_unit_price
	)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Setup addresses
	creatorAddr := sdk.AccAddress([]byte("creator_address_____"))
	ecosystemAddr := sdk.AccAddress([]byte("ecosystem_address___"))
	grantorAddr := sdk.AccAddress([]byte("grantor_address_____"))
	issuerAddr := sdk.AccAddress([]byte("issuer_address______"))
	agentAddr := sdk.AccAddress([]byte("agent_address_______"))
	walletAgentAddr := sdk.AccAddress([]byte("wallet_agent_addr___"))

	creator := creatorAddr.String()
	ecosystem := ecosystemAddr.String()
	grantor := grantorAddr.String()
	_ = issuerAddr.String()
	agent := agentAddr.String()
	walletAgent := walletAgentAddr.String()

	validDid := "did:example:123456789abcdefghi"

	// Setup initial balance for creator (enough to cover all fees)
	initialBalance := sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, 100000))
	bankKeeper.SetBalance(creator, initialBalance)

	// Create trust registry
	trID := trkKeeper.CreateMockEcosystem(ecosystem, validDid)

	// Create credential schema with GRANTOR mode
	csKeeper.UpdateMockCredentialSchema(1, trID,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

	now := sdkCtx.BlockTime()
	pastTime := now.Add(-1 * time.Hour) // Set effective_from to past to make it ACTIVE

	// Create ECOSYSTEM participant (fees: 100)
	ecosystemParticipant := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ECOSYSTEM,
		CorporationId: trkKeeper.RegisterCorp(ecosystem),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		IssuanceFees:  100, // Ecosystem charges 100 trust units for issuance
		EffectiveFrom: &pastTime,
	}
	ecosystemParticipantID, err := k.CreateParticipant(sdkCtx, ecosystemParticipant)
	require.NoError(t, err)

	// Create ISSUER_GRANTOR participant (fees: 50)
	grantorParticipant := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId:          trkKeeper.RegisterCorp(grantor),
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		ValidatorParticipantId: ecosystemParticipantID,
		OpState:                types.OnboardingState_VALIDATED,
		IssuanceFees:           50, // Grantor charges 50 trust units
		EffectiveFrom:          &pastTime,
	}
	grantorParticipantID, err := k.CreateParticipant(sdkCtx, grantorParticipant)
	require.NoError(t, err)

	// Create ISSUER participant (the executor)
	issuerParticipant := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(creator),
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		ValidatorParticipantId: grantorParticipantID,
		OpState:                types.OnboardingState_VALIDATED,
		EffectiveFrom:          &pastTime,
		VsOperator:             creator,
	}
	issuerParticipantID, err := k.CreateParticipant(sdkCtx, issuerParticipant)
	require.NoError(t, err)

	// Create agent participant (User Agent)
	agentParticipant := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(agent),
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		ValidatorParticipantId: issuerParticipantID,
		OpState:                types.OnboardingState_VALIDATED,
		EffectiveFrom:          &pastTime,
	}
	agentParticipantID, err := k.CreateParticipant(sdkCtx, agentParticipant)
	require.NoError(t, err)

	// Create wallet agent participant (Wallet User Agent)
	walletAgentParticipant := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(walletAgent),
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		ValidatorParticipantId: issuerParticipantID,
		OpState:                types.OnboardingState_VALIDATED,
		EffectiveFrom:          &pastTime,
	}
	walletAgentParticipantID, err := k.CreateParticipant(sdkCtx, walletAgentParticipant)
	require.NoError(t, err)

	// ==================== Execute CreateOrUpdateParticipantSession ====================
	msg := &types.MsgCreateOrUpdateParticipantSession{
		Corporation:              creator,
		Operator:                 creator,
		Id:                       uuid.New().String(),
		IssuerParticipantId:      issuerParticipantID,
		VerifierParticipantId:    0,
		AgentParticipantId:       agentParticipantID,
		WalletAgentParticipantId: walletAgentParticipantID,
	}

	resp, err := ms.CreateOrUpdateParticipantSession(ctx, msg)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// ==================== Calculate Expected Values ====================
	// beneficiary_fees = ecosystem.issuance_fees + grantor.issuance_fees = 100 + 50 = 150
	beneficiaryFees := uint64(150)

	// trust_fees = beneficiary_fees * (1 + ua + wua + td) * trust_unit_price
	// trust_fees = 150 * (1 + 0.1 + 0.05 + 0.2) * 1 = 150 * 1.35 = 202.5 -> 202 (truncated)
	// Note: The formula in the code uses additive not multiplicative
	uaRate := math.LegacyMustNewDecFromStr("0.1")
	wuaRate := math.LegacyMustNewDecFromStr("0.05")
	tdRate := math.LegacyMustNewDecFromStr("0.2")

	// For each beneficiary (ecosystem and grantor):
	// participant_total_trust_fees = fees * trust_unit_price
	// user_agent_reward accumulates = participant_total_trust_fees * ua_rate
	// wallet_user_agent_reward accumulates = participant_total_trust_fees * wua_rate

	// Ecosystem: 100 * 1 = 100
	ecosystemTrustFees := math.LegacyNewDec(100)
	ecosystemToTD := ecosystemTrustFees.Mul(tdRate).TruncateInt64()          // 100 * 0.2 = 20
	ecosystemToAccount := ecosystemTrustFees.TruncateInt64() - ecosystemToTD // 100 - 20 = 80

	// Grantor: 50 * 1 = 50
	grantorTrustFees := math.LegacyNewDec(50)
	grantorToTD := grantorTrustFees.Mul(tdRate).TruncateInt64()        // 50 * 0.2 = 10
	grantorToAccount := grantorTrustFees.TruncateInt64() - grantorToTD // 50 - 10 = 40

	// Agent rewards:
	// user_agent_reward = (100 * 0.1) + (50 * 0.1) = 10 + 5 = 15
	// wallet_user_agent_reward = (100 * 0.05) + (50 * 0.05) = 5 + 2.5 = 7.5 -> 7 (truncated)
	totalUserAgentReward := ecosystemTrustFees.Mul(uaRate).Add(grantorTrustFees.Mul(uaRate))     // 15
	totalWalletAgentReward := ecosystemTrustFees.Mul(wuaRate).Add(grantorTrustFees.Mul(wuaRate)) // 7.5

	uaToTD := totalUserAgentReward.Mul(tdRate).TruncateInt64()   // 15 * 0.2 = 3
	uaToAccount := totalUserAgentReward.TruncateInt64() - uaToTD // 15 - 3 = 12

	wuaToTD := totalWalletAgentReward.Mul(tdRate).TruncateInt64()    // 7.5 * 0.2 = 1.5 -> 1
	wuaToAccount := totalWalletAgentReward.TruncateInt64() - wuaToTD // 7 - 1 = 6

	t.Run("Verify beneficiary balance updates", func(t *testing.T) {
		// Ecosystem should receive only direct fees (TD funded via AdjustTrustDepositOnBehalf)
		ecosystemReceived := bankKeeper.GetTotalReceived(ecosystem)
		require.Equal(t, int64(ecosystemToAccount), ecosystemReceived.AmountOf(types.BondDenom).Int64(),
			"Ecosystem should receive only direct fees")

		// Grantor should receive only direct fees
		grantorReceived := bankKeeper.GetTotalReceived(grantor)
		require.Equal(t, int64(grantorToAccount), grantorReceived.AmountOf(types.BondDenom).Int64(),
			"Grantor should receive only direct fees")
	})

	t.Run("Verify agent balance updates", func(t *testing.T) {
		// User agent should receive only direct reward (TD funded via AdjustTrustDepositOnBehalf)
		agentReceived := bankKeeper.GetTotalReceived(agent)
		require.Equal(t, uaToAccount, agentReceived.AmountOf(types.BondDenom).Int64(),
			"User agent should receive only direct reward")

		// Wallet agent should receive only direct reward
		walletAgentReceived := bankKeeper.GetTotalReceived(walletAgent)
		require.Equal(t, wuaToAccount, walletAgentReceived.AmountOf(types.BondDenom).Int64(),
			"Wallet agent should receive only direct reward")
	})

	t.Run("Verify trust deposit updates", func(t *testing.T) {
		// Ecosystem TD should increase by 20
		ecosystemTD := tdKeeper.GetTotalAdjustment(ecosystem)
		require.Equal(t, ecosystemToTD, ecosystemTD,
			"Ecosystem trust deposit should increase")

		// Grantor TD should increase by 10
		grantorTD := tdKeeper.GetTotalAdjustment(grantor)
		require.Equal(t, grantorToTD, grantorTD,
			"Grantor trust deposit should increase")

		// Creator (executor) TD should also increase (by sum of beneficiary TD amounts)
		creatorTD := tdKeeper.GetTotalAdjustment(creator)
		expectedCreatorTD := ecosystemToTD + grantorToTD // 20 + 10 = 30
		require.Equal(t, expectedCreatorTD, creatorTD,
			"Creator trust deposit should increase by sum of beneficiary TD contributions")

		// Agent TD should increase by 3
		agentTD := tdKeeper.GetTotalAdjustment(agent)
		require.Equal(t, uaToTD, agentTD,
			"Agent trust deposit should increase")

		// Wallet Agent TD should increase by 1
		walletAgentTD := tdKeeper.GetTotalAdjustment(walletAgent)
		require.Equal(t, wuaToTD, walletAgentTD,
			"Wallet agent trust deposit should increase")
	})

	t.Run("Verify total deducted equals total distributed", func(t *testing.T) {
		totalDeducted := bankKeeper.GetTotalDeducted(creator)

		// Direct transfers to accounts (fees only)
		directToEcosystem := int64(ecosystemToAccount)
		directToGrantor := int64(grantorToAccount)
		directToAgent := uaToAccount
		directToWalletAgent := wuaToAccount
		totalDirectTransfers := directToEcosystem + directToGrantor + directToAgent + directToWalletAgent

		// Total to module (payer's own TD + on-behalf TD via SendCoinsFromAccountToModule)
		var totalToModule int64
		for _, record := range bankKeeper.ModuleTransferLog {
			totalToModule += record.Amount.AmountOf(types.BondDenom).Int64()
		}

		totalDistributed := totalDirectTransfers + totalToModule

		require.Equal(t, totalDeducted.AmountOf(types.BondDenom).Int64(), totalDistributed,
			"Total deducted from creator should equal total distributed")

		t.Logf("=== Fee Distribution Summary ===")
		t.Logf("Beneficiary fees: %d trust units", beneficiaryFees)
		t.Logf("To ecosystem (fees): %d", directToEcosystem)
		t.Logf("To grantor (fees): %d", directToGrantor)
		t.Logf("To user agent (fees): %d", directToAgent)
		t.Logf("To wallet agent (fees): %d", directToWalletAgent)
		t.Logf("Total to trust deposits (module): %d", totalToModule)
		t.Logf("Total deducted from creator: %d", totalDeducted.AmountOf(types.BondDenom).Int64())
		t.Logf("Total distributed: %d", totalDistributed)
	})

	t.Run("Verify participant deposits updated", func(t *testing.T) {
		// Check that executor participant deposit is updated
		updatedIssuerParticipant, err := k.GetParticipantByID(sdkCtx, issuerParticipantID)
		require.NoError(t, err)
		expectedIssuerDeposit := uint64(ecosystemToTD + grantorToTD)
		require.Equal(t, expectedIssuerDeposit, updatedIssuerParticipant.Deposit,
			"Issuer participant deposit should be updated")

		// Check agent participant deposits
		updatedAgentParticipant, err := k.GetParticipantByID(sdkCtx, agentParticipantID)
		require.NoError(t, err)
		require.Equal(t, uint64(uaToTD), updatedAgentParticipant.Deposit,
			"Agent participant deposit should be updated")

		updatedWalletAgentParticipant, err := k.GetParticipantByID(sdkCtx, walletAgentParticipantID)
		require.NoError(t, err)
		require.Equal(t, uint64(wuaToTD), updatedWalletAgentParticipant.Deposit,
			"Wallet agent participant deposit should be updated")
	})
}

// TestAgentRewardsWithZeroFees tests that no distributions happen when beneficiary fees are zero
func TestAgentRewardsWithZeroFees(t *testing.T) {
	k, ms, csKeeper, trkKeeper, bankKeeper, tdKeeper, ctx := setupTrackingMsgServer(t,
		"0.1",  // user_agent_reward_rate
		"0.05", // wallet_user_agent_reward_rate
		"0.2",  // trust_deposit_rate
		1,      // trust_unit_price
	)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	creatorAddr := sdk.AccAddress([]byte("creator_address_____"))
	ecosystemAddr := sdk.AccAddress([]byte("ecosystem_address___"))
	agentAddr := sdk.AccAddress([]byte("agent_address_______"))
	walletAgentAddr := sdk.AccAddress([]byte("wallet_agent_addr___"))

	creator := creatorAddr.String()
	ecosystem := ecosystemAddr.String()
	agent := agentAddr.String()
	walletAgent := walletAgentAddr.String()

	validDid := "did:example:123456789abcdefghi"

	// Setup initial balance
	initialBalance := sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, 100000))
	bankKeeper.SetBalance(creator, initialBalance)

	// Create trust registry
	trID := trkKeeper.CreateMockEcosystem(ecosystem, validDid)

	// Create credential schema
	csKeeper.UpdateMockCredentialSchema(1, trID,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

	now := sdkCtx.BlockTime()
	pastTime := now.Add(-1 * time.Hour) // Set effective_from to past to make it ACTIVE

	// Create participants with ZERO fees
	ecosystemParticipant := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ECOSYSTEM,
		CorporationId: trkKeeper.RegisterCorp(ecosystem),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		IssuanceFees:  0, // Zero fees
		EffectiveFrom: &pastTime,
	}
	ecosystemParticipantID, err := k.CreateParticipant(sdkCtx, ecosystemParticipant)
	require.NoError(t, err)

	issuerParticipant := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(creator),
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		ValidatorParticipantId: ecosystemParticipantID,
		OpState:                types.OnboardingState_VALIDATED,
		EffectiveFrom:          &pastTime,
		VsOperator:             creator,
	}
	issuerParticipantID, err := k.CreateParticipant(sdkCtx, issuerParticipant)
	require.NoError(t, err)

	agentParticipant := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(agent),
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		ValidatorParticipantId: issuerParticipantID,
		OpState:                types.OnboardingState_VALIDATED,
		EffectiveFrom:          &pastTime,
	}
	agentParticipantID, err := k.CreateParticipant(sdkCtx, agentParticipant)

	require.NoError(t, err)

	// Create wallet agent participant
	walletAgentParticipant := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(walletAgent),
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		ValidatorParticipantId: issuerParticipantID,
		OpState:                types.OnboardingState_VALIDATED,
		EffectiveFrom:          &pastTime,
	}
	walletAgentParticipantID, err := k.CreateParticipant(sdkCtx, walletAgentParticipant)
	require.NoError(t, err)

	msg := &types.MsgCreateOrUpdateParticipantSession{
		Corporation:              creator,
		Operator:                 creator,
		Id:                       uuid.New().String(),
		IssuerParticipantId:      issuerParticipantID,
		VerifierParticipantId:    0,
		AgentParticipantId:       agentParticipantID,
		WalletAgentParticipantId: walletAgentParticipantID,
	}

	resp, err := ms.CreateOrUpdateParticipantSession(ctx, msg)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// No transfers should have occurred
	require.Equal(t, 0, len(bankKeeper.TransferLog), "No direct transfers should occur with zero fees")
	require.Equal(t, 0, len(bankKeeper.ModuleTransferLog), "No module transfers should occur with zero fees")
	require.Equal(t, 0, len(tdKeeper.AdjustmentLog), "No TD adjustments should occur with zero fees")
}

// TestAgentRewardsWithDiscount tests fee distribution with discounts applied
func TestAgentRewardsWithDiscount(t *testing.T) {
	k, ms, csKeeper, trkKeeper, bankKeeper, tdKeeper, ctx := setupTrackingMsgServer(t,
		"0.1",  // user_agent_reward_rate
		"0.05", // wallet_user_agent_reward_rate
		"0.2",  // trust_deposit_rate
		1,      // trust_unit_price
	)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	creatorAddr := sdk.AccAddress([]byte("creator_address_____"))
	ecosystemAddr := sdk.AccAddress([]byte("ecosystem_address___"))
	agentAddr := sdk.AccAddress([]byte("agent_address_______"))
	walletAgentAddr := sdk.AccAddress([]byte("wallet_agent_addr___"))

	creator := creatorAddr.String()
	ecosystem := ecosystemAddr.String()
	agent := agentAddr.String()
	walletAgent := walletAgentAddr.String()

	validDid := "did:example:123456789abcdefghi"

	// Setup initial balance
	initialBalance := sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, 100000))
	bankKeeper.SetBalance(creator, initialBalance)

	// Create trust registry
	trID := trkKeeper.CreateMockEcosystem(ecosystem, validDid)

	// Create credential schema
	csKeeper.UpdateMockCredentialSchema(1, trID,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

	now := sdkCtx.BlockTime()
	pastTime := now.Add(-1 * time.Hour) // Set effective_from to past to make it ACTIVE

	// Create ecosystem participant with 100 fees
	ecosystemParticipant := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ECOSYSTEM,
		CorporationId: trkKeeper.RegisterCorp(ecosystem),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		IssuanceFees:  100,
		EffectiveFrom: &pastTime,
	}
	ecosystemParticipantID, err := k.CreateParticipant(sdkCtx, ecosystemParticipant)
	require.NoError(t, err)

	// Create issuer participant with 50% discount (per Issue #94: use discount instead of exemption)
	issuerParticipant := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(creator),
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		ValidatorParticipantId: ecosystemParticipantID,
		OpState:                types.OnboardingState_VALIDATED,
		IssuanceFeeDiscount:    5000, // 50% discount (per Issue #94)
		EffectiveFrom:          &pastTime,
		VsOperator:             creator,
	}
	issuerParticipantID, err := k.CreateParticipant(sdkCtx, issuerParticipant)
	require.NoError(t, err)

	agentParticipant := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(agent),
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		ValidatorParticipantId: issuerParticipantID,
		OpState:                types.OnboardingState_VALIDATED,
		EffectiveFrom:          &pastTime,
	}
	agentParticipantID, err := k.CreateParticipant(sdkCtx, agentParticipant)

	require.NoError(t, err)

	// Create wallet agent participant
	walletAgentParticipant := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(walletAgent),
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		ValidatorParticipantId: issuerParticipantID,
		OpState:                types.OnboardingState_VALIDATED,
		EffectiveFrom:          &pastTime,
	}
	walletAgentParticipantID, err := k.CreateParticipant(sdkCtx, walletAgentParticipant)
	require.NoError(t, err)

	msg := &types.MsgCreateOrUpdateParticipantSession{
		Corporation:              creator,
		Operator:                 creator,
		Id:                       uuid.New().String(),
		IssuerParticipantId:      issuerParticipantID,
		VerifierParticipantId:    0,
		AgentParticipantId:       agentParticipantID,
		WalletAgentParticipantId: walletAgentParticipantID,
	}

	resp, err := ms.CreateOrUpdateParticipantSession(ctx, msg)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// With 50% discount on 100 fees:
	// discounted_fees = 100 * (1 - 0.5) = 50
	// participant_total_trust_fees = 50 * 1 = 50
	// to_td = 50 * 0.2 = 10
	// to_account = 50 - 10 = 40
	// user_agent_reward = 50 * 0.1 = 5
	// ua_to_td = 5 * 0.2 = 1
	// ua_to_account = 5 - 1 = 4

	t.Run("Verify discount applied correctly", func(t *testing.T) {
		// Ecosystem receives only direct fees (TD funded via AdjustTrustDepositOnBehalf)
		ecosystemReceived := bankKeeper.GetTotalReceived(ecosystem)
		require.Equal(t, int64(40), ecosystemReceived.AmountOf(types.BondDenom).Int64(),
			"Ecosystem should receive 40 direct fees (after 50% discount)")

		// Agent receives only direct reward (TD funded via AdjustTrustDepositOnBehalf)
		agentReceived := bankKeeper.GetTotalReceived(agent)
		require.Equal(t, int64(4), agentReceived.AmountOf(types.BondDenom).Int64(),
			"Agent should receive 4 direct reward")

		ecosystemTD := tdKeeper.GetTotalAdjustment(ecosystem)
		require.Equal(t, int64(10), ecosystemTD,
			"Ecosystem TD should increase by 10")

		agentTD := tdKeeper.GetTotalAdjustment(agent)
		require.Equal(t, int64(1), agentTD,
			"Agent TD should increase by 1")
	})
}
