package keeper_test

import (
	"context"
	"sync"
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

// --- Task 8: read methods ---

func TestHasBalance_TrueWhenSufficient(t *testing.T) {
	m := newMock(t)
	m.SetBalance(alice, denom, 100)
	require.True(t, m.HasBalance(context.Background(), alice, sdk.NewInt64Coin(denom, 100)))
	require.True(t, m.HasBalance(context.Background(), alice, sdk.NewInt64Coin(denom, 50)))
}

func TestHasBalance_FalseWhenInsufficient(t *testing.T) {
	m := newMock(t)
	m.SetBalance(alice, denom, 100)
	require.False(t, m.HasBalance(context.Background(), alice, sdk.NewInt64Coin(denom, 101)))
	require.False(t, m.HasBalance(context.Background(), bob, sdk.NewInt64Coin(denom, 1)))
}

func TestGetBalance_ReflectsState(t *testing.T) {
	m := newMock(t)
	m.SetBalance(alice, denom, 250)
	got := m.GetBalance(context.Background(), alice, denom)
	require.Equal(t, denom, got.Denom)
	require.Equal(t, int64(250), got.Amount.Int64())
}

func TestGetBalance_UnsetIsZeroCoin(t *testing.T) {
	m := newMock(t)
	got := m.GetBalance(context.Background(), alice, denom)
	require.Equal(t, denom, got.Denom)
	require.True(t, got.Amount.IsZero())
}

func TestGetAllBalances_AllDenoms(t *testing.T) {
	m := newMock(t)
	m.SetBalance(alice, "uvna", 100)
	m.SetBalance(alice, "ustake", 50)

	got := m.GetAllBalances(context.Background(), alice)
	require.Equal(t, sdk.NewCoins(
		sdk.NewInt64Coin("uvna", 100),
		sdk.NewInt64Coin("ustake", 50),
	), got)
}

func TestGetAllBalances_UnsetReturnsEmpty(t *testing.T) {
	m := newMock(t)
	require.Empty(t, m.GetAllBalances(context.Background(), alice))
}

func TestGetAllBalances_SkipsZero(t *testing.T) {
	// After a transfer that drains a denom to zero, GetAllBalances should
	// not surface the zero entry — sdk.Coins is canonically sorted with
	// only non-zero amounts.
	m := newMock(t)
	m.SetBalance(alice, denom, 100)
	require.NoError(t, m.SendCoins(context.Background(), alice, bob,
		sdk.NewCoins(sdk.NewInt64Coin(denom, 100))))

	got := m.GetAllBalances(context.Background(), alice)
	require.Empty(t, got)
}

func TestSpendableCoins_MatchesGetAllBalances(t *testing.T) {
	m := newMock(t)
	m.SetBalance(alice, "uvna", 100)
	m.SetBalance(alice, "ustake", 50)
	require.Equal(t, m.GetAllBalances(context.Background(), alice),
		m.SpendableCoins(context.Background(), alice))
}

// --- Task 9: Calls() accessor and call-recording verification ---

func TestCalls_Empty(t *testing.T) {
	m := newMock(t)
	require.Empty(t, m.Calls())
}

func TestCalls_RecordsEachMethodInOrder(t *testing.T) {
	m := newMock(t)
	m.SetBalance(alice, denom, 10_000)
	tdAddr := authtypes.NewModuleAddress("td")
	yipAddr := authtypes.NewModuleAddress("yield_intermediate_pool")
	m.SetBalance(tdAddr, denom, 10_000)
	m.SetBalance(yipAddr, denom, 0)

	require.NoError(t, m.SendCoins(context.Background(), alice, bob,
		sdk.NewCoins(sdk.NewInt64Coin(denom, 10))))
	require.NoError(t, m.SendCoinsFromAccountToModule(context.Background(), alice, "td",
		sdk.NewCoins(sdk.NewInt64Coin(denom, 20))))
	require.NoError(t, m.SendCoinsFromModuleToAccount(context.Background(), "td", bob,
		sdk.NewCoins(sdk.NewInt64Coin(denom, 30))))
	require.NoError(t, m.SendCoinsFromModuleToModule(context.Background(), "td", "yield_intermediate_pool",
		sdk.NewCoins(sdk.NewInt64Coin(denom, 40))))
	require.NoError(t, m.BurnCoins(context.Background(), "td",
		sdk.NewCoins(sdk.NewInt64Coin(denom, 50))))

	calls := m.Calls()
	require.Len(t, calls, 5)
	require.Equal(t, "SendCoins", calls[0].Method)
	require.Equal(t, alice.String(), calls[0].From)
	require.Equal(t, bob.String(), calls[0].To)
	require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin(denom, 10)), calls[0].Amount)

	require.Equal(t, "SendCoinsFromAccountToModule", calls[1].Method)
	require.Equal(t, tdAddr.String(), calls[1].To)

	require.Equal(t, "SendCoinsFromModuleToAccount", calls[2].Method)
	require.Equal(t, tdAddr.String(), calls[2].From)
	require.Equal(t, bob.String(), calls[2].To)

	require.Equal(t, "SendCoinsFromModuleToModule", calls[3].Method)
	require.Equal(t, tdAddr.String(), calls[3].From)
	require.Equal(t, yipAddr.String(), calls[3].To)

	require.Equal(t, "BurnCoins", calls[4].Method)
	require.Equal(t, tdAddr.String(), calls[4].From)
	require.Equal(t, "", calls[4].To)
}

