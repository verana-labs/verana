# Step 7: TR AddGovernanceFrameworkDocument — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace `TestMsgServerAddGovernanceFrameworkDocument` with a fixture-based test that fully asserts GFVersion + GFDocument state, emitted events, and all spec precondition failure modes.

**Architecture:** Extend the `Fixture` from step 6 (`fixture_test.go`) with no structural changes — all helpers are already available. The new `TestMsgAddGovernanceFrameworkDocument` in `msg_server_test.go` replaces the legacy `TestMsgServerAddGovernanceFrameworkDocument` table-driven test. Each `t.Run` gets its own `NewFixture(t)` so state is isolated; the setup helper `createTestTR` creates a valid trust registry as a precondition for each sub-test.

**Tech Stack:** Go 1.24, Cosmos SDK v0.50.x, testify/require

---

## Pre-flight

- [ ] **Worktree.** Branch name: `test/step-7-tr-add-gf-doc`.
- [ ] **Gate.** Confirm step 6 PR is merged (fixture_test.go and TrustregistryKeeperWithDelegation exist).
- [ ] **Sanity check.**

  Run: `go build ./... && go vet ./...`
  Expected: exit 0.

---

## File Structure

- Modify: `x/tr/keeper/msg_server_test.go` — delete `TestMsgServerAddGovernanceFrameworkDocument`, add `TestMsgAddGovernanceFrameworkDocument` using fixture.

---

## Task 1: Write the fixture-based `TestMsgAddGovernanceFrameworkDocument`

**File:** `x/tr/keeper/msg_server_test.go`

Delete `TestMsgServerAddGovernanceFrameworkDocument` and insert the new fixture-based test. All other test functions remain intact.

- [ ] **Step 1.1: Add a shared `createTestTR` helper at the top of the new test block.**

The helper creates a TR with a fixed block time, returns the TR id and the creation timestamp.

```go
// createTestTR is a test-local helper that creates a single trust registry
// using f and returns (trID, createdAt).
func createTestTR(t *testing.T, f *Fixture) (trID uint64, createdAt time.Time) {
	t.Helper()
	createdAt = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	f.SetBlockTime(createdAt)
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
	trID, err = f.K.TrustRegistryDIDIndex.Get(f.Ctx, testDID)
	require.NoError(t, err)
	return trID, createdAt
}
```

- [ ] **Step 1.2: Write `TestMsgAddGovernanceFrameworkDocument`.**

