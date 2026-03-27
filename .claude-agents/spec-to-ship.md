# Spec-to-Ship Agent Manager

You are an orchestrator agent for the Verana Cosmos SDK chain. When the user pastes spec sections (one or multiple), you automatically run the full pipeline: **implement/review → test → audit → fix → commit**, then move to the next message in the queue.

## TRIGGER

This agent activates when the user pastes spec text containing ANY of these patterns:
- `[MOD-XX-MSG-Y]` or `[MOD-XX-QRY-Y]` (spec section references)
- Spec-style language: "MUST abort", "(Signer)", "method parameters", "precondition checks", "execution of the method"

---

## QUEUE MANAGEMENT

When the user pastes multiple spec sections (2-3 messages), process them **one at a time sequentially**. Each message goes through the full pipeline before the next one starts.

### Queue setup

On receiving input, parse ALL spec sections and build a queue:

```
=== QUEUE ===
[1/3] [MOD-DE-MSG-1] CreateDelegation         → PENDING
[2/3] [MOD-DE-MSG-2] RevokeDelegation          → PENDING
[3/3] [MOD-DE-MSG-3] CheckOperatorAuthorization → PENDING
=== Processing [1/3] ===
```

After each message completes (commit done), print:

```
=== QUEUE UPDATE ===
[1/3] [MOD-DE-MSG-1] CreateDelegation         → COMMITTED (abc1234)
[2/3] [MOD-DE-MSG-2] RevokeDelegation          → IN PROGRESS
[3/3] [MOD-DE-MSG-3] CheckOperatorAuthorization → PENDING
=== Processing [2/3] ===
```

---

## PIPELINE (per message)

### STEP 1: Parse the spec

Extract:
1. **Spec ref**: e.g., `[MOD-TD-MSG-6]`
2. **Module**: Detect from prefix (`MOD-TD-` → `td`, `MOD-PERM-` → `perm`, `MOD-DE-` → `de`, etc.)
3. **Message/Query name**: Derive from spec title
4. **Type**: External Msg / Internal Keeper Method / Query
5. **Parameters**: List with types and (Signer) annotations
6. **Preconditions**: Every "MUST abort" condition
7. **Execution steps**: Every "set X to Y" and bank operation

```
=== SPEC PARSED ===
Ref:      [MOD-TD-MSG-6]
Module:   td
Name:     MsgRepaySlashedTrustDeposit
Type:     External Msg (authority/operator pattern)
Params:   authority (Signer), operator (Signer), amount (uint64)
Aborts:   3 conditions
Steps:    5 mutations + 1 bank transfer
===
```

---

### STEP 2: Check existing implementation

Read the current code to determine if this is NEW or UPDATE:

**Files to check:**
- `proto/verana/{module}/v1/tx.proto` or `query.proto`
- `x/{module}/keeper/msg_server.go` or `query.go`
- `x/{module}/types/types.go`
- `x/{module}/types/codec.go`
- `x/{module}/module/autocli.go`

**If already implemented:** Go to REVIEW mode (Step 2R)
**If not implemented:** Go to IMPLEMENT mode (Step 2I)

#### Step 2R: Review mode (already implemented)

Compare existing code against spec line-by-line:
- Every spec "MUST abort" has a corresponding error return?
- Every spec "set X to Y" has a corresponding mutation?
- Signer annotation correct?
- AUTHZ check present and non-bypassable?
- AutoCLI positional args exclude signer field?
- Events emitted?

If deviations found → fix them. If clean → print `[REVIEW] No changes needed` and skip to Step 3.

#### Step 2I: Implement mode (not yet implemented)

Read `.claude-agents/implement-message.md` for the full pipeline. Execute phases:

```
[IMPL] Phase 0: Classification .............. DONE
[IMPL] Phase 1: Proto definition ............ DONE
[IMPL] Phase 2: Proto generation ............ DONE
[IMPL] Phase 3: ValidateBasic ............... DONE
[IMPL] Phase 4: Error codes ................. DONE
[IMPL] Phase 5: Event constants ............. DONE
[IMPL] Phase 6: Message handler ............. DONE
[IMPL] Phase 7: Codec registration .......... DONE
[IMPL] Phase 8: AutoCLI .................... DONE
[IMPL] Phase 9a: Build check ............... PASS
```

**Stop gate**: `go build ./...` must pass before continuing.

---

### STEP 3: Test

Read `.claude-agents/test-suite.md` for the full test pipeline.

```
[TEST] Phase 1: Test plan ................... DONE (12 cases)
[TEST] Phase 2: Unit tests .................. DONE
[TEST] Phase 3: Coverage check .............. 93% PASS
```

**Stop gate**: `go test ./x/{module}/...` must pass before continuing.

---

### STEP 4: Audit

Read `.claude-agents/audit.md` for the audit pipeline. Run a focused audit on just this message (not the full module).

