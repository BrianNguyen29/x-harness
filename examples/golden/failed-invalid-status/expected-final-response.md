# Expected Final Response

The completion claim was withheld by the verify gate due to a canonical contradiction.

## Result

- **Outcome:** failed
- **Acceptance:** withheld
- **Tier:** light

## Blocking Predicate

Invalid status combination: `acceptance_status` is `accepted` but `admission.outcome` is `failed`.

## Handoff

- **Next action:** Resolve the status contradiction. Either set `admission.outcome: success` (if the task truly passed) or set `acceptance_status: withheld` (if the task failed).
- **Owner:** alice

## Notes

The completion card schema enforces that `acceptance_status: accepted` can only appear when `admission.outcome: success`. Agents must not self-admit completion.
