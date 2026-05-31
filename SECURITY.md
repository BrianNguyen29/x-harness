# Security

Thank you for helping keep x-harness safe.

## Supported versions

Security fixes target the current `main` branch and the latest released `0.x` version.

## Reporting a vulnerability

Please do **not** open a public issue for suspected vulnerabilities.

Use one of these private channels instead:

- [GitHub private vulnerability reporting](https://github.com/BrianNguyen29/x-harness/security/advisories/new) (preferred)
- Email the maintainer if private reporting is unavailable

When reporting, include:

- A short description of the issue
- Reproduction steps or a minimal proof of concept
- Impact and affected files/commands, if known
- Whether the issue involves secrets or sensitive logs

## Disclosure timeline

| Phase | Action |
|-------|--------|
| Day 0 | Report received and acknowledged |
| Day 1-3 | Initial assessment and severity rating |
| Day 4-14 | Fix development and testing |
| Day 15-30 | Release and public disclosure |

Critical vulnerabilities may follow an accelerated timeline.

## Handling sensitive data

- Do not commit secrets, tokens, private keys, or real credentials.
- Redact command output, traces, completion cards, and examples before sharing.
- Prefer synthetic examples for bug reports.

## Scope

x-harness is a local, file-first Go-native CLI. It should not require a daemon,
database, server, MCP service, or external credential by default.
