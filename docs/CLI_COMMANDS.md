# CLI Commands

<!-- generated-by: scripts/generate-cli-docs.mjs -->
<!-- source: internal/cli/commands.json -->

This file is generated from the canonical CLI command registry. Do not edit command tables manually; update `internal/cli/commands.json` and run `npm run cli-metadata:write`.

## Onboarding Commands

The default onboarding path is intentionally narrow: initialize the workspace, run the health check, then verify a completion card.

| Command | Description |
| :-- | :-- |
| `init` | Install harness assets into a workspace |
| `doctor` | Validate workspace health and configuration |
| `verify` | Run read-only verification against a completion card |
| `check` | Alias for verify |

## Full Command Matrix

### stable

| Command | Maturity | Description |
| :-- | :-- | :-- |
| `verify` | stable | Run read-only verification against a completion card |
| `check` | stable | Alias for verify |
| `doctor` | stable | Validate workspace health and configuration |
| `examples` | stable | Verify bundled examples |
| `context` | stable | Show canonical context and runtime contract |
| `benchmark` | stable | Measure latency and verification benchmark behavior |
| `handoff` | stable | Generate structured handoff prompts |
| `prepare` | stable | Alias for handoff readiness |
| `report` | stable | Show trace summary or metrics report |
| `status` | stable | Alias for report |
| `trace` | stable | Append or verify trace events |
| `clean` | stable | Clean generated harness state |
| `reset` | stable | Alias for safe generated-state cleanup |
| `init` | stable | Install harness assets into a workspace |
| `add` | stable | Add claim, evidence, or completion card helpers |
| `recovery` | stable | Generate recovery suggestions |
| `recover` | stable | Alias for recovery suggest |

### beta

| Command | Maturity | Description |
| :-- | :-- | :-- |
| `packet` | beta | Work with claim/evidence packets |
| `profile` | beta | Recommend installation profiles |
| `repair` | beta | Repair managed files from manifest |
| `uninstall` | beta | Uninstall managed files using manifest |
| `start` | beta | Guided onboarding: doctor, examples verify, init wizard, next steps |
| `learn` | beta | Read-only concept tour for beginners |
| `quick` | beta | Read-only next-action recommender for newcomers |
| `run` | beta | Run a built-in workflow recipe |
| `ci` | beta | Run the built-in CI workflow |
| `actions` | beta | List beginner-friendly actions |
| `card` | beta | Generate or verify admission cards |
| `conformance` | beta | Run conformance checks |
| `readiness` | beta | Evaluate readiness levels |
| `release` | beta | Generate or verify release evidence |
| `adapters` | beta | Inspect adapter matrix |
| `scan` | beta | Run static security scan on adapter or skill files |
| `policy` | beta | Show policy enforcement matrix and rule explainers |
| `explain` | beta | Explain a completion card's admission/withheld state |
| `boundary` | beta | Lint/check/explain boundary policy against repo source files |

### experimental

| Command | Maturity | Description |
| :-- | :-- | :-- |
| `intake` | experimental | Evaluate task intake tiering |
| `decision` | experimental | Record or list decision memory records (ADR-lite) |
| `governance` | experimental | Evaluate governance rules |
| `intervention` | experimental | Record governance interventions |
| `prediction` | experimental | Evaluate prediction/checklist claims |
| `components` | experimental | Inspect component registry coverage |
| `evidence` | experimental | Manage evidence corpus entries |
| `episode` | experimental | Create episode packages |
| `attribution` | experimental | Evaluate attribution metadata |
| `permissions` | experimental | Evaluate permission rules |
| `evolve` | experimental | Evaluate evolution candidates |
| `export` | experimental | Export frozen artifacts |
| `import` | experimental | Import frozen artifacts |
| `frozen` | experimental | Inspect frozen manifests |
| `federation` | experimental | Evaluate federation patterns |
| `approval-risk` | experimental | Evaluate approval risk |
| `agent-profile` | experimental | Inspect agent profiles |
| `cost` | experimental | Evaluate cost budget data |
| `contract` | experimental | Run contract oracle checks |

## Maturity Labels

- `stable`: core command; tested and relied on in CI.
- `beta`: functional but may change before 1.0.
- `experimental`: advanced or exploratory; semantics may shift.
- `skeletal`: declared but not yet implemented.
