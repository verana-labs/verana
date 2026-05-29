# Step 10: TR ArchiveTrustRegistry — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace `TestMsgServerArchiveTrustRegistry` with a fixture-based test that fully asserts `Archived` pointer state, `Modified` timestamp, emitted events, and all spec precondition failure modes for both the archive and unarchive directions of the bidirectional toggle.

**Architecture:** Uses `Fixture` from step 6 and `createTestTR` from step 7. Each `t.Run` gets an isolated `NewFixture(t)` so archive/unarchive state cannot leak between cases. This step also completes the TR module migration: after all 5 message tests are fixture-based, `setupMsgServerLegacy` can be removed from `msg_server_test.go` and `TestMsgServer` can use `NewFixture`. Additionally, step 10 finalises `RequireInvariant` (already correct from step 6) and adds `TestGenesisRoundTrip` as the module-completion marker.

**Tech Stack:** Go 1.24, Cosmos SDK v0.50.x, testify/require

---

## Pre-flight

- [ ] **Worktree.** Branch name: `test/step-10-tr-archive`.
- [ ] **Gate.** Confirm steps 6-9 PRs are merged.
- [ ] **Sanity check.**

  Run: `go build ./... && go vet ./...`
  Expected: exit 0.

---

## File Structure

- Modify: `x/tr/keeper/msg_server_test.go` — delete `TestMsgServerArchiveTrustRegistry` and `setupMsgServerLegacy`, replace `TestMsgServer`, add `TestMsgArchiveTrustRegistry`.
- Create: `x/tr/keeper/genesis_test.go` — `TestGenesisRoundTrip`.

---

## Task 1: Write the fixture-based `TestMsgArchiveTrustRegistry`

**File:** `x/tr/keeper/msg_server_test.go`

- [ ] **Step 1.1: Write `TestMsgArchiveTrustRegistry`.**

