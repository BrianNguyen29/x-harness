import { Command } from "commander";
import * as path from "node:path";
import fs from "fs-extra";
import { readYamlOrJson } from "../core/schema.js";
import {
  listEpisodeDirectories,
  withEpisodeDirectory,
} from "../core/episode.js";

interface PredictionCheckOptions {
  card?: string;
  json?: boolean;
  verbose?: boolean;
}

interface PredictionVerifyOptions {
  episode?: string;
  json?: boolean;
}

interface PredictionReportOptions {
  since?: string;
  episodesDir?: string;
  json?: boolean;
}

const DEFAULT_CARD_PATHS = [
  "completion-card.yaml",
  "completion-card.yml",
  ".x-harness/completion-card.yaml",
];

const ALLOWED_HORIZONS = [
  "same_verify",
  "next_ci_run",
  "next_release",
  "manual_review",
  "production_7d",
  "production_30d",
];

interface PredictionValidationResult {
  valid: boolean;
  errors: string[];
  warnings: string[];
}

interface PredictionVerificationResult {
  ok: boolean;
  status: "confirmed" | "falsified" | "inconclusive";
  reason: string;
  episode_id: string | null;
  task_id: string | null;
  horizon: string | null;
  prediction: Record<string, unknown> | null;
  validation: PredictionValidationResult | null;
  verdict: {
    admission_outcome: string | null;
    acceptance_status: string | null;
  };
}

function validatePrediction(
  prediction: Record<string, unknown>
): PredictionValidationResult {
  const errors: string[] = [];
  const warnings: string[] = [];

  // Required fields
  const claim = prediction.claim as string | undefined;
  const expectedEffect = prediction.expected_effect as string | undefined;
  const falsificationMethod = prediction.falsification_method as
    | string
    | undefined;
  const horizon = prediction.horizon as string | undefined;

  if (!claim || typeof claim !== "string" || claim.trim().length === 0) {
    errors.push("prediction.claim is required and must be a non-empty string");
  }

  if (
    !expectedEffect ||
    typeof expectedEffect !== "string" ||
    expectedEffect.trim().length === 0
  ) {
    errors.push(
      "prediction.expected_effect is required and must be a non-empty string"
    );
  }

  if (
    !falsificationMethod ||
    typeof falsificationMethod !== "string" ||
    falsificationMethod.trim().length === 0
  ) {
    errors.push(
      "prediction.falsification_method is required and must be a non-empty string"
    );
  }

  if (!horizon) {
    errors.push("prediction.horizon is required");
  } else if (!ALLOWED_HORIZONS.includes(horizon)) {
    errors.push(
      `prediction.horizon must be one of: ${ALLOWED_HORIZONS.join(", ")}`
    );
  }

  // Optional: measurability check
  const measurableSignal = prediction.measurable_signal as string | undefined;
  if (
    !measurableSignal ||
    typeof measurableSignal !== "string" ||
    measurableSignal.trim().length === 0
  ) {
    warnings.push(
      "prediction.measurable_signal is recommended for falsifiable predictions"
    );
  }

  // Optional: confidence check
  const confidence = prediction.confidence as string | undefined;
  if (confidence && !["low", "medium", "high"].includes(confidence)) {
    warnings.push("prediction.confidence should be one of: low, medium, high");
  }

  return {
    valid: errors.length === 0,
    errors,
    warnings,
  };
}

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

async function loadEpisodeCard(
  episodeDir: string
): Promise<Record<string, unknown> | null> {
  const yamlPath = path.join(episodeDir, "completion-card.yaml");
  if (await fs.pathExists(yamlPath)) {
    return (await readYamlOrJson(yamlPath)) as Record<string, unknown>;
  }
  const jsonPath = path.join(episodeDir, "completion-card.json");
  if (await fs.pathExists(jsonPath)) {
    return (await fs.readJson(jsonPath)) as Record<string, unknown>;
  }
  return null;
}

