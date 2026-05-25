# Component Registry

`components/registry.yaml` maps important harness files to owned components. Each component declares paths, owner, stability, edit authority, and validation commands.

## Commands

```bash
xh components validate
xh components list
xh components explain --id admission_policy
xh components changed --files packages/cli/src/core/admission.ts
xh components changed --base main
```

## Doctor Integration

`xh doctor --root .` validates the registry and checks that every protected path in `policies/authority.yaml` is registered to at least one component.

This is a coverage check for harness observability. It does not grant edit authority; `policies/authority.yaml` and admission rules still decide whether work needs approval.
