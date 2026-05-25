# CI Strict Verify Fixture

This fixture is used by the GitHub Actions verify workflow to exercise `xh verify --strict` against a representative accepted completion card.

It is separate from the golden corpus so the CI read-only guard can evolve without changing admission behavior snapshots.
