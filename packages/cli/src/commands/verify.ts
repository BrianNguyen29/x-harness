import { Command } from "commander";
import * as path from "node:path";
import fs from "fs-extra";
import { readYamlOrJson } from "../core/schema.js";
import { runAdmission, acceptanceStatus } from "../core/admission.js";
import { appendTrace } from "../core/trace.js";
import { sha256File, sha256String } from "../core/hash.js";
import { suggestRecovery, getRecoveryRoute } from "../core/recovery.js";
import { runMutationGuard } from "../core/mutation-guard.js";
import { validate as validateClaim } from "../validators/claim.js";
import { validate as validateEvidence } from "../validators/evidence.js";
import { validate as validateSubagentReturn } from "../validators/subagentReturn.js";
import { validate as validateCompletionCard } from "../validators/completionCard.js";

interface VerifyOptions {
  card?: string;
  claim?: string;
  evidence?: string;
  subagentReturn?: string;
  tier?: string;
  taskId?: string;
  storyId?: string;
  trace?: boolean;
  traceDir?: string;
  json?: boolean;
  verbose?: boolean;
  mutationGuard?: boolean;
  staleGround?: boolean;
}

const DEFAULT_CARD_PATHS = [
  "completion-card.yaml",
  "completion-card.yml",
  ".x-harness/completion-card.yaml",
];

async function resolveCardPath(
  cwd: string,
  explicit?: string
): Promise<string | undefined> {
  if (explicit) {
    const p = path.resolve(cwd, explicit);
    return (await fs.pathExists(p)) ? p : undefined;
  }
  for (const rel of DEFAULT_CARD_PATHS) {
    const p = path.resolve(cwd, rel);
    if (await fs.pathExists(p)) return p;
  }
  return undefined;
}

