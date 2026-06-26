# Production Readiness Checklist

> **Current status**: `0.99.0-rc7` — **Release Candidate / pre-stable**  
> This project is **not yet production-stable**. It will remain RC until:
> - All P1/P2 hardening items are completed
> - A stable `1.0.0` tag is cut and the verify gate passes
> - Branch protection and required status checks are fully enforced

## Version Stance

- `0.99.0-rc7` is a release candidate.
- Do **not** market or deploy as production-stable.
- Consumers should pin to the exact RC tag or commit SHA.
- The stable release will be tagged only after this checklist is completed and the full verify gate passes.

## Checklist

### Security
- [x] Dependency vulnerability scanning (`npm audit --audit-level=high`, `govulncheck`) runs on every PR/push
- [ ] Secret scanning enabled via GitHub Advanced Security (repo setting) — documented requirement in `.github/workflows/security-audit.yml`
- [ ] SLSA provenance generator enabled for tagged releases using `@v2.1.0` with audited semver exception (all other actions remain SHA-pinned) — see `.github/workflows/slsa-provenance.yml`
- [ ] Backup CODEOWNERS owner added for critical paths — documented requirement in `.github/CODEOWNERS`

### CI / Quality
- [ ] Scanner report-only → blocking/waiver roadmap approved and scheduled (owner: user)
- [ ] Approval-risk scoring enabled and calibrated (`policies/approval-risk.yaml` currently `enabled: false`) (owner: user)
- [ ] Coverage thresholds defined and enforced in CI (currently report-only; owner: user)
- [ ] CI parallelization tuned (no unnecessary job delays; owner: user)
- [ ] Verify pipeline refactor completed (Go-native primary, TS compatibility secondary) — Go is already primary; TS compatibility remains as parity gate
- [ ] Go/TS drift controls automated and blocking — schema/policy sync is blocking in supplemental gate; admission engine drift test deferred (owner: user)

### Governance
- [ ] Policy change audit log implemented (owner: user)
- [ ] Context floor enforcement runtime-enabled in at least one profile (owner: user)
- [ ] Frozen manifest integrity gate implemented (owner: user)

## Roadmap References

| Topic | Document | Status |
|-------|----------|--------|
| Audit roadmap | `docs/AUDIT_ROADMAP.md` | P1 complete; P2/P3 deferred |
| Release security | `docs/RELEASE_SECURITY.md` | Active |
| Threat model | `docs/THREAT_MODEL.md` | Active |
| Release candidate | `docs/RELEASE_CANDIDATE.md` | Active |
| CI integration | `docs/CI.md` | Active |

## Deferrals

Items with owner `user` are tracked in `docs/AUDIT_ROADMAP.md` and will be scheduled post-RC. No fake owners or guessed timelines are assigned.
