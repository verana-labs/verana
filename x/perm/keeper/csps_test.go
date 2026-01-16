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

	cstypes "github.com/verana-labs/verana/x/cs/types"
	"github.com/verana-labs/verana/x/perm/keeper"
	"github.com/verana-labs/verana/x/perm/types"
	trtypes "github.com/verana-labs/verana/x/tr/types"
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
	UserAgentRewardRate math.LegacyDec
	WalletUARewardRate  math.LegacyDec
	TrustDepositRate    math.LegacyDec
}

type TrustDepositAdjustment struct {
	Account string
	Amount  int64
}

func NewTrackingTrustDepositKeeper(uaRate, wuaRate, tdRate string) *TrackingTrustDepositKeeper {
	ua, _ := math.LegacyNewDecFromStr(uaRate)
	wua, _ := math.LegacyNewDecFromStr(wuaRate)
	td, _ := math.LegacyNewDecFromStr(tdRate)
	return &TrackingTrustDepositKeeper{
		TrustDeposits:       make(map[string]int64),
		AdjustmentLog:       make([]TrustDepositAdjustment, 0),
		UserAgentRewardRate: ua,
		WalletUARewardRate:  wua,
		TrustDepositRate:    td,
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

func (m *TrackingTrustDepositKeeper) AdjustTrustDeposit(ctx sdk.Context, account string, augend int64) error {
	m.TrustDeposits[account] += augend
	m.AdjustmentLog = append(m.AdjustmentLog, TrustDepositAdjustment{Account: account, Amount: augend})
	return nil
}

func (m *TrackingTrustDepositKeeper) GetTotalAdjustment(account string) int64 {
	total := int64(0)
	for _, adj := range m.AdjustmentLog {
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

func (k *TrackingCredentialSchemaKeeper) UpdateMockCredentialSchema(id uint64, trId uint64, issuerPermMode, verifierPermMode cstypes.CredentialSchemaPermManagementMode) {
	k.credentialSchemas[id] = cstypes.CredentialSchema{
		Id:                         id,
		TrId:                       trId,
		IssuerPermManagementMode:   issuerPermMode,
		VerifierPermManagementMode: verifierPermMode,
	}
}

func (k *TrackingCredentialSchemaKeeper) GetCredentialSchemaById(ctx sdk.Context, id uint64) (cstypes.CredentialSchema, error) {
	if cs, ok := k.credentialSchemas[id]; ok {
		return cs, nil
	}
	return cstypes.CredentialSchema{}, cstypes.ErrCredentialSchemaNotFound
}

// TrackingTrustRegistryKeeper for test setup
type TrackingTrustRegistryKeeper struct {
	trustRegistries map[uint64]trtypes.TrustRegistry
	trustUnitPrice  uint64
}

func NewTrackingTrustRegistryKeeper(trustUnitPrice uint64) *TrackingTrustRegistryKeeper {
	return &TrackingTrustRegistryKeeper{
		trustRegistries: make(map[uint64]trtypes.TrustRegistry),
		trustUnitPrice:  trustUnitPrice,
	}
}

func (k *TrackingTrustRegistryKeeper) GetTrustUnitPrice(ctx sdk.Context) uint64 {
	return k.trustUnitPrice
}

func (k *TrackingTrustRegistryKeeper) GetTrustRegistry(ctx sdk.Context, id uint64) (trtypes.TrustRegistry, error) {
	if tr, ok := k.trustRegistries[id]; ok {
		return tr, nil
	}
	return trtypes.TrustRegistry{}, trtypes.ErrTrustRegistryNotFound
}

func (k *TrackingTrustRegistryKeeper) CreateMockTrustRegistry(creator string, did string) uint64 {
	id := uint64(len(k.trustRegistries) + 1)
	k.trustRegistries[id] = trtypes.TrustRegistry{
		Id:            id,
		Did:           did,
		Controller:    creator,
		ActiveVersion: 1,
		Language:      "en",
	}
	return id
}

// setupTrackingMsgServer creates msg server with tracking mocks
func setupTrackingMsgServer(t testing.TB, uaRate, wuaRate, tdRate string, trustUnitPrice uint64) (
	keeper.Keeper,
	types.MsgServer,
	*TrackingCredentialSchemaKeeper,
	*TrackingTrustRegistryKeeper,
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
	trkKeeper := NewTrackingTrustRegistryKeeper(trustUnitPrice)
	bankKeeper := NewTrackingBankKeeper()
	tdKeeper := NewTrackingTrustDepositKeeper(uaRate, wuaRate, tdRate)

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority.String(),
		csKeeper,
		trkKeeper,
		tdKeeper,
		bankKeeper,
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	// Initialize params
	if err := k.SetParams(ctx, types.DefaultParams()); err != nil {
		panic(err)
	}

	return k, keeper.NewMsgServerImpl(k), csKeeper, trkKeeper, bankKeeper, tdKeeper, ctx
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
	issuer := issuerAddr.String()
	agent := agentAddr.String()
	walletAgent := walletAgentAddr.String()

	validDid := "did:example:123456789abcdefghi"

	// Setup initial balance for creator (enough to cover all fees)
	initialBalance := sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, 100000))
	bankKeeper.SetBalance(creator, initialBalance)

	// Create trust registry
	trID := trkKeeper.CreateMockTrustRegistry(ecosystem, validDid)

	// Create credential schema with GRANTOR mode
	csKeeper.UpdateMockCredentialSchema(1, trID,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION)

	now := time.Now()

	// Create ECOSYSTEM permission (fees: 100)
	ecosystemPerm := types.Permission{
		SchemaId:     1,
		Type:         types.PermissionType_ECOSYSTEM,
		Grantee:      ecosystem,
		Created:      &now,
		CreatedBy:    ecosystem,
		Extended:     &now,
		ExtendedBy:   ecosystem,
		Modified:     &now,
		Country:      "US",
		VpState:      types.ValidationState_VALIDATED,
		IssuanceFees: 100, // Ecosystem charges 100 trust units for issuance
	}
	ecosystemPermID, err := k.CreatePermission(sdkCtx, ecosystemPerm)
	require.NoError(t, err)

	// Create ISSUER_GRANTOR permission (fees: 50)
	grantorPerm := types.Permission{
		SchemaId:        1,
		Type:            types.PermissionType_ISSUER_GRANTOR,
		Grantee:         grantor,
		Created:         &now,
		CreatedBy:       ecosystem,
		Extended:        &now,
		ExtendedBy:      ecosystem,
		Modified:        &now,
		Country:         "US",
		ValidatorPermId: ecosystemPermID,
		VpState:         types.ValidationState_VALIDATED,
		IssuanceFees:    50, // Grantor charges 50 trust units
	}
	grantorPermID, err := k.CreatePermission(sdkCtx, grantorPerm)
	require.NoError(t, err)

	// Create ISSUER permission (the executor)
	issuerPerm := types.Permission{
		SchemaId:        1,
		Type:            types.PermissionType_ISSUER,
		Grantee:         issuer,
		Created:         &now,
		CreatedBy:       grantor,
		Extended:        &now,
		ExtendedBy:      grantor,
		Modified:        &now,
		Country:         "US",
		ValidatorPermId: grantorPermID,
		VpState:         types.ValidationState_VALIDATED,
	}
	issuerPermID, err := k.CreatePermission(sdkCtx, issuerPerm)
	require.NoError(t, err)

	// Create agent permission (User Agent)
	agentPerm := types.Permission{
		SchemaId:        1,
		Type:            types.PermissionType_ISSUER,
		Grantee:         agent,
		Created:         &now,
		CreatedBy:       issuer,
		Extended:        &now,
		ExtendedBy:      issuer,
		Modified:        &now,
		Country:         "US",
		ValidatorPermId: issuerPermID,
		VpState:         types.ValidationState_VALIDATED,
	}
	agentPermID, err := k.CreatePermission(sdkCtx, agentPerm)
	require.NoError(t, err)

	// Create wallet agent permission (Wallet User Agent)
	walletAgentPerm := types.Permission{
		SchemaId:        1,
		Type:            types.PermissionType_ISSUER,
		Grantee:         walletAgent,
		Created:         &now,
		CreatedBy:       issuer,
		Extended:        &now,
		ExtendedBy:      issuer,
		Modified:        &now,
		Country:         "US",
		ValidatorPermId: issuerPermID,
		VpState:         types.ValidationState_VALIDATED,
	}
	walletAgentPermID, err := k.CreatePermission(sdkCtx, walletAgentPerm)
	require.NoError(t, err)

	// ==================== Execute CreateOrUpdatePermissionSession ====================
	msg := &types.MsgCreateOrUpdatePermissionSession{
		Creator:           creator,
		Id:                uuid.New().String(),
		IssuerPermId:      issuerPermID,
		VerifierPermId:    0,
		AgentPermId:       agentPermID,
		WalletAgentPermId: walletAgentPermID,
	}

	resp, err := ms.CreateOrUpdatePermissionSession(ctx, msg)
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
	// perm_total_trust_fees = fees * trust_unit_price
	// user_agent_reward accumulates = perm_total_trust_fees * ua_rate
	// wallet_user_agent_reward accumulates = perm_total_trust_fees * wua_rate

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
		// Ecosystem should receive 80 directly
		ecosystemReceived := bankKeeper.GetTotalReceived(ecosystem)
		require.Equal(t, int64(ecosystemToAccount), ecosystemReceived.AmountOf(types.BondDenom).Int64(),
			"Ecosystem should receive direct fees")

		// Grantor should receive 40 directly
		grantorReceived := bankKeeper.GetTotalReceived(grantor)
		require.Equal(t, int64(grantorToAccount), grantorReceived.AmountOf(types.BondDenom).Int64(),
			"Grantor should receive direct fees")
	})

	t.Run("Verify agent balance updates", func(t *testing.T) {
		// User agent should receive 12 directly
		agentReceived := bankKeeper.GetTotalReceived(agent)
		require.Equal(t, uaToAccount, agentReceived.AmountOf(types.BondDenom).Int64(),
			"User agent should receive direct reward")

		// Wallet agent should receive 6 directly
		walletAgentReceived := bankKeeper.GetTotalReceived(walletAgent)
		require.Equal(t, wuaToAccount, walletAgentReceived.AmountOf(types.BondDenom).Int64(),
			"Wallet agent should receive direct reward")
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

		// Total distributed to accounts (direct transfers)
		directToEcosystem := int64(ecosystemToAccount)
		directToGrantor := int64(grantorToAccount)
		directToAgent := uaToAccount
		directToWalletAgent := wuaToAccount
		totalDirectTransfers := directToEcosystem + directToGrantor + directToAgent + directToWalletAgent

		// Total to module (trust deposits)
		var totalToModule int64
		for _, record := range bankKeeper.ModuleTransferLog {
			totalToModule += record.Amount.AmountOf(types.BondDenom).Int64()
		}

		totalDistributed := totalDirectTransfers + totalToModule

		require.Equal(t, totalDeducted.AmountOf(types.BondDenom).Int64(), totalDistributed,
			"Total deducted from creator should equal total distributed")

		t.Logf("=== Fee Distribution Summary ===")
		t.Logf("Beneficiary fees: %d trust units", beneficiaryFees)
		t.Logf("Direct to ecosystem: %d", directToEcosystem)
		t.Logf("Direct to grantor: %d", directToGrantor)
		t.Logf("Direct to user agent: %d", directToAgent)
		t.Logf("Direct to wallet agent: %d", directToWalletAgent)
		t.Logf("Total to trust deposits (module): %d", totalToModule)
		t.Logf("Total deducted from creator: %d", totalDeducted.AmountOf(types.BondDenom).Int64())
		t.Logf("Total distributed: %d", totalDistributed)
	})

	t.Run("Verify permission deposits updated", func(t *testing.T) {
		// Check that executor permission deposit is updated
		updatedIssuerPerm, err := k.GetPermissionByID(sdkCtx, issuerPermID)
		require.NoError(t, err)
		expectedIssuerDeposit := uint64(ecosystemToTD + grantorToTD)
		require.Equal(t, expectedIssuerDeposit, updatedIssuerPerm.Deposit,
			"Issuer permission deposit should be updated")

		// Check agent permission deposits
		updatedAgentPerm, err := k.GetPermissionByID(sdkCtx, agentPermID)
		require.NoError(t, err)
		require.Equal(t, uint64(uaToTD), updatedAgentPerm.Deposit,
			"Agent permission deposit should be updated")

		updatedWalletAgentPerm, err := k.GetPermissionByID(sdkCtx, walletAgentPermID)
		require.NoError(t, err)
		require.Equal(t, uint64(wuaToTD), updatedWalletAgentPerm.Deposit,
			"Wallet agent permission deposit should be updated")
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
	issuerAddr := sdk.AccAddress([]byte("issuer_address______"))
	agentAddr := sdk.AccAddress([]byte("agent_address_______"))

	creator := creatorAddr.String()
	ecosystem := ecosystemAddr.String()
	issuerAcc := issuerAddr.String()
	agent := agentAddr.String()

	validDid := "did:example:123456789abcdefghi"

	// Setup initial balance
	initialBalance := sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, 100000))
	bankKeeper.SetBalance(creator, initialBalance)

	// Create trust registry
	trID := trkKeeper.CreateMockTrustRegistry(ecosystem, validDid)

	// Create credential schema
	csKeeper.UpdateMockCredentialSchema(1, trID,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION)

	now := time.Now()

	// Create permissions with ZERO fees
	ecosystemPerm := types.Permission{
		SchemaId:     1,
		Type:         types.PermissionType_ECOSYSTEM,
		Grantee:      ecosystem,
		Created:      &now,
		CreatedBy:    ecosystem,
		Extended:     &now,
		ExtendedBy:   ecosystem,
		Modified:     &now,
		Country:      "US",
		VpState:      types.ValidationState_VALIDATED,
		IssuanceFees: 0, // Zero fees
	}
	ecosystemPermID, err := k.CreatePermission(sdkCtx, ecosystemPerm)
	require.NoError(t, err)

	issuerPerm := types.Permission{
		SchemaId:        1,
		Type:            types.PermissionType_ISSUER,
		Grantee:         issuerAcc,
		Created:         &now,
		CreatedBy:       ecosystem,
		Extended:        &now,
		ExtendedBy:      ecosystem,
		Modified:        &now,
		Country:         "US",
		ValidatorPermId: ecosystemPermID,
		VpState:         types.ValidationState_VALIDATED,
	}
	issuerPermID, err := k.CreatePermission(sdkCtx, issuerPerm)
	require.NoError(t, err)

	agentPerm := types.Permission{
		SchemaId:        1,
		Type:            types.PermissionType_ISSUER,
		Grantee:         agent,
		Created:         &now,
		CreatedBy:       issuerAcc,
		Extended:        &now,
		ExtendedBy:      issuerAcc,
		Modified:        &now,
		Country:         "US",
		ValidatorPermId: issuerPermID,
		VpState:         types.ValidationState_VALIDATED,
	}
	agentPermID, err := k.CreatePermission(sdkCtx, agentPerm)
	require.NoError(t, err)

	msg := &types.MsgCreateOrUpdatePermissionSession{
		Creator:           creator,
		Id:                uuid.New().String(),
		IssuerPermId:      issuerPermID,
		VerifierPermId:    0,
		AgentPermId:       agentPermID,
		WalletAgentPermId: 0, // No wallet agent
	}

	resp, err := ms.CreateOrUpdatePermissionSession(ctx, msg)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// No transfers should have occurred
	require.Equal(t, 0, len(bankKeeper.TransferLog), "No direct transfers should occur with zero fees")
	require.Equal(t, 0, len(bankKeeper.ModuleTransferLog), "No module transfers should occur with zero fees")
	require.Equal(t, 0, len(tdKeeper.AdjustmentLog), "No TD adjustments should occur with zero fees")
}

