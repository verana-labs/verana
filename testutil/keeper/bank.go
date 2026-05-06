package keeper

import (
	"context"
	"fmt"
	"sync"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
)

// BankCall records one invocation of a StatefulBankMock send/burn method.
// From and To are the resolved bech32 addresses (module names are resolved
// via the modAddrs map at the time of the call). To is empty for BurnCoins.
type BankCall struct {
	Method string
	From   string
	To     string
	Amount sdk.Coins
}

// StatefulBankMock is a stateful test double for the BankKeeper interface
// used across verana modules. It tracks balances per (addr, denom), enforces
// deductions, errors on insufficient funds with sdkerrors.ErrInsufficientFunds,
// and records every send/burn call so tests can assert exact movements.
//
// Construct with NewStatefulBankMock and a module-name → module-account-address
// map (use DefaultModuleAddrs for the standard verana set).
type StatefulBankMock struct {
	mu       sync.Mutex
	balances map[string]map[string]int64 // addr -> denom -> amount
	baseline map[string]map[string]int64 // addr -> denom -> baseline (set by SetBalance)
	modAddrs map[string]sdk.AccAddress   // module name -> module account address
	calls    []BankCall
}

// NewStatefulBankMock returns a new mock pre-wired with the supplied
// module-name → module-account-address mapping. Every module name referenced
// by SendCoinsFromAccountToModule, SendCoinsFromModuleToAccount,
// SendCoinsFromModuleToModule, or BurnCoins MUST be in modAddrs; unknown
// names panic to surface test wiring bugs loudly.
func NewStatefulBankMock(modAddrs map[string]sdk.AccAddress) *StatefulBankMock {
	return &StatefulBankMock{
		balances: map[string]map[string]int64{},
		baseline: map[string]map[string]int64{},
		modAddrs: modAddrs,
	}
}

// DefaultModuleAddrs returns the verana module-name → module-account-address
// mapping for every module that moves funds, plus the yield_intermediate_pool
// sub-account used by td. Sourced from authtypes.NewModuleAddress, mirroring
// what the real app does at startup.
func DefaultModuleAddrs() map[string]sdk.AccAddress {
	names := []string{
		"tr", "cs", "td", "perm", "de", "xr", "di",
		"gov", "yield_intermediate_pool",
	}
	out := make(map[string]sdk.AccAddress, len(names))
	for _, n := range names {
		out[n] = authtypes.NewModuleAddress(n)
	}
	return out
}

// --- Mock-only helpers (not part of the BankKeeper interface) ---

// SetBalance sets the balance for (addr, denom) and snapshots it as the
// baseline used by RequireBalanceDelta. Subsequent SetBalance calls on the
// same (addr, denom) reset both current and baseline.
func (m *StatefulBankMock) SetBalance(addr sdk.AccAddress, denom string, amount int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := addr.String()
	if m.balances[key] == nil {
		m.balances[key] = map[string]int64{}
	}
	if m.baseline[key] == nil {
		m.baseline[key] = map[string]int64{}
	}
	m.balances[key][denom] = amount
	m.baseline[key][denom] = amount
}

// BalanceOf returns the int64 balance for (addr, denom). It is the
// assertion-friendly sibling of GetBalance (which returns sdk.Coin to
// satisfy the BankKeeper interface).
func (m *StatefulBankMock) BalanceOf(addr sdk.AccAddress, denom string) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	if d, ok := m.balances[addr.String()]; ok {
		return d[denom]
	}
	return 0
}

// Calls returns a defensive copy of the recorded call history. Tests can
// mutate the returned slice without affecting the mock.
func (m *StatefulBankMock) Calls() []BankCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]BankCall, len(m.calls))
	copy(out, m.calls)
	return out
}

