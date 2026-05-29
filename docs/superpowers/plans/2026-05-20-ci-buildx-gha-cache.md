# CI Buildx + GHA Cache Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the plain `docker build` step in the PR test-harness workflow with a buildx + GitHub Actions cache (`type=gha`) setup so unchanged layers are reused across PR runs, addressing Ariel's review comment on PR #302.

**Architecture:** Single workflow file change in [.github/workflows/testHarness.yml](.github/workflows/testHarness.yml). Add `docker/setup-buildx-action@v3` to enable BuildKit, then replace the `run: docker build ...` step with `docker/build-push-action@v5` using `cache-from: type=gha` / `cache-to: type=gha,mode=max` and `load: true` so the resulting `verana-node:ci` image is available to the host Docker daemon for the downstream `docker run` step at [.github/workflows/testHarness.yml:373-417](.github/workflows/testHarness.yml#L373-L417). Both new steps gated on `env.HARNESS_MODE_EFFECTIVE == 'docker'` so k8s mode is untouched.

**Tech Stack:** GitHub Actions, Docker BuildKit (buildx), `docker/setup-buildx-action@v3`, `docker/build-push-action@v5`, GitHub Actions cache backend (`type=gha`).

**Scope boundary:** This plan addresses *only* the build-step caching change. Dockerfile layer optimization is tracked in [#317](https://github.com/verana-labs/verana/issues/317) and is explicitly out of scope here.

**Verification model:** CI workflows can't be meaningfully unit-tested locally. Verification is two-stage:
1. **Local:** YAML syntax validation via `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/testHarness.yml'))"`.
2. **Remote:** Observe the next PR workflow run on GitHub — first run should cache-miss (uploads cache, expected duration ~ current 24 min); next run on the same branch should cache-hit on unchanged layers and complete faster. Image `verana-node:ci` must be loadable into the host docker daemon (downstream `docker run --rm ... verana-node:ci ...` step must succeed).

---

### Task 1: Replace plain `docker build` with buildx + GHA cache

**Files:**
- Modify: [.github/workflows/testHarness.yml:112-114](.github/workflows/testHarness.yml#L112-L114)

- [ ] **Step 1: Read the target lines to lock in exact whitespace**

Run: `sed -n '105,115p' .github/workflows/testHarness.yml`

Expected output:

```yaml
      - name: Login to Docker Hub
        if: env.HARNESS_MODE_EFFECTIVE == 'k8s'
        uses: docker/login-action@v3
        with:
          username: ${{ env.DH_USERNAME }}
          password: ${{ env.DH_TOKEN }}

      - name: Build canonical image locally
        if: env.HARNESS_MODE_EFFECTIVE == 'docker'
        run: docker build -f docker/veranad.Dockerfile -t verana-node:ci .

```

- [ ] **Step 2: Replace lines 112-114 with the buildx setup + cached build**

Use Edit tool:

`old_string`:
```
      - name: Build canonical image locally
        if: env.HARNESS_MODE_EFFECTIVE == 'docker'
        run: docker build -f docker/veranad.Dockerfile -t verana-node:ci .
```

`new_string`:
```
      - name: Set up Docker Buildx
        if: env.HARNESS_MODE_EFFECTIVE == 'docker'
        uses: docker/setup-buildx-action@v3

      - name: Build canonical image locally
        if: env.HARNESS_MODE_EFFECTIVE == 'docker'
        uses: docker/build-push-action@v5
        with:
          context: .
          file: docker/veranad.Dockerfile
          load: true
          tags: verana-node:ci
          cache-from: type=gha
          cache-to: type=gha,mode=max
```

Note: `load: true` is critical — without it, buildx leaves the image in BuildKit's internal store and the downstream `docker run --rm ... verana-node:ci ...` step at line 373-417 cannot find the image. Single-platform build means `load: true` is supported (multi-platform builds reject `load`).

- [ ] **Step 3: Validate YAML syntax**

Run: `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/testHarness.yml'))"`

Expected: exits 0 with no output.

- [ ] **Step 4: Confirm step ordering and conditionals visually**

Run: `sed -n '105,125p' .github/workflows/testHarness.yml`

Expected: shows Docker Hub login (k8s-only) → Buildx setup (docker-only) → build-push-action (docker-only) → kubectl setup (k8s-only). All `if:` conditionals preserve mode gating; docker mode does buildx + build; k8s mode skips both.

- [ ] **Step 5: Confirm no other consumers of `verana-node:ci` need adjustment**

Run: `grep -n 'verana-node:ci' .github/workflows/testHarness.yml`

Expected: only the original build step (now via build-push-action with `tags: verana-node:ci`) and the downstream `docker run ... verana-node:ci ...` at line ~381. The `load: true` flag in step 2 ensures the tag is loaded into the host daemon, so the downstream step works unchanged.

- [ ] **Step 6: Commit**

```bash
git add .github/workflows/testHarness.yml
git commit -m "$(cat <<'EOF'
ci: cache docker build layers via buildx + GHA cache
EOF
)"
```

(Commit message follows project convention: `{type}({module}): {short description}`, max 2 lines, no Co-Authored-By per CLAUDE.md.)

- [ ] **Step 7: Push to PR branch**

Run: `git push origin ci/local-docker-build-on-pr`

Expected: push succeeds; PR #302 picks up the new commit and GitHub triggers a fresh workflow run.

- [ ] **Step 8: Observe the first cache-populating run**

Open: https://github.com/verana-labs/verana/pull/302
Navigate to the new "2.Test Harness Execution" run. Expected:
- "Set up Docker Buildx" step succeeds.
- "Build canonical image locally" step runs build-push-action; shows `cache-to` writing to GHA cache at the end.
- Downstream "Run test harness in canonical image" step finds `verana-node:ci` and runs successfully.
- Total duration likely similar to today's ~24 min (cache miss on first run).

- [ ] **Step 9: Trigger a second run to confirm cache hit**

After step 8 finishes successfully, push a trivial no-op commit (or re-run the workflow from the GitHub UI) and verify:
- Build-push-action logs show `importing cache manifest from gha` and skips importing unchanged layers from cache (look for `CACHED` lines in BuildKit output).
- Total duration is shorter — primarily depends on which layers in `docker/veranad.Dockerfile` actually changed. Expect meaningful savings on base image / dependency-fetch layers if the Dockerfile orders them before `COPY . .`.

- [ ] **Step 10: Reply to Ariel on PR #302**

Run:

```bash
gh pr comment 302 --repo verana-labs/verana --body "Applied your suggestion in <commit-sha>. Buildx + GHA cache is now wired up. First run after this commit is the cache-priming run; subsequent runs should benefit. Dockerfile layer reordering for better cache utilization will follow in #317 as you proposed."
```

Replace `<commit-sha>` with the actual SHA from step 7.

---

## Self-Review

**Spec coverage:**
- Ariel's suggested code block: ✅ Task 1 step 2 applies it verbatim (modulo whitespace alignment with surrounding steps).
- Preserve k8s mode behavior: ✅ Both new steps are `if: env.HARNESS_MODE_EFFECTIVE == 'docker'`-gated; k8s path is unchanged.
- Downstream consumer (`docker run ... verana-node:ci`): ✅ `load: true` ensures host daemon has the tag; step 5 verifies no other references need touching.
- Verification: ✅ Local YAML parse + remote cache-miss / cache-hit observation across two runs.

**Placeholder scan:** No TBDs, no "add error handling", no "similar to Task N". Every command is concrete. `<commit-sha>` in step 10 is a runtime value, not a placeholder for design.

**Type consistency:** N/A — no code symbols. Action versions are consistent (`docker/setup-buildx-action@v3`, `docker/build-push-action@v5`) and match Ariel's suggestion.
