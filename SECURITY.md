# Security

Thank you for helping keep x-harness safe.

## Supported versions

Security fixes target the current `main` branch and the latest released `0.x` version.

## Reporting a vulnerability

Please do **not** open a public issue for suspected vulnerabilities.

Use one of these private channels instead:

- GitHub private vulnerability reporting, if enabled for the repository.
- Email/DM the maintainer listed on the GitHub repository profile if private reporting is unavailable.

Include:

- A short description of the issue.
- Reproduction steps or a minimal proof of concept.
- Impact and affected files/commands, if known.
- Whether the issue involves secrets or sensitive logs.

## Handling sensitive data

- Do not commit secrets, tokens, private keys, or real credentials.
- Redact command output, traces, completion cards, and examples before sharing.
- Prefer synthetic examples for bug reports.

## Scope

x-harness is a local, file-first TypeScript CLI. It should not require a daemon,
database, server, MCP service, or external credential by default.
