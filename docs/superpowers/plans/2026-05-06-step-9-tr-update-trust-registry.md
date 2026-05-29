# Step 9: TR UpdateTrustRegistry — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace `TestMsgServerUpdateTrustRegistry` with a fixture-based test that gives a full struct assertion on the updated TR row, DID-index consistency, emitted event, and all spec precondition failure modes.

**Architecture:** Uses `Fixture` from step 6 and `createTestTR` from step 7. Each sub-test gets an isolated `NewFixture(t)`. Key behaviors under test: DID update rewrites the DID index (old DID removed, new DID added), Aka can be cleared to empty string, Modified timestamp advances while Created stays unchanged, Language is NOT updatable. No bank operations.

**Tech Stack:** Go 1.24, Cosmos SDK v0.50.x, testify/require

---

## Pre-flight

- [ ] **Worktree.** Branch name: `test/step-9-tr-update`.
- [ ] **Gate.** Confirm steps 6-8 PRs are merged.
- [ ] **Sanity check.**

  Run: `go build ./... && go vet ./...`
  Expected: exit 0.

---

## File Structure

- Modify: `x/tr/keeper/msg_server_test.go` — delete `TestMsgServerUpdateTrustRegistry`, add `TestMsgUpdateTrustRegistry` using fixture.

---

## Task 1: Write the fixture-based `TestMsgUpdateTrustRegistry`

**File:** `x/tr/keeper/msg_server_test.go`

- [ ] **Step 1.1: Write `TestMsgUpdateTrustRegistry`.**

```go
func TestMsgUpdateTrustRegistry(t *testing.T) {
	const (
		newDID = "did:example:updated-9876543210"
		newAka = "https://updated.example.com/registry"
	)

	t.Run("MOD-TR-MSG-4: happy path updates DID, Aka, Modified; preserves Created and Language", func(t *testing.T) {
		f := NewFixture(t)
		trID, createdAt := createTestTR(t, f)

		updatedAt := createdAt.Add(time.Hour)
		f.SetBlockTime(updatedAt)

		resp, err := f.MS.UpdateTrustRegistry(f.Ctx, &types.MsgUpdateTrustRegistry{
			Corporation: testCorp,
			Operator:    testOperator,
			TrId:        trID,
			Did:         newDID,
			Aka:         newAka,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		// Full struct assertion
		f.RequireTrustRegistry(trID, types.TrustRegistry{
			Id:            trID,
			Did:           newDID,
			Corporation:   testCorp,
			Created:       createdAt,  // unchanged
			Modified:      updatedAt,  // advanced
			Archived:      nil,
			Aka:           newAka,
			ActiveVersion: 1,          // unchanged
			Language:      "en",       // NOT updatable
		})

		// DID index: old DID must be gone, new DID must map to trID
		f.RequireNoDIDIndex(testDID)
		f.RequireDIDIndex(newDID, trID)

		// Event emitted with new values
		f.RequireEvent(types.EventTypeUpdateTrustRegistry, map[string]string{
			types.AttributeKeyCorporation: testCorp,
			types.AttributeKeyDID:         newDID,
			types.AttributeKeyAka:         newAka,
		})

		f.RequireInvariant()
	})

	t.Run("MOD-TR-MSG-4: update with same DID does not duplicate DID index entry", func(t *testing.T) {
		f := NewFixture(t)
		trID, createdAt := createTestTR(t, f)
		f.SetBlockTime(createdAt.Add(time.Hour))

		// Update DID to same value — index must not be corrupted
		resp, err := f.MS.UpdateTrustRegistry(f.Ctx, &types.MsgUpdateTrustRegistry{
			Corporation: testCorp,
			Operator:    testOperator,
			TrId:        trID,
			Did:         testDID, // same DID as original
			Aka:         newAka,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		// DID index still maps testDID -> trID
		f.RequireDIDIndex(testDID, trID)

		tr, err := f.K.TrustRegistry.Get(f.Ctx, trID)
		require.NoError(t, err)
		require.Equal(t, testDID, tr.Did)
		require.Equal(t, newAka, tr.Aka)

		f.RequireInvariant()
	})

	t.Run("MOD-TR-MSG-4: update clears Aka when set to empty string", func(t *testing.T) {
		f := NewFixture(t)
		trID, createdAt := createTestTR(t, f)
		f.SetBlockTime(createdAt.Add(time.Hour))

		resp, err := f.MS.UpdateTrustRegistry(f.Ctx, &types.MsgUpdateTrustRegistry{
			Corporation: testCorp,
			Operator:    testOperator,
			TrId:        trID,
			Did:         testDID,
			Aka:         "", // intentionally clear
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		tr, err := f.K.TrustRegistry.Get(f.Ctx, trID)
		require.NoError(t, err)
		require.Equal(t, "", tr.Aka)

		f.RequireEvent(types.EventTypeUpdateTrustRegistry, map[string]string{
			types.AttributeKeyAka: "",
		})
		f.RequireInvariant()
	})

	t.Run("MOD-TR-MSG-4-2-1: fails if operator not authorized", func(t *testing.T) {
		f := NewFixture(t)
		trID, createdAt := createTestTR(t, f)
		f.SetBlockTime(createdAt.Add(time.Hour))

		f.DelKeeper.ErrToReturn = errors.New("not authorized")

		_, err := f.MS.UpdateTrustRegistry(f.Ctx, &types.MsgUpdateTrustRegistry{
			Corporation: testCorp, Operator: testOperator,
			TrId: trID, Did: newDID, Aka: newAka,
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "authorization check failed")

		// TR must be unchanged
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

		// Old DID index still intact; new DID must not appear
		f.RequireDIDIndex(testDID, trID)
		f.RequireNoDIDIndex(newDID)
		f.RequireNoEvent(types.EventTypeUpdateTrustRegistry)
	})

	t.Run("MOD-TR-MSG-4-2-1: fails if TR not found", func(t *testing.T) {
		f := NewFixture(t)

		_, err := f.MS.UpdateTrustRegistry(f.Ctx, &types.MsgUpdateTrustRegistry{
			Corporation: testCorp, Operator: testOperator,
			TrId: 99999, Did: newDID, Aka: newAka,
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "trust registry not found")

		f.RequireTrustRegistryCount(0)
		f.RequireNoEvent(types.EventTypeUpdateTrustRegistry)
	})

	t.Run("MOD-TR-MSG-4-2-1: fails if corporation mismatch", func(t *testing.T) {
		f := NewFixture(t)
		trID, createdAt := createTestTR(t, f)
		f.SetBlockTime(createdAt.Add(time.Hour))

		_, err := f.MS.UpdateTrustRegistry(f.Ctx, &types.MsgUpdateTrustRegistry{
			Corporation: "attacker-corp",
			Operator:    testOperator,
			TrId:        trID,
			Did:         newDID,
			Aka:         newAka,
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "only trust registry corporation can update")

		// TR is unchanged
		tr, err := f.K.TrustRegistry.Get(f.Ctx, trID)
		require.NoError(t, err)
		require.Equal(t, testDID, tr.Did, "DID must not change on corporation mismatch")
		require.Equal(t, testAka, tr.Aka, "Aka must not change on corporation mismatch")
		require.Equal(t, createdAt, tr.Modified, "Modified must not advance on failure")

		f.RequireDIDIndex(testDID, trID)
		f.RequireNoDIDIndex(newDID)
		f.RequireNoEvent(types.EventTypeUpdateTrustRegistry)
	})

	t.Run("edge: Modified advances by exact block-time delta", func(t *testing.T) {
		f := NewFixture(t)
		trID, createdAt := createTestTR(t, f)

		delta := 7*time.Hour + 23*time.Minute + 11*time.Second
		f.SetBlockTime(createdAt.Add(delta))

		_, err := f.MS.UpdateTrustRegistry(f.Ctx, &types.MsgUpdateTrustRegistry{
			Corporation: testCorp, Operator: testOperator,
			TrId: trID, Did: testDID, Aka: "",
		})
		require.NoError(t, err)

		tr, err := f.K.TrustRegistry.Get(f.Ctx, trID)
		require.NoError(t, err)
		require.Equal(t, createdAt, tr.Created)
		require.Equal(t, createdAt.Add(delta), tr.Modified)
	})
}
```

