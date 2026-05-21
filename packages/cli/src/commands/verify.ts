import { Command } from "commander";
import * as path from "node:path";
import { readYamlOrJson } from "../core/schema.js";
import { runAdmission, acceptanceStatus } from "../core/admission.js";
import { appendTrace } from "../core/trace.js";
import { validate as validateClaim } from "../validators/claim.js";
import { validate as validateEvidence } from "../validators/evidence.js";
import { validate as validateSubagentReturn } from "../validators/subagentReturn.js";

interface VerifyOptions {
  claim?: string;
  evidence?: string;
  subagentReturn?: string;
  tier?: string;
  taskId?: string;
  storyId?: string;
  trace?: boolean;
}

export function verifyCommand(): Command {
  return new Command("verify")
    .description("Run read-only verification against claim/evidence/subagent-return")
    .option("-c, --claim <path>", "Path to claim YAML/JSON")
    .option("-e, --evidence <path>", "Path to evidence YAML/JSON")
    .option("-s, --subagent-return <path>", "Path to subagent return YAML/JSON")
    .option("-t, --tier <tier>", "Tier: light, standard, deep", "standard")
    .option("--task-id <id>", "Task ID")
    .option("--story-id <id>", "Story ID")
    .option("--trace", "Append verify event to trace", false)
    .action(async (opts: VerifyOptions) => {
      const errors: string[] = [];
      const notes: string[] = [];
      let claim: Record<string, unknown> | undefined;
      let evidence: Record<string, unknown> | undefined;
      let subagentReturn: Record<string, unknown> | undefined;

      // Load and validate claim
      if (opts.claim) {
        try {
          const data = await readYamlOrJson(path.resolve(opts.claim));
          const result = await validateClaim(data);
          if (!result.valid) {
            errors.push(`claim validation failed: ${result.errors.join("; ")}`);
          } else {
            claim = data as Record<string, unknown>;
            notes.push("claim schema valid");
          }
        } catch (err) {
          errors.push(`claim load error: ${err instanceof Error ? err.message : String(err)}`);
        }
      }

      // Load and validate evidence
      if (opts.evidence) {
        try {
          const data = await readYamlOrJson(path.resolve(opts.evidence));
          const result = await validateEvidence(data);
          if (!result.valid) {
            errors.push(`evidence validation failed: ${result.errors.join("; ")}`);
          } else {
            evidence = data as Record<string, unknown>;
            notes.push("evidence schema valid");
          }
        } catch (err) {
          errors.push(`evidence load error: ${err instanceof Error ? err.message : String(err)}`);
        }
      }

      // Load and validate subagent return
      if (opts.subagentReturn) {
        try {
          const data = await readYamlOrJson(path.resolve(opts.subagentReturn));
          const result = await validateSubagentReturn(data);
          if (!result.valid) {
            errors.push(`subagent-return validation failed: ${result.errors.join("; ")}`);
          } else {
            subagentReturn = data as Record<string, unknown>;
            notes.push("subagent-return schema valid");
          }
        } catch (err) {
          errors.push(`subagent-return load error: ${err instanceof Error ? err.message : String(err)}`);
        }
      }

      // Admission checks
      const tier = (opts.tier as "light" | "standard" | "deep") ?? "standard";
      const admission = runAdmission({ claim, evidence, subagentReturn, tier, staleGround: false });

      // Merge admission results
      errors.push(...admission.errors);
      notes.push(...admission.notes);

      const outcome = errors.length > 0 ? "failed" : admission.outcome;
      const acceptance = acceptanceStatus(outcome);

      const event = {
        event_id: `VE-${Date.now()}`,
        event_type: "verify_completed",
        task_id: opts.taskId ?? "TASK-UNKNOWN",
        story_id: opts.storyId ?? null,
        tier,
        claim_id: claim?.id as string | undefined ?? null,
        evidence_id: evidence?.id as string | undefined ?? null,
        verifier: "claimgate",
        verifier_mode: "read_only",
        outcome,
        acceptance_status: acceptance,
        blocking_predicate: outcome === "blocked" ? "admission_failed" : null,
        blocked_reason_class: outcome === "blocked" ? "policy_violation" : null,
        next_owner: null,
        next_action: errors.length > 0 ? "resolve validation errors" : null,
        created_at: new Date().toISOString(),
        notes,
        errors,
      };

      if (opts.trace) {
        await appendTrace(event);
      }

      console.log(JSON.stringify(event, null, 2));
      process.exit(errors.length > 0 ? 1 : 0);
    });
}