async function verifyPredictionFromEpisodeDir(
  episodeDir: string
): Promise<PredictionVerificationResult> {
  const manifestPath = path.join(episodeDir, "manifest.json");
  const manifest = (await fs.readJson(manifestPath)) as Record<string, unknown>;
  const verdict =
    (manifest.verdict as Record<string, unknown> | undefined) ?? {};
  const card = await loadEpisodeCard(episodeDir);
  const prediction = card?.prediction as Record<string, unknown> | undefined;

  if (!prediction) {
    return {
      ok: false,
      status: "inconclusive",
      reason: "missing_prediction",
      episode_id: String(manifest.episode_id ?? "") || null,
      task_id: String(manifest.task_id ?? "") || null,
      horizon: null,
      prediction: null,
      validation: null,
      verdict: {
        admission_outcome:
          typeof verdict.admission_outcome === "string"
            ? verdict.admission_outcome
            : null,
        acceptance_status:
          typeof verdict.acceptance_status === "string"
            ? verdict.acceptance_status
            : null,
      },
    };
  }

  const validation = validatePrediction(prediction);
  const horizon =
    typeof prediction.horizon === "string" ? prediction.horizon : null;
  const admissionOutcome =
    typeof verdict.admission_outcome === "string"
      ? verdict.admission_outcome
      : null;
  const acceptanceStatus =
    typeof verdict.acceptance_status === "string"
      ? verdict.acceptance_status
      : null;

  if (!validation.valid) {
    return {
      ok: false,
      status: "inconclusive",
      reason: "invalid_prediction",
      episode_id: String(manifest.episode_id ?? "") || null,
      task_id: String(manifest.task_id ?? "") || null,
      horizon,
      prediction,
      validation,
      verdict: {
        admission_outcome: admissionOutcome,
        acceptance_status: acceptanceStatus,
      },
    };
  }

  if (horizon !== "same_verify") {
    return {
      ok: true,
      status: "inconclusive",
      reason: `unsupported_horizon:${horizon ?? "missing"}`,
      episode_id: String(manifest.episode_id ?? "") || null,
      task_id: String(manifest.task_id ?? "") || null,
      horizon,
      prediction,
      validation,
      verdict: {
        admission_outcome: admissionOutcome,
        acceptance_status: acceptanceStatus,
      },
    };
  }

  const confirmed =
    admissionOutcome === "success" && acceptanceStatus === "accepted";
  return {
    ok: confirmed,
    status: confirmed ? "confirmed" : "falsified",
    reason: confirmed
      ? "same_verify_episode_accepted"
      : "same_verify_episode_withheld",
    episode_id: String(manifest.episode_id ?? "") || null,
    task_id: String(manifest.task_id ?? "") || null,
    horizon,
    prediction,
    validation,
    verdict: {
      admission_outcome: admissionOutcome,
      acceptance_status: acceptanceStatus,
    },
  };
}

function parseSinceMs(since?: string): number | null {
  if (!since) return null;
  const match = /^(\d+)([dh])$/.exec(since);
  if (!match) return null;
  const value = Number(match[1]);
  return match[2] === "d"
    ? value * 24 * 60 * 60 * 1000
    : value * 60 * 60 * 1000;
}

export async function predictionCheckAction(
  opts: PredictionCheckOptions
): Promise<void> {
  const cwd = process.cwd();
  const cardPath = await resolveCardPath(cwd, opts.card);

  if (!cardPath) {
    console.error(
      "Error: No completion card found. Searched: " +
        DEFAULT_CARD_PATHS.join(", ")
    );
    console.error("Provide --card <path> to specify a card.");
    process.exit(1);
  }

  let card: Record<string, unknown>;
  try {
    const data = await readYamlOrJson(cardPath);
    card = data as Record<string, unknown>;
  } catch (err) {
    console.error(
      `Error loading card: ${err instanceof Error ? err.message : String(err)}`
    );
    process.exit(1);
  }

  const prediction = card.prediction as Record<string, unknown> | undefined;
  const tier = card.tier as string | undefined;

  if (!prediction) {
    if (opts.json) {
      console.log(
        JSON.stringify(
          {
            ok: false,
            error: "No prediction found in completion card",
            tier,
          },
          null,
          2
        )
      );
    } else {
      console.error("Error: No prediction found in completion card.");
      if (tier === "standard" || tier === "deep") {
        console.error(`Tier "${tier}" requires a prediction.`);
      }
    }
    process.exit(1);
  }

  const result = validatePrediction(prediction);

  if (opts.json) {
    console.log(
      JSON.stringify(
        {
          ok: result.valid,
          errors: result.errors,
          warnings: result.warnings,
          prediction,
          tier,
        },
        null,
        2
      )
    );
  } else if (opts.verbose) {
    if (result.valid) {
      console.log("✓ Prediction is valid");
    } else {
      console.log("✗ Prediction has errors:");
    }
    for (const err of result.errors) {
      console.log(`  - ${err}`);
    }
    if (result.warnings.length > 0) {
      console.log("\nWarnings:");
      for (const warn of result.warnings) {
        console.log(`  - ${warn}`);
      }
    }
    if (result.valid && result.warnings.length === 0) {
      console.log("\nPrediction structure:");
      console.log(`  claim: ${prediction.claim}`);
      console.log(`  expected_effect: ${prediction.expected_effect}`);
      console.log(`  falsification_method: ${prediction.falsification_method}`);
      console.log(`  horizon: ${prediction.horizon}`);
      if (prediction.measurable_signal) {
        console.log(`  measurable_signal: ${prediction.measurable_signal}`);
      }
      if (prediction.confidence) {
        console.log(`  confidence: ${prediction.confidence}`);
      }
    }
  } else {
    if (result.valid) {
      console.log("Prediction is valid.");
    } else {
      console.error("Prediction validation failed:");
      for (const err of result.errors) {
        console.error(`  - ${err}`);
      }
    }
  }

  process.exit(result.valid ? 0 : 1);
}

