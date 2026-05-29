# blocked-strict-mutation-guard

Strict mutation-guard failure fixture.

A completion card cannot faithfully encode a mutation-guard failure because the
mutation guard operates on the **repository worktree** itself (it compares a
before/after snapshot of the filesystem while the conformance suite runs).  A
static fixture therefore cannot represent the dynamic delta detection that
`mutation_guard_verified` performs.

This check is exercised in the Go unit tests instead:

- `TestRunStrictNonGit` in `internal/conformance/conformance_test.go`
- `TestConformanceStrictPasses` in `internal/cli/conformance_test.go`

Those tests verify that:
1. A non-git repository causes `worktree_metadata_valid` to fail.
2. A non-git repository (or any repo where the snapshot mechanism cannot run)
   causes `mutation_guard_verified` to fail.
3. A healthy git repository passes all 15 strict checks including
   `mutation_guard_verified`.
