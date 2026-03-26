package keeper_test

import (
	"fmt"
	"testing"

	storetypes "cosmossdk.io/store/types"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana/testutil/keeper"
	"github.com/verana-labs/verana/x/di/keeper"
	module "github.com/verana-labs/verana/x/di/module"
	"github.com/verana-labs/verana/x/di/types"
)

// initFixtureWithMock creates a test fixture with a MockDelegationKeeper wired in.
func initFixtureWithMock(t *testing.T, mock *keepertest.MockDelegationKeeper) *fixture {
	t.Helper()

	encCfg := moduletestutil.MakeTestEncodingConfig(module.AppModule{})
	addrCodec := addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	storeService := runtime.NewKVStoreService(storeKey)
	ctx := testutil.DefaultContextWithDB(t, storeKey, storetypes.NewTransientStoreKey("transient_test")).Ctx

	authority := authtypes.NewModuleAddress(types.GovModuleName)

	k := keeper.NewKeeper(
		storeService,
		encCfg.Codec,
		addrCodec,
		authority,
		mock,
	)

	if err := k.Params.Set(ctx, types.DefaultParams()); err != nil {
		t.Fatalf("failed to set params: %v", err)
	}

	return &fixture{
		ctx:          ctx,
		keeper:       k,
		addressCodec: addrCodec,
	}
}

func TestStoreDigest_HappyPath(t *testing.T) {
	mock := &keepertest.MockDelegationKeeper{}
	f := initFixtureWithMock(t, mock)
	ms := keeper.NewMsgServerImpl(f.keeper)

	authority := sdk.AccAddress([]byte("authority_address_")).String()
	operator := sdk.AccAddress([]byte("operator_address__")).String()
	digest := "sha256-abc123def456"

	resp, err := ms.StoreDigest(f.ctx, &types.MsgStoreDigest{
		Authority: authority,
		Operator:  operator,
		Digest:    digest,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify digest was stored
	stored, err := f.keeper.Digests.Get(f.ctx, digest)
	require.NoError(t, err)
	require.Equal(t, digest, stored.Digest)
}

func TestStoreDigest_EmptyDigest(t *testing.T) {
	// ValidateBasic is called before the handler in a real chain, so test it directly.
	authority := sdk.AccAddress([]byte("authority_address_")).String()
	operator := sdk.AccAddress([]byte("operator_address__")).String()

	msg := &types.MsgStoreDigest{
		Authority: authority,
		Operator:  operator,
		Digest:    "",
	}
	err := msg.ValidateBasic()
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrDigestEmpty)
}

func TestStoreDigest_InvalidAuthority(t *testing.T) {
	msg := &types.MsgStoreDigest{
		Authority: "bad-address",
		Operator:  sdk.AccAddress([]byte("operator_address__")).String(),
		Digest:    "sha256-abc123",
	}
	err := msg.ValidateBasic()
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid authority address")
}

func TestStoreDigest_InvalidOperator(t *testing.T) {
	msg := &types.MsgStoreDigest{
		Authority: sdk.AccAddress([]byte("authority_address_")).String(),
		Operator:  "bad-operator",
		Digest:    "sha256-abc123",
	}
	err := msg.ValidateBasic()
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid operator address")
}

func TestStoreDigest_AuthzFailure(t *testing.T) {
	mock := &keepertest.MockDelegationKeeper{
		ErrToReturn: fmt.Errorf("not authorized"),
	}
	f := initFixtureWithMock(t, mock)
	ms := keeper.NewMsgServerImpl(f.keeper)

	authority := sdk.AccAddress([]byte("authority_address_")).String()
	operator := sdk.AccAddress([]byte("operator_address__")).String()

	resp, err := ms.StoreDigest(f.ctx, &types.MsgStoreDigest{
		Authority: authority,
		Operator:  operator,
		Digest:    "sha256-abc123",
	})
	require.Error(t, err)
	require.Nil(t, resp)
	require.Contains(t, err.Error(), "authorization check failed")
}

func TestStoreDigest_AuthzSuccess(t *testing.T) {
	// MockDelegationKeeper with nil ErrToReturn → authorization passes.
	mock := &keepertest.MockDelegationKeeper{}
	f := initFixtureWithMock(t, mock)
	ms := keeper.NewMsgServerImpl(f.keeper)

	authority := sdk.AccAddress([]byte("authority_address_")).String()
	operator := sdk.AccAddress([]byte("operator_address__")).String()
	digest := "sha256-authz-success"

	resp, err := ms.StoreDigest(f.ctx, &types.MsgStoreDigest{
		Authority: authority,
		Operator:  operator,
		Digest:    digest,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify state
	stored, err := f.keeper.Digests.Get(f.ctx, digest)
	require.NoError(t, err)
	require.Equal(t, digest, stored.Digest)
}

func TestStoreDigest_DuplicateDigest(t *testing.T) {
	mock := &keepertest.MockDelegationKeeper{}
	f := initFixtureWithMock(t, mock)
	ms := keeper.NewMsgServerImpl(f.keeper)

	authority := sdk.AccAddress([]byte("authority_address_")).String()
	operator := sdk.AccAddress([]byte("operator_address__")).String()
	digest := "sha256-duplicate"

	msg := &types.MsgStoreDigest{
		Authority: authority,
		Operator:  operator,
		Digest:    digest,
	}

	// Store first time
	resp, err := ms.StoreDigest(f.ctx, msg)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Store second time — should succeed (overwrite)
	resp2, err := ms.StoreDigest(f.ctx, msg)
	require.NoError(t, err)
	require.NotNil(t, resp2)

	// Verify digest still present
	stored, err := f.keeper.Digests.Get(f.ctx, digest)
	require.NoError(t, err)
	require.Equal(t, digest, stored.Digest)
}

func TestStoreDigestModuleCall(t *testing.T) {
	// StoreDigestModuleCall bypasses AUTHZ, so nil delegation keeper is fine.
	f := initFixture(t)

	authority := sdk.AccAddress([]byte("authority_address_")).String()

	testCases := []struct {
		name      string
		authority string
		digest    string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid digest stored",
			authority: authority,
			digest:    "sha256-modulecall",
			wantErr:   false,
		},
		{
			name:      "empty digest rejected",
			authority: authority,
			digest:    "",
			wantErr:   true,
			errMsg:    "digest must not be empty",
		},
		{
			name:      "duplicate digest succeeds",
			authority: authority,
			digest:    "sha256-modulecall",
			wantErr:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := f.keeper.StoreDigestModuleCall(f.ctx, tc.authority, tc.digest)
			if tc.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
			} else {
				require.NoError(t, err)

				// Verify state
				stored, err := f.keeper.Digests.Get(f.ctx, tc.digest)
				require.NoError(t, err)
				require.Equal(t, tc.digest, stored.Digest)
			}
		})
	}
}