```go
func TestMsgAddGovernanceFrameworkDocument(t *testing.T) {
	const (
		docURL2En  = "https://example.com/gf-v2-en.html"
		docURL2Fr  = "https://example.com/gf-v2-fr.html"
		docURL3En  = "https://example.com/gf-v3-en.html"
		digest2    = "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26"
	)

	t.Run("MOD-TR-MSG-2: happy path adds GFDoc to new version", func(t *testing.T) {
		f := NewFixture(t)
		trID, createdAt := createTestTR(t, f)

		addedAt := createdAt.Add(time.Hour)
		f.SetBlockTime(addedAt)

		resp, err := f.MS.AddGovernanceFrameworkDocument(f.Ctx, &types.MsgAddGovernanceFrameworkDocument{
			Corporation: testCorp,
			Operator:    testOperator,
			TrId:        trID,
			Language:    "en",
			Url:         docURL2En,
			DigestSri:   digest2,
			Version:     2,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		// A new GFVersion row must exist for version 2
		f.RequireGFVersionCount(2) // v1 from CreateTrustRegistry + v2 just added
		f.RequireGFVersion(trID, 2, func(gfv types.GovernanceFrameworkVersion) {
			require.Equal(t, trID, gfv.TrId)
			require.Equal(t, int32(2), gfv.Version)
			require.Equal(t, addedAt, gfv.Created)
			// ActiveSince must be zero (not yet activated)
			require.True(t, gfv.ActiveSince.IsZero(),
				"new GFVersion must have zero ActiveSince until IncreaseActiveGFVersion is called")
		})

		// GFDocument created and linked to the new GFVersion
		f.RequireGFDocumentCount(2) // 1 from CreateTrustRegistry + 1 just added
		var gfv2ID uint64
		_ = f.K.GFVersion.Walk(f.Ctx, nil, func(id uint64, gfv types.GovernanceFrameworkVersion) (bool, error) {
			if gfv.TrId == trID && gfv.Version == 2 {
				gfv2ID = id
				return true, nil
			}
			return false, nil
		})
		require.NotZero(t, gfv2ID)
		f.RequireGFDocument(gfv2ID, "en", docURL2En, digest2)

		// TrustRegistry state must be unchanged (AddGFDoc does not mutate the TR row)
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

		// Event emitted
		f.RequireEvent(types.EventTypeAddGovernanceFrameworkDocument, map[string]string{
			types.AttributeKeyCorporation: testCorp,
			types.AttributeKeyVersion:     "2",
			types.AttributeKeyLanguage:    "en",
			types.AttributeKeyDocURL:      docURL2En,
			types.AttributeKeyDigestSri:   digest2,
		})

		f.RequireInvariant()
	})

	t.Run("MOD-TR-MSG-2: adding same version different language adds second GFDoc, same GFVersion", func(t *testing.T) {
		f := NewFixture(t)
		trID, _ := createTestTR(t, f)
		f.AdvanceTime(time.Hour)

		// Add English doc for v2
		_, err := f.MS.AddGovernanceFrameworkDocument(f.Ctx, &types.MsgAddGovernanceFrameworkDocument{
			Corporation: testCorp, Operator: testOperator,
			TrId: trID, Language: "en", Url: docURL2En, DigestSri: digest2, Version: 2,
		})
		require.NoError(t, err)

		f.AdvanceTime(time.Hour)

		// Add French doc for same v2
		_, err = f.MS.AddGovernanceFrameworkDocument(f.Ctx, &types.MsgAddGovernanceFrameworkDocument{
			Corporation: testCorp, Operator: testOperator,
			TrId: trID, Language: "fr", Url: docURL2Fr, DigestSri: digest2, Version: 2,
		})
		require.NoError(t, err)

		// GFVersion count stays at 2 (no new version row created)
		f.RequireGFVersionCount(2)
		// GFDocument count grows to 3
		f.RequireGFDocumentCount(3)

		var gfv2ID uint64
		_ = f.K.GFVersion.Walk(f.Ctx, nil, func(id uint64, gfv types.GovernanceFrameworkVersion) (bool, error) {
			if gfv.TrId == trID && gfv.Version == 2 {
				gfv2ID = id
				return true, nil
			}
			return false, nil
		})
		require.NotZero(t, gfv2ID)
		f.RequireGFDocument(gfv2ID, "en", docURL2En, digest2)
		f.RequireGFDocument(gfv2ID, "fr", docURL2Fr, digest2)

		f.RequireInvariant()
	})

	t.Run("MOD-TR-MSG-2: replacing existing doc for same (version, language) updates URL and digest in-place", func(t *testing.T) {
		f := NewFixture(t)
		trID, _ := createTestTR(t, f)
		f.AdvanceTime(time.Hour)

		_, err := f.MS.AddGovernanceFrameworkDocument(f.Ctx, &types.MsgAddGovernanceFrameworkDocument{
			Corporation: testCorp, Operator: testOperator,
			TrId: trID, Language: "en", Url: docURL2En, DigestSri: digest2, Version: 2,
		})
		require.NoError(t, err)

		newURL := "https://example.com/gf-v2-en-revised.html"
		newDigest := "sha384-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA1"
		f.AdvanceTime(time.Hour)

		_, err = f.MS.AddGovernanceFrameworkDocument(f.Ctx, &types.MsgAddGovernanceFrameworkDocument{
			Corporation: testCorp, Operator: testOperator,
			TrId: trID, Language: "en", Url: newURL, DigestSri: newDigest, Version: 2,
		})
		require.NoError(t, err)

		// Still 2 GFVersions (v1 + v2) and 2 GFDocuments (v1-en + v2-en replaced)
		f.RequireGFVersionCount(2)
		f.RequireGFDocumentCount(2)

		var gfv2ID uint64
		_ = f.K.GFVersion.Walk(f.Ctx, nil, func(id uint64, gfv types.GovernanceFrameworkVersion) (bool, error) {
			if gfv.TrId == trID && gfv.Version == 2 {
				gfv2ID = id
				return true, nil
			}
			return false, nil
		})
		require.NotZero(t, gfv2ID)
		f.RequireGFDocument(gfv2ID, "en", newURL, newDigest)

		f.RequireInvariant()
	})

	t.Run("MOD-TR-MSG-2-2-1: fails if operator not authorized", func(t *testing.T) {
		f := NewFixture(t)
		trID, _ := createTestTR(t, f)

		// Now make AUTHZ fail
		f.DelKeeper.ErrToReturn = errors.New("not authorized")
		f.AdvanceTime(time.Hour)

		_, err := f.MS.AddGovernanceFrameworkDocument(f.Ctx, &types.MsgAddGovernanceFrameworkDocument{
			Corporation: testCorp, Operator: testOperator,
			TrId: trID, Language: "en", Url: docURL2En, DigestSri: digest2, Version: 2,
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "authorization check failed")

		// No new state written
		f.RequireGFVersionCount(1)  // only v1 from CreateTrustRegistry
		f.RequireGFDocumentCount(1) // only the initial doc
		f.RequireNoEvent(types.EventTypeAddGovernanceFrameworkDocument)
	})

	t.Run("MOD-TR-MSG-2-2-1: fails if TR not found", func(t *testing.T) {
		f := NewFixture(t)
		_, err := f.MS.AddGovernanceFrameworkDocument(f.Ctx, &types.MsgAddGovernanceFrameworkDocument{
			Corporation: testCorp, Operator: testOperator,
			TrId: 99999, Language: "en", Url: docURL2En, DigestSri: digest2, Version: 2,
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "does not exist")

		f.RequireTrustRegistryCount(0)
		f.RequireGFVersionCount(0)
		f.RequireGFDocumentCount(0)
		f.RequireNoEvent(types.EventTypeAddGovernanceFrameworkDocument)
	})

	t.Run("MOD-TR-MSG-2-2-1: fails if corporation mismatch", func(t *testing.T) {
		f := NewFixture(t)
		trID, _ := createTestTR(t, f)
		f.AdvanceTime(time.Hour)

		_, err := f.MS.AddGovernanceFrameworkDocument(f.Ctx, &types.MsgAddGovernanceFrameworkDocument{
			Corporation: "wrong-corp",
			Operator:    testOperator,
			TrId:        trID,
			Language:    "en",
			Url:         docURL2En,
			DigestSri:   digest2,
			Version:     2,
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "corporation is not the controller")

		f.RequireGFVersionCount(1)  // no new version
		f.RequireGFDocumentCount(1) // no new doc
		f.RequireNoEvent(types.EventTypeAddGovernanceFrameworkDocument)
	})

	t.Run("MOD-TR-MSG-2-2-1: fails if version equals active version (immutable)", func(t *testing.T) {
		f := NewFixture(t)
		trID, _ := createTestTR(t, f)
		f.AdvanceTime(time.Hour)

		// Version 1 is the current active version; it must be rejected
		_, err := f.MS.AddGovernanceFrameworkDocument(f.Ctx, &types.MsgAddGovernanceFrameworkDocument{
			Corporation: testCorp, Operator: testOperator,
			TrId: trID, Language: "en", Url: docURL2En, DigestSri: digest2, Version: 1,
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "greater than")

		f.RequireGFVersionCount(1)
		f.RequireGFDocumentCount(1)
		f.RequireNoEvent(types.EventTypeAddGovernanceFrameworkDocument)
	})

	t.Run("MOD-TR-MSG-2-2-1: fails if version skips a gap (e.g. 4 when max is 2)", func(t *testing.T) {
		f := NewFixture(t)
		trID, _ := createTestTR(t, f)
		f.AdvanceTime(time.Hour)

		// Add v2 first
		_, err := f.MS.AddGovernanceFrameworkDocument(f.Ctx, &types.MsgAddGovernanceFrameworkDocument{
			Corporation: testCorp, Operator: testOperator,
			TrId: trID, Language: "en", Url: docURL2En, DigestSri: digest2, Version: 2,
		})
		require.NoError(t, err)

		// Attempt to add v4 (skipping v3)
		_, err = f.MS.AddGovernanceFrameworkDocument(f.Ctx, &types.MsgAddGovernanceFrameworkDocument{
			Corporation: testCorp, Operator: testOperator,
			TrId: trID, Language: "en", Url: docURL3En, DigestSri: digest2, Version: 4,
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "invalid version")

		f.RequireGFVersionCount(2) // v1 + v2 only
		f.RequireGFDocumentCount(2)
		f.RequireNoEvent(types.EventTypeAddGovernanceFrameworkDocument)
	})

	t.Run("MOD-TR-MSG-2-2-1: fails on invalid BCP47 language tag", func(t *testing.T) {
		f := NewFixture(t)
		trID, _ := createTestTR(t, f)
		f.AdvanceTime(time.Hour)

		_, err := f.MS.AddGovernanceFrameworkDocument(f.Ctx, &types.MsgAddGovernanceFrameworkDocument{
			Corporation: testCorp, Operator: testOperator,
			TrId: trID, Language: "not_a_real_lang", Url: docURL2En, DigestSri: digest2, Version: 2,
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "invalid language")

		f.RequireGFVersionCount(1)
		f.RequireGFDocumentCount(1)
		f.RequireNoEvent(types.EventTypeAddGovernanceFrameworkDocument)
	})
}
```