export async function predictionVerifyAction(
  opts: PredictionVerifyOptions
): Promise<void> {
  if (!opts.episode) {
    console.error("Error: prediction verify requires --episode");
    process.exit(2);
  }

  const result = await withEpisodeDirectory(opts.episode, (episodeDir) =>
    verifyPredictionFromEpisodeDir(episodeDir)
  );
  if (opts.json) {
    console.log(JSON.stringify(result, null, 2));
  } else {
    console.log("# x-harness Prediction Verify");
    console.log(`- status: ${result.status}`);
    console.log(`- reason: ${result.reason}`);
    console.log(`- episode_id: ${result.episode_id ?? "unknown"}`);
    console.log(`- task_id: ${result.task_id ?? "unknown"}`);
    console.log(`- horizon: ${result.horizon ?? "unknown"}`);
    console.log(
      `- verdict: ${result.verdict.admission_outcome ?? "unknown"} / ${result.verdict.acceptance_status ?? "unknown"}`
    );
  }
  process.exit(result.status === "falsified" ? 1 : 0);
}

export async function predictionReportAction(
  opts: PredictionReportOptions
): Promise<void> {
  const episodes = await listEpisodeDirectories(
    opts.episodesDir ?? ".x-harness/episodes"
  );
  const sinceMs = parseSinceMs(opts.since);
  const cutoff = sinceMs ? Date.now() - sinceMs : null;
  const filtered = cutoff
    ? episodes.filter(
        (episode) => new Date(episode.manifest.created_at).getTime() >= cutoff
      )
    : episodes;
  const results = await Promise.all(
    filtered.map((episode) => verifyPredictionFromEpisodeDir(episode.dir))
  );
  const report = {
    ok: true,
    since: opts.since ?? null,
    episodes_analyzed: results.length,
    confirmed: results.filter((result) => result.status === "confirmed").length,
    falsified: results.filter((result) => result.status === "falsified").length,
    inconclusive: results.filter((result) => result.status === "inconclusive")
      .length,
    results,
  };

  if (opts.json) {
    console.log(JSON.stringify(report, null, 2));
  } else {
    console.log("# x-harness Prediction Report");
    console.log(`- episodes_analyzed: ${report.episodes_analyzed}`);
    console.log(`- confirmed: ${report.confirmed}`);
    console.log(`- falsified: ${report.falsified}`);
    console.log(`- inconclusive: ${report.inconclusive}`);
    if (results.length > 0) {
      console.log("");
      console.log("## Episodes");
      for (const result of results) {
        console.log(
          `- ${result.episode_id ?? "unknown"}: ${result.status} (${result.reason})`
        );
      }
    }
  }
  process.exit(0);
}

export function predictionCommand(): Command {
  const cmd = new Command("prediction");

  cmd
    .description("Validate and verify predictions in completion cards")
    .addCommand(
      new Command("check")
        .description("Validate prediction structure and falsifiability")
        .option(
          "--card <path>",
          "Path to completion card YAML/JSON (default: auto-detect)"
        )
        .option("--json", "Output JSON instead of text", false)
        .option("--verbose", "Output detailed human-readable text", false)
        .action(predictionCheckAction)
    )
    .addCommand(
      new Command("verify")
        .description("Falsify same-verify prediction from an episode")
        .option("--episode <path>", "Path to episode directory or bundle")
        .option("--json", "Output JSON instead of text", false)
        .action(predictionVerifyAction)
    )
    .addCommand(
      new Command("report")
        .description("Show prediction falsification history from episodes")
        .option("--since <duration>", "Filter by time (e.g., 7d, 30d)")
        .option(
          "--episodes-dir <dir>",
          "Episodes directory",
          ".x-harness/episodes"
        )
        .option("--json", "Output JSON instead of text", false)
        .action(predictionReportAction)
    );

  return cmd;
}
