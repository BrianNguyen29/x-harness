# Golden Examples Verification Tests

## Purpose

This directory contains tests validating the core `x-harness verify` command outcomes against all golden reference scenarios.

## Operations

- The suite loads each golden folder (e.g. `examples/golden/success-light/`).
- Validates that quiet and verbose CLI verify calls return the exact expected exit codes, strings, outcomes, and recovery objects.
- Ensures regression safety across versions.
