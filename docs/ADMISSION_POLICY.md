# Admission Policy

ClaimGate admission is fail-closed.

Success requires claim or completion card, evidence, owner, accountable, mapped success criteria, evidence floor, no unresolved blocker, stale ground resolved, no active recovery, no active veto, verify invoked, and read-only verifier.

Reject success if `fix_status` is `partial` or `not_fixed`, if verification failed/skipped/blocked, if evidence is missing or weak, if stale ground remains, if active recovery remains, if unresolved questions remain, or if timeout/error occurred.
