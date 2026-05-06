package keeper_test

import (
	"context"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	testkeeper "github.com/verana-labs/verana/testutil/keeper"
	cstypes "github.com/verana-labs/verana/x/cs/types"
	detypes "github.com/verana-labs/verana/x/de/types"
	ditypes "github.com/verana-labs/verana/x/di/types"
	permtypes "github.com/verana-labs/verana/x/perm/types"
	tdtypes "github.com/verana-labs/verana/x/td/types"
	trtypes "github.com/verana-labs/verana/x/tr/types"
	xrtypes "github.com/verana-labs/verana/x/xr/types"
)

// Compile-time assertions: StatefulBankMock satisfies every module's
// BankKeeper interface. If any module's interface grows, these will fail.
var (
	_ permtypes.BankKeeper = (*testkeeper.StatefulBankMock)(nil)
	_ tdtypes.BankKeeper   = (*testkeeper.StatefulBankMock)(nil)
	_ cstypes.BankKeeper   = (*testkeeper.StatefulBankMock)(nil)
	_ trtypes.BankKeeper   = (*testkeeper.StatefulBankMock)(nil)
	_ detypes.BankKeeper   = (*testkeeper.StatefulBankMock)(nil)
	_ xrtypes.BankKeeper   = (*testkeeper.StatefulBankMock)(nil)
	_ ditypes.BankKeeper   = (*testkeeper.StatefulBankMock)(nil)
)

// Test addresses, reused across tests.
var (
	denom = "uvna"
	alice = sdk.AccAddress([]byte("alice_______________"))
	bob   = sdk.AccAddress([]byte("bob_________________"))
)

func newMock(t *testing.T) *testkeeper.StatefulBankMock {
	t.Helper()
	return testkeeper.NewStatefulBankMock(testkeeper.DefaultModuleAddrs())
}

// captureT implements require.TestingT and records whether an assertion
// would have failed. Used to test RequireBalanceDelta itself.
type captureT struct {
	failed bool
}

func (c *captureT) Errorf(format string, args ...interface{}) { c.failed = true }
func (c *captureT) FailNow()                                  { c.failed = true }

// Compile-time assertion: captureT implements require.TestingT.
var _ require.TestingT = (*captureT)(nil)

// --- Task 2: SetBalance and BalanceOf ---

func TestSetBalance_GetSet(t *testing.T) {
	m := newMock(t)
	m.SetBalance(alice, denom, 1_000)
	require.Equal(t, int64(1_000), m.BalanceOf(alice, denom))
}

func TestSetBalance_OverwritesPreviousValue(t *testing.T) {
	m := newMock(t)
	m.SetBalance(alice, denom, 1_000)
	m.SetBalance(alice, denom, 2_500)
	require.Equal(t, int64(2_500), m.BalanceOf(alice, denom))
}

func TestBalanceOf_UnsetReturnsZero(t *testing.T) {
	m := newMock(t)
	require.Equal(t, int64(0), m.BalanceOf(alice, denom))
}

func TestSetBalance_PerDenomIsolation(t *testing.T) {
	m := newMock(t)
	m.SetBalance(alice, "uvna", 1_000)
	m.SetBalance(alice, "ustake", 500)
	require.Equal(t, int64(1_000), m.BalanceOf(alice, "uvna"))
	require.Equal(t, int64(500), m.BalanceOf(alice, "ustake"))
}

// --- Task 3: SendCoins ---

func TestSendCoins_HappyPath(t *testing.T) {
	m := newMock(t)
	m.SetBalance(alice, denom, 1_000)
	m.SetBalance(bob, denom, 200)

	err := m.SendCoins(context.Background(), alice, bob,
		sdk.NewCoins(sdk.NewInt64Coin(denom, 300)))
	require.NoError(t, err)

	require.Equal(t, int64(700), m.BalanceOf(alice, denom))
	require.Equal(t, int64(500), m.BalanceOf(bob, denom))
}

func TestSendCoins_MultiCoin(t *testing.T) {
	m := newMock(t)
	m.SetBalance(alice, "uvna", 1_000)
	m.SetBalance(alice, "ustake", 500)

	err := m.SendCoins(context.Background(), alice, bob, sdk.NewCoins(
		sdk.NewInt64Coin("uvna", 100),
		sdk.NewInt64Coin("ustake", 50),
	))
	require.NoError(t, err)

	require.Equal(t, int64(900), m.BalanceOf(alice, "uvna"))
	require.Equal(t, int64(450), m.BalanceOf(alice, "ustake"))
	require.Equal(t, int64(100), m.BalanceOf(bob, "uvna"))
	require.Equal(t, int64(50), m.BalanceOf(bob, "ustake"))
}

func TestSendCoins_InsufficientFundsDoesNotMutate(t *testing.T) {
	m := newMock(t)
	m.SetBalance(alice, denom, 100)
	m.SetBalance(bob, denom, 0)

	err := m.SendCoins(context.Background(), alice, bob,
		sdk.NewCoins(sdk.NewInt64Coin(denom, 200)))
	require.ErrorIs(t, err, sdkerrors.ErrInsufficientFunds)

	require.Equal(t, int64(100), m.BalanceOf(alice, denom))
	require.Equal(t, int64(0), m.BalanceOf(bob, denom))
}

func TestSendCoins_InsufficientOnSecondDenomDoesNotMutateFirst(t *testing.T) {
	// Atomicity: if the second coin has insufficient funds, the first must
	// NOT be debited. Otherwise tests can't reason about partial state.
	m := newMock(t)
	m.SetBalance(alice, "uvna", 1_000)
	m.SetBalance(alice, "ustake", 10)

	err := m.SendCoins(context.Background(), alice, bob, sdk.NewCoins(
		sdk.NewInt64Coin("uvna", 100),
		sdk.NewInt64Coin("ustake", 50),
	))
	require.ErrorIs(t, err, sdkerrors.ErrInsufficientFunds)
	require.Equal(t, int64(1_000), m.BalanceOf(alice, "uvna"))
	require.Equal(t, int64(10), m.BalanceOf(alice, "ustake"))
	require.Equal(t, int64(0), m.BalanceOf(bob, "uvna"))
}
