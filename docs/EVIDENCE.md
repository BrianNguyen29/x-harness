# Evidence Corpus

x-harness evidence corpus is a file-first observability layer for replaying what an agent attached, captured, or referenced. It does not replace the verify gate and does not grant admission authority.

## Layers

```text
evidence/
  README.md
  index.jsonl
  raw/<task-id>/
  redacted/<task-id>/
  digest/<task-id>.md
  digest/<task-id>.json
```

- `raw` contains original evidence artifacts.
- `redacted` contains text artifacts with known secret/token patterns removed.
- `index.jsonl` contains deterministic evidence records with hashes and metadata.
- `digest` is replayed from `index.jsonl`; it is a report, not admission evidence.

## Commands

Build an index from an episode directory:

```bash
node packages/cli/dist/index.js evidence index --episode .x-harness/episodes/TASK-123 --task-id TASK-123 --redact
```

Build an index from a completion card:

```bash
node packages/cli/dist/index.js evidence index --card completion-card.yaml
```

Render a digest from the index:

```bash
node packages/cli/dist/index.js report --digest --task-id TASK-123
```

Write digest artifacts:

```bash
node packages/cli/dist/index.js report --digest --task-id TASK-123 --write
```

## Strict Provenance

`verify --strict` requires stronger provenance for `standard` and `deep` completion cards. Every `evidence.command_evidence[]` entry must include `command`, `exit_code`, `runner`, and `started_at`. Every `evidence.verification_artifacts[]` entry, when present, must include `command`, `exit_code`, `runner`, and `started_at`.

This rule is intentionally limited to strict mode so existing light/manual flows remain usable while CI and release paths can fail closed on anonymous or unauditable evidence.

## Redaction

The built-in redactor covers common secret shapes:

- API key assignments
- Bearer tokens
- JWTs
- Private keys
- Password assignments
- Database/queue connection strings
- GitHub tokens
- npm tokens

Redaction is deterministic and local. Optional external secret scanners are not required for this CLI path.

## Admission Boundary

Digest output is advisory observability. It can summarize indexed evidence, but accepted completion still requires:

```yaml
admission.outcome: success
acceptance_status: accepted
```

The digest cannot admit completion and must not be used as a substitute for raw command evidence or verification artifacts.
