# CLI Execution Scripts Tests

## Purpose

This directory contains tests validating the operational capabilities of the build, typecheck, clean, and doctor execution processes.

## Scripts

- `verify-adapters.sh` — Integration/snapshot test for adapter matrix, eval, and doctor.
- `verify-cli-docs.sh` — Drift check for generated CLI command reference docs.

## Tasks

- Confirms the doctor command successfully flags missing files or wrong wording.
- Ensures the build outputs clean TS compilation files.
- Assures command line interface stability across all platforms.
- Verifies adapter files exist and meet minimum contract via snapshot and eval/doctor checks.
- Validates CLI command reference docs are current against the canonical registry.
