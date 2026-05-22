# JSON Schemas Validation Tests

## Purpose

This directory contains tests checking the validity and compile actions of core JSON schemas (`completion-card`, `subagent-return`, `verify-event`, and `pgv-advice`).

## Actions

- Compiles schemas using Ajv.
- Ensures all Zod validation models perfectly align with core JSON schemas.
- Prevents structural regression of critical metadata properties.