// RequireBalanceDelta asserts that (current - baseline) for (addr, denom)
// equals delta. The baseline is whatever SetBalance most recently wrote
// for (addr, denom); if SetBalance was never called for that pair, the
// baseline is 0.
func (m *StatefulBankMock) RequireBalanceDelta(t require.TestingT, addr sdk.AccAddress, denom string, delta int64) {
	if h, ok := t.(interface{ Helper() }); ok {
		h.Helper()
	}
	m.mu.Lock()
	current, baseline := int64(0), int64(0)
	if d, ok := m.balances[addr.String()]; ok {
		current = d[denom]
	}
	if d, ok := m.baseline[addr.String()]; ok {
		baseline = d[denom]
	}
	m.mu.Unlock()
	got := current - baseline
	require.Equalf(t, delta, got,
		"balance delta for %s [%s]: want %d, got %d (baseline=%d, current=%d)",
		addr.String(), denom, delta, got, baseline, current)
}

// --- BankKeeper interface methods ---

// SendCoins moves amt from->to. It is atomic: if any denom is insufficient,
// no balance is mutated and sdkerrors.ErrInsufficientFunds is returned.
func (m *StatefulBankMock) SendCoins(ctx context.Context, from, to sdk.AccAddress, amt sdk.Coins) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	fromKey, toKey := from.String(), to.String()
	if err := m.transfer(fromKey, toKey, amt); err != nil {
		return err
	}
	m.calls = append(m.calls, BankCall{Method: "SendCoins", From: fromKey, To: toKey, Amount: copyCoins(amt)})
	return nil
}

// SendCoinsFromAccountToModule debits sender, credits the resolved address
// of recipientModule. Panics if recipientModule is not registered in modAddrs.
func (m *StatefulBankMock) SendCoinsFromAccountToModule(ctx context.Context, sender sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	modAddr := m.resolveModule(recipientModule)
	m.mu.Lock()
	defer m.mu.Unlock()
	fromKey, toKey := sender.String(), modAddr.String()
	if err := m.transfer(fromKey, toKey, amt); err != nil {
		return err
	}
	m.calls = append(m.calls, BankCall{Method: "SendCoinsFromAccountToModule", From: fromKey, To: toKey, Amount: copyCoins(amt)})
	return nil
}

// SendCoinsFromModuleToAccount debits the resolved address of senderModule,
// credits recipient. Panics if senderModule is not registered.
func (m *StatefulBankMock) SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipient sdk.AccAddress, amt sdk.Coins) error {
	modAddr := m.resolveModule(senderModule)
	m.mu.Lock()
	defer m.mu.Unlock()
	fromKey, toKey := modAddr.String(), recipient.String()
	if err := m.transfer(fromKey, toKey, amt); err != nil {
		return err
	}
	m.calls = append(m.calls, BankCall{Method: "SendCoinsFromModuleToAccount", From: fromKey, To: toKey, Amount: copyCoins(amt)})
	return nil
}

// SendCoinsFromModuleToModule debits senderModule's resolved address,
// credits recipientModule's. Panics if either is not registered.
func (m *StatefulBankMock) SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error {
	fromAddr := m.resolveModule(senderModule)
	toAddr := m.resolveModule(recipientModule)
	m.mu.Lock()
	defer m.mu.Unlock()
	fromKey, toKey := fromAddr.String(), toAddr.String()
	if err := m.transfer(fromKey, toKey, amt); err != nil {
		return err
	}
	m.calls = append(m.calls, BankCall{Method: "SendCoinsFromModuleToModule", From: fromKey, To: toKey, Amount: copyCoins(amt)})
	return nil
}

// BurnCoins debits the resolved address of name; nothing is credited
// elsewhere. Errors atomically with sdkerrors.ErrInsufficientFunds if the
// module account is short. Panics if name is not registered.
func (m *StatefulBankMock) BurnCoins(ctx context.Context, name string, amt sdk.Coins) error {
	modAddr := m.resolveModule(name)
	m.mu.Lock()
	defer m.mu.Unlock()
	fromKey := modAddr.String()
	for _, c := range amt {
		need := nonNegativeAmount(c)
		have := int64(0)
		if d, ok := m.balances[fromKey]; ok {
			have = d[c.Denom]
		}
		if have < need {
			return sdkerrors.ErrInsufficientFunds.Wrapf(
				"%s: have %d%s, need %d%s to burn", fromKey, have, c.Denom, need, c.Denom)
		}
	}
	for _, c := range amt {
		m.balances[fromKey][c.Denom] -= nonNegativeAmount(c)
	}
	m.calls = append(m.calls, BankCall{Method: "BurnCoins", From: fromKey, To: "", Amount: copyCoins(amt)})
	return nil
}