func TestCalls_FailedTransferIsNotRecorded(t *testing.T) {
	m := newMock(t)
	m.SetBalance(alice, denom, 1)
	err := m.SendCoins(context.Background(), alice, bob,
		sdk.NewCoins(sdk.NewInt64Coin(denom, 100)))
	require.Error(t, err)
	require.Empty(t, m.Calls(), "failed transfers must not appear in call history")
}

func TestCalls_ReturnsCopyNotInternalSlice(t *testing.T) {
	m := newMock(t)
	m.SetBalance(alice, denom, 100)
	require.NoError(t, m.SendCoins(context.Background(), alice, bob,
		sdk.NewCoins(sdk.NewInt64Coin(denom, 1))))

	c1 := m.Calls()
	c1[0].Method = "tampered"

	c2 := m.Calls()
	require.Equal(t, "SendCoins", c2[0].Method, "Calls() must return a defensive copy")
}

// --- Task 10: RequireBalanceDelta ---

func TestRequireBalanceDelta_NoChange(t *testing.T) {
	m := newMock(t)
	m.SetBalance(alice, denom, 1_000)
	m.RequireBalanceDelta(t, alice, denom, 0)
}

func TestRequireBalanceDelta_NegativeAfterSend(t *testing.T) {
	m := newMock(t)
	m.SetBalance(alice, denom, 1_000)
	m.SetBalance(bob, denom, 0)
	require.NoError(t, m.SendCoins(context.Background(), alice, bob,
		sdk.NewCoins(sdk.NewInt64Coin(denom, 300))))
	m.RequireBalanceDelta(t, alice, denom, -300)
	m.RequireBalanceDelta(t, bob, denom, +300)
}

func TestRequireBalanceDelta_UnsetBaselineTreatedAsZero(t *testing.T) {
	m := newMock(t)
	m.SetBalance(alice, denom, 500)
	require.NoError(t, m.SendCoinsFromAccountToModule(context.Background(), alice, "perm",
		sdk.NewCoins(sdk.NewInt64Coin(denom, 200))))
	// perm module addr was never SetBalance'd → baseline 0 → delta +200.
	m.RequireBalanceDelta(t, authtypes.NewModuleAddress("perm"), denom, +200)
}

func TestRequireBalanceDelta_FailsOnWrongDelta(t *testing.T) {
	m := newMock(t)
	m.SetBalance(alice, denom, 1_000)
	m.SetBalance(bob, denom, 0)
	require.NoError(t, m.SendCoins(context.Background(), alice, bob,
		sdk.NewCoins(sdk.NewInt64Coin(denom, 300))))

	cap := &captureT{}
	m.RequireBalanceDelta(cap, alice, denom, -100) // wrong: actual is -300
	require.True(t, cap.failed, "RequireBalanceDelta must fail on wrong delta")
}

func TestRequireBalanceDelta_RebaselinesOnSetBalance(t *testing.T) {
	// Calling SetBalance again resets the baseline.
	m := newMock(t)
	m.SetBalance(alice, denom, 1_000)
	require.NoError(t, m.SendCoins(context.Background(), alice, bob,
		sdk.NewCoins(sdk.NewInt64Coin(denom, 100))))
	require.Equal(t, int64(900), m.BalanceOf(alice, denom))

	m.SetBalance(alice, denom, 5_000) // new baseline
	m.RequireBalanceDelta(t, alice, denom, 0)
}

// --- Task 11: Concurrency safety ---

func TestStatefulBankMock_ConcurrentTransfersAreRaceFree(t *testing.T) {
	m := newMock(t)
	addrs := make([]sdk.AccAddress, 16)
	for i := range addrs {
		var a [20]byte
		a[0] = byte(i)
		addrs[i] = sdk.AccAddress(a[:])
		m.SetBalance(addrs[i], denom, 1_000_000)
	}

	const workers = 8
	const ops = 200
	var wg sync.WaitGroup
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func(w int) {
			defer wg.Done()
			for i := 0; i < ops; i++ {
				from := addrs[(w+i)%len(addrs)]
				to := addrs[(w+i+1)%len(addrs)]
				_ = m.SendCoins(context.Background(), from, to,
					sdk.NewCoins(sdk.NewInt64Coin(denom, 1)))
				_ = m.HasBalance(context.Background(), from, sdk.NewInt64Coin(denom, 1))
				_ = m.GetAllBalances(context.Background(), to)
				_ = m.Calls()
			}
		}(w)
	}
	wg.Wait()

	// Conservation: total amount across all addrs is unchanged.
	var total int64
	for _, a := range addrs {
		total += m.BalanceOf(a, denom)
	}
	require.Equal(t, int64(16_000_000), total)
}
