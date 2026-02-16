package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	dekeeper "github.com/verana-labs/verana/x/de/keeper"
	demodule "github.com/verana-labs/verana/x/de/module"
	detypes "github.com/verana-labs/verana/x/de/types"
	trkeeper "github.com/verana-labs/verana/x/tr/keeper"
	trtypes "github.com/verana-labs/verana/x/tr/types"
)

// integrationFixture holds both DE and TR keepers wired together so that the
// TR module's AUTHZ-CHECK calls the real DE module's CheckOperatorAuthorization.
type integrationFixture struct {
	deKeeper       dekeeper.Keeper
	deMsgServer    detypes.MsgServer
	trKeeper       trkeeper.Keeper
	trMsgServer    trtypes.MsgServer
	ctx            sdk.Context
	addressCodec   address.Codec
}

// MockTrustDepositKeeper is a no-op trust deposit keeper for integration tests.
type mockTrustDepositKeeper struct{}

func (m *mockTrustDepositKeeper) AdjustTrustDeposit(_ sdk.Context, _ string, _ int64) error {
	return nil
}

func setupIntegrationFixture(t *testing.T) *integrationFixture {
	t.Helper()

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())

	// Store keys for both modules
	deStoreKey := storetypes.NewKVStoreKey(detypes.StoreKey)
	trStoreKey := storetypes.NewKVStoreKey(trtypes.StoreKey)

	stateStore.MountStoreWithDB(deStoreKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(trStoreKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	// DE module setup
	deEncCfg := moduletestutil.MakeTestEncodingConfig(demodule.AppModule{})
	deAddressCodec := addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	deAuthority := authtypes.NewModuleAddress(detypes.GovModuleName)

	deK := dekeeper.NewKeeper(
		runtime.NewKVStoreService(deStoreKey),
		deEncCfg.Codec,
		deAddressCodec,
		deAuthority,
	)

	// TR module setup — uses the real DE keeper for AUTHZ-CHECK
	trRegistry := codectypes.NewInterfaceRegistry()
	trCdc := codec.NewProtoCodec(trRegistry)
	trAuthority := authtypes.NewModuleAddress(govtypes.ModuleName)

	trK := trkeeper.NewKeeper(
		trCdc,
		runtime.NewKVStoreService(trStoreKey),
		log.NewNopLogger(),
		trAuthority.String(),
		&mockTrustDepositKeeper{},
		deK, // real DE keeper as DelegationKeeper
	)

	// Shared context with a deterministic block time
	ctx := sdk.NewContext(stateStore, cmtproto.Header{
		Time: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}, false, log.NewNopLogger())

	// Initialize params for both modules
	require.NoError(t, deK.Params.Set(ctx, detypes.DefaultParams()))
	require.NoError(t, trK.SetParams(ctx, trtypes.DefaultParams()))

	return &integrationFixture{
		deKeeper:     deK,
		deMsgServer:  dekeeper.NewMsgServerImpl(deK),
		trKeeper:     trK,
		trMsgServer:  trkeeper.NewMsgServerImpl(trK),
		ctx:          ctx,
		addressCodec: deAddressCodec,
	}
}

// ---------------------------------------------------------------------------
// Integration tests: DE operator authorization → TR create trust registry
// ---------------------------------------------------------------------------

// TestIntegration_OperatorCreatesTrustRegistry models the full real-world flow:
//
//  1. Group proposal onboards an operator with MsgCreateTrustRegistry permission.
//  2. The operator creates a trust registry on behalf of the group authority.
//  3. Verify the trust registry was created with the group as controller.
func TestIntegration_OperatorCreatesTrustRegistry(t *testing.T) {
	f := setupIntegrationFixture(t)

	groupAccount := sdk.AccAddress([]byte("group_policy_addr___")).String()
	operator := sdk.AccAddress([]byte("test_operator_______")).String()

	// ---- Step 1: Group proposal onboards operator via DE module ----
	_, err := f.deMsgServer.GrantOperatorAuthorization(f.ctx, &detypes.MsgGrantOperatorAuthorization{
		Authority: groupAccount,
		Operator:  "", // group proposal
		Grantee:   operator,
		MsgTypes:  []string{"/verana.tr.v1.MsgCreateTrustRegistry"},
	})
	require.NoError(t, err)

	// ---- Step 2: Operator creates trust registry via TR module ----
	_, err = f.trMsgServer.CreateTrustRegistry(f.ctx, &trtypes.MsgCreateTrustRegistry{
		Authority:    groupAccount,
		Operator:     operator,
		Did:          "did:example:integration-test-123",
		Language:     "en",
		DocUrl:       "https://example.com/governance-framework",
		DocDigestSri: "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
	})
	require.NoError(t, err)

	// ---- Step 3: Verify the trust registry ----
	trID, err := f.trKeeper.TrustRegistryDIDIndex.Get(f.ctx, "did:example:integration-test-123")
	require.NoError(t, err)

	tr, err := f.trKeeper.TrustRegistry.Get(f.ctx, trID)
	require.NoError(t, err)
	require.Equal(t, "did:example:integration-test-123", tr.Did)
	require.Equal(t, groupAccount, tr.Controller) // authority is the controller
	require.Equal(t, "en", tr.Language)
	require.Equal(t, int32(1), tr.ActiveVersion)
}

// TestIntegration_UnauthorizedOperatorCannotCreateTrustRegistry verifies
// that an operator without authorization is rejected by AUTHZ-CHECK.
func TestIntegration_UnauthorizedOperatorCannotCreateTrustRegistry(t *testing.T) {
	f := setupIntegrationFixture(t)

	groupAccount := sdk.AccAddress([]byte("group_policy_addr___")).String()
	unauthorizedOp := sdk.AccAddress([]byte("unauthorized_op_____")).String()

	// No operator authorization granted — attempt should fail
	_, err := f.trMsgServer.CreateTrustRegistry(f.ctx, &trtypes.MsgCreateTrustRegistry{
		Authority:    groupAccount,
		Operator:     unauthorizedOp,
		Did:          "did:example:should-fail",
		Language:     "en",
		DocUrl:       "https://example.com/doc",
		DocDigestSri: "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "authorization check failed")
	require.Contains(t, err.Error(), "operator authorization not found")
}

// TestIntegration_OperatorWithWrongMsgTypeCannotCreateTrustRegistry verifies
// that an operator authorized for a different msg type is rejected.
func TestIntegration_OperatorWithWrongMsgTypeCannotCreateTrustRegistry(t *testing.T) {
	f := setupIntegrationFixture(t)

	groupAccount := sdk.AccAddress([]byte("group_policy_addr___")).String()
	operator := sdk.AccAddress([]byte("wrong_type_op_______")).String()

	// Grant operator authorization for a DIFFERENT msg type
	_, err := f.deMsgServer.GrantOperatorAuthorization(f.ctx, &detypes.MsgGrantOperatorAuthorization{
		Authority: groupAccount,
		Operator:  "", // group proposal
		Grantee:   operator,
		MsgTypes:  []string{"/verana.cs.v1.MsgCreateCredentialSchema"}, // NOT MsgCreateTrustRegistry
	})
	require.NoError(t, err)

	// Operator tries to create trust registry — should fail
	_, err = f.trMsgServer.CreateTrustRegistry(f.ctx, &trtypes.MsgCreateTrustRegistry{
		Authority:    groupAccount,
		Operator:     operator,
		Did:          "did:example:wrong-type",
		Language:     "en",
		DocUrl:       "https://example.com/doc",
		DocDigestSri: "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "authorization check failed")
	require.Contains(t, err.Error(), "does not include requested message type")
}

// TestIntegration_ExpiredOperatorCannotCreateTrustRegistry verifies that
// an operator with an expired authorization is rejected.
func TestIntegration_ExpiredOperatorCannotCreateTrustRegistry(t *testing.T) {
	f := setupIntegrationFixture(t)

	groupAccount := sdk.AccAddress([]byte("group_policy_addr___")).String()
	operator := sdk.AccAddress([]byte("expired_op__________")).String()

	// Manually create an expired OperatorAuthorization in the DE store
	pastTime := f.ctx.BlockTime().Add(-1 * time.Hour)
	oaKey := collections.Join(groupAccount, operator)
	err := f.deKeeper.OperatorAuthorizations.Set(f.ctx, oaKey, detypes.OperatorAuthorization{
		Authority:  groupAccount,
		Operator:   operator,
		MsgTypes:   []string{"/verana.tr.v1.MsgCreateTrustRegistry"},
		Expiration: &pastTime,
	})
	require.NoError(t, err)

	// Operator tries to create trust registry — should fail
	_, err = f.trMsgServer.CreateTrustRegistry(f.ctx, &trtypes.MsgCreateTrustRegistry{
		Authority:    groupAccount,
		Operator:     operator,
		Did:          "did:example:expired",
		Language:     "en",
		DocUrl:       "https://example.com/doc",
		DocDigestSri: "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "authorization check failed")
	require.Contains(t, err.Error(), "expired")
}

// TestIntegration_FullBootstrapToTrustRegistryCreation models the complete
// end-to-end flow from group proposal to trust registry creation:
//
//  1. Group proposal onboards first operator (admin + TR permissions).
//  2. First operator onboards a second operator (TR-only permissions).
//  3. Second operator creates a trust registry.
//  4. Second operator cannot grant further operators (no DE permissions).
//  5. Group revokes second operator.
//  6. Revoked operator cannot create trust registries anymore.
func TestIntegration_FullBootstrapToTrustRegistryCreation(t *testing.T) {
	f := setupIntegrationFixture(t)

	groupAccount := sdk.AccAddress([]byte("group_policy_addr___")).String()
	adminOperator := sdk.AccAddress([]byte("admin_operator______")).String()
	trOperator := sdk.AccAddress([]byte("tr_operator_________")).String()

	// ---- Step 1: Group proposal onboards admin operator ----
	_, err := f.deMsgServer.GrantOperatorAuthorization(f.ctx, &detypes.MsgGrantOperatorAuthorization{
		Authority: groupAccount,
		Operator:  "", // group proposal
		Grantee:   adminOperator,
		MsgTypes: []string{
			"/verana.de.v1.MsgGrantOperatorAuthorization",
			"/verana.de.v1.MsgRevokeOperatorAuthorization",
			"/verana.tr.v1.MsgCreateTrustRegistry",
		},
	})
	require.NoError(t, err)

	// ---- Step 2: Admin operator onboards TR-only operator ----
	_, err = f.deMsgServer.GrantOperatorAuthorization(f.ctx, &detypes.MsgGrantOperatorAuthorization{
		Authority: groupAccount,
		Operator:  adminOperator, // admin cosigns
		Grantee:   trOperator,
		MsgTypes:  []string{"/verana.tr.v1.MsgCreateTrustRegistry"},
	})
	require.NoError(t, err)

	// ---- Step 3: TR operator creates a trust registry ----
	_, err = f.trMsgServer.CreateTrustRegistry(f.ctx, &trtypes.MsgCreateTrustRegistry{
		Authority:    groupAccount,
		Operator:     trOperator,
		Did:          "did:example:full-bootstrap-test",
		Aka:          "https://example.com/my-registry",
		Language:     "en",
		DocUrl:       "https://example.com/governance-framework-v1",
		DocDigestSri: "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
	})
	require.NoError(t, err)

	// Verify trust registry created
	trID, err := f.trKeeper.TrustRegistryDIDIndex.Get(f.ctx, "did:example:full-bootstrap-test")
	require.NoError(t, err)
	tr, err := f.trKeeper.TrustRegistry.Get(f.ctx, trID)
	require.NoError(t, err)
	require.Equal(t, groupAccount, tr.Controller)
	require.Equal(t, "https://example.com/my-registry", tr.Aka)

	// ---- Step 4: TR operator CANNOT grant further operators ----
	_, err = f.deMsgServer.GrantOperatorAuthorization(f.ctx, &detypes.MsgGrantOperatorAuthorization{
		Authority: groupAccount,
		Operator:  trOperator, // TR operator doesn't have DE grant permission
		Grantee:   sdk.AccAddress([]byte("another_operator____")).String(),
		MsgTypes:  []string{"/verana.tr.v1.MsgCreateTrustRegistry"},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not include requested message type")

	// ---- Step 5: Group revokes TR operator ----
	_, err = f.deMsgServer.RevokeOperatorAuthorization(f.ctx, &detypes.MsgRevokeOperatorAuthorization{
		Authority: groupAccount,
		Operator:  "", // group proposal
		Grantee:   trOperator,
	})
	require.NoError(t, err)

	// ---- Step 6: Revoked operator cannot create trust registries ----
	_, err = f.trMsgServer.CreateTrustRegistry(f.ctx, &trtypes.MsgCreateTrustRegistry{
		Authority:    groupAccount,
		Operator:     trOperator,
		Did:          "did:example:should-fail-revoked",
		Language:     "en",
		DocUrl:       "https://example.com/doc",
		DocDigestSri: "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "authorization check failed")
}
