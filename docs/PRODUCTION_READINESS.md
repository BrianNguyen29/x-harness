# Production Readiness Checklist

> **Current status**: `1.0.0` — **Stable**
> This project is **production-stable**. The `1.0.0` release completes the RC cycle.

## Version Stance

- `1.0.0` is the first stable release.
- The CLI is feature-complete for the v1.0 contract.
- Consumers can pin to the exact tag or use `latest`.

## Checklist

### Security
- [x] Dependency vulnerability scanning (`npm audit --audit-level=high`, `govulncheck`) runs on every PR/push
- [ ] Secret scanning enabled via GitHub Advanced Security (repo setting) — documented requirement in `.github/workflows/security-audit.yml`
- [x] SLSA provenance generator enabled for tagged releases using `@v2.1.0` with audited semver exception (all other actions remain SHA-pinned) — see `.github/workflows/slsa-provenance.yml`
- [ ] Backup CODEOWNERS owner added for critical paths — **accepted single-maintainer governance risk for 1.0.0**; tracked as post-1.0 follow-up. See `docs/RELEASE_SECURITY.md` and `.github/CODEOWNERS`

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
| Release candidate | `docs/RELEASE_CANDIDATE.md` | Historical (RC cycle complete) |
| CI integration | `docs/CI.md` | Active |

## Deferrals

Items with owner `user` are tracked in `docs/AUDIT_ROADMAP.md` and will be scheduled post-1.0. No fake owners or guessed timelines are assigned.
