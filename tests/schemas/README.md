# JSON Schemas Validation Tests

## Purpose

This directory contains tests checking the validity and compile actions of core JSON schemas (`completion-card`, `subagent-return`, `verify-event`, and `pgv-advice`).

## Actions

- Compiles schemas using Ajv.
- Validates that JSON Schema structures correctly enforce the expected metadata properties.
- Prevents structural regression of critical metadata properties.
