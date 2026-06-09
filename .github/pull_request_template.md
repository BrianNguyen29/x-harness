## Summary

-

## Change type

- [ ] Documentation only
- [ ] CLI behavior
- [ ] Schema/policy/admission contract
- [ ] Templates/adapters/examples
- [ ] Tests only

## Verification

- [ ] `npm run typecheck`
- [ ] `npm run build`
- [ ] `npm test`
- [ ] `npm run verify`
- [ ] `xh doctor --root .`
- [ ] `xh examples verify`

## Harness invariants

- [ ] Verifier remains read-only
- [ ] PGV remains advisory-only
- [ ] Non-success outcomes remain withheld
- [ ] No daemon/database/server/MCP requirement added
- [ ] Docs/templates/examples updated if contracts changed

## Harness Change Contract

If this PR changes admission policy, schemas, templates, CLI verify, adapters, or skills, attach or reference a completed `templates/HARNESS_CHANGE_CONTRACT.md`.