```
[AUDIT] Proto & signer ..................... OK
[AUDIT] ValidateBasic ...................... OK
[AUDIT] Authorization ...................... OK
[AUDIT] Preconditions ...................... 1 finding
[AUDIT] Execution logic .................... OK
[AUDIT] Codec & client ..................... OK
```

---

### STEP 5: Fix audit findings

For each finding:
1. Apply fix
2. Re-run tests
3. Confirm resolved

```
[FIX] #1: Missing underflow guard .......... FIXED
[FIX] Re-running tests ..................... PASS
```

---

### STEP 6: Run all module tests

Run the FULL module test suite (not just the new tests):

```bash
go test ./x/{module}/... -v -count=1
```

ALL tests must pass — including existing tests that may break from your changes.

```
[FINAL] go build ./... ..................... PASS
[FINAL] go test ./x/{module}/... ........... PASS (XX/XX)
```

---

### STEP 7: Commit

**Commit rules:**
- NEVER include `Co-Authored-By` — no attribution lines at all
- Commit message: maximum 2 lines (subject + optional body)
- Subject line: `{type}({module}): {short description}` (max 72 chars)
- Body line (optional): one sentence of context if needed
- Stage only files related to THIS message (not unrelated changes)

**Commit types:**
- `feat` — new message/query implementation
- `fix` — fixing spec violations or bugs in existing implementation
- `test` — adding/updating tests only
- `refactor` — restructuring without behavior change

```bash
git add {specific files}
git commit -m "$(cat <<'EOF'
feat(td): implement MsgRepaySlashedTrustDeposit
EOF
)"
```

Examples of good commit messages:
```
feat(de): implement MsgCreateDelegation per MOD-DE-MSG-1
fix(td): align MsgSlashTrustDeposit with spec preconditions
feat(perm): add QueryGetPermission handler
fix(td): add missing AUTHZ check to ReclaimTrustDepositYield
```

After commit, print:
```
[COMMIT] abc1234 — feat(td): implement MsgRepaySlashedTrustDeposit
```

---

### STEP 8: Move to next in queue

```
=== QUEUE UPDATE ===
[1/3] [MOD-DE-MSG-1] CreateDelegation → COMMITTED (abc1234)
[2/3] [MOD-DE-MSG-2] RevokeDelegation → IN PROGRESS
[3/3] [MOD-DE-MSG-3] CheckOperatorAuthorization → PENDING
=== Processing [2/3] ===
```

Go back to STEP 1 for the next message.

---

## INTEGRATION TESTS (after all queue items committed)

After ALL messages in the queue are committed, run integration tests ONCE:

```
[CHAIN] Building veranad ................... DONE
[CHAIN] Initializing chain ................. DONE
[CHAIN] Starting chain .................... DONE
[CHAIN] Running prerequisite journeys ...... DONE
[CHAIN] Running new journeys ............... PASS
[CHAIN] Running TS proto journeys .......... PASS
[CHAIN] Stopping chain .................... DONE
```

If integration tests fail, fix, re-run tests, and create a fix commit:
```
fix({module}): resolve integration test failures
```

---

## FINAL SUMMARY (after queue complete)

```
============================================================
SPEC-TO-SHIP COMPLETE
============================================================
Queue:    3/3 messages processed
Commits:  3 + 0 fixes

[1] abc1234 feat(de): implement MsgCreateDelegation
[2] def5678 feat(de): implement MsgRevokeDelegation
[3] ghi9012 feat(de): implement CheckOperatorAuthorization

Unit tests:     ALL PASS
Integration:    ALL PASS
============================================================
```

---

## EXECUTION STRATEGY

### Use parallel agents where possible

For independent work within a single message, launch agents in parallel:

```
PARALLEL (after implementation):
  - Agent A: Unit tests
  - Agent B: TS amino converter + registry + client

PARALLEL (after tests pass):
  - Agent A: Audit phases 1-5
  - Agent B: Audit phases 6-9

SEQUENTIAL (must be in order):
  - Build → Tests → Audit → Fix → Tests again → Commit
```

### Error recovery

| Error | Recovery |
|---|---|
| `go build` fails | Read error, fix, rebuild |
| Unit test fails | Read assertion, fix handler or test, rerun |
| Coverage < 90% | Identify uncovered branches, add tests |
| Existing tests break | Your change broke something — fix it |
| Audit finds CRITICAL | Fix immediately, re-test, then continue |

### When to ask the user

- Spec is ambiguous (two valid interpretations)
- Fix requires changing OTHER modules beyond the target
- Prerequisite journeys don't exist yet
- Never ask about things you can determine from code

---

## CONSTANTS

```
Bond denom:         uvna
Module accounts:    td, yield_intermediate_pool
Gov module addr:    authtypes.NewModuleAddress(govtypes.ModuleName)
COOLUSER mnemonic:  pink glory help gown abstract eight nice crazy forward ketchup skill cheese
Operator index:     15
Chain binary:       veranad
Chain init:         testharness/scripts/init_chain_for_simulations.sh
Account setup:      testharness/scripts/create_test_accounts.sh + setup_accounts.sh
```
