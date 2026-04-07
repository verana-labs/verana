package prompt

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/verana-labs/verana/tools/cipher/internal/config"
)

const preamble = `You are an expert Cosmos SDK / Go developer on the Verana blockchain.

Verana is a Cosmos SDK 0.50.x appchain implementing the Verifiable Public Registry (VPR) spec.
Every module in x/ implements the VPR/VT spec normatively — never guess, never invent behavior.
Spec: https://verana-labs.github.io/verifiable-trust-vpr-spec/

Modules: x/trustregistry x/credentialschema x/permission x/diddirectory x/tddistrib
Each: keeper/ types/ msgs.go genesis.go module.go query.go tx.go

Proto: edit .proto → make proto-gen. NEVER edit *.pb.go *.pb.gw.go *.pulsar.go
Build: make build (must pass). Test: make test. Lint: make lint.
Commits: conventional (feat: fix: refactor: test: docs:)
Tools: Read LS Glob Grep Edit Write Bash`

func ctx(cfg *config.Config, f string) string {
	b, _ := os.ReadFile(fmt.Sprintf("%s/context/%s", cfg.PromptsDir, f))
	return string(b)
}

func ImplementIssue(cfg *config.Config, num int, title, body, commits, branch string) string {
	return fmt.Sprintf("%s\n\n## VPR Spec\n%s\n\n## Module Patterns\n%s\n\n## Recent commits\n```\n%s\n```\n\n---\n\n## Task: Implement Issue #%d\n\n### %s\n\n%s\n\nBranch: %s\n1. Read AGENTS.md first\n2. Explore relevant module(s)\n3. Plan: types → keeper → handler → events → tests\n4. Proto changes? edit .proto → make proto-gen\n5. Implement per spec\n6. make build && make test && make lint\n7. Conventional commit\n\nDo NOT push. Do NOT open PR.",
		preamble, ctx(cfg, "vpr_spec_summary.md"), ctx(cfg, "verana_modules.md"),
		commits, num, title, body, branch)
}

func FixCI(cfg *config.Config, prNum int, branch, report string, attempt, max int) string {
	return fmt.Sprintf("%s\n\n---\n\n## Task: Fix CI on PR #%d (attempt %d/%d)\n\nBranch: %s\n\n## Failures\n%s\n\n1. Find root cause precisely\n2. Fix the actual problem — no test hacks, no nolint without reason\n3. make build && make test && make lint\n4. Commit: fix: resolve CI failure - <desc>\n\nDo NOT push. Do NOT modify workflows.",
		preamble, prNum, attempt, max, branch, report)
}

func AddressReviews(cfg *config.Config, prNum int, branch, feedback string) string {
	return fmt.Sprintf("%s\n\n---\n\n## Task: Address Reviews on PR #%d\n\nBranch: %s\n\n## Feedback\n%s\n\n1. Address every comment — skip nothing\n2. Spec-breaking request? Follow spec, note it\n3. make build && make test && make lint\n4. Single commit: fix: address review feedback on PR #%d\n\nDo NOT push.",
		preamble, prNum, branch, feedback, prNum)
}

func RebaseResolve(cfg *config.Config, branch, base string) string {
	return fmt.Sprintf("%s\n\n---\n\n## Task: Resolve Rebase Conflicts on %s\n\nBase: %s\n\n1. git fetch origin && git rebase origin/%s\n2. Each conflict: read both sides, preserve intent of both\n3. After each: git add <file> then git rebase --continue\n4. make build must pass\n5. Never git rebase --skip unless commit is genuinely empty\n\nDo NOT push.",
		preamble, branch, base, base)
}

func Freeform(cfg *config.Config, task, commits string) string {
	return fmt.Sprintf("%s\n\n## VPR Spec\n%s\n\n## Module Patterns\n%s\n\n## Recent commits\n```\n%s\n```\n\n---\n\n## Task\n\n%s\n\n1. Read AGENTS.md first\n2. Explore before changing\n3. make build && make test && make lint\n4. Conventional commit\n\nDo NOT push.",
		preamble, ctx(cfg, "vpr_spec_summary.md"), ctx(cfg, "verana_modules.md"), commits, task)
}

