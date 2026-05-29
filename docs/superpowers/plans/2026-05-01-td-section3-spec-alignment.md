# TD Module — Section 3 Spec Alignment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Align Trust Deposit module to the three section-3 spec gaps from issue #232 comment 4354823665 (MOD-TD-MSG-2 reclaim formula, MOD-TD-MSG-5 slash burn timing, MOD-TD-MSG-6 cumulative repay accounting), and add spec-driven tests that would have caught the divergence.

**Architecture:** Three coupled fixes in `x/td/keeper`. Repay accounting (gap 3) is fixed first because it changes `slashed_deposit` from "outstanding balance" to "cumulative counter" — gap 1's abort precondition and gap 2's burn timing both reference these fields. The shared `MockBankKeeper` in `testutil/keeper/credentialschema.go` is enhanced (additively) to track burns and module-account balances so spec assertions can observe them. Existing tests that encoded the wrong semantics are surgically updated with citations to the spec section that contradicts them.

**Tech Stack:** Go 1.24, Cosmos SDK 0.50.x, Cosmos `cosmossdk.io/math`, testify, existing `testutil/keeper` mock infrastructure.

**Authoritative spec source:** https://verana-labs.github.io/verifiable-trust-vpr-spec/ (verbatim quotes inlined per task).

---

## File Map

**Modify:**
- [x/td/keeper/msg_server.go](x/td/keeper/msg_server.go) — `ReclaimTrustDepositYield` (formula), `SlashTrustDeposit` (burn-at-slash), `RepaySlashedTrustDeposit` (cumulative semantics, drop burn-on-repay).
- [x/td/keeper/msg_server_test.go](x/td/keeper/msg_server_test.go) — new spec-driven tests; surgical updates to tests that encoded wrong semantics.
- [testutil/keeper/credentialschema.go](testutil/keeper/credentialschema.go) — enhance `MockBankKeeper` to track per-module balances and burn totals (additive only; preserves existing call sites).

**No changes needed:**
- `x/td/keeper/adjust_td.go` — `td.Claimable` semantics for adjust are spec-correct (MOD-TD-MSG-1 stores the field; gap 1 only concerns reclaim).
- `x/td/keeper/burn_slashed_td.go` — MOD-TD-MSG-7 (ecosystem burn) is independent from MOD-TD-MSG-5/6.
- `x/td/types/genesis.go` validation — only checks `deposit >= claimable`; no `slashed_deposit` invariants change.

**Out of scope:**
- Journey fixtures in `journey_results/*.json` may shift when these fixes land. Regenerating those is a follow-up after the test suite is green.

---

## Task 1: Extend MockBankKeeper to Track Burns and Module Balances

The current mock returns `nil` from `BurnCoins` and zero from `GetBalance`, so no test can observe "did slash burn the right amount." We need additive tracking to write spec assertions for gap 2.

