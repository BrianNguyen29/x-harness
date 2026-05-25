# Experimental Evolution

Evolution is disabled by default and cannot modify source files directly. The local MVP can evaluate the configured budget, check candidate manifests against the constitution, and generate change-request markdown for human review.

The default loop is:

1. `xh evolve evaluate`
2. `xh evolve analyze --run <run-id>`
3. `xh evolve propose --component <component-id>`
4. `xh evolve constitution-check --candidate <candidate>`
5. `xh evolve compare --candidate <candidate>`
6. `xh evolve promote --candidate <candidate>`
7. Human reviews the generated change request.
8. `xh evolve rollback --candidate <candidate>` can generate a rollback request.

`promote` only writes a promotion request. It does not merge, rewrite policies, update schemas, or run git commands.