- [ ] **Step 1.2: Build.**

  Run: `go build ./x/tr/keeper/... && go vet ./x/tr/keeper/...`
  Expected: exit 0.

- [ ] **Step 1.3: Run new tests.**

  Run: `go test ./x/tr/keeper/... -run TestMsgUpdateTrustRegistry -v -count=1`
  Expected: PASS (all sub-tests green).

- [ ] **Step 1.4: Run full TR keeper suite.**

  Run: `go test ./x/tr/keeper/... -count=1 -v`
  Expected: PASS.

- [ ] **Step 1.5: Commit.**

```bash
git add x/tr/keeper/msg_server_test.go
git commit -m "test(tr): replace TestMsgServerUpdateTrustRegistry with fixture-based test"
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
git push -u origin test/step-9-tr-update
gh pr create --title "test(tr): step 9 — UpdateTrustRegistry fixture-based tests" --body "$(cat <<'EOF'
## Summary
- Replaces legacy TestMsgServerUpdateTrustRegistry with fixture-based TestMsgUpdateTrustRegistry
- Asserts full TR struct equality after update (DID, Aka, Modified, Created preserved, Language preserved)
- DID index consistency: old DID removed, new DID inserted
- Negative cases: AUTHZ fail, TR not found, corporation mismatch — all assert zero mutation and no event

## Test plan
- [ ] go test ./x/tr/keeper/... -count=1 passes
- [ ] go test ./... -count=1 passes
EOF
)"
```

---

## "Done" Criteria — Step 9

- [ ] `TestMsgServerUpdateTrustRegistry` deleted.
- [ ] `TestMsgUpdateTrustRegistry` added with fixture.
- [ ] Happy paths (3): full TR struct equality, DID index, event attributes, invariant.
- [ ] Negative cases (3): error string, TR unchanged, DID index unchanged, no event.
- [ ] Edge case (1): exact timestamp delta.
- [ ] `go test ./x/tr/keeper/... -count=1` PASS.
- [ ] `go test ./... -count=1` PASS.

---

## Self-Review

- No TBD/TODO items.
- `testDID` and `testAka` are defined in `fixture_test.go` as package-level constants — usable across all test files in `package keeper_test`.
- Happy path "same DID update" covers the `if tr.Did != msg.Did` branch in msg_server.go:183 (the remove+set path is NOT taken); without this test, the else-branch has no coverage.
- "Clears Aka" covers `Aka: ""` — the spec says Aka is optional and can be set to empty.
- Language field: the spec says Language is NOT updatable (msg_server.go does not include Language in the update). The happy-path struct assertion enforces this: `Language: "en"` must survive the update.
- Negative-case "corporation mismatch" checks `tr.Modified == createdAt` to confirm the timestamp did not advance — stronger than just checking no error was returned.
- DID index removal check (`RequireNoDIDIndex(newDID)`) on failures confirms no partial index corruption.