- [ ] **Step 1.3: Remove the `errors` import if not already present; add it to the import block.**

The test file must import `"errors"` for `errors.New(...)` in the AUTHZ failure case. Verify it is in the import block.

- [ ] **Step 1.4: Build.**

  Run: `go build ./x/tr/keeper/... && go vet ./x/tr/keeper/...`
  Expected: exit 0.

- [ ] **Step 1.5: Run new tests.**

  Run: `go test ./x/tr/keeper/... -run TestMsgAddGovernanceFrameworkDocument -v -count=1`
  Expected: PASS (all sub-tests green).

- [ ] **Step 1.6: Run full TR keeper suite.**

  Run: `go test ./x/tr/keeper/... -count=1 -v`
  Expected: PASS — no regressions.

- [ ] **Step 1.7: Commit.**

```bash
git add x/tr/keeper/msg_server_test.go
git commit -m "test(tr): replace TestMsgServerAddGovernanceFrameworkDocument with fixture-based test"
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
git push -u origin test/step-7-tr-add-gf-doc
gh pr create --title "test(tr): step 7 — AddGovernanceFrameworkDocument fixture-based tests" --body "$(cat <<'EOF'
## Summary
- Replaces legacy TestMsgServerAddGovernanceFrameworkDocument with fixture-based TestMsgAddGovernanceFrameworkDocument
- Full GFVersion + GFDocument struct assertions per sub-test
- Zero-state and no-event assertions for every negative case
- Negative cases cover: AUTHZ fail, TR not found, corporation mismatch, version <= active_version, version skips gap, invalid language tag

## Test plan
- [ ] go test ./x/tr/keeper/... -count=1 passes
- [ ] go test ./... -count=1 passes (no regressions)
EOF
)"
```