```go
func TestMsgArchiveTrustRegistry(t *testing.T) {
	t.Run("MOD-TR-MSG-5: happy path archives TR — sets Archived pointer and advances Modified", func(t *testing.T) {
		f := NewFixture(t)
		trID, createdAt := createTestTR(t, f)

		archivedAt := createdAt.Add(time.Hour)
		f.SetBlockTime(archivedAt)

		resp, err := f.MS.ArchiveTrustRegistry(f.Ctx, &types.MsgArchiveTrustRegistry{
			Corporation: testCorp,
			Operator:    testOperator,
			TrId:        trID,
			Archive:     true,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		// Full struct assertion
		f.RequireTrustRegistry(trID, types.TrustRegistry{
			Id:            trID,
			Did:           testDID,
			Corporation:   testCorp,
			Created:       createdAt,
			Modified:      archivedAt,
			Archived:      &archivedAt,
			Aka:           testAka,
			ActiveVersion: 1,
			Language:      "en",
		})

		// Event emitted with archive_status=archived
		f.RequireEvent(types.EventTypeArchiveTrustRegistry, map[string]string{
			types.AttributeKeyCorporation:  testCorp,
			types.AttributeKeyArchiveStatus: "archived",
		})

		f.RequireInvariant()
	})

	t.Run("MOD-TR-MSG-5-3: happy path unarchives TR — sets Archived to nil", func(t *testing.T) {
		f := NewFixture(t)
		trID, createdAt := createTestTR(t, f)

		// Archive first
		archivedAt := createdAt.Add(time.Hour)
		f.SetBlockTime(archivedAt)
		_, err := f.MS.ArchiveTrustRegistry(f.Ctx, &types.MsgArchiveTrustRegistry{
			Corporation: testCorp, Operator: testOperator, TrId: trID, Archive: true,
		})
		require.NoError(t, err)

		// Unarchive
		unarchivedAt := archivedAt.Add(time.Hour)
		f.SetBlockTime(unarchivedAt)

		resp, err := f.MS.ArchiveTrustRegistry(f.Ctx, &types.MsgArchiveTrustRegistry{
			Corporation: testCorp,
			Operator:    testOperator,
			TrId:        trID,
			Archive:     false,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		// Full struct assertion — Archived must be nil again
		f.RequireTrustRegistry(trID, types.TrustRegistry{
			Id:            trID,
			Did:           testDID,
			Corporation:   testCorp,
			Created:       createdAt,
			Modified:      unarchivedAt,
			Archived:      nil,
			Aka:           testAka,
			ActiveVersion: 1,
			Language:      "en",
		})

		// Event emitted with archive_status=unarchived
		f.RequireEvent(types.EventTypeArchiveTrustRegistry, map[string]string{
			types.AttributeKeyArchiveStatus: "unarchived",
		})

		f.RequireInvariant()
	})

	t.Run("MOD-TR-MSG-5: archive then unarchive then re-archive works", func(t *testing.T) {
		f := NewFixture(t)
		trID, createdAt := createTestTR(t, f)

		f.SetBlockTime(createdAt.Add(time.Hour))
		_, err := f.MS.ArchiveTrustRegistry(f.Ctx, &types.MsgArchiveTrustRegistry{
			Corporation: testCorp, Operator: testOperator, TrId: trID, Archive: true,
		})
		require.NoError(t, err)

		f.AdvanceTime(time.Hour)
		_, err = f.MS.ArchiveTrustRegistry(f.Ctx, &types.MsgArchiveTrustRegistry{
			Corporation: testCorp, Operator: testOperator, TrId: trID, Archive: false,
		})
		require.NoError(t, err)

		reArchivedAt := f.Ctx.BlockTime().Add(time.Hour)
		f.SetBlockTime(reArchivedAt)
		_, err = f.MS.ArchiveTrustRegistry(f.Ctx, &types.MsgArchiveTrustRegistry{
			Corporation: testCorp, Operator: testOperator, TrId: trID, Archive: true,
		})
		require.NoError(t, err)

		tr, err := f.K.TrustRegistry.Get(f.Ctx, trID)
		require.NoError(t, err)
		require.NotNil(t, tr.Archived)
		require.Equal(t, reArchivedAt, *tr.Archived)

		f.RequireInvariant()
	})

	t.Run("MOD-TR-MSG-5-2-1: fails if operator not authorized", func(t *testing.T) {
		f := NewFixture(t)
		trID, createdAt := createTestTR(t, f)
		f.SetBlockTime(createdAt.Add(time.Hour))

		f.DelKeeper.ErrToReturn = errors.New("not authorized")

		_, err := f.MS.ArchiveTrustRegistry(f.Ctx, &types.MsgArchiveTrustRegistry{
			Corporation: testCorp, Operator: testOperator, TrId: trID, Archive: true,
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "authorization check failed")

		// TR must be unchanged — Archived still nil, Modified still createdAt
		f.RequireTrustRegistry(trID, types.TrustRegistry{
			Id:            trID,
			Did:           testDID,
			Corporation:   testCorp,
			Created:       createdAt,
			Modified:      createdAt,
			Archived:      nil,
			Aka:           testAka,
			ActiveVersion: 1,
			Language:      "en",
		})
		f.RequireNoEvent(types.EventTypeArchiveTrustRegistry)
	})

	t.Run("MOD-TR-MSG-5-2-1: fails if TR not found", func(t *testing.T) {
		f := NewFixture(t)

		_, err := f.MS.ArchiveTrustRegistry(f.Ctx, &types.MsgArchiveTrustRegistry{
			Corporation: testCorp, Operator: testOperator, TrId: 99999, Archive: true,
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "trust registry not found")

		f.RequireTrustRegistryCount(0)
		f.RequireNoEvent(types.EventTypeArchiveTrustRegistry)
	})

	t.Run("MOD-TR-MSG-5-2-1: fails if corporation mismatch on archive", func(t *testing.T) {
		f := NewFixture(t)
		trID, createdAt := createTestTR(t, f)
		f.SetBlockTime(createdAt.Add(time.Hour))

		_, err := f.MS.ArchiveTrustRegistry(f.Ctx, &types.MsgArchiveTrustRegistry{
			Corporation: "attacker-corp",
			Operator:    testOperator,
			TrId:        trID,
			Archive:     true,
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "only trust registry corporation can archive")

		tr, err := f.K.TrustRegistry.Get(f.Ctx, trID)
		require.NoError(t, err)
		require.Nil(t, tr.Archived, "Archived must remain nil on corporation mismatch")
		require.Equal(t, createdAt, tr.Modified, "Modified must not advance on failure")

		f.RequireNoEvent(types.EventTypeArchiveTrustRegistry)
	})

	t.Run("MOD-TR-MSG-5: fails if already archived (archive=true on archived TR)", func(t *testing.T) {
		f := NewFixture(t)
		trID, createdAt := createTestTR(t, f)
		f.SetBlockTime(createdAt.Add(time.Hour))

		// Archive it
		_, err := f.MS.ArchiveTrustRegistry(f.Ctx, &types.MsgArchiveTrustRegistry{
			Corporation: testCorp, Operator: testOperator, TrId: trID, Archive: true,
		})
		require.NoError(t, err)

		f.AdvanceTime(time.Hour)

		// Try to archive again
		_, err = f.MS.ArchiveTrustRegistry(f.Ctx, &types.MsgArchiveTrustRegistry{
			Corporation: testCorp, Operator: testOperator, TrId: trID, Archive: true,
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "already archived")

		// Archived timestamp must not have changed
		tr, err := f.K.TrustRegistry.Get(f.Ctx, trID)
		require.NoError(t, err)
		require.NotNil(t, tr.Archived)
		require.Equal(t, createdAt.Add(time.Hour), *tr.Archived, "Archived timestamp must not advance on double-archive")
	})

	t.Run("MOD-TR-MSG-5-3: fails if archive=false on non-archived TR", func(t *testing.T) {
		f := NewFixture(t)
		trID, createdAt := createTestTR(t, f)
		f.SetBlockTime(createdAt.Add(time.Hour))

		// TR is not archived — unarchive must fail
		_, err := f.MS.ArchiveTrustRegistry(f.Ctx, &types.MsgArchiveTrustRegistry{
			Corporation: testCorp, Operator: testOperator, TrId: trID, Archive: false,
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "not archived")

		tr, err := f.K.TrustRegistry.Get(f.Ctx, trID)
		require.NoError(t, err)
		require.Nil(t, tr.Archived, "Archived must remain nil on spurious unarchive")
		require.Equal(t, createdAt, tr.Modified, "Modified must not advance on failure")

		f.RequireNoEvent(types.EventTypeArchiveTrustRegistry)
	})
}
```