// TestAgentRewardsWithExemption tests fee distribution with exemptions applied
func TestAgentRewardsWithExemption(t *testing.T) {
	k, ms, csKeeper, trkKeeper, bankKeeper, tdKeeper, ctx := setupTrackingMsgServer(t,
		"0.1",  // user_agent_reward_rate
		"0.05", // wallet_user_agent_reward_rate
		"0.2",  // trust_deposit_rate
		1,      // trust_unit_price
	)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	creatorAddr := sdk.AccAddress([]byte("creator_address_____"))
	ecosystemAddr := sdk.AccAddress([]byte("ecosystem_address___"))
	issuerAddr := sdk.AccAddress([]byte("issuer_address______"))
	agentAddr := sdk.AccAddress([]byte("agent_address_______"))

	creator := creatorAddr.String()
	ecosystem := ecosystemAddr.String()
	issuerAcc := issuerAddr.String()
	agent := agentAddr.String()

	validDid := "did:example:123456789abcdefghi"

	// Setup initial balance
	initialBalance := sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, 100000))
	bankKeeper.SetBalance(creator, initialBalance)

	// Create trust registry
	trID := trkKeeper.CreateMockTrustRegistry(ecosystem, validDid)

	// Create credential schema
	csKeeper.UpdateMockCredentialSchema(1, trID,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION)

	now := time.Now()

	// Create ecosystem permission with 100 fees
	ecosystemPerm := types.Permission{
		SchemaId:     1,
		Type:         types.PermissionType_ECOSYSTEM,
		Grantee:      ecosystem,
		Created:      &now,
		CreatedBy:    ecosystem,
		Extended:     &now,
		ExtendedBy:   ecosystem,
		Modified:     &now,
		Country:      "US",
		VpState:      types.ValidationState_VALIDATED,
		IssuanceFees: 100,
	}
	ecosystemPermID, err := k.CreatePermission(sdkCtx, ecosystemPerm)
	require.NoError(t, err)

	// Create issuer permission with 50% exemption
	issuerPerm := types.Permission{
		SchemaId:             1,
		Type:                 types.PermissionType_ISSUER,
		Grantee:              issuerAcc,
		Created:              &now,
		CreatedBy:            ecosystem,
		Extended:             &now,
		ExtendedBy:           ecosystem,
		Modified:             &now,
		Country:              "US",
		ValidatorPermId:      ecosystemPermID,
		VpState:              types.ValidationState_VALIDATED,
		IssuanceFeeExemption: 5000, // 50% exemption
	}
	issuerPermID, err := k.CreatePermission(sdkCtx, issuerPerm)
	require.NoError(t, err)

	agentPerm := types.Permission{
		SchemaId:        1,
		Type:            types.PermissionType_ISSUER,
		Grantee:         agent,
		Created:         &now,
		CreatedBy:       issuerAcc,
		Extended:        &now,
		ExtendedBy:      issuerAcc,
		Modified:        &now,
		Country:         "US",
		ValidatorPermId: issuerPermID,
		VpState:         types.ValidationState_VALIDATED,
	}
	agentPermID, err := k.CreatePermission(sdkCtx, agentPerm)
	require.NoError(t, err)

	msg := &types.MsgCreateOrUpdatePermissionSession{
		Creator:           creator,
		Id:                uuid.New().String(),
		IssuerPermId:      issuerPermID,
		VerifierPermId:    0,
		AgentPermId:       agentPermID,
		WalletAgentPermId: 0,
	}

	resp, err := ms.CreateOrUpdatePermissionSession(ctx, msg)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// With 50% exemption on 100 fees:
	// exempted_fees = 100 * (1 - 0.5) = 50
	// perm_total_trust_fees = 50 * 1 = 50
	// to_td = 50 * 0.2 = 10
	// to_account = 50 - 10 = 40
	// user_agent_reward = 50 * 0.1 = 5
	// ua_to_td = 5 * 0.2 = 1
	// ua_to_account = 5 - 1 = 4

	t.Run("Verify exemption applied correctly", func(t *testing.T) {
		ecosystemReceived := bankKeeper.GetTotalReceived(ecosystem)
		require.Equal(t, int64(40), ecosystemReceived.AmountOf(types.BondDenom).Int64(),
			"Ecosystem should receive 40 (after 50% exemption)")

		agentReceived := bankKeeper.GetTotalReceived(agent)
		require.Equal(t, int64(4), agentReceived.AmountOf(types.BondDenom).Int64(),
			"Agent should receive 4")

		ecosystemTD := tdKeeper.GetTotalAdjustment(ecosystem)
		require.Equal(t, int64(10), ecosystemTD,
			"Ecosystem TD should increase by 10")

		agentTD := tdKeeper.GetTotalAdjustment(agent)
		require.Equal(t, int64(1), agentTD,
			"Agent TD should increase by 1")
	})
}
