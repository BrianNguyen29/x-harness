import { Command } from "commander";
import * as path from "node:path";
import fs from "fs-extra";
import { readTrace } from "../core/trace.js";
import { computeMetrics } from "../core/metrics.js";
import { readYamlOrJson } from "../core/schema.js";
import { runAdmission } from "../core/admission.js";
import { sha256File, sha256String } from "../core/hash.js";

interface ReportOptions {
  traceDir?: string;
  json?: boolean;
  metrics?: boolean;
  card?: string;
}

export function reportCommand(): Command {
  return new Command("report")
    .description("Summarize trace events or compute metrics for a completion card")
    .option("--trace-dir <dir>", "Trace directory", ".x-harness/traces")
    .option("--json", "Output JSON instead of Markdown", false)
    .option("--metrics", "Compute deterministic local metrics for a completion card", false)
    .option("--card <path>", "Path to completion card for --metrics", "completion-card.yaml")
    .action(async (opts: ReportOptions) => {
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
        const policyPath = path.resolve(process.cwd(), "policies", "admission.yaml");
        const policyHash = await sha256File(policyPath);

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
          denominator_warning: "Verify-event success must not be interpreted as task-level success, production reliability, benchmark success, or safety guarantee.",
        };

        if (opts.json) {
          console.log(JSON.stringify(report, null, 2));
        } else {
          console.log("# x-harness Metrics Report");
          console.log("");
          console.log("## Verification strength");
          console.log(`- command_evidence_count: ${metrics.verification_strength.command_evidence_count}`);
          console.log(`- oracle_kinds: ${metrics.verification_strength.oracle_kinds.join(", ") || "none"}`);
          console.log(`- untested_regions_count: ${metrics.verification_strength.untested_regions_count}`);
          console.log(`- remaining_risks_count: ${metrics.verification_strength.remaining_risks_count}`);
          console.log("");
          console.log("## State consistency");
          console.log(`- owner_present: ${metrics.state_consistency.owner_present}`);
          console.log(`- accountable_present: ${metrics.state_consistency.accountable_present}`);
          console.log(`- files_changed_present: ${metrics.state_consistency.files_changed_present}`);
          console.log(`- admission_mapping_valid: ${metrics.state_consistency.admission_mapping_valid}`);
          console.log("");
          console.log("## Recovery ability");
          console.log(`- blocked_has_next_action: ${metrics.recovery_ability.blocked_has_next_action}`);
          console.log(`- blocked_has_owner: ${metrics.recovery_ability.blocked_has_owner}`);
          console.log(`- recovery_route_present: ${metrics.recovery_ability.recovery_route_present}`);
          console.log("");
          console.log("## Replayability");
          console.log(`- completion_card_present: ${metrics.replayability.completion_card_present}`);
          console.log(`- input_card_hash_present: ${metrics.replayability.input_card_hash_present}`);
          console.log(`- policy_hash_present: ${metrics.replayability.policy_hash_present}`);
          console.log("");
          console.log("## Cost");
          console.log(`- default_context_class: ${metrics.cost.default_context_class}`);
          console.log(`- verify_runtime_ms: ${metrics.cost.verify_runtime_ms}`);
          console.log("");
          console.log("## Denominator warning");
          console.log("> Verify-event success must not be interpreted as task-level success, production reliability, benchmark success, or safety guarantee."
          );
        }
        return;
      }

      const events = await readTrace(path.resolve(opts.traceDir ?? ".x-harness/traces"));

      const total = events.length;
      const accepted = events.filter((e) => e.acceptance_status === "accepted").length;
      const withheld = events.filter((e) => e.acceptance_status === "withheld").length;
      const blocked = events.filter((e) => e.outcome === "blocked").length;
      const byOutcome: Record<string, number> = {};
      for (const e of events) {
        const o = String(e.outcome ?? "unknown");
        byOutcome[o] = (byOutcome[o] ?? 0) + 1;
      }

      if (opts.json) {
        const report = {
          total_events: total,
          accepted,
          withheld,
          by_outcome: byOutcome,
          latest: events.length > 0 ? events[events.length - 1] : null,
        };
        console.log(JSON.stringify(report, null, 2));
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
      console.log("## Verification summary");
      if (total === 0) {
        console.log("No verification events recorded.");
      } else {
        for (const [outcome, count] of Object.entries(byOutcome)) {
          console.log(`- ${outcome}: ${count}/${total}`);
        }
      }
      console.log("");
      console.log("## Blocked items");
      if (blocked === 0) {
        console.log("None.");
      } else {
        console.log(`${blocked}/${total} blocked`);
      }
      console.log("");
      console.log("## Denominator warning");
      console.log("> Verify-event success must not be interpreted as task-level success without denominator review."
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