export function verifyCommand(): Command {
  return new Command("verify")
    .description(
      "Run read-only verification against a completion card or claim/evidence/subagent-return"
    )
    .option(
      "--card <path>",
      "Path to completion card YAML/JSON (default: auto-detect)"
    )
    .option(
      "-c, --claim <path>",
      "Path to claim YAML/JSON (advanced/compatibility mode)"
    )
    .option(
      "-e, --evidence <path>",
      "Path to evidence YAML/JSON (advanced/compatibility mode)"
    )
    .option(
      "-s, --subagent-return <path>",
      "Path to subagent return YAML/JSON (advanced/compatibility mode)"
    )
    .option("-t, --tier <tier>", "Tier: light, standard, deep", "standard")
    .option("--task-id <id>", "Task ID")
    .option("--story-id <id>", "Story ID")
    .option("--trace", "Append verify event to trace", false)
    .option(
      "--trace-dir <dir>",
      "Directory for trace output (default: .x-harness/traces)",
      ".x-harness/traces"
    )
    .option("--json", "Output JSON instead of human-readable text", false)
    .option("--verbose", "Output detailed human-readable text", false)
    .option(
      "--mutation-guard",
      "Block verification if unexpected file mutations are detected",
      false
    )
    .option(
      "--stale-ground",
      "Mark the task as having stale ground (blocks admission)",
      false
    )
    .action(async (opts: VerifyOptions) => {
      const guard = await runMutationGuard(
        opts.mutationGuard ?? false,
        process.cwd()
      );
      await guard.takeSnapshot();

      // Test-only hook: gate by X_HARNESS_ENABLE_TEST_HOOKS to avoid accidental activation.
      // When enabled, deterministically injects a mutation after the guard snapshot
      // so integration tests can verify blocked trace behavior without timing races.
      // Safety: restrict injection path to within cwd to prevent arbitrary file writes.
      if (
        process.env.X_HARNESS_ENABLE_TEST_HOOKS === "1" &&
        process.env.X_HARNESS_TEST_INJECT_MUTATION
      ) {
        const injectPath = path.resolve(
          process.cwd(),
          process.env.X_HARNESS_TEST_INJECT_MUTATION
        );
        if (injectPath.startsWith(process.cwd() + path.sep)) {
          fs.writeFileSync(injectPath, "test-mutation");
        } else {
          console.error(
            `test hook: rejected injection path ${injectPath} outside cwd (${process.cwd()})`
          );
        }
      }

      const startTime = Date.now();
      const errors: string[] = [];
      const notes: string[] = [];
      let claim: Record<string, unknown> | undefined;
      let evidence: Record<string, unknown> | undefined;
      let subagentReturn: Record<string, unknown> | undefined;
      let card: Record<string, unknown> | undefined;
      let cardPath: string | undefined;
      let inputCardHash: string | null = null;
      let policyHash: string | null = null;

      const useLegacy = opts.claim || opts.evidence || opts.subagentReturn;
      const useCard = !useLegacy;

      if (useCard) {
        cardPath = await resolveCardPath(process.cwd(), opts.card);
        if (!cardPath) {
          console.error(
            "Error: No completion card found. Searched: " +
              DEFAULT_CARD_PATHS.join(", ")
          );
          console.error(
            "Provide --card <path> or use --claim/--evidence/--subagent-return for compatibility mode."
          );
          process.exit(2);
        }
        try {
          const data = await readYamlOrJson(cardPath);
          const result = await validateCompletionCard(data);
          if (!result.valid) {
            errors.push(
              `completion card validation failed: ${result.errors.join("; ")}`
            );
          } else {
            card = data as Record<string, unknown>;
            notes.push(
              `completion card valid: ${path.relative(process.cwd(), cardPath)}`
            );
          }
          inputCardHash = sha256String(JSON.stringify(data));
        } catch (err) {
          errors.push(
            `completion card load error: ${err instanceof Error ? err.message : String(err)}`
          );
        }
      }

      // Load policy hash
      const policyPath = path.resolve(
        process.cwd(),
        "policies",
        "admission.yaml"
      );
      policyHash = await sha256File(policyPath);

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
          errors.push(
            `claim load error: ${err instanceof Error ? err.message : String(err)}`
          );
        }
      }

      // Load and validate evidence
      if (opts.evidence) {
        try {
          const data = await readYamlOrJson(path.resolve(opts.evidence));
          const result = await validateEvidence(data);
          if (!result.valid) {
            errors.push(
              `evidence validation failed: ${result.errors.join("; ")}`
            );
          } else {
            evidence = data as Record<string, unknown>;
            notes.push("evidence schema valid");
          }
        } catch (err) {
          errors.push(
            `evidence load error: ${err instanceof Error ? err.message : String(err)}`
          );
        }
      }

      // Load and validate subagent return
      if (opts.subagentReturn) {
        try {
          const data = await readYamlOrJson(path.resolve(opts.subagentReturn));
          const result = await validateSubagentReturn(data);
          if (!result.valid) {
            errors.push(
              `subagent-return validation failed: ${result.errors.join("; ")}`
            );
          } else {
            subagentReturn = data as Record<string, unknown>;
            notes.push("subagent-return schema valid");
          }
        } catch (err) {
          errors.push(
            `subagent-return load error: ${err instanceof Error ? err.message : String(err)}`
          );
        }
      }

      // Admission checks
      const tier = (opts.tier as "light" | "standard" | "deep") ?? "standard";
      // CLI flag overrides card field; card field is also accepted from completion card.
      // Use explicit true check for CLI flag since false is the default and also a valid card value.
      const staleGroundFromCard =
        card &&
        typeof (card as Record<string, unknown>).stale_ground === "boolean"
          ? ((card as Record<string, unknown>).stale_ground as boolean)
          : false;
      const effectiveStaleGround =
        opts.staleGround === true ? true : staleGroundFromCard;
      const admissionInput: Parameters<typeof runAdmission>[0] = card
        ? {
            schema_version: String(card.schema_version ?? ""),
            task_id: String(card.task_id ?? opts.taskId ?? ""),
            tier: (card.tier as "light" | "standard" | "deep") ?? tier,
            owner: String(card.owner ?? ""),
            accountable: String(card.accountable ?? ""),
            claim: card.claim as Record<string, unknown>,
            verification: card.verification as Record<string, unknown>,
            admission: card.admission as Record<string, unknown>,
            acceptance_status: card.acceptance_status as
              | "accepted"
              | "withheld",
            handoff: card.handoff as Record<string, unknown>,
            evidence: card.evidence as Record<string, unknown> | undefined,
            state: card.state as Record<string, unknown> | undefined,
            governance: card.governance as Record<string, unknown> | undefined,
            staleGround: effectiveStaleGround,
          }
        : {
            claim,
            evidence,
            subagentReturn,
            tier,
            staleGround: effectiveStaleGround,
          };

      const admission = runAdmission(admissionInput);

      // Merge admission results
      errors.push(...admission.errors);
      notes.push(...admission.notes);

      // Preserve blocked/failed/skipped outcomes from admission; only fall back to
      // "failed" when there are errors but admission had not yet decided.
      const outcome =
        admission.outcome !== "success"
          ? admission.outcome
          : errors.length > 0
            ? "failed"
            : admission.outcome;
      const _acceptance = acceptanceStatus(outcome);
      const _verifyRuntimeMs = Date.now() - startTime;

      const recovery = suggestRecovery(errors, outcome);
      const blockingPredicate =
        admission.blocking_predicate ??
        recovery.predicate ??
        (outcome === "blocked" || outcome === "failed"
          ? "admission_failed"
          : null);
      const recoveryRoute = recovery.route;

      const cardId = (card?.id as string | undefined) ?? null;
      const taskId =
        opts.taskId ?? (card?.task_id as string | undefined) ?? "TASK-UNKNOWN";

      const guardResult = await guard.evaluate();
      if (guardResult.skippedReason) {
        notes.push(`mutation guard skipped: ${guardResult.skippedReason}`);
      } else if (guardResult.enabled && !guardResult.violated) {
        notes.push("mutation guard passed");
      }
      if (guardResult.violated && guardResult.unexpectedDeltas) {
        const paths = guardResult.unexpectedDeltas.map((d) => d.path);
        errors.push(
          `mutation guard blocked: unexpected changes detected: ${paths.join(", ")}`
        );
      }

      // Recompute outcome if guard detected mutations
      const finalOutcome =
        guardResult.violated && guardResult.unexpectedDeltas
          ? "blocked"
          : outcome;
      const finalAcceptance = acceptanceStatus(finalOutcome);
      const finalBlockingPredicate = guardResult.violated
        ? "verifier_not_read_only"
        : blockingPredicate;
      const finalRecoveryRoute = guardResult.violated
        ? getRecoveryRoute("verifier_not_read_only")
        : recoveryRoute;

      // Build enriched checks (after guard errors are finalized)
      const checks: {
        name: string;
        status: string;
        severity: string;
        note?: string;
      }[] = [];
      for (const note of notes) {
        const isWarning =
          note.includes("recommends") || note.includes("warning");
        checks.push({
          name: note.split(":")[0] || "note",
          status: isWarning ? "warning" : "passed",
          severity: isWarning ? "warning" : "info",
          note,
        });
      }
      for (const err of errors) {
        checks.push({
          name: err.split(":")[0] || "error",
          status: "failed",
          severity: "error",
          note: err,
        });
      }

      const event = {
        event_id: `VE-${Date.now()}`,
        event_type: "verify_completed",
        task_id: taskId,
        story_id: opts.storyId ?? null,
        tier: card?.tier ?? tier,
        claim_id:
          (claim?.id as string | undefined) ??
          (card?.task_id as string | undefined) ??
          null,
        evidence_id: (evidence?.id as string | undefined) ?? null,
        verifier: "x-harness",
        verifier_mode: "read_only",
        outcome: finalOutcome,
        acceptance_status: finalAcceptance,
        blocking_predicate: finalBlockingPredicate,
        blocked_reason_class:
          finalOutcome === "blocked" ? "policy_violation" : null,
        next_owner:
          finalRecoveryRoute?.owner ??
          ((card?.handoff as Record<string, unknown> | undefined)?.owner as
            | string
            | null) ??
          null,
        next_action:
          finalRecoveryRoute?.next_action ??
          ((card?.handoff as Record<string, unknown> | undefined)
            ?.next_action as string | null) ??
          (errors.length > 0 ? "resolve validation errors" : null),
        created_at: new Date().toISOString(),
        notes,
        errors,
      };

      if (opts.trace) {
        await appendTrace(event, opts.traceDir);
      }

      if (opts.json) {
        console.log(
          JSON.stringify(
            {
              ok: errors.length === 0 && finalOutcome === "success",
              acceptance_status: finalAcceptance,
              admission_outcome: finalOutcome,
              withheld_reason: errors.length > 0 ? errors.join("; ") : null,
              card_id: cardId,
              task_id: taskId,
              schema_version: 1,
              input_card_hash: inputCardHash ? `sha256:${inputCardHash}` : null,
              policy_hash: policyHash ? `sha256:${policyHash}` : null,
              checks,
              decision: {
                outcome: finalOutcome,
                acceptance_status: finalAcceptance,
              },
              recovery: finalRecoveryRoute
                ? {
                    predicate: finalBlockingPredicate,
                    next_action: finalRecoveryRoute.next_action,
                    owner: finalRecoveryRoute.owner,
                  }
                : null,
              denominator_warning:
                "Verify-event success must not be interpreted as task-level success, production reliability, benchmark success, or safety guarantee.",
            },
            null,
            2
          )
        );
      } else if (opts.verbose) {
        const cardName = cardPath ? path.basename(cardPath) : "N/A";
        const cardTier = String(card?.tier ?? tier);
        const cardClaim = card?.claim
          ? (card.claim as Record<string, unknown>).fix_status
          : ((claim?.fix_status as string | undefined) ?? "N/A");
        const cardVerification = card?.verification
          ? (card.verification as Record<string, unknown>).status
          : "N/A";
        const cardAdmission = card?.admission
          ? (card.admission as Record<string, unknown>).outcome
          : finalOutcome;
        const cardAcceptance = finalAcceptance;

        console.log(`Card: ${cardName}`);
        console.log(`Tier: ${cardTier}`);
        console.log(`Claim: ${cardClaim}`);
        console.log(`Verification: ${cardVerification}`);
        console.log(`Admission: ${cardAdmission}`);
        console.log(`Acceptance: ${cardAcceptance}`);
        console.log(
          `Result: ${errors.length === 0 && finalOutcome === "success" ? "ACCEPTED" : "WITHHELD"}`
        );
        if (errors.length > 0) {
          console.log("Errors:");
          for (const err of errors) {
            console.log(`  - ${err}`);
          }
        }
        if (event.next_action && event.next_owner) {
          console.log(`Handoff: ${event.next_action} -> ${event.next_owner}`);
        }
        if (finalRecoveryRoute) {
          console.log(
            `Recovery: ${finalRecoveryRoute.next_action} -> ${finalRecoveryRoute.owner}`
          );
        }
      } else {
        // Quiet default: <=3 lines
        const passedChecks = notes.filter(
          (n) =>
            n.includes("valid") ||
            n.includes("passed") ||
            n.includes("checks passed")
        ).length;
        const failedChecks = errors.length;
        console.log(`outcome: ${finalOutcome}`);
        console.log(`acceptance_status: ${finalAcceptance}`);
        if (failedChecks > 0) {
          console.log(`checks: ${passedChecks} passed, ${failedChecks} failed`);
        } else {
          console.log(`checks: ${passedChecks} passed, 0 failed`);
        }
      }

      const accepted =
        finalOutcome === "success" && finalAcceptance === "accepted";
      process.exit(accepted ? 0 : 1);
    });
}
