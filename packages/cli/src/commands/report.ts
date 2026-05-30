import { Command } from "commander";
import * as path from "node:path";
import fs from "fs-extra";
import { readTrace } from "../core/trace.js";
import { computeMetrics } from "../core/metrics.js";
import { readYamlOrJson } from "../core/schema.js";
import { runAdmission } from "../core/admission.js";
import { sha256File, sha256String } from "../core/hash.js";
import {
  buildEvidenceDigest,
  readEvidenceIndex,
  renderEvidenceDigestMarkdown,
} from "../core/evidence-corpus.js";

interface ReportOptions {
  traceDir?: string;
  json?: boolean;
  metrics?: boolean;
  digest?: boolean;
  card?: string;
  format?: string;
  taskId?: string;
  index?: string;
  write?: boolean;
  outDir?: string;
}

function escapeHtml(input: string): string {
  return input
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

function renderHtmlReport(data: {
  total: number;
  accepted: number;
  withheld: number;
  blocked: number;
  failed: number;
  skipped: number;
  timeout: number;
  error: number;
  unknownEvents: number;
  byOutcome: Record<string, number>;
  latest: unknown;
}): string {
  const {
    total,
    accepted,
    withheld,
    blocked,
    failed,
    skipped,
    timeout,
    error: errorCount,
    unknownEvents,
    byOutcome,
    latest,
  } = data;

  const rows = Object.entries(byOutcome)
    .map(
      ([outcome, count]) =>
        `<tr><td>${escapeHtml(outcome)}</td><td>${count}</td><td>${total > 0 ? Math.round((count / total) * 100) : 0}%</td></tr>`
    )
    .join("\n");

  return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>x-harness Audit Report</title>
<style>
  body { font-family: system-ui, -apple-system, sans-serif; max-width: 900px; margin: 40px auto; padding: 0 20px; color: #333; }
  h1 { border-bottom: 2px solid #333; padding-bottom: 8px; }
  h2 { margin-top: 32px; color: #222; }
  table { border-collapse: collapse; width: 100%; margin-top: 12px; }
  th, td { border: 1px solid #ddd; padding: 8px 12px; text-align: left; }
  th { background: #f5f5f5; }
  .badge { display: inline-block; padding: 4px 10px; border-radius: 4px; font-weight: 600; font-size: 14px; }
  .badge-success { background: #d4edda; color: #155724; }
  .badge-failed { background: #f8d7da; color: #721c24; }
  .badge-blocked { background: #fff3cd; color: #856404; }
  .warning { background: #fff3cd; border-left: 4px solid #ffc107; padding: 12px; margin-top: 16px; }
  .muted { color: #666; font-size: 14px; }
  code { background: #f4f4f4; padding: 2px 6px; border-radius: 3px; }
</style>
</head>
<body>
<h1>x-harness Audit Report</h1>
<p class="muted">Generated at ${escapeHtml(new Date().toISOString())}</p>

<h2>Summary</h2>
<table>
  <tr><th>Metric</th><th>Value</th></tr>
  <tr><td>Total events</td><td>${total}</td></tr>
  <tr><td>Accepted</td><td><span class="badge badge-success">${accepted}</span></td></tr>
  <tr><td>Withheld</td><td><span class="badge badge-failed">${withheld}</span></td></tr>
  <tr><td>Blocked</td><td><span class="badge badge-blocked">${blocked}</span></td></tr>
  <tr><td>Failed</td><td><span class="badge badge-failed">${failed}</span></td></tr>
  <tr><td>Skipped</td><td>${skipped}</td></tr>
  <tr><td>Timeout</td><td>${timeout}</td></tr>
  <tr><td>Error</td><td>${errorCount}</td></tr>
  <tr><td>Unknown / unlinked</td><td>${unknownEvents}</td></tr>
</table>

<h2>Outcome breakdown</h2>
<table>
  <tr><th>Outcome</th><th>Count</th><th>Share</th></tr>
  ${rows || '<tr><td colspan="3">No events</td></tr>'}
</table>

<h2>Latest event</h2>
${latest ? `<pre><code>${escapeHtml(JSON.stringify(latest, null, 2))}</code></pre>` : '<p class="muted">No events recorded.</p>'}

<div class="warning">
  <strong>Denominator warning:</strong> Verify-event success must not be interpreted as task-level success, production reliability, benchmark success, or safety guarantee.
</div>

<footer class="muted" style="margin-top: 40px; border-top: 1px solid #ddd; padding-top: 12px;">
  x-harness CLI-only mode (no daemon / no database / no MCP)
</footer>
</body>
</html>`;
}

function renderBooleanCell(value: boolean): string {
  return value
    ? '<span class="badge badge-success">yes</span>'
    : '<span class="badge badge-failed">no</span>';
}

function renderHtmlMetricsReport(data: {
  metrics: Record<string, unknown>;
  admission: Record<string, unknown>;
}): string {
  const { metrics, admission } = data;
  const vs = (metrics.verification_strength ?? {}) as Record<string, unknown>;
  const sc = (metrics.state_consistency ?? {}) as Record<string, unknown>;
  const ra = (metrics.recovery_ability ?? {}) as Record<string, unknown>;
  const rp = (metrics.replayability ?? {}) as Record<string, unknown>;
  const cost = (metrics.cost ?? {}) as Record<string, unknown>;

  const out = (admission.outcome as string) ?? "unknown";
  const status = (admission.acceptance_status as string) ?? "unknown";

  return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>x-harness Metrics Report</title>
<style>
  body { font-family: system-ui, -apple-system, sans-serif; max-width: 900px; margin: 40px auto; padding: 0 20px; color: #333; }
  h1 { border-bottom: 2px solid #333; padding-bottom: 8px; }
  h2 { margin-top: 32px; color: #222; }
  table { border-collapse: collapse; width: 100%; margin-top: 12px; }
  th, td { border: 1px solid #ddd; padding: 8px 12px; text-align: left; }
  th { background: #f5f5f5; }
  .badge { display: inline-block; padding: 4px 10px; border-radius: 4px; font-weight: 600; font-size: 14px; }
  .badge-success { background: #d4edda; color: #155724; }
  .badge-failed { background: #f8d7da; color: #721c24; }
  .badge-blocked { background: #fff3cd; color: #856404; }
  .warning { background: #fff3cd; border-left: 4px solid #ffc107; padding: 12px; margin-top: 16px; }
  .muted { color: #666; font-size: 14px; }
  code { background: #f4f4f4; padding: 2px 6px; border-radius: 3px; }
  .card { border: 1px solid #ddd; border-radius: 6px; padding: 16px; margin-top: 16px; background: #fafafa; }
  .card h3 { margin-top: 0; font-size: 16px; color: #444; }
  details { margin-top: 16px; border: 1px solid #ddd; border-radius: 6px; padding: 12px; background: #f9f9f9; }
  summary { cursor: pointer; font-weight: 600; }
</style>
</head>
<body>
<h1>x-harness Metrics Report</h1>
<p class="muted">Generated at ${escapeHtml(new Date().toISOString())}</p>

<h2>Admission Outcome</h2>
<table>
  <tr><th>Field</th><th>Value</th></tr>
  <tr><td>Outcome</td><td><span class="badge badge-${out === "success" ? "success" : out === "blocked" ? "blocked" : "failed"}">${escapeHtml(out)}</span></td></tr>
  <tr><td>Acceptance</td><td><span class="badge badge-${status === "accepted" ? "success" : "failed"}">${escapeHtml(status)}</span></td></tr>
</table>

<h2>Verification Strength</h2>
<table>
  <tr><th>Metric</th><th>Value</th></tr>
  <tr><td>Command evidence count</td><td>${Number(vs.command_evidence_count ?? 0)}</td></tr>
  <tr><td>Oracle kinds</td><td>${escapeHtml((vs.oracle_kinds as string[])?.join(", ") || "none")}</td></tr>
  <tr><td>Untested regions</td><td>${Number(vs.untested_regions_count ?? 0)}</td></tr>
  <tr><td>Remaining risks</td><td>${Number(vs.remaining_risks_count ?? 0)}</td></tr>
</table>

<h2>State Consistency</h2>
<table>
  <tr><th>Check</th><th>Status</th></tr>
  <tr><td>Owner present</td><td>${renderBooleanCell(Boolean(sc.owner_present))}</td></tr>
  <tr><td>Accountable present</td><td>${renderBooleanCell(Boolean(sc.accountable_present))}</td></tr>
  <tr><td>Files changed present</td><td>${renderBooleanCell(Boolean(sc.files_changed_present))}</td></tr>
  <tr><td>Admission mapping valid</td><td>${renderBooleanCell(Boolean(sc.admission_mapping_valid))}</td></tr>
</table>

<h2>Recovery Ability</h2>
<table>
  <tr><th>Check</th><th>Status</th></tr>
  <tr><td>Blocked has next action</td><td>${renderBooleanCell(Boolean(ra.blocked_has_next_action))}</td></tr>
  <tr><td>Blocked has owner</td><td>${renderBooleanCell(Boolean(ra.blocked_has_owner))}</td></tr>
  <tr><td>Recovery route present</td><td>${renderBooleanCell(Boolean(ra.recovery_route_present))}</td></tr>
</table>

<h2>Replayability</h2>
<table>
  <tr><th>Check</th><th>Status</th></tr>
  <tr><td>Completion card present</td><td>${renderBooleanCell(Boolean(rp.completion_card_present))}</td></tr>
  <tr><td>Input card hash present</td><td>${renderBooleanCell(Boolean(rp.input_card_hash_present))}</td></tr>
  <tr><td>Policy hash present</td><td>${renderBooleanCell(Boolean(rp.policy_hash_present))}</td></tr>
</table>

<h2>Cost</h2>
<table>
  <tr><th>Metric</th><th>Value</th></tr>
  <tr><td>Default context class</td><td>${escapeHtml(String(cost.default_context_class ?? "unknown"))}</td></tr>
  <tr><td>Verify runtime (ms)</td><td>${Number(cost.verify_runtime_ms ?? 0)}</td></tr>
</table>

<details>
<summary>Raw JSON (for debugging)</summary>
<h3>Metrics</h3>
<pre><code>${escapeHtml(JSON.stringify(metrics, null, 2))}</code></pre>
<h3>Admission</h3>
<pre><code>${escapeHtml(JSON.stringify(admission, null, 2))}</code></pre>
</details>

<div class="warning">
  <strong>Denominator warning:</strong> Verify-event success must not be interpreted as task-level success, production reliability, benchmark success, or safety guarantee.
</div>

<footer class="muted" style="margin-top: 40px; border-top: 1px solid #ddd; padding-top: 12px;">
  x-harness CLI-only mode (no daemon / no database / no MCP)
</footer>
</body>
</html>`;
}

export function reportCommand(): Command {
  return new Command("report")
    .description(
      "Summarize trace events, metrics, or replayable evidence digests"
    )
    .option("--trace-dir <dir>", "Trace directory", ".x-harness/traces")
    .option("--json", "Output JSON instead of Markdown", false)
    .option(
      "--format <format>",
      "Output format: markdown, html, json",
      "markdown"
    )
    .option(
      "--metrics",
      "Compute deterministic local metrics for a completion card",
      false
    )
    .option(
      "--card <path>",
      "Path to completion card for --metrics",
      "completion-card.yaml"
    )
    .option("--digest", "Render an evidence digest from an index", false)
    .option("--task-id <id>", "Task id for --digest")
    .option(
      "--index <path>",
      "Evidence index path for --digest",
      "evidence/index.jsonl"
    )
    .option("--write", "Write digest markdown/json artifacts", false)
    .option("--out-dir <path>", "Digest output directory", "evidence/digest")
    .action(async (opts: ReportOptions) => {
      const format = opts.json ? "json" : (opts.format ?? "markdown");
      if (opts.digest) {
        if (opts.metrics) {
          console.error("Error: --digest cannot be combined with --metrics");
          process.exit(2);
        }
        if (!opts.taskId) {
          console.error("Error: --digest requires --task-id");
          process.exit(2);
        }
        const indexPath = path.resolve(opts.index ?? "evidence/index.jsonl");
        const entries = await readEvidenceIndex(indexPath);
        const digest = buildEvidenceDigest({
          taskId: opts.taskId,
          entries,
        });
        const markdown = renderEvidenceDigestMarkdown(digest);

        if (opts.write) {
          const outDir = path.resolve(opts.outDir ?? "evidence/digest");
          await fs.ensureDir(outDir);
          await fs.writeFile(
            path.join(outDir, `${opts.taskId}.md`),
            markdown,
            "utf-8"
          );
          await fs.writeJson(path.join(outDir, `${opts.taskId}.json`), digest, {
            spaces: 2,
          });
        }

        if (format === "json") {
          console.log(JSON.stringify(digest, null, 2));
        } else {
          console.log(markdown);
        }
        return;
      }

      if (opts.metrics) {
        const cardPath = path.resolve(opts.card ?? "completion-card.yaml");
        if (!(await fs.pathExists(cardPath))) {
          console.error(`Error: Completion card not found at ${cardPath}`);
          process.exit(2);
        }

        const startTime = Date.now();
        const data = await readYamlOrJson(cardPath);
        const card = data as Record<string, unknown>;
        const inputCardHash = sha256String(JSON.stringify(data));
        const policyPath = path.resolve(
          process.cwd(),
          "policies",
          "admission.yaml"
        );
        let policyHash: string | null = null;
        try {
          policyHash = await sha256File(policyPath);
        } catch (err) {
          console.error(
            `warning: could not compute policy hash for ${policyPath}: ${err instanceof Error ? err.message : String(err)}`
          );
        }

        const admissionInput = {
          schema_version: String(card.schema_version ?? ""),
          task_id: String(card.task_id ?? ""),
          tier: (card.tier as "light" | "standard" | "deep") ?? "standard",
          owner: String(card.owner ?? ""),
          accountable: String(card.accountable ?? ""),
          claim: card.claim as Record<string, unknown>,
          verification: card.verification as Record<string, unknown>,
          admission: card.admission as Record<string, unknown>,
          acceptance_status: card.acceptance_status as "accepted" | "withheld",
          handoff: card.handoff as Record<string, unknown>,
          evidence: card.evidence as Record<string, unknown> | undefined,
          state: card.state as Record<string, unknown> | undefined,
          governance: card.governance as Record<string, unknown> | undefined,
          intake: card.intake as Record<string, unknown> | undefined,
          context_acknowledged:
            typeof card.context_acknowledged === "boolean"
              ? card.context_acknowledged
              : undefined,
          done_checklist: card.done_checklist as
            | Record<string, unknown>
            | undefined,
          prediction: card.prediction as Record<string, unknown> | undefined,
          pgv_advice: card.pgv_advice as Record<string, unknown> | undefined,
          isCardMode: true,
          staleGround: false,
        };

        const admission = runAdmission(admissionInput);
        const verifyRuntimeMs = Date.now() - startTime;

        const metrics = computeMetrics(admissionInput, {
          inputCardHash,
          policyHash,
          verifyRuntimeMs,
        });

        const report = {
          card_id: card.id ?? null,
          task_id: card.task_id ?? null,
          tier: card.tier ?? "standard",
          metrics,
          admission: {
            outcome: admission.outcome,
            acceptance_status: admission.acceptance_status,
            errors: admission.errors,
            notes: admission.notes,
          },
          verify_event_accounting: {
            cards_analyzed: 1,
            note: "Single-card analysis; aggregate task denominator is not inferred.",
          },
          task_lifecycle_accounting: {
            admitted: admission.acceptance_status === "accepted" ? 1 : 0,
            withheld: admission.acceptance_status === "withheld" ? 1 : 0,
            note: "Lifecycle state reflects only the analyzed completion card.",
          },
          admission_accounting: {
            accepted: admission.acceptance_status === "accepted" ? 1 : 0,
            total_analyzed: 1,
            note: "Admission requires outcome=success; non-success outcomes are withheld.",
          },
          withheld_accounting: {
            failed: admission.outcome === "failed" ? 1 : 0,
            blocked: admission.outcome === "blocked" ? 1 : 0,
            skipped: admission.outcome === "skipped" ? 1 : 0,
            timeout: admission.outcome === "timeout" ? 1 : 0,
            error: admission.outcome === "error" ? 1 : 0,
            note: "Withheld breakdown reflects only the analyzed completion card.",
          },
          unknown_or_unlinked_events: {
            count: 0,
            note: "Not applicable for single-card metrics analysis.",
          },
          denominator_warning:
            "Verify-event success must not be interpreted as task-level success, production reliability, benchmark success, or safety guarantee.",
        };

        if (format === "json") {
          console.log(JSON.stringify(report, null, 2));
        } else if (format === "html") {
          console.log(
            renderHtmlMetricsReport({
              metrics: metrics as unknown as Record<string, unknown>,
              admission: report.admission as unknown as Record<string, unknown>,
            })
          );
        } else {
          console.log("# x-harness Metrics Report");
          console.log("");
          console.log("## Verification strength");
          console.log(
            `- command_evidence_count: ${metrics.verification_strength.command_evidence_count}`
          );
          console.log(
            `- oracle_kinds: ${metrics.verification_strength.oracle_kinds.join(", ") || "none"}`
          );
          console.log(
            `- untested_regions_count: ${metrics.verification_strength.untested_regions_count}`
          );
          console.log(
            `- remaining_risks_count: ${metrics.verification_strength.remaining_risks_count}`
          );
          console.log("");
          console.log("## State consistency");
          console.log(
            `- owner_present: ${metrics.state_consistency.owner_present}`
          );
          console.log(
            `- accountable_present: ${metrics.state_consistency.accountable_present}`
          );
          console.log(
            `- files_changed_present: ${metrics.state_consistency.files_changed_present}`
          );
          console.log(
            `- admission_mapping_valid: ${metrics.state_consistency.admission_mapping_valid}`
          );
          console.log("");
          console.log("## Recovery ability");
          console.log(
            `- blocked_has_next_action: ${metrics.recovery_ability.blocked_has_next_action}`
          );
          console.log(
            `- blocked_has_owner: ${metrics.recovery_ability.blocked_has_owner}`
          );
          console.log(
            `- recovery_route_present: ${metrics.recovery_ability.recovery_route_present}`
          );
          console.log("");
          console.log("## Replayability");
          console.log(
            `- completion_card_present: ${metrics.replayability.completion_card_present}`
          );
          console.log(
            `- input_card_hash_present: ${metrics.replayability.input_card_hash_present}`
          );
          console.log(
            `- policy_hash_present: ${metrics.replayability.policy_hash_present}`
          );
          console.log("");
          console.log("## Cost");
          console.log(
            `- default_context_class: ${metrics.cost.default_context_class}`
          );
          console.log(`- verify_runtime_ms: ${metrics.cost.verify_runtime_ms}`);
          console.log("");
          console.log("## Rate metrics");
          console.log(
            `- verify_event_success_rate: ${metrics.verify_event_success_rate.numerator}/${metrics.verify_event_success_rate.denominator} ${metrics.verify_event_success_rate.unit} (not_task_level)`
          );
          console.log(
            `- task_completion_coverage: ${metrics.task_completion_coverage.status} (${metrics.task_completion_coverage.reason})`
          );
          console.log(
            `- withheld_rate: ${metrics.withheld_rate.numerator}/${metrics.withheld_rate.denominator} ${metrics.withheld_rate.unit} (not_task_level)`
          );
          console.log("");
          console.log("## Verify event accounting");
          console.log("- cards_analyzed: 1");
          console.log(
            "> Single-card analysis; aggregate task denominator is not inferred."
          );
          console.log("");
          console.log("## Task lifecycle accounting");
          console.log(
            `- admitted: ${admission.acceptance_status === "accepted" ? 1 : 0}/1`
          );
          console.log(
            `- withheld: ${admission.acceptance_status === "withheld" ? 1 : 0}/1`
          );
          console.log(
            "> Lifecycle state reflects only the analyzed completion card."
          );
          console.log("");
          console.log("## Admission accounting");
          console.log(
            `- accepted: ${admission.acceptance_status === "accepted" ? 1 : 0}/1`
          );
          console.log(
            "> Admission requires outcome=success; non-success outcomes are withheld."
          );
          console.log("");
          console.log("## Withheld accounting");
          if (admission.acceptance_status === "accepted") {
            console.log("None.");
          } else {
            if (admission.outcome === "failed") console.log("- failed: 1/1");
            if (admission.outcome === "blocked") console.log("- blocked: 1/1");
            if (admission.outcome === "skipped") console.log("- skipped: 1/1");
            if (admission.outcome === "timeout") console.log("- timeout: 1/1");
            if (admission.outcome === "error") console.log("- error: 1/1");
            console.log(
              "> Withheld breakdown reflects only the analyzed completion card."
            );
          }
          console.log("");
          console.log("## Unknown or unlinked events");
          console.log("Not applicable for single-card metrics analysis.");
          console.log("");
          console.log("## Denominator warning");
          console.log(
            "> Verify-event success must not be interpreted as task-level success, production reliability, benchmark success, or safety guarantee."
          );
        }
        return;
      }

      const events = await readTrace(
        path.resolve(opts.traceDir ?? ".x-harness/traces")
      );

      const total = events.length;
      const accepted = events.filter(
        (e) => e.acceptance_status === "accepted"
      ).length;
      const withheld = events.filter(
        (e) => e.acceptance_status === "withheld"
      ).length;
      const blocked = events.filter((e) => e.outcome === "blocked").length;
      const failed = events.filter((e) => e.outcome === "failed").length;
      const skipped = events.filter((e) => e.outcome === "skipped").length;
      const timeout = events.filter((e) => e.outcome === "timeout").length;
      const error = events.filter((e) => e.outcome === "error").length;
      const unknownEvents = events.filter(
        (e) =>
          !e.outcome ||
          !e.acceptance_status ||
          ![
            "success",
            "failed",
            "blocked",
            "skipped",
            "timeout",
            "error",
          ].includes(String(e.outcome))
      ).length;
      const byOutcome: Record<string, number> = {};
      for (const e of events) {
        const o = String(e.outcome ?? "unknown");
        byOutcome[o] = (byOutcome[o] ?? 0) + 1;
      }

      if (format === "json") {
        const report = {
          total_events: total,
          accepted,
          withheld,
          by_outcome: byOutcome,
          verify_event_accounting: {
            total_trace_events: total,
            note: "Counts are based only on traced verify events; total task denominator may differ.",
          },
          task_lifecycle_accounting: {
            admitted: accepted,
            withheld,
            note: "Lifecycle accounting covers only events present in the trace log.",
          },
          admission_accounting: {
            accepted,
            total_trace_events: total,
            note: "Admission requires outcome=success; non-success outcomes are withheld.",
          },
          withheld_accounting: {
            failed,
            blocked,
            skipped,
            timeout,
            error,
            note: "Withheld breakdown is only as complete as the trace event set.",
          },
          unknown_or_unlinked_events: {
            count: unknownEvents,
            note: "Events with missing or unrecognized outcome/acceptance_status.",
          },
          latest: events.length > 0 ? events[events.length - 1] : null,
          verify_event_success_rate: {
            numerator: accepted,
            denominator: total,
            unit: "verify_event",
            not_task_level: true,
          },
          task_completion_coverage: {
            status: "not_computable",
            reason: "missing_aligned_task_denominator",
          },
          withheld_rate: {
            numerator: withheld,
            denominator: total,
            unit: "verify_event",
            not_task_level: true,
          },
        };
        console.log(JSON.stringify(report, null, 2));
        return;
      }

      if (format === "html") {
        console.log(
          renderHtmlReport({
            total,
            accepted,
            withheld,
            blocked,
            failed,
            skipped,
            timeout,
            error,
            unknownEvents,
            byOutcome,
            latest: events.length > 0 ? events[events.length - 1] : null,
          })
        );
        return;
      }

      // Markdown output
      console.log("# x-harness Report");
      console.log("");
      console.log("## Installed mode");
      console.log("CLI-only (no daemon / no database / no MCP)");
      console.log("");
      console.log("## Templates");
      console.log("- COMPLETION_CARD.md");
      console.log("- SUBAGENT_TASK_light.md");
      console.log("- SUBAGENT_TASK_standard.md");
      console.log("- SUBAGENT_TASK_deep.md");
      console.log("- VERIFY_REPORT.md");
      console.log("");
      console.log("## Completion card");
      if (total === 0) {
        console.log("No completion cards found in trace.");
      } else {
        console.log(`${total} card(s) in trace.`);
      }
      console.log("");
      console.log("## Verify event accounting");
      if (total === 0) {
        console.log("No verify events recorded.");
        console.log("Denominator: NOT_COMPUTABLE (no events)");
      } else {
        console.log(`- total_trace_events: ${total}`);
        for (const [outcome, count] of Object.entries(byOutcome)) {
          console.log(`- ${outcome}: ${count}/${total}`);
        }
        console.log(
          "> Counts are based only on traced verify events; total task denominator may differ."
        );
      }
      console.log("");
      console.log("## Task lifecycle accounting");
      if (total === 0) {
        console.log("No lifecycle data available.");
      } else {
        console.log(`- admitted: ${accepted}/${total}`);
        console.log(`- withheld: ${withheld}/${total}`);
        console.log(
          "> Lifecycle accounting covers only events present in the trace log."
        );
      }
      console.log("");
      console.log("## Admission accounting");
      if (total === 0) {
        console.log("No admission data available.");
      } else {
        console.log(`- accepted: ${accepted}/${total}`);
        console.log(
          "> Admission requires outcome=success; non-success outcomes are withheld."
        );
      }
      console.log("");
      console.log("## Withheld accounting");
      if (total === 0) {
        console.log("No withheld data available.");
      } else if (withheld === 0) {
        console.log("None.");
      } else {
        if (failed > 0) console.log(`- failed: ${failed}/${total}`);
        if (blocked > 0) console.log(`- blocked: ${blocked}/${total}`);
        if (skipped > 0) console.log(`- skipped: ${skipped}/${total}`);
        if (timeout > 0) console.log(`- timeout: ${timeout}/${total}`);
        if (error > 0) console.log(`- error: ${error}/${total}`);
        console.log(
          "> Withheld breakdown is only as complete as the trace event set."
        );
      }
      console.log("");
      console.log("## Unknown or unlinked events");
      if (unknownEvents === 0) {
        console.log("None.");
      } else {
        console.log(
          `${unknownEvents}/${total} events with missing or unrecognized outcome/acceptance_status.`
        );
      }
      console.log("");
      console.log("## Rate metrics");
      if (total === 0) {
        console.log("No rate metrics available (no events).");
      } else {
        console.log(
          `- verify_event_success_rate: ${accepted}/${total} verify_event (not_task_level)`
        );
        console.log(
          "- task_completion_coverage: not_computable (missing_aligned_task_denominator)"
        );
        console.log(
          `- withheld_rate: ${withheld}/${total} verify_event (not_task_level)`
        );
      }
      console.log("");
      console.log("## Denominator warning");
      console.log(
        "> Verify-event success must not be interpreted as task-level success without denominator review."
      );
      if (total === 0) {
        console.log("Denominator: NOT_COMPUTABLE (no events)");
      } else {
        console.log(`- accepted: ${accepted}/${total} cards`);
        console.log(`- blocked: ${blocked}/${total} cards`);
        console.log(`- withheld: ${withheld}/${total} cards`);
      }
    });
}
