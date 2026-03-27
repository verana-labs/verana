# Verana Chain - Claude Code Instructions

## Spec-to-Ship Pipeline

When the user pastes VPR spec text (containing patterns like `[MOD-XX-MSG-Y]`, `MUST abort`, `(Signer)`, `precondition checks`, `execution of the method`), read and follow the orchestrator agent at `.claude-agents/spec-to-ship.md`.

The orchestrator references these sub-agents:
- `.claude-agents/implement-message.md` — 15-phase implementation pipeline
- `.claude-agents/test-suite.md` — 6-phase test generation pipeline
- `.claude-agents/audit.md` — 9-phase spec compliance audit

## Commit Rules

- NEVER include `Co-Authored-By` lines in commits
- Commit messages: max 2 lines, format: `{type}({module}): {short description}`
- Types: `feat`, `fix`, `test`, `refactor`, `chore`
