package keeper_test

import (
	"context"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
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

// --- Task 4: SendCoinsFromAccountToModule ---

func TestSendCoinsFromAccountToModule_HappyPath(t *testing.T) {
	m := newMock(t)
	m.SetBalance(alice, denom, 1_000)
	modAddr := authtypes.NewModuleAddress("perm")

	err := m.SendCoinsFromAccountToModule(context.Background(), alice, "perm",
		sdk.NewCoins(sdk.NewInt64Coin(denom, 300)))
	require.NoError(t, err)

	require.Equal(t, int64(700), m.BalanceOf(alice, denom))
	require.Equal(t, int64(300), m.BalanceOf(modAddr, denom))
}

func TestSendCoinsFromAccountToModule_InsufficientFunds(t *testing.T) {
	m := newMock(t)
	m.SetBalance(alice, denom, 50)

	err := m.SendCoinsFromAccountToModule(context.Background(), alice, "perm",
		sdk.NewCoins(sdk.NewInt64Coin(denom, 300)))
	require.ErrorIs(t, err, sdkerrors.ErrInsufficientFunds)
	require.Equal(t, int64(50), m.BalanceOf(alice, denom))
	require.Equal(t, int64(0), m.BalanceOf(authtypes.NewModuleAddress("perm"), denom))
}

func TestSendCoinsFromAccountToModule_UnknownModulePanics(t *testing.T) {
	m := newMock(t)
	m.SetBalance(alice, denom, 1_000)
	require.PanicsWithValue(t,
		`StatefulBankMock: unknown module "no_such_mod" (register it in NewStatefulBankMock's modAddrs)`,
		func() {
			_ = m.SendCoinsFromAccountToModule(context.Background(), alice, "no_such_mod",
				sdk.NewCoins(sdk.NewInt64Coin(denom, 1)))
		})
}

// --- Task 5: SendCoinsFromModuleToAccount ---

func TestSendCoinsFromModuleToAccount_HappyPath(t *testing.T) {
	m := newMock(t)
	modAddr := authtypes.NewModuleAddress("td")
	m.SetBalance(modAddr, denom, 5_000)

	err := m.SendCoinsFromModuleToAccount(context.Background(), "td", alice,
		sdk.NewCoins(sdk.NewInt64Coin(denom, 1_200)))
	require.NoError(t, err)
	require.Equal(t, int64(3_800), m.BalanceOf(modAddr, denom))
	require.Equal(t, int64(1_200), m.BalanceOf(alice, denom))
}

func TestSendCoinsFromModuleToAccount_InsufficientFunds(t *testing.T) {
	m := newMock(t)
	modAddr := authtypes.NewModuleAddress("td")
	m.SetBalance(modAddr, denom, 100)

	err := m.SendCoinsFromModuleToAccount(context.Background(), "td", alice,
		sdk.NewCoins(sdk.NewInt64Coin(denom, 200)))
	require.ErrorIs(t, err, sdkerrors.ErrInsufficientFunds)
	require.Equal(t, int64(100), m.BalanceOf(modAddr, denom))
	require.Equal(t, int64(0), m.BalanceOf(alice, denom))
}

func TestSendCoinsFromModuleToAccount_UnknownModulePanics(t *testing.T) {
	m := newMock(t)
	require.Panics(t, func() {
		_ = m.SendCoinsFromModuleToAccount(context.Background(), "ghost", alice,
			sdk.NewCoins(sdk.NewInt64Coin(denom, 1)))
	})
}

// --- Task 6: SendCoinsFromModuleToModule ---

func TestSendCoinsFromModuleToModule_HappyPath(t *testing.T) {
	m := newMock(t)
	tdAddr := authtypes.NewModuleAddress("td")
	yipAddr := authtypes.NewModuleAddress("yield_intermediate_pool")
	m.SetBalance(tdAddr, denom, 10_000)

	err := m.SendCoinsFromModuleToModule(context.Background(), "td", "yield_intermediate_pool",
		sdk.NewCoins(sdk.NewInt64Coin(denom, 4_000)))
	require.NoError(t, err)
	require.Equal(t, int64(6_000), m.BalanceOf(tdAddr, denom))
	require.Equal(t, int64(4_000), m.BalanceOf(yipAddr, denom))
}

func TestSendCoinsFromModuleToModule_InsufficientFunds(t *testing.T) {
	m := newMock(t)
	tdAddr := authtypes.NewModuleAddress("td")
	m.SetBalance(tdAddr, denom, 50)

	err := m.SendCoinsFromModuleToModule(context.Background(), "td", "yield_intermediate_pool",
		sdk.NewCoins(sdk.NewInt64Coin(denom, 100)))
	require.ErrorIs(t, err, sdkerrors.ErrInsufficientFunds)
}

func TestSendCoinsFromModuleToModule_UnknownSenderPanics(t *testing.T) {
	m := newMock(t)
	require.Panics(t, func() {
		_ = m.SendCoinsFromModuleToModule(context.Background(), "ghost", "td",
			sdk.NewCoins(sdk.NewInt64Coin(denom, 1)))
	})
}

func TestSendCoinsFromModuleToModule_UnknownRecipientPanics(t *testing.T) {
	m := newMock(t)
	require.Panics(t, func() {
		_ = m.SendCoinsFromModuleToModule(context.Background(), "td", "ghost",
			sdk.NewCoins(sdk.NewInt64Coin(denom, 1)))
	})
}

// --- Task 7: BurnCoins ---

func TestBurnCoins_HappyPath(t *testing.T) {
	m := newMock(t)
	tdAddr := authtypes.NewModuleAddress("td")
	m.SetBalance(tdAddr, denom, 5_000)

	err := m.BurnCoins(context.Background(), "td",
		sdk.NewCoins(sdk.NewInt64Coin(denom, 1_500)))
	require.NoError(t, err)
	require.Equal(t, int64(3_500), m.BalanceOf(tdAddr, denom))
}

func TestBurnCoins_InsufficientFunds(t *testing.T) {
	m := newMock(t)
	tdAddr := authtypes.NewModuleAddress("td")
	m.SetBalance(tdAddr, denom, 100)

	err := m.BurnCoins(context.Background(), "td",
		sdk.NewCoins(sdk.NewInt64Coin(denom, 1_000)))
	require.ErrorIs(t, err, sdkerrors.ErrInsufficientFunds)
	require.Equal(t, int64(100), m.BalanceOf(tdAddr, denom))
}

func TestBurnCoins_UnknownModulePanics(t *testing.T) {
	m := newMock(t)
	require.Panics(t, func() {
		_ = m.BurnCoins(context.Background(), "ghost",
			sdk.NewCoins(sdk.NewInt64Coin(denom, 1)))
	})
}
