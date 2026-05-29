# Step 0: StatefulBankMock Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a stateful `StatefulBankMock` in `testutil/keeper/bank.go` that satisfies every module's `BankKeeper` interface, tracks balances, enforces deductions, errors on insufficient funds, and records call history — plus its own test suite. This is the gate for the test overhaul (issue #292): no per-module fixture work begins until this is solid.

**Architecture:** New `bank.go` and `bank_test.go` in package `keeper` under `testutil/keeper/`. The new mock lives **alongside** the existing no-op `MockBankKeeper` (in `testutil/keeper/credentialschema.go`) — the legacy mock is intentionally not removed in this step (per spec: "the no-op mock remains for legacy test files until they are migrated"). Internal balance map is `map[addr]map[denom]int64`; concurrency via `sync.Mutex`; module-name → module-account-address mapping is supplied at construction time. A baseline snapshot is captured per `(addr, denom)` whenever `SetBalance` is called, so `RequireBalanceDelta` can make assertions independent of the implementation under test.

**Tech Stack:** Go 1.24, Cosmos SDK v0.50.x (`github.com/cosmos/cosmos-sdk/types`, `cosmossdk.io/math`, `github.com/cosmos/cosmos-sdk/x/auth/types`, `github.com/cosmos/cosmos-sdk/types/errors` aliased as `sdkerrors`), `github.com/stretchr/testify/require`, `sync`.

---

## Pre-flight (one-time, before Task 1)

- [ ] **Worktree.** Create an isolated worktree for this step (use the `superpowers:using-git-worktrees` skill if not already in one). Do not work on `main`.
- [ ] **Branch.** Branch name: `test/step-0-stateful-bank-mock`.
- [ ] **Sanity check current tree builds.**

  Run: `go build ./... && go vet ./...`
  Expected: no output, exit 0.

---

## File Structure

- Create: `testutil/keeper/bank.go` — `StatefulBankMock`, `BankCall`, `NewStatefulBankMock`, `DefaultModuleAddrs`, full BankKeeper-interface method set.
- Create: `testutil/keeper/bank_test.go` — black-box-style tests in the same package, plus compile-time `var _ <module>types.BankKeeper = (*StatefulBankMock)(nil)` assertions for every module that defines a `BankKeeper` interface.
- **No edits to existing files.** The existing no-op `MockBankKeeper` in `testutil/keeper/credentialschema.go:27-72` stays untouched; existing module wiring keeps working.

### Method set on `StatefulBankMock`

The mock must satisfy the **union** of every module's `BankKeeper` interface. Verified set (from `x/{perm,td,cs,tr,de,xr,di}/types/expected_keepers.go`):

| Method | Signature | Modules requiring it |
|---|---|---|
| `SpendableCoins` | `(ctx context.Context, addr sdk.AccAddress) sdk.Coins` | all |
| `SendCoinsFromAccountToModule` | `(ctx context.Context, sender sdk.AccAddress, recipientModule string, amt sdk.Coins) error` | perm, td |
| `SendCoinsFromModuleToAccount` | `(ctx context.Context, senderModule string, recipient sdk.AccAddress, amt sdk.Coins) error` | perm, td |
| `SendCoinsFromModuleToModule` | `(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error` | td |
| `SendCoins` | `(ctx context.Context, from, to sdk.AccAddress, amt sdk.Coins) error` | perm |
| `HasBalance` | `(ctx context.Context, addr sdk.AccAddress, amt sdk.Coin) bool` | perm |
| `GetBalance` | `(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin` | td, cs |
| `GetAllBalances` | `(ctx context.Context, addr sdk.AccAddress) sdk.Coins` | td |
| `BurnCoins` | `(ctx context.Context, name string, amt sdk.Coins) error` | td |

Plus mock-only assertion helpers (not on any interface):

