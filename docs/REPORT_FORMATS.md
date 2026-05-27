# Report Formats

x-harness supports multiple output formats for the `report` command.

Source checkouts can use the Go binary (`./x-harness`) or the TypeScript compatibility entrypoint (`node packages/cli/dist/index.js`). The npm wrapper currently defaults to TypeScript compatibility and can opt into a packaged Go binary with `X_HARNESS_GO=1`.

## Supported formats

| Format | CLI flag | Use case |
|--------|----------|----------|
| Markdown (default) | `--format markdown` or no flag | Human-readable terminal output |
| JSON | `--json` or `--format json` | Machine parsing, CI integration |
| HTML | `--format html` | Static single-file audit report |

## Markdown (default)

```bash
./x-harness report --trace-dir .x-harness/traces
# compatibility: node packages/cli/dist/index.js report --trace-dir .x-harness/traces
```

Produces a structured Markdown report with sections for:
- Verify event accounting
- Task lifecycle accounting
- Admission accounting
- Withheld breakdown
- Denominator warning

## JSON

```bash
./x-harness report --trace-dir .x-harness/traces --json
# compatibility: node packages/cli/dist/index.js report --trace-dir .x-harness/traces --json
```

Produces a JSON object with exact counts, outcome breakdowns, and the latest event.

## HTML

```bash
node packages/cli/dist/index.js report --trace-dir .x-harness/traces --format html
```

HTML rendering is currently provided by the TypeScript compatibility CLI. The Go CLI supports Markdown and JSON report rendering.

Produces a single self-contained HTML file with:
- Inline CSS (no external assets)
- Escaped user-provided fields (safe from XSS)
- Summary tables and outcome badges
- Denominator warning

### HTML safety

All dynamic content is HTML-escaped before rendering. The output contains no external scripts, stylesheets, or network requests.

```bash
# Write HTML to a file
node packages/cli/dist/index.js report --trace-dir .x-harness/traces --format html > audit-report.html
```

## Metrics report

```bash
./x-harness report --metrics --card completion-card.yaml --json
# HTML compatibility: node packages/cli/dist/index.js report --metrics --card completion-card.yaml --format html
```

When `--metrics` is combined with `--format html`, the report renders a self-contained HTML page with structured sections for:

- Admission outcome
- Verification strength
- State consistency
- Recovery ability
- Replayability
- Cost

The raw metrics/admission JSON remains available in a collapsed `Raw JSON` details block for debugging.

## Non-goals

- No PDF export
- No live dashboard server
- No email/Slack notification integration
- No external CDN dependencies in HTML output
