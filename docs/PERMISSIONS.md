# Permissions

The permissions model turns vague safe-command language into a file-first policy. It evaluates command strings and capabilities; it does not execute commands and does not grant admission authority.

## Policy Files

```text
policies/permissions.yaml
schemas/permissions.schema.json
packages/cli/src/core/permissions.ts
packages/cli/src/commands/permissions.ts
```

The policy defines command sets:

- `safe_readonly`: exact read-only commands such as `git status --porcelain`.
- `safe_tests`: test/typecheck command patterns.
- `dangerous`: deny patterns for destructive, release, cloud, and pipe-to-shell commands.

It also defines role/tier profiles for `worker`, `verifier`, and `maintainer`.

## Commands

```bash
node packages/cli/dist/index.js permissions check --role verifier --tier deep --command "npm test"
node packages/cli/dist/index.js permissions explain --role worker --tier deep --capability dependency_install
node packages/cli/dist/index.js permissions test-fixtures
```

`permissions check` exits non-zero when a command or capability is denied or needs an intervention. `permissions explain` reports the same decision but exits zero for inspection.

## Verifier Boundary

The verifier role can read files, read evidence, and run readonly/test command sets. It cannot use source mutation capabilities:

```text
write_source
repair_code
release_publish
destructive_filesystem
```

Those denials are hard denials for the verifier role.

## Intervention Exceptions

Some deep worker capabilities, such as `dependency_install`, require a valid intervention artifact. The intervention must validate against `schemas/intervention.schema.json`, be unexpired, have `decision: allow` or `decision: override`, and cover the requested target, for example:

```yaml
paths:
  - capability:dependency_install
```

Permission decisions remain advisory guardrails. Accepted completion still requires the verify/admission contract.