- `SetBalance(addr sdk.AccAddress, denom string, amount int64)` — sets balance and snapshots baseline.
- `BalanceOf(addr sdk.AccAddress, denom string) int64` — int64 accessor (avoids name clash with interface `GetBalance`).
- `Calls() []BankCall` — copy of full call history.
- `RequireBalanceDelta(t require.TestingT, addr sdk.AccAddress, denom string, delta int64)` — asserts current minus baseline equals `delta`.
- `DefaultModuleAddrs() map[string]sdk.AccAddress` — package-level helper returning the canonical verana module map.

---

## Task 1: Skeleton — types, constructor, stubbed methods, compile-time interface assertions

**Files:**
- Create: `testutil/keeper/bank.go`
- Create: `testutil/keeper/bank_test.go`

The skeleton must compile and the interface assertions must pass; methods panic so they will fail loudly if accidentally invoked before implementation.

- [ ] **Step 1.1: Create `testutil/keeper/bank.go` with types, constructor, default-addr helper, and stubbed methods.**

```go
package keeper

import (
	"context"
	"fmt"
	"sort"
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

func (m *StatefulBankMock) SetBalance(addr sdk.AccAddress, denom string, amount int64) {
	panic("not implemented")
}

func (m *StatefulBankMock) BalanceOf(addr sdk.AccAddress, denom string) int64 {
	panic("not implemented")
}

func (m *StatefulBankMock) Calls() []BankCall {
	panic("not implemented")
}

func (m *StatefulBankMock) RequireBalanceDelta(t require.TestingT, addr sdk.AccAddress, denom string, delta int64) {
	panic("not implemented")
}

// --- BankKeeper interface methods ---

func (m *StatefulBankMock) SendCoins(ctx context.Context, from, to sdk.AccAddress, amt sdk.Coins) error {
	panic("not implemented")
}

func (m *StatefulBankMock) SendCoinsFromAccountToModule(ctx context.Context, sender sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	panic("not implemented")
}

func (m *StatefulBankMock) SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipient sdk.AccAddress, amt sdk.Coins) error {
	panic("not implemented")
}

func (m *StatefulBankMock) SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error {
	panic("not implemented")
}

func (m *StatefulBankMock) BurnCoins(ctx context.Context, name string, amt sdk.Coins) error {
	panic("not implemented")
}

func (m *StatefulBankMock) HasBalance(ctx context.Context, addr sdk.AccAddress, amt sdk.Coin) bool {
	panic("not implemented")
}

func (m *StatefulBankMock) GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	panic("not implemented")
}

func (m *StatefulBankMock) GetAllBalances(ctx context.Context, addr sdk.AccAddress) sdk.Coins {
	panic("not implemented")
}

func (m *StatefulBankMock) SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins {
	panic("not implemented")
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

// sortedDenoms returns coin denoms sorted ascending — matches sdk.Coins ordering.
func sortedDenoms(c sdk.Coins) []string {
	d := make([]string, 0, len(c))
	for _, coin := range c {
		d = append(d, coin.Denom)
	}
	sort.Strings(d)
	return d
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
```

- [ ] **Step 1.2: Create `testutil/keeper/bank_test.go` skeleton with compile-time interface assertions.**

```go
package keeper_test

import (
	"context"
	"sync"
	"testing"

	"cosmossdk.io/math"
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
	last   string
}

func (c *captureT) Errorf(format string, args ...interface{}) {
	c.failed = true
}
func (c *captureT) FailNow() { c.failed = true }
```

- [ ] **Step 1.3: Run build + interface assertions.**

  Run: `go build ./testutil/keeper/... && go vet ./testutil/keeper/...`
  Expected: no output, exit 0. (Compile-time interface assertions all pass; methods are unimplemented but their signatures match.)

- [ ] **Step 1.4: Run tests to confirm package builds.**

  Run: `go test ./testutil/keeper/... -run TestNotARealTest -v`
  Expected: `PASS` (no tests match the filter, but the test binary compiles).

- [ ] **Step 1.5: Commit.**

```bash
git add testutil/keeper/bank.go testutil/keeper/bank_test.go
git commit -m "test(testutil): scaffold StatefulBankMock with stubbed methods"
```

