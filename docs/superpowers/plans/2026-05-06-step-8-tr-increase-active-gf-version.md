# Step 8: TR IncreaseActiveGovernanceFrameworkVersion — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace `TestMsgServerIncreaseActiveGovernanceFrameworkVersion` with a fixture-based test that asserts the full TR state transition, GFVersion `ActiveSince` timestamp, emitted event, and all spec precondition failure modes.

**Architecture:** Uses the `Fixture` from step 6 and the `createTestTR` helper from step 7. A local `setupV2ForIncrease` helper adds a pending v2 GFVersion with an English document (the required precondition). Each sub-test gets an isolated `NewFixture(t)`. No bank operations.

**Tech Stack:** Go 1.24, Cosmos SDK v0.50.x, testify/require

---

## Pre-flight

- [ ] **Worktree.** Branch name: `test/step-8-tr-increase-gfv`.
- [ ] **Gate.** Confirm steps 6 and 7 PRs are merged.
- [ ] **Sanity check.**

  Run: `go build ./... && go vet ./...`
  Expected: exit 0.

---

## File Structure

- Modify: `x/tr/keeper/msg_server_test.go` — delete `TestMsgServerIncreaseActiveGovernanceFrameworkVersion`, add `TestMsgIncreaseActiveGovernanceFrameworkVersion` using fixture.

---

## Task 1: Write the fixture-based `TestMsgIncreaseActiveGovernanceFrameworkVersion`

**File:** `x/tr/keeper/msg_server_test.go`

- [ ] **Step 1.1: Add a local `setupV2ForIncrease` helper.**

This helper builds a fresh TR, adds a v2 GFVersion with a document in the TR's primary language ("en"), and returns the TR id. It leaves the context's block time advanced by 1 hour past creation.

```go
// setupV2ForIncrease creates a TR and adds a pending v2 with an "en" document
// so that IncreaseActiveGovernanceFrameworkVersion can succeed. Returns trID.
func setupV2ForIncrease(t *testing.T, f *Fixture) uint64 {
	t.Helper()
	trID, _ := createTestTR(t, f)
	f.AdvanceTime(time.Hour)

	_, err := f.MS.AddGovernanceFrameworkDocument(f.Ctx, &types.MsgAddGovernanceFrameworkDocument{
		Corporation: testCorp,
		Operator:    testOperator,
		TrId:        trID,
		Language:    "en",
		Url:         "https://example.com/gf-v2-en.html",
		DigestSri:   testDigestSRI,
		Version:     2,
	})
	require.NoError(t, err)
	f.AdvanceTime(time.Hour)
	return trID
}
```

- [ ] **Step 1.2: Write `TestMsgIncreaseActiveGovernanceFrameworkVersion`.**

