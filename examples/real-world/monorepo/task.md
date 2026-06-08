# Task: Add Shared Email/Display-Name Validator and Consume in Web App

**Task ID:** TASK-MONOREPO-VALIDATOR-001
**Tier:** standard
**Owner:** charlie
**Accountable:** dana

## Objective

Add a shared email and display-name validator in `packages/shared/src/validators/settings.ts` and consume it from the settings form in `packages/web/app/settings/form.tsx`. Include unit tests in both packages.

## Requirements

- Create `packages/shared/src/validators/settings.ts` — shared validator functions.
  - `validateEmail(email: string): boolean`
  - `validateDisplayName(name: string): boolean`
- Create `packages/shared/src/validators/settings.test.ts` — unit tests for shared validators.
- Update `packages/web/app/settings/form.tsx` — consume shared validators (import from `packages/shared`).
- Create `packages/web/app/settings/form.test.tsx` — unit tests for the form component using shared validators.
- Ensure `npm run typecheck` passes across the monorepo.

## Evidence needed

- Changed files list scoped to `packages/shared/src/validators/*` and `packages/web/app/settings/*`.
- Test output for both packages with `verifies` / `does_not_verify` scope.
- Typecheck output.
- Untested regions note (e.g., no E2E test for the settings flow).
- Prediction with measurable signal and falsification method.