---

## Task 2: `SetBalance` and `BalanceOf` (the foundation)

**Files:**
- Modify: `testutil/keeper/bank.go` (replace `SetBalance` and `BalanceOf` stubs)
- Modify: `testutil/keeper/bank_test.go`

- [ ] **Step 2.1: Write failing test.**

Append to `bank_test.go`:

```go
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
```

- [ ] **Step 2.2: Run to verify failure.**

  Run: `go test ./testutil/keeper/ -run 'TestSetBalance|TestBalanceOf' -v`
  Expected: panic "not implemented" → FAIL.

- [ ] **Step 2.3: Implement.**

In `bank.go`, replace the two stubs:

```go
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

func (m *StatefulBankMock) BalanceOf(addr sdk.AccAddress, denom string) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	if d, ok := m.balances[addr.String()]; ok {
		return d[denom]
	}
	return 0
}
```

- [ ] **Step 2.4: Run to verify passing.**

  Run: `go test ./testutil/keeper/ -run 'TestSetBalance|TestBalanceOf' -v`
  Expected: PASS.

- [ ] **Step 2.5: Commit.**

```bash
git add testutil/keeper/bank.go testutil/keeper/bank_test.go
git commit -m "test(testutil): implement SetBalance and BalanceOf on StatefulBankMock"
```

---

## Task 3: `SendCoins` (peer-to-peer)

**Files:**
- Modify: `testutil/keeper/bank.go` (replace `SendCoins` stub; add internal `transfer` helper)
- Modify: `testutil/keeper/bank_test.go`

- [ ] **Step 3.1: Write failing tests for happy path, multi-coin, and insufficient funds.**

```go
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

	// balances unchanged
	require.Equal(t, int64(100), m.BalanceOf(alice, denom))
	require.Equal(t, int64(0), m.BalanceOf(bob, denom))
}

func TestSendCoins_InsufficientOnSecondDenomDoesNotMutateFirst(t *testing.T) {
	// Atomicity: if the second coin has insufficient funds, the first
	// must NOT be debited. Otherwise tests can't reason about partial state.
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
```

- [ ] **Step 3.2: Run to verify failure.**

  Run: `go test ./testutil/keeper/ -run TestSendCoins -v`
  Expected: panic "not implemented" → FAIL.

- [ ] **Step 3.3: Implement.**

In `bank.go`, replace the `SendCoins` stub and add a private `transfer` helper:

```go
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
	for _, c := range amt {
		amount := nonNegativeAmount(c)
		if m.balances[fromKey] == nil {
			m.balances[fromKey] = map[string]int64{}
		}
		if m.balances[toKey] == nil {
			m.balances[toKey] = map[string]int64{}
		}
		m.balances[fromKey][c.Denom] -= amount
		m.balances[toKey][c.Denom] += amount
	}
	return nil
}

func (m *StatefulBankMock) SendCoins(ctx context.Context, from, to sdk.AccAddress, amt sdk.Coins) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	fromKey, toKey := from.String(), to.String()
	if err := m.transfer(fromKey, toKey, amt); err != nil {
		return err
	}
	m.calls = append(m.calls, BankCall{Method: "SendCoins", From: fromKey, To: toKey, Amount: amt})
	return nil
}
```

- [ ] **Step 3.4: Run to verify passing.**

  Run: `go test ./testutil/keeper/ -run TestSendCoins -v`
  Expected: PASS (4 tests).

- [ ] **Step 3.5: Commit.**

```bash
git add testutil/keeper/bank.go testutil/keeper/bank_test.go
git commit -m "test(testutil): implement SendCoins with atomic insufficient-funds check"
```

---

## Task 4: `SendCoinsFromAccountToModule`

**Files:**
- Modify: `testutil/keeper/bank.go`
- Modify: `testutil/keeper/bank_test.go`

- [ ] **Step 4.1: Write failing tests.**

```go
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
```