- [ ] **Step 1.2: Remove `setupMsgServerLegacy` and migrate `TestMsgServer`.**

Since all 5 TR message tests are now fixture-based, `setupMsgServerLegacy` is no longer needed. Replace it and `TestMsgServer` with:

```go
func TestMsgServer(t *testing.T) {
	f := NewFixture(t)
	require.NotNil(t, f.MS)
	require.NotNil(t, f.Ctx)
	require.NotEmpty(t, f.K)
}
```

Remove the `setupMsgServerLegacy` function entirely.

- [ ] **Step 1.3: Remove unused imports.**

After removing `setupMsgServerLegacy`, check that `sdk` and `keepertest` and `keeper` (the package import, not the type) are still used. If `sdk` is no longer referenced (all test functions now go through `NewFixture`), remove it. Adjust the import block accordingly.

- [ ] **Step 1.4: Build.**

  Run: `go build ./x/tr/keeper/... && go vet ./x/tr/keeper/...`
  Expected: exit 0.

- [ ] **Step 1.5: Run new tests.**

  Run: `go test ./x/tr/keeper/... -run TestMsgArchiveTrustRegistry -v -count=1`
  Expected: PASS.

- [ ] **Step 1.6: Run full TR keeper suite.**

  Run: `go test ./x/tr/keeper/... -count=1 -v`
  Expected: PASS.

- [ ] **Step 1.7: Commit.**

```bash
git add x/tr/keeper/msg_server_test.go
git commit -m "test(tr): replace TestMsgServerArchiveTrustRegistry with fixture-based test"
```

---

## Task 2: Genesis round-trip test (module-completion requirement)

**File:** `x/tr/keeper/genesis_test.go`

Per the design spec: "Each module gets a single `TestGenesisRoundTrip` test using the fixture: build non-trivial state, export → import → assert `require.Equal` on full GenesisState."

- [ ] **Step 2.1: Locate genesis export/import functions.**

  Run: `grep -rn "ExportGenesis\|InitGenesis" /Users/pratik/verana/x/tr/ --include="*.go" | grep -v "_test.go"`

  Identify the function signatures for `ExportGenesis` and `InitGenesis` in the TR module.