---

## "Done" Criteria — Step 7

- [ ] `TestMsgServerAddGovernanceFrameworkDocument` deleted.
- [ ] `TestMsgAddGovernanceFrameworkDocument` added with fixture.
- [ ] Happy path (3 sub-tests): GFVersion count, GFDocument count, full field assertions, event, invariant.
- [ ] Negative cases (6 sub-tests): error string, zero new state, no event.
- [ ] `go test ./x/tr/keeper/... -count=1` PASS.
- [ ] `go test ./... -count=1` PASS.

---

## Self-Review

- No TBD/TODO items.
- `createTestTR` is a file-local helper (lower-case `c`); does not pollute fixture_test.go.
- Happy path "new version" sub-test asserts: GFVersion count = 2, GFVersion fields (trID, version, Created, ActiveSince==zero), GFDocument count = 2, GFDocument URL+digest, TR struct unchanged (AddGFDoc does NOT mutate the TR row), event attributes.
- Negative cases all assert `RequireNoEvent(types.EventTypeAddGovernanceFrameworkDocument)`.
- `errors` package must be in the import block — noted in step 1.3.
- `GFVersionByTR` is `collections.Map[Pair[uint64, int32], uint64]` — walking `GFVersion` to resolve IDs avoids importing `collections` in test files.
- Event attribute `AttributeKeyGFVersionID` maps to `tr_id` (see events.go line 19 — the constant is misnamed "gf_version_id" but its value in events.go is `"gf_version_id"` while in msg_server.go line 99 it is `strconv.FormatUint(msg.TrId, 10)`). Test uses the constant name; the assertion is on the value.