- [ ] **Step 4.2: Run to verify failure.**

  Run: `go test ./testutil/keeper/ -run TestSendCoinsFromAccountToModule -v`
  Expected: panic "not implemented" → FAIL.

- [ ] **Step 4.3: Implement.**

```go
func (m *StatefulBankMock) SendCoinsFromAccountToModule(ctx context.Context, sender sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	modAddr := m.resolveModule(recipientModule)
	m.mu.Lock()
	defer m.mu.Unlock()
	fromKey, toKey := sender.String(), modAddr.String()
	if err := m.transfer(fromKey, toKey, amt); err != nil {
		return err
	}
	m.calls = append(m.calls, BankCall{Method: "SendCoinsFromAccountToModule", From: fromKey, To: toKey, Amount: amt})
	return nil
}
```

- [ ] **Step 4.4: Run to verify passing.**

  Run: `go test ./testutil/keeper/ -run TestSendCoinsFromAccountToModule -v`
  Expected: PASS (3 tests).

- [ ] **Step 4.5: Commit.**

```bash
git add testutil/keeper/bank.go testutil/keeper/bank_test.go
git commit -m "test(testutil): implement SendCoinsFromAccountToModule with module resolution"
```

---

## Task 5: `SendCoinsFromModuleToAccount`

**Files:**
- Modify: `testutil/keeper/bank.go`
- Modify: `testutil/keeper/bank_test.go`

- [ ] **Step 5.1: Write failing tests.**

```go
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
```

- [ ] **Step 5.2: Run to verify failure.**

  Run: `go test ./testutil/keeper/ -run TestSendCoinsFromModuleToAccount -v`
  Expected: panic "not implemented" → FAIL.

- [ ] **Step 5.3: Implement.**

```go
func (m *StatefulBankMock) SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipient sdk.AccAddress, amt sdk.Coins) error {
	modAddr := m.resolveModule(senderModule)
	m.mu.Lock()
	defer m.mu.Unlock()
	fromKey, toKey := modAddr.String(), recipient.String()
	if err := m.transfer(fromKey, toKey, amt); err != nil {
		return err
	}
	m.calls = append(m.calls, BankCall{Method: "SendCoinsFromModuleToAccount", From: fromKey, To: toKey, Amount: amt})
	return nil
}
```

- [ ] **Step 5.4: Run to verify passing.**

  Run: `go test ./testutil/keeper/ -run TestSendCoinsFromModuleToAccount -v`
  Expected: PASS (3 tests).

- [ ] **Step 5.5: Commit.**

```bash
git add testutil/keeper/bank.go testutil/keeper/bank_test.go
git commit -m "test(testutil): implement SendCoinsFromModuleToAccount"
```

---

## Task 6: `SendCoinsFromModuleToModule`

**Files:**
- Modify: `testutil/keeper/bank.go`
- Modify: `testutil/keeper/bank_test.go`

- [ ] **Step 6.1: Write failing tests.**

```go
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
```

- [ ] **Step 6.2: Run to verify failure.**

  Run: `go test ./testutil/keeper/ -run TestSendCoinsFromModuleToModule -v`
  Expected: panic "not implemented" → FAIL.

- [ ] **Step 6.3: Implement.**

```go
func (m *StatefulBankMock) SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error {
	fromAddr := m.resolveModule(senderModule)
	toAddr := m.resolveModule(recipientModule)
	m.mu.Lock()
	defer m.mu.Unlock()
	fromKey, toKey := fromAddr.String(), toAddr.String()
	if err := m.transfer(fromKey, toKey, amt); err != nil {
		return err
	}
	m.calls = append(m.calls, BankCall{Method: "SendCoinsFromModuleToModule", From: fromKey, To: toKey, Amount: amt})
	return nil
}
```

- [ ] **Step 6.4: Run to verify passing.**

  Run: `go test ./testutil/keeper/ -run TestSendCoinsFromModuleToModule -v`
  Expected: PASS (4 tests).

- [ ] **Step 6.5: Commit.**