func ReviewPR(cfg *config.Config, prNum int, title, diff, history string) string {
	historySection := ""
	if history != "" {
		historySection = fmt.Sprintf("\n## Past Review Patterns (learn from these)\n%s\n", history)
	}
	return fmt.Sprintf(`%s

## VPR Spec
%s

## Module Patterns
%s
%s
---

## Task: Review PR #%d — %s

Below is the full diff. Your review MUST follow this process:

### Phase 1 — Understand (do not write findings yet)
- Read the entire diff to understand what changed and why
- Identify the PR type: Go code, proto, CI/CD, docs, config, tests
- Trace control flow: if value X is set in block A, verify your claim about X by checking all paths

### Phase 2 — Analyze
For each potential finding, BEFORE reporting it:
- Verify the claim is true by re-reading the relevant diff lines
- Check if a guard/condition elsewhere already handles the case
- Confirm the issue can actually be triggered given the control flow

### Phase 3 — Report
For each finding, provide:
- **Severity**: BLOCKING (must fix before merge) or NOTE (improvement suggestion)
- **Evidence**: quote the exact diff line(s)
- **Impact**: what specifically goes wrong if unfixed
- **Confidence**: HIGH (verified in diff) / MEDIUM (likely but not 100%%) / LOW (possible but uncertain)

Only mark as BLOCKING if: security vulnerability, data loss, crash, or correctness bug.
Config mismatches and style issues are NOTE unless they cause runtime failure.

Review areas (adapt to PR type):
1. Correctness — logic matches intended behavior
2. Security — injection, overflow, missing validation, supply chain
3. Cosmos patterns — keeper keys, events, fee math, proto
4. Tests — adequate coverage for changes
5. CI/CD — workflow correctness, permissions, pinning

Do NOT flag issues you are not confident about. Fewer accurate findings > many speculative ones.

`+"```diff\n%s\n```",
		preamble, ctx(cfg, "vpr_spec_summary.md"), ctx(cfg, "verana_modules.md"),
		historySection, prNum, title, diff)
}

func CheckSpec(cfg *config.Config, prNum int, title, diff, history string) string {
	historySection := ""
	if history != "" {
		historySection = fmt.Sprintf("\n## Past Spec Check Patterns (learn from these)\n%s\n", history)
	}
	return fmt.Sprintf(`%s

## VPR Spec (AUTHORITATIVE)
%s
%s
---

## Task: Spec Compliance Check for PR #%d — %s

### Phase 1 — Identify message handlers in the diff
List every MsgXxx handler that appears in the diff. If none, state that this PR has no message handlers and focus on whatever is relevant (queries, genesis, params, etc.).

### Phase 2 — Check each handler against spec
For each message handler found:
1. Are all MUST/MUST NOT requirements met?
2. Are precondition checks complete and in correct order?
3. Is the (Signer) field validated correctly?
4. Are state transitions correct per spec?
5. Are events emitted for every state change?
6. Is fee distribution using exact integer math (sdkmath.Int)?

### Phase 3 — Report
For each violation:
- **Spec ref**: [MOD-XX-MSG-Y] if identifiable, or describe the spec requirement
- **Evidence**: quote the exact code
- **What spec requires vs what code does**
- **Confidence**: HIGH / MEDIUM / LOW

If fully compliant, say so explicitly with brief justification.
If the PR doesn't touch spec-relevant code, say so.

`+"```diff\n%s\n```",
		preamble, ctx(cfg, "vpr_spec_summary.md"),
		historySection, prNum, title, diff)
}

func Slugify(text string, max int) string {
	text = strings.ToLower(text)
	text = regexp.MustCompile(`[^a-z0-9\s-]`).ReplaceAllString(text, "")
	text = strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(text, "-"))
	if len(text) > max {
		text = text[:max]
	}
	return strings.TrimRight(text, "-")
}