```go
func TestMsgIncreaseActiveGovernanceFrameworkVersion(t *testing.T) {
	t.Run("MOD-TR-MSG-3: happy path increases active_version and sets GFV.ActiveSince", func(t *testing.T) {
		f := NewFixture(t)
		trID := setupV2ForIncrease(t, f)

		// Remember the current time — this becomes the new GFV.ActiveSince
		activatedAt := f.Ctx.BlockTime()

		resp, err := f.MS.IncreaseActiveGovernanceFrameworkVersion(f.Ctx,
			&types.MsgIncreaseActiveGovernanceFrameworkVersion{
				Corporation: testCorp,
				Operator:    testOperator,
				TrId:        trID,
			})
		require.NoError(t, err)
		require.NotNil(t, resp)

		// TR must reflect new active version and updated Modified timestamp
		tr, err := f.K.TrustRegistry.Get(f.Ctx, trID)
		require.NoError(t, err)
		require.Equal(t, int32(2), tr.ActiveVersion)
		require.Equal(t, activatedAt, tr.Modified)
		// Created must not change
		require.True(t, tr.Created.Before(tr.Modified))

		// Full TR struct assertion
		f.RequireTrustRegistry(trID, types.TrustRegistry{
			Id:            trID,
			Did:           testDID,
			Corporation:   testCorp,
			Created:       tr.Created, // preserved from createTestTR
			Modified:      activatedAt,
			Archived:      nil,
			Aka:           testAka,
			ActiveVersion: 2,
			Language:      "en",
		})

		// GFVersion v2 must have ActiveSince set to activatedAt (was zero before)
		f.RequireGFVersion(trID, 2, func(gfv types.GovernanceFrameworkVersion) {
			require.Equal(t, activatedAt, gfv.ActiveSince,
				"GFVersion.ActiveSince must be set to block time at promotion")
		})

		// Event emitted
		f.RequireEvent(types.EventTypeIncreaseActiveGFVersion, map[string]string{
			types.AttributeKeyTrustRegistryID: func() string {
				return fmt.Sprintf("%d", trID)
			}(),
			types.AttributeKeyCorporation: testCorp,
		})

		f.RequireInvariant()
	})

	t.Run("MOD-TR-MSG-3: sequential increase v1->v2->v3 works", func(t *testing.T) {
		f := NewFixture(t)
		trID := setupV2ForIncrease(t, f)

		// Add v3 with English doc
		_, err := f.MS.AddGovernanceFrameworkDocument(f.Ctx, &types.MsgAddGovernanceFrameworkDocument{
			Corporation: testCorp, Operator: testOperator,
			TrId: trID, Language: "en",
			Url:       "https://example.com/gf-v3-en.html",
			DigestSri: testDigestSRI, Version: 3,
		})
		require.NoError(t, err)

		// Increase to v2
		_, err = f.MS.IncreaseActiveGovernanceFrameworkVersion(f.Ctx,
			&types.MsgIncreaseActiveGovernanceFrameworkVersion{
				Corporation: testCorp, Operator: testOperator, TrId: trID,
			})
		require.NoError(t, err)

		tr, err := f.K.TrustRegistry.Get(f.Ctx, trID)
		require.NoError(t, err)
		require.Equal(t, int32(2), tr.ActiveVersion)

		f.AdvanceTime(time.Hour)

		// Increase to v3
		_, err = f.MS.IncreaseActiveGovernanceFrameworkVersion(f.Ctx,
			&types.MsgIncreaseActiveGovernanceFrameworkVersion{
				Corporation: testCorp, Operator: testOperator, TrId: trID,
			})
		require.NoError(t, err)

		tr, err = f.K.TrustRegistry.Get(f.Ctx, trID)
		require.NoError(t, err)
		require.Equal(t, int32(3), tr.ActiveVersion)

		f.RequireInvariant()
	})

	t.Run("MOD-TR-MSG-3-2-1: fails if operator not authorized", func(t *testing.T) {
		f := NewFixture(t)
		trID := setupV2ForIncrease(t, f)

		f.DelKeeper.ErrToReturn = errors.New("not authorized")

		_, err := f.MS.IncreaseActiveGovernanceFrameworkVersion(f.Ctx,
			&types.MsgIncreaseActiveGovernanceFrameworkVersion{
				Corporation: testCorp, Operator: testOperator, TrId: trID,
			})
		require.Error(t, err)
		require.ErrorContains(t, err, "authorization check failed")

		// TR active_version must remain 1
		tr, err := f.K.TrustRegistry.Get(f.Ctx, trID)
		require.NoError(t, err)
		require.Equal(t, int32(1), tr.ActiveVersion)

		// GFVersion v2 must still have zero ActiveSince
		f.RequireGFVersion(trID, 2, func(gfv types.GovernanceFrameworkVersion) {
			require.True(t, gfv.ActiveSince.IsZero(), "GFVersion.ActiveSince must remain zero on failure")
		})

		f.RequireNoEvent(types.EventTypeIncreaseActiveGFVersion)
	})

	t.Run("MOD-TR-MSG-3-2-1: fails if TR not found", func(t *testing.T) {
		f := NewFixture(t)

		_, err := f.MS.IncreaseActiveGovernanceFrameworkVersion(f.Ctx,
			&types.MsgIncreaseActiveGovernanceFrameworkVersion{
				Corporation: testCorp, Operator: testOperator, TrId: 99999,
			})
		require.Error(t, err)
		require.ErrorContains(t, err, "does not exist")

		f.RequireTrustRegistryCount(0)
		f.RequireNoEvent(types.EventTypeIncreaseActiveGFVersion)
	})

	t.Run("MOD-TR-MSG-3-2-1: fails if corporation mismatch", func(t *testing.T) {
		f := NewFixture(t)
		trID := setupV2ForIncrease(t, f)

		_, err := f.MS.IncreaseActiveGovernanceFrameworkVersion(f.Ctx,
			&types.MsgIncreaseActiveGovernanceFrameworkVersion{
				Corporation: "wrong-corp", Operator: testOperator, TrId: trID,
			})
		require.Error(t, err)
		require.ErrorContains(t, err, "corporation is not the controller")

		tr, err := f.K.TrustRegistry.Get(f.Ctx, trID)
		require.NoError(t, err)
		require.Equal(t, int32(1), tr.ActiveVersion, "active_version must be unchanged")

		f.RequireNoEvent(types.EventTypeIncreaseActiveGFVersion)
	})

	t.Run("MOD-TR-MSG-3-2-1: fails if next version GFV does not exist", func(t *testing.T) {
		f := NewFixture(t)
		trID, _ := createTestTR(t, f)
		f.AdvanceTime(time.Hour)
		// No v2 GFVersion was created — IncreaseActiveGFVersion must fail

		_, err := f.MS.IncreaseActiveGovernanceFrameworkVersion(f.Ctx,
			&types.MsgIncreaseActiveGovernanceFrameworkVersion{
				Corporation: testCorp, Operator: testOperator, TrId: trID,
			})
		require.Error(t, err)
		require.ErrorContains(t, err, "no governance framework version found")

		tr, err := f.K.TrustRegistry.Get(f.Ctx, trID)
		require.NoError(t, err)
		require.Equal(t, int32(1), tr.ActiveVersion)
		f.RequireNoEvent(types.EventTypeIncreaseActiveGFVersion)
	})

	t.Run("MOD-TR-MSG-3-2-1: fails if next version has no document in primary language", func(t *testing.T) {
		f := NewFixture(t)
		trID, _ := createTestTR(t, f)
		f.AdvanceTime(time.Hour)

		// Add v2 with only a French doc — missing English (primary language of TR)
		_, err := f.MS.AddGovernanceFrameworkDocument(f.Ctx, &types.MsgAddGovernanceFrameworkDocument{
			Corporation: testCorp, Operator: testOperator,
			TrId: trID, Language: "fr",
			Url: "https://example.com/gf-v2-fr.html", DigestSri: testDigestSRI, Version: 2,
		})
		require.NoError(t, err)
		f.AdvanceTime(time.Hour)

		_, err = f.MS.IncreaseActiveGovernanceFrameworkVersion(f.Ctx,
			&types.MsgIncreaseActiveGovernanceFrameworkVersion{
				Corporation: testCorp, Operator: testOperator, TrId: trID,
			})
		require.Error(t, err)
		require.ErrorContains(t, err, "no document found for the default language")

		tr, err := f.K.TrustRegistry.Get(f.Ctx, trID)
		require.NoError(t, err)
		require.Equal(t, int32(1), tr.ActiveVersion)
		f.RequireNoEvent(types.EventTypeIncreaseActiveGFVersion)
	})
}
```

