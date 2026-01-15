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

## Verana Public Registry (VPR) spec

- Verana implementation is mostly derived from the VPR spec
- VPR spec is available under: [this repo]/../verifiable-trust-vpr-spec/spec.md

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
  - `npx tsx ts-proto/test/scripts/benches/amino/perm/ts.ts`
- Compare TS vs Go bench outputs:
  - `node ts-proto/test/scripts/benches/amino/perm/compare.js`

When debugging signing, always log:
- chain-id
- account_number
- sequence
- the JSON used to build sign bytes
- sign bytes hex

## Go Debugging

- Run the Go Amino bench:
  - `go run ts-proto/test/scripts/benches/amino/perm/go.go`

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

## Infra Access (Devnet/Testnet)

### SSH
Use local SSH keys or agent (not in repo). Default user is `ubuntu`.

- Devnet nodes:
  - node1: `ssh ubuntu@node1.devnet.verana.network`
  - node2: `ssh ubuntu@node2.devnet.verana.network`
  - node3: `ssh ubuntu@node3.devnet.verana.network`

- Testnet nodes:
  - node1: `ssh ubuntu@node1.testnet.verana.network`
  - node2: `ssh ubuntu@node2.testnet.verana.network`
  - node3: `ssh ubuntu@node3.testnet.verana.network`

### Veranad service (VMs)
Systemd unit and cosmovisor paths are standard on devnet/testnet nodes.

- Service:
  - `sudo systemctl status veranad --no-pager`
  - `sudo systemctl restart veranad`
  - Unit file: `/etc/systemd/system/veranad.service` (User=ubuntu, ExecStart=`/home/ubuntu/.verana/cosmovisor/start.sh`)
- Cosmovisor:
  - Home: `/home/ubuntu/.verana/cosmovisor`
  - Current symlink: `/home/ubuntu/.verana/cosmovisor/current`
  - Upgrades live in `/home/ubuntu/.verana/cosmovisor/upgrades/<version>/bin/veranad`
  - Upgrade info file: `/home/ubuntu/.verana/data/upgrade-info.json`
  - `cosmovisor.env` has `DAEMON_ALLOW_DOWNLOAD_BINARIES=false` (binaries must be present)

### Remote RPC (no SSH)
Use `veranad` locally with `--chain-id` and `--node $NODE_RPC` to query testnet/devnet.

- Testnet:
  - `CHAIN_ID=vna-testnet-1`
  - `NODE_RPC=http://node1.testnet.verana.network:26657`
  - Example: `veranad q gov proposals --chain-id $CHAIN_ID --node $NODE_RPC -o json`
- Devnet:
  - `CHAIN_ID=vna-devnet-1`
  - `NODE_RPC=http://node1.devnet.verana.network:26657`
  - Example: `veranad q gov proposals --chain-id $CHAIN_ID --node $NODE_RPC -o json`

### Disk cleanup (common offenders)
Large space usage tends to come from chain data and backups.

- Chain data: `/home/ubuntu/.verana/data`
- Cosmovisor backups: `/home/ubuntu/.verana_backup_*`
- Data backups: `/home/ubuntu/.verana/data-backup-*`
- Logs: `/var/log/journal` (clean via `sudo journalctl --vacuum-size=200M`)

### Kubernetes
Kubeconfig is stored locally (not in repo). Export it before running kubectl.

- `export KUBECONFIG=<PATH_TO_KUBECONFIG>`
- `kubectl get nodes`
- `kubectl get pods -n <NAMESPACE>`
- `kubectl describe pod <POD> -n <NAMESPACE>`

### S3 (OVH)
s3cmd config is stored locally (not in repo). Verify config, then list buckets/paths.

- `s3cmd --config=<PATH_TO_S3CFG> ls`
- `s3cmd --config=<PATH_TO_S3CFG> ls s3://<BUCKET>/<PREFIX>/`
- `s3cmd --config=<PATH_TO_S3CFG> get s3://<BUCKET>/<OBJECT> <LOCAL_PATH>`
