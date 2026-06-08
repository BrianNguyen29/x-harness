# Task: Add Next.js Settings Page with Form Validation

**Task ID:** TASK-NEXTJS-SETTINGS-001
**Tier:** standard
**Owner:** charlie
**Accountable:** dana

## Objective

Add a TypeScript settings page at `app/settings/page.tsx` with a reusable form component in `app/settings/form.tsx`. The form must validate user input client-side and include unit tests.

## Requirements

- Create `app/settings/page.tsx` — Next.js page component.
- Create `app/settings/form.tsx` — form component with client-side validation.
- Reject empty display names and invalid email formats.
- Add unit tests in `app/settings/form.test.tsx`.
- Ensure `npm run typecheck` passes.
- Ensure `npm run lint` passes.

## Evidence needed

- Changed files list scoped to `app/settings/*`.
- Test output with `verifies` / `does_not_verify` scope.
- Typecheck and lint output.
- Untested regions note (e.g., no E2E test for the settings flow).
- Prediction with measurable signal and falsification method.