// HasBalance returns true iff addr's balance of amt.Denom is at least amt.Amount.
func (m *StatefulBankMock) HasBalance(ctx context.Context, addr sdk.AccAddress, amt sdk.Coin) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	have := int64(0)
	if d, ok := m.balances[addr.String()]; ok {
		have = d[amt.Denom]
	}
	return have >= nonNegativeAmount(amt)
}

// GetBalance returns the sdk.Coin for (addr, denom). Unset balances yield a
// zero coin in the requested denom (denom preserved).
func (m *StatefulBankMock) GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	m.mu.Lock()
	defer m.mu.Unlock()
	have := int64(0)
	if d, ok := m.balances[addr.String()]; ok {
		have = d[denom]
	}
	return sdk.NewCoin(denom, math.NewInt(have))
}

// GetAllBalances returns all non-zero balances held by addr, sorted in
// canonical sdk.Coins order. Zero-amount denoms are omitted.
func (m *StatefulBankMock) GetAllBalances(ctx context.Context, addr sdk.AccAddress) sdk.Coins {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := sdk.Coins{}
	if d, ok := m.balances[addr.String()]; ok {
		for denom, amt := range d {
			if amt > 0 {
				out = append(out, sdk.NewCoin(denom, math.NewInt(amt)))
			}
		}
	}
	return out.Sort()
}

// SpendableCoins matches GetAllBalances for the mock — there are no
// vesting or escrow constraints in the test surface.
func (m *StatefulBankMock) SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins {
	return m.GetAllBalances(ctx, addr)
}

// resolveModule returns the module account address for name, or panics if
// the module is not registered in modAddrs.
func (m *StatefulBankMock) resolveModule(name string) sdk.AccAddress {
	a, ok := m.modAddrs[name]
	if !ok {
		panic(fmt.Sprintf("StatefulBankMock: unknown module %q (register it in NewStatefulBankMock's modAddrs)", name))
	}
	return a
}

// transfer is the atomic core. It assumes m.mu is held. It first verifies
// the sender has enough of every denom, then performs all debits and
// credits. Returns sdkerrors.ErrInsufficientFunds (wrapped with detail) on
// failure without mutating any balance.
func (m *StatefulBankMock) transfer(fromKey, toKey string, amt sdk.Coins) error {
	for _, c := range amt {
		need := nonNegativeAmount(c)
		have := int64(0)
		if d, ok := m.balances[fromKey]; ok {
			have = d[c.Denom]
		}
		if have < need {
			return sdkerrors.ErrInsufficientFunds.Wrapf(
				"%s: have %d%s, need %d%s", fromKey, have, c.Denom, need, c.Denom)
		}
	}
	// fromKey's inner map is guaranteed non-nil at this point: the check
	// above only passes when have>=need>0, which means balances[fromKey]
	// was already initialized when have was read. toKey may still be new.
	for _, c := range amt {
		amount := nonNegativeAmount(c)
		if m.balances[toKey] == nil {
			m.balances[toKey] = map[string]int64{}
		}
		m.balances[fromKey][c.Denom] -= amount
		m.balances[toKey][c.Denom] += amount
	}
	return nil
}

// copyCoins returns a shallow copy of the slice header so that the
// caller-supplied amt cannot be mutated post-record. sdk.Coin elements are
// value-copied; their math.Int internals are immutable.
func copyCoins(amt sdk.Coins) sdk.Coins {
	out := make(sdk.Coins, len(amt))
	copy(out, amt)
	return out
}

// nonNegativeAmount returns coin.Amount.Int64() and panics if it would
// overflow or be negative — sdk.Coins must always be sorted, non-negative.
func nonNegativeAmount(coin sdk.Coin) int64 {
	if coin.Amount.IsNegative() {
		panic(fmt.Sprintf("StatefulBankMock: negative amount for denom %s", coin.Denom))
	}
	if !coin.Amount.IsInt64() {
		panic(fmt.Sprintf("StatefulBankMock: amount for denom %s exceeds int64", coin.Denom))
	}
	return coin.Amount.Int64()
}