```bash
git add testutil/keeper/bank.go testutil/keeper/bank_test.go
git commit -m "test(testutil): implement SendCoinsFromModuleToModule"
```

---

## Task 7: `BurnCoins`

**Files:**
- Modify: `testutil/keeper/bank.go`
- Modify: `testutil/keeper/bank_test.go`

- [ ] **Step 7.1: Write failing tests.**

```go
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
```

- [ ] **Step 7.2: Run to verify failure.**

  Run: `go test ./testutil/keeper/ -run TestBurnCoins -v`
  Expected: panic "not implemented" → FAIL.

- [ ] **Step 7.3: Implement.**

```go
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
	m.calls = append(m.calls, BankCall{Method: "BurnCoins", From: fromKey, To: "", Amount: amt})
	return nil
}
```

- [ ] **Step 7.4: Run to verify passing.**

  Run: `go test ./testutil/keeper/ -run TestBurnCoins -v`
  Expected: PASS (3 tests).

- [ ] **Step 7.5: Commit.**

```bash
git add testutil/keeper/bank.go testutil/keeper/bank_test.go
git commit -m "test(testutil): implement BurnCoins"
```

---

## Task 8: Read methods — `HasBalance`, `GetBalance`, `GetAllBalances`, `SpendableCoins`

These are pure reads. Implement together since their logic is trivial and tests for one share fixture with another.

**Files:**
- Modify: `testutil/keeper/bank.go`
- Modify: `testutil/keeper/bank_test.go`

- [ ] **Step 8.1: Write failing tests.**

```go
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
```

- [ ] **Step 8.2: Run to verify failure.**

  Run: `go test ./testutil/keeper/ -run 'TestHasBalance|TestGetBalance|TestGetAllBalances|TestSpendableCoins' -v`
  Expected: panic "not implemented" → FAIL.

- [ ] **Step 8.3: Implement.**

```go
func (m *StatefulBankMock) HasBalance(ctx context.Context, addr sdk.AccAddress, amt sdk.Coin) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	have := int64(0)
	if d, ok := m.balances[addr.String()]; ok {
		have = d[amt.Denom]
	}
	return have >= nonNegativeAmount(amt)
}

func (m *StatefulBankMock) GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	m.mu.Lock()
	defer m.mu.Unlock()
	have := int64(0)
	if d, ok := m.balances[addr.String()]; ok {
		have = d[denom]
	}
	return sdk.NewCoin(denom, math.NewInt(have))
}

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

func (m *StatefulBankMock) SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins {
	return m.GetAllBalances(ctx, addr)
}
```

- [ ] **Step 8.4: Run to verify passing.**

  Run: `go test ./testutil/keeper/ -run 'TestHasBalance|TestGetBalance|TestGetAllBalances|TestSpendableCoins' -v`
  Expected: PASS (8 tests).

- [ ] **Step 8.5: Commit.**

```bash
git add testutil/keeper/bank.go testutil/keeper/bank_test.go
git commit -m "test(testutil): implement BankKeeper read methods on StatefulBankMock"
```

---

## Task 9: `Calls()` accessor and call-recording verification

The send/burn methods already append to `m.calls`. Now expose a safe accessor and verify recording end-to-end.

**Files:**
- Modify: `testutil/keeper/bank.go`
- Modify: `testutil/keeper/bank_test.go`

- [ ] **Step 9.1: Write failing tests.**

```go
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
```

- [ ] **Step 9.2: Run to verify failure.**

  Run: `go test ./testutil/keeper/ -run TestCalls -v`
  Expected: panic "not implemented" → FAIL.

- [ ] **Step 9.3: Implement.**

Replace the `Calls` stub in `bank.go`:

```go
// Calls returns a defensive copy of the recorded call history. Tests can
// mutate the returned slice without affecting the mock.
func (m *StatefulBankMock) Calls() []BankCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]BankCall, len(m.calls))
	copy(out, m.calls)
	return out
}
```

