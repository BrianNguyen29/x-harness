# Verify Gate

The verify gate is read-only. It may inspect handoff templates, sub-agent returns, completion cards, claim packets, evidence packets, diffs, command outputs, and trace events. It must not edit source files or repair work while verifying.

Activation may occur when a specialist returns `fix_status: fixed`, returns `verification.status: passed`, creates claim/evidence, or when the orchestrator is about to declare work done.

Outcomes:

```yaml
success: accepted
failed: withheld
blocked: withheld
skipped: withheld
timeout: withheld
error: withheld
```

Blocked verification must identify blocking predicate, blocked reason class, next owner, and next action.

## Evidence scope verification

The verify gate evaluates evidence scope according to tier:

- `light`: `verification_artifacts`, `verifies`, `does_not_verify`, `untested_regions` are optional.
- `standard`: The above are recommended; missing scope produces a warning, not a hard failure.
- `deep`: The above are required. Missing `remaining_risks`, `state.read_set`, or `state.write_set` blocks admission.

Allowed artifact kinds:

```txt
typecheck, unit_test, integration_test, e2e_test, lint, build,
static_analysis, security_scan, fuzz, performance_profile,
manual_review, model_critique, custom
```

Allowed artifact statuses:

```txt
passed, failed, blocked, skipped, timeout, error
```

## Read-only mutation guard (Phase 3.1)

The verify command supports an opt-in `--mutation-guard` flag. `--strict` enables the same guard automatically. When enabled, verify snapshots the repository state before and after its work and compares the delta. This does not require a clean worktree.

Behavior:

- In git workspaces, the guard uses `git status --porcelain=v1 -z --untracked-files=all` and content hashes for dirty/untracked files.
- In non-git workspaces, the guard falls back to a directory snapshot with bounded-concurrency content hashing.
- Hashing concurrency defaults to `16`. Set `X_HARNESS_MUTATION_GUARD_HASH_CONCURRENCY` for high file-count dirty/untracked worktrees; values below `1` fall back to the default and values above `64` are capped at `64`.
- Non-git fallback always ignores `.git/`, `node_modules/`, and `.x-harness/`. It also reads root `.gitignore` entries and optional `policies/mutation-guard.yaml` `fallback_ignore` rules.
- If no baseline can be established, strict verification is fail-closed because read-only verification cannot be proven.
- Writes under `.x-harness/` (including trace output in `.x-harness/traces/`) are allowlisted and do not trigger the guard.
- If any unexpected file change is detected, verify produces a `blocked` outcome with `blocking_predicate: verifier_not_read_only` and recovery routed to `admission-verifier`.
- Without `--mutation-guard`, existing behavior is unchanged.

Latency can be measured with:

```bash
./x-harness benchmark --filter mutation-guard --json
# compatibility: node packages/cli/dist/index.js benchmark --filter mutation-guard --json
```

By default this measures git and non-git fallback snapshots with `100`, `1000`, and `5000` files across concurrency levels `1`, `4`, `16`, and `64`. Use `--mutation-files` and `--mutation-concurrency` to override those lists.

## Governance verification

For `deep` tasks with `governance.requires_human_approval: true`, verify checks that `approval_status` is `approved`. Pending or missing approval blocks admission.