- [ ] **Step 2.2: Create `x/tr/keeper/genesis_test.go`.**

The test builds a TR with a v1+v2 GFVersion, multiple GFDocuments, then exports and re-imports:

```go
package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/verana-labs/verana/x/tr/keeper"
	"github.com/verana-labs/verana/x/tr/types"
)

func TestGenesisRoundTrip(t *testing.T) {
	f := NewFixture(t)
	now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	f.SetBlockTime(now)

	// Create TR 1
	_, err := f.MS.CreateTrustRegistry(f.Ctx, &types.MsgCreateTrustRegistry{
		Corporation:  testCorp,
		Operator:     testOperator,
		Did:          testDID,
		Aka:          testAka,
		Language:     "en",
		DocUrl:       testDocURL,
		DocDigestSri: testDigestSRI,
	})
	require.NoError(t, err)

	trID, err := f.K.TrustRegistryDIDIndex.Get(f.Ctx, testDID)
	require.NoError(t, err)

	f.AdvanceTime(time.Hour)

	// Add v2 English doc
	_, err = f.MS.AddGovernanceFrameworkDocument(f.Ctx, &types.MsgAddGovernanceFrameworkDocument{
		Corporation: testCorp, Operator: testOperator,
		TrId: trID, Language: "en",
		Url:       "https://example.com/gf-v2-en.html",
		DigestSri: testDigestSRI,
		Version:   2,
	})
	require.NoError(t, err)

	// Add v2 French doc
	_, err = f.MS.AddGovernanceFrameworkDocument(f.Ctx, &types.MsgAddGovernanceFrameworkDocument{
		Corporation: testCorp, Operator: testOperator,
		TrId: trID, Language: "fr",
		Url:       "https://example.com/gf-v2-fr.html",
		DigestSri: testDigestSRI,
		Version:   2,
	})
	require.NoError(t, err)

	// Promote v2
	f.AdvanceTime(time.Hour)
	_, err = f.MS.IncreaseActiveGovernanceFrameworkVersion(f.Ctx,
		&types.MsgIncreaseActiveGovernanceFrameworkVersion{
			Corporation: testCorp, Operator: testOperator, TrId: trID,
		})
	require.NoError(t, err)

	// Create TR 2 (archived)
	did2 := "did:example:second-tr"
	f.AdvanceTime(time.Hour)
	_, err = f.MS.CreateTrustRegistry(f.Ctx, &types.MsgCreateTrustRegistry{
		Corporation:  testCorp,
		Operator:     testOperator,
		Did:          did2,
		Language:     "de",
		DocUrl:       testDocURL,
		DocDigestSri: testDigestSRI,
	})
	require.NoError(t, err)
	tr2ID, err := f.K.TrustRegistryDIDIndex.Get(f.Ctx, did2)
	require.NoError(t, err)

	f.AdvanceTime(time.Hour)
	_, err = f.MS.ArchiveTrustRegistry(f.Ctx, &types.MsgArchiveTrustRegistry{
		Corporation: testCorp, Operator: testOperator, TrId: tr2ID, Archive: true,
	})
	require.NoError(t, err)

	// Export genesis
	gs := keeper.ExportGenesis(f.Ctx, f.K)
	require.NotNil(t, gs)

	// Import genesis into a fresh keeper
	f2 := NewFixture(t)
	keeper.InitGenesis(f2.Ctx, f2.K, gs)

	// Re-export from the fresh keeper
	gs2 := keeper.ExportGenesis(f2.Ctx, f2.K)

	// Full equality on the exported state
	require.Equal(t, gs, gs2, "genesis export must be identical after import+re-export")
}
```

**Note:** If `keeper.ExportGenesis` / `keeper.InitGenesis` have different signatures (e.g., they accept a `*types.GenesisState` or return a different type), adjust accordingly after running step 2.1.

- [ ] **Step 2.3: Build.**

  Run: `go build ./x/tr/keeper/... && go vet ./x/tr/keeper/...`
  Expected: exit 0.

- [ ] **Step 2.4: Run genesis test.**

  Run: `go test ./x/tr/keeper/... -run TestGenesisRoundTrip -v -count=1`
  Expected: PASS.