- [ ] **Step 9.4: Run to verify passing.**

  Run: `go test ./testutil/keeper/ -run TestCalls -v`
  Expected: PASS (4 tests).

- [ ] **Step 9.5: Commit.**

```bash
git add testutil/keeper/bank.go testutil/keeper/bank_test.go
git commit -m "test(testutil): expose Calls() accessor with defensive copy"
```

---

## Task 10: `RequireBalanceDelta`

The baseline is set by `SetBalance`; for unset (addr, denom) the baseline is 0.

**Files:**
- Modify: `testutil/keeper/bank.go`
- Modify: `testutil/keeper/bank_test.go`

- [ ] **Step 10.1: Write failing tests.**

```go
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
	// alice never SetBalance'd; module account gets credited via send.
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
	// Calling SetBalance again resets the baseline. Documented behavior.
	m := newMock(t)
	m.SetBalance(alice, denom, 1_000)
	require.NoError(t, m.SendCoins(context.Background(), alice, bob,
		sdk.NewCoins(sdk.NewInt64Coin(denom, 100))))
	require.Equal(t, int64(900), m.BalanceOf(alice, denom))

	m.SetBalance(alice, denom, 5_000) // new baseline
	m.RequireBalanceDelta(t, alice, denom, 0)
}
```

- [ ] **Step 10.2: Run to verify failure.**

  Run: `go test ./testutil/keeper/ -run TestRequireBalanceDelta -v`
  Expected: panic "not implemented" → FAIL.

- [ ] **Step 10.3: Implement.**

Replace the `RequireBalanceDelta` stub:

```go
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
```

- [ ] **Step 10.4: Run to verify passing.**

  Run: `go test ./testutil/keeper/ -run TestRequireBalanceDelta -v`
  Expected: PASS (5 tests).

- [ ] **Step 10.5: Commit.**

```bash
git add testutil/keeper/bank.go testutil/keeper/bank_test.go
git commit -m "test(testutil): implement RequireBalanceDelta with baseline snapshot"
```

---

## Task 11: Concurrency safety (race-test)

The mock must be race-safe — fixtures may run table-driven tests with `t.Parallel()`, and downstream keeper code may invoke bank methods from goroutines.

**Files:**
- Modify: `testutil/keeper/bank_test.go`

- [ ] **Step 11.1: Add a stress test.**

```go
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
```

- [ ] **Step 11.2: Run with the race detector.**

  Run: `go test ./testutil/keeper/ -race -run TestStatefulBankMock_ConcurrentTransfersAreRaceFree -v`
  Expected: PASS, no `DATA RACE` reports.

- [ ] **Step 11.3: Commit.**

```bash
git add testutil/keeper/bank_test.go
git commit -m "test(testutil): add race-detector stress test for StatefulBankMock"
```

---

## Task 12: Final pass — full test suite, vet, lint, coverage

- [ ] **Step 12.1: Run the full mock test suite.**

  Run: `go test ./testutil/keeper/... -v -count=1`
  Expected: PASS, all tests green.

- [ ] **Step 12.2: Run race detector across the package.**

  Run: `go test ./testutil/keeper/... -race -count=1`
  Expected: PASS.

- [ ] **Step 12.3: Run vet.**

  Run: `go vet ./testutil/keeper/...`
  Expected: no output.

- [ ] **Step 12.4: Run linter.**

  Run: `golangci-lint run ./testutil/keeper/...`
  Expected: no findings. Fix any reported issues, then re-run.

- [ ] **Step 12.5: Coverage check.**

  Run: `go test ./testutil/keeper/ -cover -count=1`
  Expected: coverage ≥95% on `bank.go`. If below 95%, run `go test ./testutil/keeper/ -coverprofile=/tmp/bank.cov && go tool cover -func=/tmp/bank.cov | grep bank.go` to find uncovered lines and add tests.

- [ ] **Step 12.6: Sanity check the rest of the repo still builds and tests pass.**

  Run: `go build ./... && go test ./... -count=1`
  Expected: PASS. The new file is additive; no existing test should regress.

