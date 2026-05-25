# Episode Packages

An x-harness episode is a local audit bundle for a verify run. It is replay and inspection material, not admission authority.

## Create

```bash
node packages/cli/dist/index.js verify --card completion-card.yaml --strict --episode --bundle
```

This writes:

```text
.x-harness/episodes/<episode-id>/
  manifest.json
  completion-card.yaml
  verdict.json
  trace.jsonl
  failure-attribution.json
  policy-snapshot/
  schema-snapshot/
  hashes.json
  mutation-guard.json
  git.json
  evidence-index.jsonl
  digest.md
  digest.json
  interventions.jsonl
  signatures/
.x-harness/episodes/<episode-id>.redacted.tar.gz
.x-harness/episodes/<episode-id>.raw.tar.gz
```

The redacted bundle is emitted alongside the raw bundle when `--bundle` is used.

## Inspect

```bash
node packages/cli/dist/index.js episode inspect .x-harness/episodes/<episode-id>
node packages/cli/dist/index.js episode inspect .x-harness/episodes/<episode-id>.redacted.tar.gz
```

Inspection validates the manifest schema, manifest hash, file hashes, trace chain, and evidence index.

Withheld episodes include `failure-attribution.json`, a deterministic advisory record that maps the blocking predicate to the failure taxonomy. It has `admission_authority: false` and never accepts completion.

## Chain

```bash
node packages/cli/dist/index.js episode verify-chain --task-id TASK-123
```

Episode chain verification checks each episode independently and confirms `previous_episode_id` links are consistent for the task.

## Boundary

Episode packages record verify outcomes. They never accept completion by themselves. Accepted completion still requires:

```yaml
admission.outcome: success
acceptance_status: accepted
```

Signing is intentionally `unsigned` in the local MVP. Release signing is a later hardening step.