**Note:** `fmt.Sprintf("%d", trID)` is used inline in the event assertion because `strconv.FormatUint` is what the implementation uses (and both produce the same string for a uint64). The `fmt` package must be in the import block.

- [ ] **Step 1.3: Verify import block.**

Ensure `"errors"` and `"fmt"` are in the import block of `msg_server_test.go`.

- [ ] **Step 1.4: Build.**

  Run: `go build ./x/tr/keeper/... && go vet ./x/tr/keeper/...`
  Expected: exit 0.

- [ ] **Step 1.5: Run new tests.**

  Run: `go test ./x/tr/keeper/... -run TestMsgIncreaseActiveGovernanceFrameworkVersion -v -count=1`
  Expected: PASS (all sub-tests green).

- [ ] **Step 1.6: Run full TR keeper suite.**

  Run: `go test ./x/tr/keeper/... -count=1 -v`
  Expected: PASS.

- [ ] **Step 1.7: Commit.**

```bash
git add x/tr/keeper/msg_server_test.go
git commit -m "test(tr): replace TestMsgServerIncreaseActiveGFVersion with fixture-based test"
```

---

## Task 2: Coverage and CI validation

- [ ] **Step 2.1: Coverage.**

  Run: `go test ./x/tr/keeper/... -cover -count=1`
  Record coverage delta.

- [ ] **Step 2.2: Full build and test.**

  Run: `go build ./... && go test ./... -count=1`
  Expected: exit 0.

- [ ] **Step 2.3: Push and open PR.**

```bash
git push -u origin test/step-8-tr-increase-gfv
gh pr create --title "test(tr): step 8 — IncreaseActiveGFVersion fixture-based tests" --body "$(cat <<'EOF'
## Summary
- Replaces legacy TestMsgServerIncreaseActiveGovernanceFrameworkVersion with fixture-based version
- Asserts TR.ActiveVersion, TR.Modified, GFVersion.ActiveSince on promotion
- Full TR struct comparison on happy path
- Negative cases: AUTHZ fail, TR not found, corporation mismatch, missing next GFV, missing primary-language doc

## Test plan
- [ ] go test ./x/tr/keeper/... -count=1 passes
- [ ] go test ./... -count=1 passes
EOF
)"
```

---

## "Done" Criteria — Step 8

- [ ] `TestMsgServerIncreaseActiveGovernanceFrameworkVersion` deleted.
- [ ] `TestMsgIncreaseActiveGovernanceFrameworkVersion` added with fixture.
- [ ] Happy paths (2): full TR struct equality, GFVersion.ActiveSince set, event, invariant.
- [ ] Negative cases (5): error string, active_version unchanged, GFVersion.ActiveSince remains zero where applicable, no event.
- [ ] `go test ./x/tr/keeper/... -count=1` PASS.
- [ ] `go test ./... -count=1` PASS.

---

## Self-Review

- No TBD/TODO items.
- `setupV2ForIncrease` is file-local (lower-case) and calls `createTestTR` from step 7.
- Happy path "sequential v1->v2->v3" verifies that active_version can be incremented more than once; this is an edge case not covered by the old test.
- All negative-case sub-tests assert: specific error string, `tr.ActiveVersion` unchanged (==1), and `RequireNoEvent`.
- The "missing primary-language doc" case explicitly uses a French-only v2 against an English-primary TR, which maps directly to `gfv.go:49`.
- `fmt.Sprintf("%d", trID)` matches `strconv.FormatUint(trID, 10)` output for uint64 — both produce the decimal string.
- `GFVersion.ActiveSince` is a `time.Time` (not a pointer); a zero `time.Time` is the sentinel for "not yet active" (gfd.go:91).
