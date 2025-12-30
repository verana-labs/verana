# Codex Agent Guide (verana-blockchain)

This file captures the expected troubleshooting approach for this repo. Use it as the default behavior for future investigations.

## Mindset

- Be inquisitive: ask clarifying questions early instead of assuming.
- Probe before patching: inspect logs, configs, endpoints, and bytes.
- Validate with tests, then iterate: run the smallest test that proves or disproves a hypothesis.
- Offer a plan and options; execute only after the user approves the plan.

## Investigation Workflow

1) **Reproduce**: run the minimal command that exhibits the issue.
2) **Observe**: capture exact errors, inputs, and outputs (sign bytes, JSON, chain-id, account number, sequence).
3) **Hypothesize**: identify the most likely mismatch (encoding, aminoType, field omissions, chain-id, account/sequence).
4) **Isolate**: create a focused bench or script that reproduces the mismatch.
5) **Fix**: make the smallest change, re-run the reproduction, and confirm resolution.
6) **Document**: update README with steps and outputs.

## Commit Discipline

- Commit every meaningful change set as you go.
- Keep commits focused and message them clearly.
- This enables backtracking, clean rebases, and precise review once the root cause is understood.

## Veranad Introspection (CLI)

Use these commands to validate chain state and debug signing issues:

- Check node status:
  - `veranad status`
- Validate chain-id and RPC/LCD:
  - `veranad config chain-id`
  - `veranad config node`
- Fetch account number/sequence:
  - `veranad q auth account <address>`
- Inspect a transaction by hash:
  - `veranad q tx <tx_hash>`
- Check recent blocks or heights:
  - `veranad status | rg \"latest_block_height\"`
- Run dry-run (gas estimation) without broadcasting:
  - `veranad tx <module> <msg> --from <key> --dry-run`

If LCD is available, the REST endpoint `GET /cosmos/auth/v1beta1/accounts/<address>` should match `veranad q auth account`.

## TypeScript Debugging (ts-proto/test)

- Run a specific journey:
  - `npm run test:create-perm-session`
- Run the Amino sign bench:
  - `npx tsx ts-proto/test/scripts/amino-sign-bench.ts`
- Compare TS vs Go bench outputs:
  - `node ts-proto/test/scripts/compare-amino-bench.js`

When debugging signing, always log:
- chain-id
- account_number
- sequence
- the JSON used to build sign bytes
- sign bytes hex

## Go Debugging

- Run the Go Amino bench:
  - `go run ts-proto/test/scripts/amino-sign-bench-go.go`

Use the legacy Amino codec (`RegisterLegacyAminoCodec`) and `legacytx.StdSignBytes` to match the chain’s sign bytes. If needed, canonicalize JSON (`sdk.MustSortJSON`) to compare with client-side outputs.

## Problem-Solving Example (Amino)

Common root cause: the chain omits zero-value fields in legacy Amino JSON (Go `omitempty`), while the client includes `"0"` values. This changes sign bytes and causes signature verification to fail.

In such cases:
- Add a focused bench to print both “server-style” (zeros omitted) and “client-style” (zeros included) sign bytes.
- Compare sign bytes to confirm the mismatch before changing production code.

## Expectations During Troubleshooting

- Ask for environment details when needed (RPC/LCD endpoints, chain-id, account).
- Confirm whether tests should be run and whether sandbox escalation is allowed.
- Report test results and include command outputs that affect conclusions.
- Prefer small, reversible steps and frequent commits until the fix is proven.
