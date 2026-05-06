# Comprehensive Behavioral Test Overhaul

**Date:** 2026-05-05
**Issue:** [#292](https://github.com/verana-labs/verana/issues/292)
**Sub-issue:** [#286](https://github.com/verana-labs/verana/issues/286) (TS smoke tests for CreateRootPermission + RenewPermissionVP)
**Spec:** https://verana-labs.github.io/verifiable-trust-vpr-spec/

---

## Problem Statement

The current `MockBankKeeper` is a no-op — `SendCoinsFromAccountToModule` returns `nil` and records nothing. Fee math is never exercised in unit tests. A wrong calculation passes because the mock silently eats every `SendCoins` call. This was observed: incorrect math was introduced, tests passed; math was corrected, tests still passed. The tests provided no signal.

Additional gaps:
- State assertions check only the fields the test author thought to check, not the full struct
- Negative cases are ad hoc — not systematically derived from the spec's precondition list
- Expected financial values are derived from the same implementation under test (self-validating, catches nothing)
- TS-proto journey files mix `LEGACY_AMINO` and `DIRECT` sign modes inconsistently

---

## Approach: Per-module Fixture + Stateful Bank Mock

### Chosen approach

**Approach B — Per-module `Fixture` struct with embedded financial assertions.**

Each module gets a `x/{module}/keeper/fixture_test.go` that owns the full test environment and exposes typed assertion helpers. Expected values are computed from spec formulas written as pure Go functions, independent of the implementation.

### Rejected alternatives

- **Approach A (upgraded shared mock):** Simpler, but doesn't force independent expected value computation — a test author could still derive expectations from the same wrong formula.
- **Approach C (`simapp` integration tests):** Most realistic, but 10–20x slower per test, much heavier setup, overkill for behavioral unit testing.

---

## Architecture

### 1. `StatefulBankMock` (`testutil/keeper/bank.go`)

New file. All fixture-based tests use this instead of the existing no-op `MockBankKeeper`. The no-op mock remains for legacy test files until they are migrated.

```go
type StatefulBankMock struct {
    mu       sync.Mutex
    balances map[string]map[string]int64 // addr -> denom -> amount
    calls    []BankCall                  // full call history for assertions
}

type BankCall struct {
    Method string
    From   string
    To     string
    Amount sdk.Coins
}

func (m *StatefulBankMock) SetBalance(addr, denom string, amount int64)
func (m *StatefulBankMock) GetBalance(addr, denom string) int64
func (m *StatefulBankMock) RequireBalanceDelta(t, addr, denom string, delta int64)
func (m *StatefulBankMock) SendCoinsFromAccountToModule(...) error  // enforces deduction
func (m *StatefulBankMock) SendCoinsFromModuleToAccount(...) error  // enforces balance
func (m *StatefulBankMock) SendCoins(...) error
func (m *StatefulBankMock) HasBalance(...) bool
// ... full BankKeeper interface
```

`SendCoinsFromAccountToModule` returns `sdkerrors.ErrInsufficientFunds` if the sender balance is insufficient. Every call is recorded in `calls`.

### 2. Per-module `Fixture` (`x/{module}/keeper/fixture_test.go`)

```go
type Fixture struct {
    t    *testing.T
    K    keeper.Keeper
    MS   types.MsgServer
    Ctx  sdk.Context
    Bank *testutil.StatefulBankMock
    // module-specific mocks:
    // CSKeeper  *testutil.MockCredentialSchemaKeeper  (perm module)
    // TRKeeper  *testutil.MockTrustRegistryKeeper     (perm, cs modules)
    // etc.
}

func NewFixture(t *testing.T) *Fixture

// Balance helpers
func (f *Fixture) SetBalance(addr, denom string, amount int64)
func (f *Fixture) RequireBalanceDelta(addr, denom string, delta int64)
func (f *Fixture) RequireNoBalanceChange(addr string)

// State helpers
func (f *Fixture) RequireObjectCount(n int)                        // module-specific count helper
func (f *Fixture) RequirePermission(id uint64, want types.Permission) // typed per-module; full require.Equal

// Time helpers
func (f *Fixture) SetBlockTime(t time.Time)
func (f *Fixture) AdvanceTime(d time.Duration)

// Invariant helper
func (f *Fixture) RequireInvariant()  // module-specific invariant check
```

### 3. Spec formula functions (per test file)

Written at the top of each test file, pure Go, no keeper imports. Named after the spec ID they implement.

```go
// specPermStartVPFees returns the validation fee deduction per MOD-PERM-MSG-1-2-3.
// validationFees is validator_perm.validation_fees, trustUnitPrice from params.
func specPermStartVPFees(validationFees, trustUnitPrice uint64) uint64 {
    return validationFees * trustUnitPrice
}

// specPermStartVPDeposit returns the trust deposit increment per MOD-PERM-MSG-1-2-3.
func specPermStartVPDeposit(validationFees, trustUnitPrice, depositRatio uint64) uint64 {
    return (validationFees * trustUnitPrice * depositRatio) / 100
}
```

The test assertion:
```go
f.RequireBalanceDelta(applicant, denom, -int64(specPermStartVPFees(valFees, tup)))
```

If the implementation uses a different formula, this breaks. That is the point.

### 4. Precondition matrix pattern

Every message test function follows this structure:

```go
func TestMsgStartPermissionVP(t *testing.T) {
    // --- Happy path ---
    t.Run("MOD-PERM-MSG-1: valid ISSUER request", func(t *testing.T) {
        f := NewFixture(t)
        f.SetBalance(applicant, denom, 10_000)
        // ... setup ...
        resp, err := f.MS.StartPermissionVP(f.Ctx, msg)
        require.NoError(t, err)
        require.NotZero(t, resp.Id)
        f.RequireBalanceDelta(applicant, denom, -int64(specPermStartVPFees(...)))
        f.RequireState(resp.Id, expectedPerm)  // full struct comparison
        f.RequireInvariant()
    })

    // --- Negative cases, one per spec precondition ---
    t.Run("MOD-PERM-MSG-1-2-1: fails if operator not authorized", func(t *testing.T) {
        f := NewFixture(t)
        _, err := f.MS.StartPermissionVP(f.Ctx, msgWithBadOperator)
        require.ErrorContains(t, err, "authorization check failed")
        f.RequireObjectCount(0)
        f.RequireNoBalanceChange(applicant)
    })

    t.Run("MOD-PERM-MSG-1-2-2a: fails if validator perm not found", func(t *testing.T) { ... })
    t.Run("MOD-PERM-MSG-1-2-2b: fails if validator perm not ACTIVE", func(t *testing.T) { ... })
    t.Run("MOD-PERM-MSG-1-2-2c: fails if perm type incompatible", func(t *testing.T) { ... })
    t.Run("MOD-PERM-MSG-1-2-4: fails on overlap", func(t *testing.T) { ... })

    // --- Edge cases ---
    t.Run("overflow: fees * trustUnitPrice exceeds uint64", func(t *testing.T) { ... })
    t.Run("zero fees: no bank transfer occurs", func(t *testing.T) { ... })
}
```

Rules:
- Every negative case: assert error, assert zero state written, assert no balance change
- Every happy path: full struct assertion + balance delta + invariant
- Each `t.Run` name references the spec ID where applicable

---

## TS-Proto Strategy

### Sign mode standardization

All journey files must use `createAccountFromMnemonic` (`Secp256k1HdWallet` — LEGACY_AMINO). Files currently using `createDirectAccountFromMnemonic` must be updated. This is Keplr's sign mode and the one the frontend uses.

### Issue #286 (Step 5)

Two deliverables:
1. `permCreateRootPermission.ts` — verify sign mode is LEGACY_AMINO (currently correct, confirm)
2. `permRenewPermissionVP.ts` — new file, same pattern as existing journeys

Both: build message in TS → sign LEGACY_AMINO → broadcast → assert no error. No business logic assertions.

### Complete journey coverage gaps

| Module | Journey files to add |
|--------|---------------------|
| PERM | `permRenewPermissionVP.ts` |
| CS | `csCreateSchemaAuthorizationPolicy.ts`, `csIncreaseActiveSAPVersion.ts`, `csRevokeSchemaAuthorizationPolicy.ts` |
| TD | `tdSlashTrustDeposit.ts`, `tdBurnEcosystemSlashedTrustDeposit.ts` |
| DE | `deGrantVsOperatorAuthorization.ts`, `deRevokeVsOperatorAuthorization.ts`, `deGrantExchangeRateAuthorization.ts`, `deRevokeExchangeRateAuthorization.ts` |
| XR | `xrCreateExchangeRate.ts`, `xrUpdateExchangeRate.ts`, `xrToggleExchangeRateState.ts` |

`runAll.ts` updated to include all new journeys in dependency order.

---

## Execution Order

Work is one message at a time. Step 0 is a hard gate.

| Step | Work unit | Type |
|------|-----------|------|
| 0 | `StatefulBankMock` in `testutil/keeper/bank.go` + its own tests | Infrastructure |
| 1 | TD: `ReclaimTrustDepositYield` | Unit |
| 2 | TD: `SlashTrustDeposit` | Unit |
| 3 | TD: `RepaySlashedTrustDeposit` | Unit |
| 4 | TD: `BurnEcosystemSlashedTrustDeposit` | Unit |
| 5 | TS #286: sign mode fix + `permRenewPermissionVP.ts` | TS-proto |
| 6 | TR: `CreateTrustRegistry` | Unit |
| 7 | TR: `AddGovernanceFrameworkDocument` | Unit |
| 8 | TR: `IncreaseActiveGovernanceFrameworkVersion` | Unit |
| 9 | TR: `UpdateTrustRegistry` | Unit |
| 10 | TR: `ArchiveTrustRegistry` | Unit |
| 11 | CS: `CreateCredentialSchema` | Unit |
| 12 | CS: `UpdateCredentialSchema` | Unit |
| 13 | CS: `ArchiveCredentialSchema` | Unit |
| 14 | CS: `CreateSchemaAuthorizationPolicy` | Unit |
| 15 | CS: `IncreaseActiveSAPVersion` | Unit |
| 16 | CS: `RevokeSchemaAuthorizationPolicy` | Unit |
| 17 | DE: `GrantOperatorAuthorization` | Unit |
| 18 | DE: `RevokeOperatorAuthorization` | Unit |
| 19 | DE: `GrantVsOperatorAuthorization` | Unit |
| 20 | DE: `RevokeVsOperatorAuthorization` | Unit |
| 21 | PERM: `CreateRootPermission` | Unit |
| 22 | PERM: `StartPermissionVP` | Unit |
| 23 | PERM: `RenewPermissionVP` | Unit |
| 24 | PERM: `SetPermissionVPToValidated` | Unit |
| 25 | PERM: `CancelPermissionVPLastRequest` | Unit |
| 26 | PERM: `AdjustPermission` | Unit |
| 27 | PERM: `RevokePermission` | Unit |
| 28 | PERM: `CreateOrUpdatePermissionSession` | Unit |
| 29 | PERM: `SlashPermissionTrustDeposit` | Unit |
| 30 | PERM: `RepayPermissionSlashedTrustDeposit` | Unit |
| 31 | PERM: `SelfCreatePermission` | Unit |
| 32 | XR: `CreateExchangeRate`, `UpdateExchangeRate`, `ToggleExchangeRateState` | Unit |
| 33 | DI: `StoreDigest` | Unit |
| 34 | Fill remaining TS-proto journey gaps | TS-proto |

---

## "Done" Criteria

### Per message (unit test)
- [ ] Happy path: full field-by-field state assertion (`require.Equal` on complete struct)
- [ ] Happy path: exact balance delta via independently-computed spec formula function
- [ ] Happy path: module-level invariant check passes
- [ ] One `t.Run` per spec precondition: correct error string, zero state written, no balance change
- [ ] Edge cases: overflow inputs, boundary timestamps, zero fees, max values per spec

### Per message (ts-proto)
- [ ] Journey file exists using `Secp256k1HdWallet` (LEGACY_AMINO)
- [ ] Builds message, signs, broadcasts, asserts no error

---

## Source of Truth

All precondition IDs, fee formulas, and state field requirements come from the spec:
https://verana-labs.github.io/verifiable-trust-vpr-spec/

If spec and implementation disagree, spec wins. Tests are written to the spec, not to the implementation.