- [ ] **Step 12.7: Push the branch and open the PR.**

```bash
git push -u origin test/step-0-stateful-bank-mock
gh pr create --title "test(testutil): add StatefulBankMock (issue #292 step 0)" --body "$(cat <<'EOF'
## Summary
- New `StatefulBankMock` in `testutil/keeper/bank.go` satisfying every module's `BankKeeper` interface
- Tracks per-address per-denom balances, enforces deductions, errors with `sdkerrors.ErrInsufficientFunds`, records full call history
- `RequireBalanceDelta` for fixture-style assertions; baseline captured on `SetBalance`
- `DefaultModuleAddrs()` helper covering tr, cs, td, perm, de, xr, di, gov, yield_intermediate_pool
- Compile-time assertions ensure all 7 module BankKeeper interfaces are satisfied
- Legacy no-op `MockBankKeeper` is intentionally untouched (per issue #292 migration policy)

This is the gate for issue #292: no per-module fixture work begins until this is merged.

## Test plan
- [ ] `go test ./testutil/keeper/... -race -count=1` passes
- [ ] `go test ./testutil/keeper/ -cover` reports ≥95%
- [ ] `golangci-lint run ./testutil/keeper/...` clean
- [ ] `go test ./...` (full repo) passes — no regressions in existing tests
EOF
)"
```

---

## "Done" Criteria — Step 0

- [ ] `testutil/keeper/bank.go` exists with `StatefulBankMock`, `BankCall`, `NewStatefulBankMock`, `DefaultModuleAddrs`, and the full BankKeeper-interface method set.
- [ ] Compile-time `var _ <module>types.BankKeeper = (*StatefulBankMock)(nil)` for **all 7** modules: perm, td, cs, tr, de, xr, di.
- [ ] `SendCoinsFromAccountToModule`, `SendCoinsFromModuleToAccount`, `SendCoinsFromModuleToModule`, `SendCoins`, `BurnCoins` all enforce deductions and return `sdkerrors.ErrInsufficientFunds` (atomically — no partial mutation) when balance is insufficient.
- [ ] Unknown module names panic with a clear message at the resolution step.
- [ ] `Calls()` returns a defensive copy; failed transfers are **not** recorded.
- [ ] `RequireBalanceDelta` works against a baseline captured at `SetBalance` time; unset baseline is treated as 0.
- [ ] All mock tests pass with `-race`.
- [ ] `bank.go` line coverage ≥95%.
- [ ] `go test ./...`, `go vet ./...`, `golangci-lint run` all pass.
- [ ] Existing no-op `MockBankKeeper` is untouched; legacy tests still compile and pass.
- [ ] PR opened and merged before any per-module Step (1+) begins.

---

## Self-Review Notes

- **Spec coverage:** every API in spec section "1. `StatefulBankMock`" maps to a task — `SetBalance` (T2), `GetBalance` exposed via `BalanceOf` int64 + interface `GetBalance` sdk.Coin (T2/T8), `RequireBalanceDelta` (T10), `SendCoinsFromAccountToModule` (T4), `SendCoinsFromModuleToAccount` (T5), `SendCoins` (T3), `HasBalance` (T8), plus full BankKeeper interface satisfied (interface assertions in T1, methods filled in T3–T8).
- **Spec note: `BankCall` shape.** Spec lists `Method/From/To/Amount` — implemented identically. `From`/`To` are bech32 addresses (resolved when a module name is involved); the recorded `Method` lets tests filter on operation type.
- **Migration policy:** the legacy no-op `MockBankKeeper` stays untouched in `testutil/keeper/credentialschema.go`. Step 0 only adds; per-module steps 1+ are responsible for migrating their own callers off the legacy mock.
- **Out of scope for Step 0:** per-module `Fixture` structs (steps 1+), spec formula functions, precondition matrix, event/invariant assertions, query handler tests, genesis round-trip tests, ts-proto changes — all are downstream consumers of this mock.