**Files:**
- Modify: [testutil/keeper/credentialschema.go:27-72](testutil/keeper/credentialschema.go#L27-L72)

- [ ] **Step 1: Add tracking fields and helpers**

Replace the existing `MockBankKeeper` block (lines 27-72) with:

```go
// MockBankKeeper is a mock implementation of types.BankKeeper.
// Tracks per-module/account balances and cumulative burns so tests can
// assert spec-required money movements (e.g. burn-at-slash).
type MockBankKeeper struct {
	bankBalances map[string]sdk.Coins // key: bech32 address
	moduleBalances map[string]sdk.Coins // key: module name
	burned       sdk.Coins            // cumulative coins burned by module
}

func NewMockBankKeeper() *MockBankKeeper {
	return &MockBankKeeper{
		bankBalances:   make(map[string]sdk.Coins),
		moduleBalances: make(map[string]sdk.Coins),
		burned:         sdk.NewCoins(),
	}
}

// CreditModule seeds a module account balance for tests.
func (k *MockBankKeeper) CreditModule(name string, amt sdk.Coins) {
	k.moduleBalances[name] = k.moduleBalances[name].Add(amt...)
}

// CreditAccount seeds an account balance for tests.
func (k *MockBankKeeper) CreditAccount(addr sdk.AccAddress, amt sdk.Coins) {
	k.bankBalances[addr.String()] = k.bankBalances[addr.String()].Add(amt...)
}

// ModuleBalance returns the tracked balance of a module account.
func (k *MockBankKeeper) ModuleBalance(name, denom string) sdk.Coin {
	return sdk.NewCoin(denom, k.moduleBalances[name].AmountOf(denom))
}

// BurnedTotal returns cumulative coins burned in `denom`.
func (k *MockBankKeeper) BurnedTotal(denom string) sdk.Coin {
	return sdk.NewCoin(denom, k.burned.AmountOf(denom))
}

func (k *MockBankKeeper) SendCoins(ctx context.Context, from, to sdk.AccAddress, amt sdk.Coins) error {
	k.bankBalances[from.String()] = k.bankBalances[from.String()].Sub(amt...)
	k.bankBalances[to.String()] = k.bankBalances[to.String()].Add(amt...)
	return nil
}

func (k *MockBankKeeper) HasBalance(ctx context.Context, addr sdk.AccAddress, amt sdk.Coin) bool {
	return true
}

func (k *MockBankKeeper) BurnCoins(ctx context.Context, name string, amt sdk.Coins) error {
	k.moduleBalances[name] = k.moduleBalances[name].Sub(amt...)
	k.burned = k.burned.Add(amt...)
	return nil
}

func (k *MockBankKeeper) SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	k.moduleBalances[senderModule] = k.moduleBalances[senderModule].Sub(amt...)
	k.bankBalances[recipientAddr.String()] = k.bankBalances[recipientAddr.String()].Add(amt...)
	return nil
}

func (k *MockBankKeeper) SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error {
	k.moduleBalances[senderModule] = k.moduleBalances[senderModule].Sub(amt...)
	k.moduleBalances[recipientModule] = k.moduleBalances[recipientModule].Add(amt...)
	return nil
}

func (k *MockBankKeeper) SpendableCoins(ctx context.Context, address sdk.AccAddress) sdk.Coins {
	return k.bankBalances[address.String()]
}

func (k *MockBankKeeper) GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	return sdk.NewCoin(denom, k.bankBalances[addr.String()].AmountOf(denom))
}

func (k *MockBankKeeper) GetAllBalances(ctx context.Context, addr sdk.AccAddress) sdk.Coins {
	return k.bankBalances[addr.String()]
}

func (k *MockBankKeeper) SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	k.bankBalances[senderAddr.String()] = k.bankBalances[senderAddr.String()].Sub(amt...)
	k.moduleBalances[recipientModule] = k.moduleBalances[recipientModule].Add(amt...)
	return nil
}
```

- [ ] **Step 2: Build all consumers to confirm no signature breakage**

Run: `go build ./...`
Expected: success.

- [ ] **Step 3: Run existing affected tests to confirm no behavioral regression**

Run: `go test ./x/td/... ./x/perm/... ./x/cs/...`
Expected: all pass (mock is additive — defaults match prior behavior except `Sub` may now go negative in untracked tests; `sdk.Coins.Sub` panics on negative, so any failure here flags a test that called bank methods without seeding state — fix by adding `bankKeeper.CreditModule(...)` in that test's setup).

- [ ] **Step 4: Commit**

```bash
git add testutil/keeper/credentialschema.go
git commit -m "test(testutil): track module balances and burns in MockBankKeeper"
```

---

## Task 2: Helper accessor — expose mock bank keeper from TD test setup

The TD `setupMsgServer` helper in [x/td/keeper/msg_server_test.go:19-22](x/td/keeper/msg_server_test.go#L19-L22) hides the bank keeper. Spec assertions need it.

**Files:**
- Modify: [testutil/keeper/trustdeposit.go](testutil/keeper/trustdeposit.go) — return bank keeper from helper
- Modify: [x/td/keeper/msg_server_test.go:19-22](x/td/keeper/msg_server_test.go#L19-L22) — expose via new helper

- [ ] **Step 1: Read current `TrustdepositKeeper` helper signature**

Run: `sed -n '1,90p' /Users/pratik/verana/testutil/keeper/trustdeposit.go`

This reveals the construction of `bankKeeper` at line 48 (and 85). We will add a sibling function that also returns it.

- [ ] **Step 2: Add a new helper `TrustdepositKeeperWithBank` that returns the mock bank keeper**

Append to `testutil/keeper/trustdeposit.go` (after the existing `TrustdepositKeeper` function — preserve the existing one):

```go
// TrustdepositKeeperWithBank is identical to TrustdepositKeeper but also
// returns the MockBankKeeper so tests can assert burn/transfer behavior.
func TrustdepositKeeperWithBank(t testing.TB) (keeper.Keeper, sdk.Context, *MockBankKeeper) {
	// Copy the body of TrustdepositKeeper, capturing bankKeeper before
	// returning. This avoids a drive-by refactor of every existing caller.
	// (Actual body must match the existing TrustdepositKeeper exactly.)
}
```

The actual body must mirror `TrustdepositKeeper`'s body — read it from the existing function, then in the copy, also return `bankKeeper`. Do not refactor the original to avoid disturbing other modules' callers.

- [ ] **Step 3: Add a sibling `setupMsgServerWithBank` to TD msg_server_test**

Insert into [x/td/keeper/msg_server_test.go](x/td/keeper/msg_server_test.go) below the existing `setupMsgServer`:

```go
func setupMsgServerWithBank(t testing.TB) (keeper.Keeper, types.MsgServer, context.Context, *keepertest.MockBankKeeper) {
	k, ctx, bk := keepertest.TrustdepositKeeperWithBank(t)
	return k, keeper.NewMsgServerImpl(k), ctx, bk
}
```

- [ ] **Step 4: Build and verify nothing broke**

Run: `go build ./... && go test ./x/td/...`
Expected: existing tests still pass; new helper unused yet so no behavior change.

- [ ] **Step 5: Commit**

```bash
git add testutil/keeper/trustdeposit.go x/td/keeper/msg_server_test.go
git commit -m "test(td): expose mock bank keeper for spec-driven assertions"
```

---

## Task 3: Gap 3 (Repay) — TDD test for cumulative `slashed_deposit`

**Spec [MOD-TD-MSG-6-2-1]:**
> `amount` MUST be exactly equal to `td.slashed_deposit` - `td.repaid_deposit`.

**Spec [MOD-TD-MSG-6-3]:**
> set `td.repaid_deposit` to `td.repaid_deposit` + `amount`

`td.slashed_deposit` is **not** decremented per spec — it's a cumulative counter. Outstanding = `slashed_deposit - repaid_deposit`.

**Files:**
- Modify: [x/td/keeper/msg_server_test.go](x/td/keeper/msg_server_test.go)

- [ ] **Step 1: Write the failing spec-driven test**

Append to `x/td/keeper/msg_server_test.go`:

```go
// TestRepaySpecCumulativeSemantics asserts MOD-TD-MSG-6-2-1 and MOD-TD-MSG-6-3:
// repay amount must equal (slashed_deposit - repaid_deposit), and slashed_deposit
// is NEVER decremented — only repaid_deposit grows cumulatively.
func TestRepaySpecCumulativeSemantics(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)

	corp := sdk.AccAddress([]byte("repay_spec_corp_addr1")).String()

	require.NoError(t, k.SetParams(ctx, defaultTestParams()))
	require.NoError(t, k.TrustDeposit.Set(ctx, corp, types.TrustDeposit{
		Corporation:    corp,
		Share:          math.LegacyNewDec(700),
		Deposit:        700,
		SlashedDeposit: 300, // cumulative: 300 has been slashed historically
		RepaidDeposit:  100, // cumulative: 100 already repaid
		// outstanding = 300 - 100 = 200
	}))

	t.Run("rejects amount that does not equal outstanding", func(t *testing.T) {
		_, err := ms.RepaySlashedTrustDeposit(ctx, &types.MsgRepaySlashedTrustDeposit{
			Corporation: corp,
			Operator:    corp,
			Deposit:     300, // wrong: outstanding is 200
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "must exactly equal outstanding")
	})

	t.Run("accepts exact outstanding and grows repaid_deposit cumulatively", func(t *testing.T) {
		_, err := ms.RepaySlashedTrustDeposit(ctx, &types.MsgRepaySlashedTrustDeposit{
			Corporation: corp,
			Operator:    corp,
			Deposit:     200, // exactly outstanding
		})
		require.NoError(t, err)

		td, err := k.TrustDeposit.Get(ctx, corp)
		require.NoError(t, err)
		// MOD-TD-MSG-6-3: slashed_deposit MUST NOT be decremented (cumulative counter)
		require.Equal(t, uint64(300), td.SlashedDeposit, "slashed_deposit must remain cumulative per MOD-TD-MSG-6-3")
		// MOD-TD-MSG-6-3: repaid_deposit grows by amount
		require.Equal(t, uint64(300), td.RepaidDeposit, "repaid_deposit must equal prior + amount")
	})
}
```

- [ ] **Step 2: Run the test — must fail**

Run: `go test ./x/td/keeper/ -run TestRepaySpecCumulativeSemantics -v`
Expected: FAIL — first sub-test fails because current precondition compares against raw `td.SlashedDeposit` (300, not 200), and second sub-test fails because current code decrements `SlashedDeposit` to 0.

- [ ] **Step 3: Commit the failing test**

```bash
git add x/td/keeper/msg_server_test.go
git commit -m "test(td): add MOD-TD-MSG-6 cumulative repay semantics test (failing)"
```

---

## Task 4: Gap 3 (Repay) — implement spec-aligned cumulative semantics

**Files:**
- Modify: [x/td/keeper/msg_server.go:225-251](x/td/keeper/msg_server.go#L225-L251)

- [ ] **Step 1: Replace the precondition and execution block**

In `RepaySlashedTrustDeposit`, replace lines 225-251 (the block from the `[MOD-TD-MSG-6-2-1]` comment through `td.RepaidDeposit += msg.Deposit`) with:

```go
	// [MOD-TD-MSG-6-2-1] Spec: amount MUST be exactly equal to
	// (td.slashed_deposit - td.repaid_deposit). slashed_deposit is a
	// cumulative counter; outstanding = slashed_deposit - repaid_deposit.
	if td.SlashedDeposit < td.RepaidDeposit {
		return nil, fmt.Errorf("invariant violated: repaid_deposit (%d) > slashed_deposit (%d)", td.RepaidDeposit, td.SlashedDeposit)
	}
	outstanding := td.SlashedDeposit - td.RepaidDeposit
	if msg.Deposit != outstanding {
		return nil, fmt.Errorf("deposit must exactly equal outstanding slashed amount: expected %d, got %d", outstanding, msg.Deposit)
	}

	// Validate corporation address for bank transfer
	corporationAddr, err := sdk.AccAddressFromBech32(account)
	if err != nil {
		return nil, fmt.Errorf("invalid corporation address: %w", err)
	}

	// [MOD-TD-MSG-6-3] Execution
	params := ms.Keeper.GetParams(ctx)
	now := ctx.BlockTime()

	// td.deposit += amount
	td.Deposit += msg.Deposit

	// td.share += amount / GlobalVariables.trust_deposit_share_value
	shareIncrease := ms.Keeper.AmountToShare(msg.Deposit, params.TrustDepositShareValue)
	td.Share = td.Share.Add(shareIncrease)

	// [MOD-TD-MSG-6-3] Spec: slashed_deposit is cumulative — DO NOT decrement.
	// Only repaid_deposit grows. Outstanding balance is derived as needed.
	td.RepaidDeposit += msg.Deposit
	td.LastRepaid = &now
```

Also remove the existing `corporationAddr` declaration that appears later in the function (around current line 232) to avoid a duplicate declaration. Search the function for the second `corporationAddr` block and delete it (the `sdk.AccAddressFromBech32(account)` call must occur exactly once after the precondition).

- [ ] **Step 2: Run the spec test — must now pass**

Run: `go test ./x/td/keeper/ -run TestRepaySpecCumulativeSemantics -v`
Expected: PASS.

- [ ] **Step 3: Run all TD tests to identify which legacy tests now break**

Run: `go test ./x/td/...`
Expected: FAIL on `TestMsgRepaySlashedTrustDeposit` and `TestMsgReclaimTrustDepositYieldEdgeCases` (they encoded the wrong "decrement" semantics — they will be fixed in the next task).

- [ ] **Step 4: Commit (test passes, legacy tests will be fixed next)**

```bash
git add x/td/keeper/msg_server.go
git commit -m "fix(td): align RepaySlashedTrustDeposit with MOD-TD-MSG-6 cumulative semantics"
```

---

## Task 5: Gap 3 (Repay) — fix legacy tests that encoded wrong semantics

The pre-existing tests at [msg_server_test.go:644-712](x/td/keeper/msg_server_test.go#L644-L712) (TestMsgRepaySlashedTrustDeposit) and [msg_server_test.go:1158+](x/td/keeper/msg_server_test.go#L1158) (TestMsgReclaimTrustDepositYieldEdgeCases) assert post-repay `SlashedDeposit == 0`. Per spec, slashed_deposit must remain at its cumulative value.

**Files:**
- Modify: [x/td/keeper/msg_server_test.go](x/td/keeper/msg_server_test.go)

- [ ] **Step 1: Update `TestMsgRepaySlashedTrustDeposit` — first sub-case (full repay from zero)**

In the test case starting around line 624 (setup with `SlashedDeposit: 300, RepaidDeposit: 0`, msg.Deposit: 300), update the assertion block:

Find:
```go
		// [MOD-TD-MSG-6-3] spec v4 draft 13: slashed_deposit decremented,
		// repaid_deposit cumulative.
		require.Equal(t, uint64(300), td.RepaidDeposit)
		require.Equal(t, uint64(0), td.SlashedDeposit)
```

Replace with:
```go
		// [MOD-TD-MSG-6-3] slashed_deposit is cumulative (never decremented).
		// repaid_deposit grows by amount. Outstanding = slashed - repaid.
		require.Equal(t, uint64(300), td.RepaidDeposit)
		require.Equal(t, uint64(300), td.SlashedDeposit, "cumulative; not decremented")
```

- [ ] **Step 2: Update `TestMsgRepaySlashedTrustDeposit` — partial-history sub-case**

In the test case around line 686 with comment "spec v4 draft 13: slashed_deposit holds outstanding balance" (setup `SlashedDeposit: 300, RepaidDeposit: 200`, msg.Deposit: 300):

The premise is wrong — outstanding here is `300 - 200 = 100`, so msg.Deposit should be `100`, not `300`. Rewrite the case:

Find the case block (use the sub-test name as anchor; it should look like a struct literal with `SlashedDeposit: 300` and `RepaidDeposit: 200`) and replace its `setup`, `msg.Deposit`, and `check` to:

```go
		{
			name: "Partial repay history — pays only outstanding",
			setup: func() {
				err := k.SetParams(ctx, defaultTestParams())
				require.NoError(t, err)
				td := types.TrustDeposit{
					Corporation:    testAccString,
					Share:          math.LegacyNewDec(700),
					Deposit:        700,
					SlashedDeposit: 300, // cumulative: total ever slashed
					RepaidDeposit:  200, // cumulative: total ever repaid
					// outstanding = 100
				}
				err = k.TrustDeposit.Set(ctx, testAccString, td)
				require.NoError(t, err)
			},
			msg: &types.MsgRepaySlashedTrustDeposit{
				Corporation: testAccString,
				Operator:    testAccString,
				Deposit:     100, // exactly outstanding
			},
			expErr: false,
			check: func() {
				td, err := k.TrustDeposit.Get(ctx, testAccString)
				require.NoError(t, err)
				// MOD-TD-MSG-6-3: slashed_deposit unchanged (cumulative).
				require.Equal(t, uint64(300), td.SlashedDeposit)
				// repaid_deposit grew from 200 → 300.
				require.Equal(t, uint64(300), td.RepaidDeposit)
				require.Equal(t, uint64(800), td.Deposit) // 700 + 100
			},
		},
```

- [ ] **Step 3: Update `TestMsgReclaimTrustDepositYieldEdgeCases`**

Around line 1197-1207 there's a setup that asserts the post-repay state has `SlashedDeposit: 0, RepaidDeposit: 100`. This is now invalid — under cumulative semantics, after slashing 100 and repaying 100, the state is `SlashedDeposit: 100, RepaidDeposit: 100`.

Find:
```go
		Claimable:      500, // pre-accrued yield
		SlashedDeposit: 0,   // fully repaid — decremented to 0
		RepaidDeposit:  100, // cumulative history preserved
```

Replace with:
```go
		Claimable:      500, // pre-accrued yield (unused; reclaim now uses formula)
		SlashedDeposit: 100, // cumulative: 100 historically slashed
		RepaidDeposit:  100, // cumulative: 100 repaid → outstanding = 0, reclaim allowed
```

Also update the comment block above it (currently says "on full repay, slashed_deposit is decremented to 0") to:

```go
		// [MOD-TD-MSG-6-3] After full repay, slashed_deposit and repaid_deposit
		// are equal. Reclaim is allowed when slashed_deposit <= repaid_deposit.
```

- [ ] **Step 4: Run all TD tests**

Run: `go test ./x/td/...`
Expected: PASS (gap 3 changes complete and consistent).

- [ ] **Step 5: Commit**

```bash
git add x/td/keeper/msg_server_test.go
git commit -m "test(td): update legacy repay/reclaim tests for cumulative slashed_deposit"
```

---

## Task 6: Gap 1 (Reclaim) — TDD test for spec formula

**Spec [MOD-TD-MSG-2-2-1]:**
> calculate `claimable_yield` = `td.share` * `GlobalVariables.trust_deposit_share_value` - `td.deposit`.

**Spec [MOD-TD-MSG-2-3]:**
> transfer `claimable_yield` from `TrustDeposit` account to corporation account.

The current code uses the stored `td.Claimable` field as the yield. That field is for MOD-TD-MSG-1 (Adjust) bookkeeping — it must not drive reclaim.

Also the current abort condition is `td.SlashedDeposit > 0` — per spec MOD-TD-MSG-2-2-1 it must be `td.slashed_deposit > 0 AND td.repaid_deposit < td.slashed_deposit`.

**Files:**
- Modify: [x/td/keeper/msg_server_test.go](x/td/keeper/msg_server_test.go)

- [ ] **Step 1: Write the failing spec-driven test**

Append to `x/td/keeper/msg_server_test.go`:

```go
// TestReclaimSpecFormula asserts MOD-TD-MSG-2-2-1 / MOD-TD-MSG-2-3:
// claimable_yield = td.share * trust_deposit_share_value - td.deposit, and is
// independent of any stored td.claimable bookkeeping field.
func TestReclaimSpecFormula(t *testing.T) {
	k, ms, ctx, bk := setupMsgServerWithBank(t)
	corp := sdk.AccAddress([]byte("reclaim_spec_corp_a1"))
	corpStr := corp.String()

	t.Run("yield computed from share/value/deposit, ignoring stored Claimable mismatch", func(t *testing.T) {
		params := types.Params{
			TrustDepositShareValue:      math.LegacyMustNewDecFromStr("1.5"),
			TrustDepositReclaimBurnRate: math.LegacyMustNewDecFromStr("0.6"),
			TrustDepositRate:            math.LegacyMustNewDecFromStr("0.2"),
			WalletUserAgentRewardRate:   math.LegacyMustNewDecFromStr("0.3"),
			UserAgentRewardRate:         math.LegacyMustNewDecFromStr("0.2"),
		}
		require.NoError(t, k.SetParams(ctx, params))

		// share=1000, value=1.5, deposit=1000 → spec yield = 500.
		// Stored Claimable=999 to prove the field is NOT consulted.
		require.NoError(t, k.TrustDeposit.Set(ctx, corpStr, types.TrustDeposit{
			Corporation: corpStr,
			Share:       math.LegacyNewDec(1000),
			Deposit:     1000,
			Claimable:   999, // intentionally mismatched
		}))
		// Seed module account with the yield so the bank transfer succeeds.
		bk.CreditModule(types.ModuleName, sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, 500)))

		resp, err := ms.ReclaimTrustDepositYield(ctx, &types.MsgReclaimTrustDepositYield{
			Corporation: corpStr,
			Operator:    corpStr,
		})
		require.NoError(t, err)
		require.Equal(t, uint64(500), resp.ClaimedAmount, "spec formula must yield 500, not stored Claimable=999")

		td, err := k.TrustDeposit.Get(ctx, corpStr)
		require.NoError(t, err)
		require.Equal(t, uint64(1000), td.Deposit, "deposit unchanged")
		// share reduced by claimable_yield/share_value = 500/1.5 ≈ 333.333…
		expected := math.LegacyNewDec(1000).Sub(math.LegacyMustNewDecFromStr("333.333333333333333333"))
		require.True(t, td.Share.Equal(expected), "expected share=%s got %s", expected, td.Share)
	})

	t.Run("aborts when slashed_deposit > repaid_deposit (outstanding slash)", func(t *testing.T) {
		corp2 := sdk.AccAddress([]byte("reclaim_outstanding_a1")).String()
		require.NoError(t, k.TrustDeposit.Set(ctx, corp2, types.TrustDeposit{
			Corporation:    corp2,
			Share:          math.LegacyNewDec(1000),
			Deposit:        1000,
			SlashedDeposit: 200, // cumulative
			RepaidDeposit:  100, // outstanding = 100 → MUST abort
		}))
		_, err := ms.ReclaimTrustDepositYield(ctx, &types.MsgReclaimTrustDepositYield{
			Corporation: corp2,
			Operator:    corp2,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "slashed")
	})

	t.Run("allows reclaim when slashed_deposit == repaid_deposit (fully repaid)", func(t *testing.T) {
		corp3 := sdk.AccAddress([]byte("reclaim_fullyrepaid_a")).String()
		params := types.Params{
			TrustDepositShareValue:      math.LegacyMustNewDecFromStr("1.5"),
			TrustDepositReclaimBurnRate: math.LegacyMustNewDecFromStr("0.6"),
			TrustDepositRate:            math.LegacyMustNewDecFromStr("0.2"),
			WalletUserAgentRewardRate:   math.LegacyMustNewDecFromStr("0.3"),
			UserAgentRewardRate:         math.LegacyMustNewDecFromStr("0.2"),
		}
		require.NoError(t, k.SetParams(ctx, params))
		require.NoError(t, k.TrustDeposit.Set(ctx, corp3, types.TrustDeposit{
			Corporation:    corp3,
			Share:          math.LegacyNewDec(1000),
			Deposit:        1000,
			SlashedDeposit: 100,
			RepaidDeposit:  100, // outstanding = 0 → reclaim allowed
		}))
		bk.CreditModule(types.ModuleName, sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, 500)))

		_, err := ms.ReclaimTrustDepositYield(ctx, &types.MsgReclaimTrustDepositYield{
			Corporation: corp3,
			Operator:    corp3,
		})
		require.NoError(t, err)
	})
}
```

- [ ] **Step 2: Run the test — must fail**

Run: `go test ./x/td/keeper/ -run TestReclaimSpecFormula -v`
Expected: FAIL — sub-test 1 returns 999 not 500; sub-test 2 may pass for the wrong reason (current code aborts on `SlashedDeposit > 0` regardless); sub-test 3 fails because current code rejects when `SlashedDeposit > 0` even after full repay.

- [ ] **Step 3: Commit failing test**

```bash
git add x/td/keeper/msg_server_test.go
git commit -m "test(td): add MOD-TD-MSG-2 spec formula reclaim test (failing)"
```

---

## Task 7: Gap 1 (Reclaim) — implement spec formula

**Files:**
- Modify: [x/td/keeper/msg_server.go:44-73](x/td/keeper/msg_server.go#L44-L73)

- [ ] **Step 1: Replace the precondition and yield computation**

In `ReclaimTrustDepositYield`, replace lines 44-73 (from the `[MOD-TD-MSG-2-2-1] Load TrustDeposit entry` comment through `td.Claimable = 0`) with:

```go
	// [MOD-TD-MSG-2-2-1] Load TrustDeposit entry
	td, err := ms.Keeper.TrustDeposit.Get(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("trust deposit not found for account: %s", account)
	}

	// [MOD-TD-MSG-2-2-1] Spec: abort if slashed_deposit > 0 AND
	// repaid_deposit < slashed_deposit (i.e. outstanding slash exists).
	// slashed_deposit is cumulative; equality with repaid_deposit means fully repaid.
	if td.SlashedDeposit > 0 && td.RepaidDeposit < td.SlashedDeposit {
		return nil, fmt.Errorf("deposit has been slashed and not repaid")
	}

	// [MOD-TD-MSG-2-2-1] / [MOD-TD-MSG-2-3] Spec: claimable_yield is computed,
	// not stored. Formula: td.share * trust_deposit_share_value - td.deposit.
	// The stored td.Claimable field is owned by MOD-TD-MSG-1 (Adjust) and MUST
	// NOT be consulted here.
	params := ms.Keeper.GetParams(ctx)
	yieldDec := td.Share.Mul(params.TrustDepositShareValue).Sub(math.LegacyNewDec(int64(td.Deposit)))
	if !yieldDec.IsPositive() {
		return nil, fmt.Errorf("no claimable yield")
	}
	claimed := yieldDec.TruncateInt().Uint64()

	// [MOD-TD-MSG-2-3] Reduce shares by claimable_yield / share_value.
	sharesToReduce := ms.Keeper.AmountToShare(claimed, params.TrustDepositShareValue)
	td.Share = td.Share.Sub(sharesToReduce)
```

Note: this removes the `td.Claimable = 0` line. The Adjust flow owns that field; reclaim must not touch it.

- [ ] **Step 2: Remove the now-redundant `params` declaration later in the function**

After your edit, the original `params := ms.Keeper.GetParams(ctx)` at line ~66 is duplicated. Delete it (keep only the one introduced in step 1).

- [ ] **Step 3: Verify imports**

`math.LegacyNewDec` requires `cosmossdk.io/math`. Confirm the file already imports it (it does, aliased or directly — check the import block at the top of `msg_server.go`).

Run: `go build ./x/td/...`
Expected: success.

- [ ] **Step 4: Run the spec test — must pass**

Run: `go test ./x/td/keeper/ -run TestReclaimSpecFormula -v`
Expected: PASS.

- [ ] **Step 5: Run all TD tests — confirm no other test relied on draining `Claimable`**

Run: `go test ./x/td/...`
Expected: PASS. If any test fails because `td.Claimable` is no longer reset to 0 by reclaim, fix it: either the test was relying on a non-spec side effect (remove the assertion) or it was using `Claimable` to seed the yield (replace with `Share`/`Deposit`/`ShareValue` setup that yields the desired formula result).

- [ ] **Step 6: Commit**

```bash
git add x/td/keeper/msg_server.go
git commit -m "fix(td): compute reclaim yield from spec formula per MOD-TD-MSG-2"
```

---

## Task 8: Gap 2 (Slash) — TDD test for burn-at-slash

**Spec [MOD-TD-MSG-5-3]:**
> set `td.deposit` to `td.deposit` - `amount`
> set `td.share` to `td.share` - `amount` / `GlobalVariables.trust_deposit_share_value`
> **burn `amount` from `TrustDeposit` account.**
> set `td.slashed_deposit` to `td.slashed_deposit` + `amount`

The burn happens during MOD-TD-MSG-5, not deferred to MOD-TD-MSG-6.

**Files:**
- Modify: [x/td/keeper/msg_server_test.go](x/td/keeper/msg_server_test.go)

- [ ] **Step 1: Write the failing spec test**

Append to `x/td/keeper/msg_server_test.go`:

```go
// TestSlashBurnsImmediately asserts MOD-TD-MSG-5-3: the burn happens during
// SlashTrustDeposit, not deferred to repay.
func TestSlashBurnsImmediately(t *testing.T) {
	k, ms, ctx, bk := setupMsgServerWithBank(t)

	corp := sdk.AccAddress([]byte("slash_burn_corp_addr1")).String()
	require.NoError(t, k.SetParams(ctx, defaultTestParams()))
	require.NoError(t, k.TrustDeposit.Set(ctx, corp, types.TrustDeposit{
		Corporation: corp,
		Share:       math.LegacyNewDec(1000),
		Deposit:     1000,
	}))
	// Seed the module account to mirror prior deposit inflows.
	bk.CreditModule(types.ModuleName, sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, 1000)))

	beforeBurn := bk.BurnedTotal(types.BondDenom).Amount.Uint64()
	beforeBalance := bk.ModuleBalance(types.ModuleName, types.BondDenom).Amount.Uint64()

	_, err := ms.SlashTrustDeposit(ctx, &types.MsgSlashTrustDeposit{
		Authority:   govAuthority(),
		Corporation: corp,
		Deposit:     math.NewInt(300),
	})
	require.NoError(t, err)

	// MOD-TD-MSG-5-3: burn `amount` from TrustDeposit account.
	require.Equal(t, beforeBurn+300, bk.BurnedTotal(types.BondDenom).Amount.Uint64(), "300 must be burned at slash time")
	require.Equal(t, beforeBalance-300, bk.ModuleBalance(types.ModuleName, types.BondDenom).Amount.Uint64(), "module balance must drop by 300")
}
```

- [ ] **Step 2: Run the test — must fail**

Run: `go test ./x/td/keeper/ -run TestSlashBurnsImmediately -v`
Expected: FAIL — current `SlashTrustDeposit` does not call `BurnCoins`; `BurnedTotal` returns 0.

- [ ] **Step 3: Commit failing test**

```bash
git add x/td/keeper/msg_server_test.go
git commit -m "test(td): add MOD-TD-MSG-5 burn-at-slash test (failing)"
```

---

## Task 9: Gap 2 (Slash) — implement burn-at-slash, drop burn-on-repay

**Files:**
- Modify: [x/td/keeper/msg_server.go:172-184](x/td/keeper/msg_server.go#L172-L184) — add burn during slash
- Modify: [x/td/keeper/msg_server.go:275-280](x/td/keeper/msg_server.go#L275-L280) — remove burn during repay

- [ ] **Step 1: Add burn to SlashTrustDeposit**

In `SlashTrustDeposit`, after `td.SlashCount++` (around line 176) and BEFORE the `Save the updated TrustDeposit entry` block, insert:

```go
	// [MOD-TD-MSG-5-3] Burn `amount` from TrustDeposit account immediately.
	if !msg.Deposit.IsUint64() {
		return nil, fmt.Errorf("deposit overflow")
	}
	burnAmt := msg.Deposit.Uint64()
	if burnAmt > uint64(mathstd.MaxInt64) {
		return nil, fmt.Errorf("burn amount exceeds int64")
	}
	burnCoins := sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, int64(burnAmt)))
	if err := ms.Keeper.bankKeeper.BurnCoins(ctx, types.ModuleName, burnCoins); err != nil {
		return nil, fmt.Errorf("failed to burn slashed coins: %w", err)
	}
```

Then update the misleading comment block (lines 178-181) to:
```go
	// Save the updated TrustDeposit entry. Coins were already burned above
	// per MOD-TD-MSG-5-3.
```

- [ ] **Step 2: Remove burn from RepaySlashedTrustDeposit**

In `RepaySlashedTrustDeposit`, locate the block (lines 275-280) starting with `// Burn the previously-slashed coins` and ending with the closing brace of the `BurnCoins` call. Delete it entirely. Per MOD-TD-MSG-6-3, repay only adds `amount` to the TrustDeposit account; it does not burn. The burn was done at slash time.

After the deletion, the repay flow is:
1. Validate `msg.Deposit == td.SlashedDeposit - td.RepaidDeposit`
2. Update `td.Deposit`, `td.Share`, `td.RepaidDeposit`, `td.LastRepaid`
3. `SendCoinsFromAccountToModule` (corporation pays in fresh coins)
4. Emit event

- [ ] **Step 3: Run the burn test — must pass**

Run: `go test ./x/td/keeper/ -run TestSlashBurnsImmediately -v`
Expected: PASS.

- [ ] **Step 4: Run all TD tests — fix any legacy slash/repay tests now affected**

Run: `go test ./x/td/...`
Expected: most PASS. If a test fails because the module account balance now requires seeding (since burn drains it), add `bk.CreditModule(types.ModuleName, ...)` to the test setup. Use `setupMsgServerWithBank` if needed.

- [ ] **Step 5: Add a complementary repay-does-not-burn test**

Append to `x/td/keeper/msg_server_test.go`:

```go
// TestRepayDoesNotBurn asserts MOD-TD-MSG-6-3: repay adds the corporation's
// coins to the TrustDeposit module account; it does NOT burn (burn happened at slash).
func TestRepayDoesNotBurn(t *testing.T) {
	k, ms, ctx, bk := setupMsgServerWithBank(t)

	corp := sdk.AccAddress([]byte("repay_noburn_corp_a1"))
	corpStr := corp.String()

	require.NoError(t, k.SetParams(ctx, defaultTestParams()))
	require.NoError(t, k.TrustDeposit.Set(ctx, corpStr, types.TrustDeposit{
		Corporation:    corpStr,
		Share:          math.LegacyNewDec(700),
		Deposit:        700,
		SlashedDeposit: 300,
		RepaidDeposit:  0,
	}))
	bk.CreditAccount(corp, sdk.NewCoins(sdk.NewInt64Coin(types.BondDenom, 300)))

	burnedBefore := bk.BurnedTotal(types.BondDenom).Amount.Uint64()
	moduleBalBefore := bk.ModuleBalance(types.ModuleName, types.BondDenom).Amount.Uint64()

	_, err := ms.RepaySlashedTrustDeposit(ctx, &types.MsgRepaySlashedTrustDeposit{
		Corporation: corpStr,
		Operator:    corpStr,
		Deposit:     300,
	})
	require.NoError(t, err)

	// MOD-TD-MSG-6-3: no burn at repay (already burned at slash).
	require.Equal(t, burnedBefore, bk.BurnedTotal(types.BondDenom).Amount.Uint64(), "repay must not burn")
	// Module balance grows by the repaid amount (corporation paid in).
	require.Equal(t, moduleBalBefore+300, bk.ModuleBalance(types.ModuleName, types.BondDenom).Amount.Uint64())
}
```

- [ ] **Step 6: Run the new test**

Run: `go test ./x/td/keeper/ -run TestRepayDoesNotBurn -v`
Expected: PASS.

- [ ] **Step 7: Run all TD tests once more**

Run: `go test ./x/td/...`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add x/td/keeper/msg_server.go x/td/keeper/msg_server_test.go
git commit -m "fix(td): burn at slash time per MOD-TD-MSG-5-3; drop redundant repay-time burn"
```

---

## Task 10: Cross-module verification

These TD changes ripple through `perm` (which calls TD via module-call for slash/adjust flows). Run dependent module tests.

- [ ] **Step 1: Run perm + cs + tr + de tests**

Run: `go test ./x/perm/... ./x/cs/... ./x/tr/... ./x/de/...`
Expected: PASS. If any test fails because it asserted post-repay `SlashedDeposit == 0` or the old reclaim semantics, update it with the same spec citations from Task 5.

- [ ] **Step 2: Run the full unit suite**

Run: `go test ./...`
Expected: PASS (excluding integration journey suites which run separately).

- [ ] **Step 3: If any failure, fix in place and commit per failing module**

Use the commit template:
```bash
git commit -m "test(<module>): align with cumulative slashed_deposit per MOD-TD-MSG-6"
```

---

## Task 11: Regenerate journey fixtures (if affected)

The `journey_results/journey*.json` files are integration snapshots. If any journey exercises slash → repay → reclaim, the cumulative `slashed_deposit` change will alter their content.

- [ ] **Step 1: Check whether any journey involves slash/repay**

Run: `grep -l "slash\|Slash\|repay\|Repay\|reclaim\|Reclaim" /Users/pratik/verana/journey_results/*.json | head`

If empty, skip to Task 12.

- [ ] **Step 2: Re-run affected journeys per repo conventions**

Run the project's journey runner (consult `Makefile` or `scripts/`). Inspect the diffs to confirm changes are limited to:
- `slashed_deposit` no longer dropping to 0 after repay
- A burn event appearing at slash time instead of repay time

- [ ] **Step 3: Commit fixture updates**

```bash
git add journey_results/
git commit -m "test(journeys): regenerate fixtures for spec-aligned TD slash/repay/reclaim"
```

---

## Task 12: Final verification and PR

- [ ] **Step 1: Run the full test suite one final time**

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 2: Run `go vet` and any linter the project uses**

Run: `go vet ./...`
Expected: clean. If the project has a custom lint script (check `Makefile`), run it.

- [ ] **Step 3: Inspect the diff vs `main`**

Run: `git diff main --stat`
Confirm the changes are scoped to:
- `x/td/keeper/msg_server.go`
- `x/td/keeper/msg_server_test.go`
- `testutil/keeper/credentialschema.go`
- `testutil/keeper/trustdeposit.go`
- (optionally) `journey_results/*.json`

- [ ] **Step 4: Open the PR**

```bash
gh pr create --title "fix(td): align reclaim/slash/repay with VPR spec section MOD-TD-MSG-2/5/6" --body "$(cat <<'EOF'
## Summary
- Reclaim Yield now uses the spec formula `td.share * share_value - td.deposit` instead of the stored `td.Claimable` field (MOD-TD-MSG-2).
- Slash burns coins immediately during the slash call instead of deferring to repay (MOD-TD-MSG-5-3).
- Repay treats `slashed_deposit` as a cumulative counter and validates `amount == slashed_deposit - repaid_deposit` per MOD-TD-MSG-6-2-1.
- Adds spec-driven tests that would have caught all three divergences (formula assertion, bank-balance/burn assertion, cumulative-counter assertion).
- Fixes legacy tests that encoded the wrong semantics.

Closes the section-3 follow-up items in #232 (comment 4354823665).

## Test plan
- [x] `go test ./x/td/...`
- [x] `go test ./x/perm/... ./x/cs/... ./x/tr/... ./x/de/...`
- [x] `go test ./...`
- [ ] Manual journey re-run (if applicable)
EOF
)"
```

---

## Self-Review Notes

- Spec coverage: every step quotes the spec rule it implements. All three section-3 gaps have a TDD test → fix → legacy-test-update triplet.
- Test design fixes: bank-balance / burn assertions (gap 2), formula-from-state assertion independent of `td.Claimable` (gap 1), cumulative-counter precondition assertion (gap 3) — all three test-design failure modes from the prior analysis are addressed.
- Type / signature consistency: `setupMsgServerWithBank` returns `*keepertest.MockBankKeeper`; `MockBankKeeper` has methods `CreditModule`, `CreditAccount`, `ModuleBalance`, `BurnedTotal`. Used consistently across Tasks 6, 8, 9.
- `BondDenom` is the canonical denom constant in `x/td/types`. `defaultTestParams()` and `govAuthority()` are existing helpers in `msg_server_test.go` — preserved.
- Sequencing rationale: gap 3 first (it changes field semantics that gap 1 & gap 2 reference), then gap 1 (depends on gap 3's `slashed > repaid` precondition), then gap 2 (largely independent but benefits from existing tests being green when the burn assertion is added).
