import { Command } from "commander";
import * as path from "node:path";
import {
  runVerifyPipeline,
  VerifyInputError,
  VerifyPipelineOptions,
  VerifyPipelineResult,
} from "../core/verify-pipeline.js";
import { createEpisodeFromVerifyResult } from "../core/episode.js";

interface VerifyOptions extends VerifyPipelineOptions {
  json?: boolean;
  verbose?: boolean;
  episode?: boolean;
  bundle?: boolean;
  episodesDir?: string;
}

function renderJson(result: VerifyPipelineResult): void {
  console.log(
    JSON.stringify(
      {
        ok: result.errors.length === 0 && result.finalOutcome === "success",
        acceptance_status: result.finalAcceptance,
        admission_outcome: result.finalOutcome,
        withheld_reason:
          result.errors.length > 0 ? result.errors.join("; ") : null,
        card_id: result.cardId,
        task_id: result.taskId,
        schema_version: 1,
        input_card_hash: result.inputCardHash
          ? `sha256:${result.inputCardHash}`
          : null,
        policy_hash: result.policyHash ? `sha256:${result.policyHash}` : null,
        product_intent_status: productIntentStatusFromCard(result.card),
        strict: result.strict,
        changed_files: result.changedFiles
          ? {
              source: result.changedFiles.source,
              card_files: result.changedFiles.card_files,
              git_files: result.changedFiles.git_files,
              files: result.changedFiles.files,
            }
          : null,
        checks: result.checks,
        decision: {
          outcome: result.finalOutcome,
          acceptance_status: result.finalAcceptance,
        },
        recovery: result.finalRecoveryRoute
          ? {
              predicate: result.finalBlockingPredicate,
              next_action: result.finalRecoveryRoute.next_action,
              owner: result.finalRecoveryRoute.owner,
            }
          : null,
        episode: result.episode ?? null,
        denominator_warning:
          "Verify-event success must not be interpreted as task-level success, production reliability, benchmark success, or safety guarantee.",
      },
      null,
      2
    )
  );
}

function renderVerbose(result: VerifyPipelineResult): void {
  const cardName = result.cardPath ? path.basename(result.cardPath) : "N/A";
  const cardClaim = result.card?.claim
    ? (result.card.claim as Record<string, unknown>).fix_status
    : ((result.claim?.fix_status as string | undefined) ?? "N/A");
  const cardVerification = result.card?.verification
    ? (result.card.verification as Record<string, unknown>).status
    : "N/A";
  const cardAdmission = result.card?.admission
    ? (result.card.admission as Record<string, unknown>).outcome
    : result.finalOutcome;

  console.log(`Card: ${cardName}`);
  console.log(`Tier: ${result.tier}`);
  console.log(`Claim: ${cardClaim}`);
  console.log(`Verification: ${cardVerification}`);
  console.log(`Admission: ${cardAdmission}`);
  console.log(`Acceptance: ${result.finalAcceptance}`);
  const intentStatus = productIntentStatusFromCard(result.card);
  if (intentStatus) {
    console.log(`Product intent: ${intentStatus}`);
  }
  console.log(
    `Result: ${
      result.errors.length === 0 && result.finalOutcome === "success"
        ? "ACCEPTED"
        : "WITHHELD"
    }`
  );
  if (result.errors.length > 0) {
    console.log("Errors:");
    for (const err of result.errors) {
      console.log(`  - ${err}`);
    }
  }
  if (result.event.next_action && result.event.next_owner) {
    console.log(
      `Handoff: ${result.event.next_action} -> ${result.event.next_owner}`
    );
  }
  if (result.finalRecoveryRoute) {
    console.log(
      `Recovery: ${result.finalRecoveryRoute.next_action} -> ${result.finalRecoveryRoute.owner}`
    );
  }
  if (result.episode) {
    console.log(`Episode: ${result.episode.episode_dir}`);
  }
}

function renderQuiet(result: VerifyPipelineResult): void {
  const passedChecks = result.notes.filter(
    (n) =>
      n.includes("valid") || n.includes("passed") || n.includes("checks passed")
  ).length;
  const failedChecks = result.errors.length;
  const intentStatus = productIntentStatusFromCard(result.card);
  console.log(`outcome: ${result.finalOutcome}`);
  console.log(`acceptance_status: ${result.finalAcceptance}`);
  if (intentStatus) {
    console.log(`product_intent: ${intentStatus}`);
  }
  console.log(`checks: ${passedChecks} passed, ${failedChecks} failed`);
  if (result.episode) {
    console.log(`episode: ${result.episode.episode_dir}`);
  }
}

function productIntentStatusFromCard(
  card?: Record<string, unknown>
): string | null {
  if (card == null) return null;
  const productIntent = card.product_intent as
    | Record<string, unknown>
    | undefined;
  if (productIntent == null) return null;
  const status = productIntent.status;
  if (typeof status !== "string") return null;
  const trimmed = status.trim();
  return trimmed === "" ? null : trimmed;
}

function renderResult(result: VerifyPipelineResult, opts: VerifyOptions): void {
  if (opts.json) {
    renderJson(result);
  } else if (opts.verbose) {
    renderVerbose(result);
  } else {
    renderQuiet(result);
  }
}

function renderInputError(err: VerifyInputError): void {
  console.error(`Error: ${err.message}`);
  for (const detail of err.details) {
    console.error(detail);
  }
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
      "--strict",
      "Enable strict read-only verification checks, including mutation guard",
      false
    )
    .option(
      "--context-floor",
      "Enable context floor checks (default for standard/deep; opt-in for light)",
      false
    )
    .option(
      "--stale-ground",
      "Mark the task as having stale ground (blocks admission)",
      false
    )
    .option(
      "--governance-enforced",
      "Require verified approval artifacts for protected path changes",
      false
    )
    .option(
      "--diff <ref>",
      "Git ref used to derive real changed files for governance checks"
    )
    .option(
      "--changed-files-source <mode>",
      "Changed files source for governance: card, git, union, strict"
    )
    .option(
      "--profile <name>",
      "Verify profile (light-local|ci-standard|ci-strict|governed-deep). Sets the default for --decision-enforce when the flag is omitted; an explicit --decision-enforce always wins."
    )
    .option(
      "--decision-enforce <mode>",
      "Enforce context_alignment.decision_refs at the verify layer (off|advisory|block). Defaults to the profile default when --profile is set; otherwise off. Mirrors the Go canonical flag in internal/cli/verify.go."
    )
    .option("--episode", "Write an audit episode package", false)
    .option(
      "--bundle",
      "Create raw and redacted episode tar.gz bundles (implies --episode)",
      false
    )
    .option(
      "--episodes-dir <dir>",
      "Directory for episode packages",
      ".x-harness/episodes"
    )
    .action(async (opts: VerifyOptions) => {
      try {
        // Auto-enable context floor for standard and deep tiers
        const effectiveTier = opts.tier ?? "standard";
        if (
          !opts.contextFloor &&
          (effectiveTier === "standard" || effectiveTier === "deep")
        ) {
          opts.contextFloor = true;
        }
        const result = await runVerifyPipeline(opts);
        if (opts.episode || opts.bundle) {
          result.episode = await createEpisodeFromVerifyResult(result, {
            episodesDir: opts.episodesDir,
            bundle: Boolean(opts.bundle),
          });
        }
        renderResult(result, opts);
        process.exit(result.accepted ? 0 : 1);
      } catch (err) {
        if (err instanceof VerifyInputError) {
          renderInputError(err);
          process.exit(err.exitCode);
        }
        throw err;
      }
    });
}
