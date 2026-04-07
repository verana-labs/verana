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

func ReviewPR(cfg *config.Config, prNum int, title, diff string) string {
	return fmt.Sprintf(`%s

## VPR Spec
%s

## Module Patterns
%s

---

## Task: Review PR #%d — %s

Below is the full diff. Review it for:
1. Correctness — does the logic match Cosmos SDK / VPR spec patterns?
2. Security — any injection, overflow, or missing validation?
3. Style — keeper keys, error handling, events, fee math (sdkmath.Int only)
4. Proto — any hand-edited generated files?
5. Tests — adequate coverage?

Be concise. Use bullet points. Flag blocking issues vs nice-to-haves.

`+"```diff\n%s\n```",
		preamble, ctx(cfg, "vpr_spec_summary.md"), ctx(cfg, "verana_modules.md"),
		prNum, title, diff)
}

func CheckSpec(cfg *config.Config, prNum int, title, diff string) string {
	return fmt.Sprintf(`%s

## VPR Spec (AUTHORITATIVE)
%s

---

## Task: Spec Compliance Check for PR #%d — %s

Compare the implementation in this diff against the VPR spec. For each message handler:
1. Are all MUST/MUST NOT requirements met?
2. Are precondition checks complete and in correct order?
3. Is the (Signer) field validated?
4. Are state transitions correct per spec?
5. Are events emitted for every state change?
6. Is fee distribution using exact integer math?

List each spec violation with the spec reference (e.g. [MOD-XX-MSG-Y]) if identifiable.
If compliant, say so explicitly.

`+"```diff\n%s\n```",
		preamble, ctx(cfg, "vpr_spec_summary.md"),
		prNum, title, diff)
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