- [ ] **Step 2.5: Commit.**

```bash
git add x/tr/keeper/genesis_test.go
git commit -m "test(tr): add TestGenesisRoundTrip for module completion"
```

---

## Task 3: Full coverage check and final CI validation

- [ ] **Step 3.1: Coverage check.**

  Run: `go test ./x/tr/keeper/... -cover -count=1`
  Target: ≥95% line coverage. If below, run:

  ```bash
  go test ./x/tr/keeper/... -coverprofile=/tmp/tr.cov -count=1
  go tool cover -func=/tmp/tr.cov | sort -k3 -n | tail -30
  ```

  Identify uncovered lines and add targeted sub-tests until ≥95%.

- [ ] **Step 3.2: Vet.**

  Run: `go vet ./x/tr/keeper/... && go vet ./testutil/keeper/...`
  Expected: no output.

- [ ] **Step 3.3: Full build.**

  Run: `go build ./...`
  Expected: exit 0.

- [ ] **Step 3.4: Full test suite.**

  Run: `go test ./... -count=1`
  Expected: PASS.

- [ ] **Step 3.5: Push and open PR.**

```bash
git push -u origin test/step-10-tr-archive
gh pr create --title "test(tr): step 10 — ArchiveTrustRegistry fixture-based tests + genesis round-trip" --body "$(cat <<'EOF'
## Summary
- Replaces TestMsgServerArchiveTrustRegistry with fixture-based TestMsgArchiveTrustRegistry
- Removes setupMsgServerLegacy (no longer needed after all 5 TR messages migrated)
- Full struct assertion on Archived pointer and Modified timestamp for both archive and unarchive directions
- Negative cases: AUTHZ fail, TR not found, corporation mismatch, double-archive, spurious unarchive
- Adds TestGenesisRoundTrip (module-completion requirement)
- TR module line coverage ≥95%

## Test plan
- [ ] go test ./x/tr/keeper/... -cover reports ≥95%
- [ ] go test ./x/tr/keeper/... -count=1 passes
- [ ] go test ./... -count=1 passes (no regressions)
EOF
)"
```

---

## "Done" Criteria — Step 10

- [ ] `TestMsgServerArchiveTrustRegistry` deleted.
- [ ] `setupMsgServerLegacy` deleted.
- [ ] `TestMsgServer` updated to use `NewFixture`.
- [ ] `TestMsgArchiveTrustRegistry` added with fixture; 8 sub-tests covering all spec preconditions.
- [ ] `TestGenesisRoundTrip` exists in `genesis_test.go` and passes.
- [ ] Happy paths: full TR struct equality (including `Archived *time.Time` pointer value), event `archive_status` attribute, invariant.
- [ ] Negative cases: error string, TR struct unchanged (Archived nil/unchanged, Modified unchanged), no event.
- [ ] `go test ./x/tr/keeper/... -cover -count=1` ≥95%.
- [ ] `go test ./... -count=1` PASS.

---

## Self-Review

- No TBD/TODO items.
- `Archived *time.Time` is a pointer; the happy-path "archive" sub-test asserts `Archived: &archivedAt` (pointer to a specific time), not just non-nil. This catches a bug where the correct pointer is set but to the wrong timestamp.
- "double-archive" sub-test explicitly checks `*tr.Archived == original archivedAt` — this verifies the timestamp was NOT overwritten even though the error returned.
- "archive→unarchive→re-archive" covers the full bidirectional toggle cycle; the old test covered this only via dependent sequential `testCases` (shared state bug risk). Each sub-test here uses an isolated fixture.
- After step 10 the `msg_server_test.go` file imports: `errors`, `fmt`, `testing`, `time`, `require`, `types`. The `sdk`, `keepertest`, and `keeper` (package) imports from `setupMsgServerLegacy` are removed — build step 1.4 will catch any missed import.
- Genesis round-trip test uses `keeper.ExportGenesis` / `keeper.InitGenesis` as package-level functions. If they are unexported or differently named, step 2.1 (`grep`) will surface the correct names before the code is written.
- `RequireInvariant` checks `tr.ActiveVersion >= 1` for every TR row; the archived TR2 still satisfies this invariant (archiving does not reset ActiveVersion).
