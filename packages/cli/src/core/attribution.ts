import * as path from "node:path";
import fs from "fs-extra";
import { compileSchema, loadSchema } from "./schema.js";
import { readTraceFromFile } from "./trace.js";
import type { EpisodeManifest } from "./episode.js";

export type FailureTaxonomy =
  | "Ftask_spec"
  | "Fcontext"
  | "Ftool"
  | "Fmemory"
  | "Fstate"
  | "Fobservability"
  | "Fattribution"
  | "Fverification"
  | "Fpermission"
  | "Fentropy"
  | "Fintervention"
  | "Fmodel"
  | "Funknown";

export interface AttributionCandidate {
  taxonomy: FailureTaxonomy;
  predicate: string;
  component_id: string;
  confidence: "low" | "medium" | "high";
  rationale: string;
}

export interface FailureAttribution {
  schema_version: "1";
  episode_id: string;
  task_id: string;
  created_at: string;
  verdict: {
    admission_outcome: string;
    acceptance_status: string;
    blocking_predicate: string | null;
  };
  primary: AttributionCandidate | null;
  candidates: AttributionCandidate[];
  unknown_rate_signal: {
    is_unknown: boolean;
    reason: string;
  };
  admission_authority: false;
}

export interface AttributionReport {
  ok: boolean;
  group_by: "predicate" | "taxonomy" | "component";
  total_episodes: number;
  withheld_episodes: number;
  unknown_count: number;
  unknown_rate: number;
  groups: Array<{
    key: string;
    count: number;
    episode_ids: string[];
    predicates: string[];
    taxonomies: string[];
    components: string[];
  }>;
  entropy_warning: string | null;
}

interface AttributionInput {
  episodeId: string;
  taskId: string;
  createdAt: string;
  admissionOutcome: string;
  acceptanceStatus: string;
  blockingPredicate?: string | null;
  errors?: string[];
  notes?: string[];
}

function candidate(
  taxonomy: FailureTaxonomy,
  predicate: string,
  componentId: string,
  confidence: "low" | "medium" | "high",
  rationale: string
): AttributionCandidate {
  return {
    taxonomy,
    predicate,
    component_id: componentId,
    confidence,
    rationale,
  };
}

export function createFailureAttribution(
  input: AttributionInput
): FailureAttribution {
  const verdict = {
    admission_outcome: input.admissionOutcome,
    acceptance_status: input.acceptanceStatus,
    blocking_predicate: input.blockingPredicate ?? null,
  };

  if (
    verdict.admission_outcome === "success" &&
    verdict.acceptance_status === "accepted"
  ) {
    return {
      schema_version: "1",
      episode_id: input.episodeId,
      task_id: input.taskId,
      created_at: input.createdAt,
      verdict,
      primary: null,
      candidates: [],
      unknown_rate_signal: {
        is_unknown: false,
        reason: "accepted episode has no failure attribution",
      },
      admission_authority: false,
    };
  }

  const text = [
    input.blockingPredicate ?? "",
    ...(input.errors ?? []),
    ...(input.notes ?? []),
  ]
    .join("\n")
    .toLowerCase();

  let primary: AttributionCandidate;
  if (
    text.includes("evidence") ||
    text.includes("prediction") ||
    text.includes("verification") ||
    text.includes("typecheck")
  ) {
    primary = candidate(
      "Fverification",
      input.blockingPredicate ?? "verification_failed",
      "admission_policy",
      "high",
      "Verify/admission evidence or prediction requirements were not satisfied."
    );
  } else if (
    text.includes("mutation guard") ||
    text.includes("verifier_not_read_only") ||
    text.includes("read-only")
  ) {
    primary = candidate(
      "Fpermission",
      input.blockingPredicate ?? "verifier_not_read_only",
      "verify_runtime",
      "high",
      "Verifier read-only or mutation guard boundary was violated."
    );
  } else if (
    text.includes("approval") ||
    text.includes("intervention") ||
    text.includes("downgrade")
  ) {
    primary = candidate(
      "Fintervention",
      input.blockingPredicate ?? "approval_missing",
      "governance_boundary",
      "high",
      "Human approval, intervention, or tier downgrade authorization was missing or invalid."
    );
  } else if (
    text.includes("context") ||
    text.includes("stale") ||
    text.includes("managed block")
  ) {
    primary = candidate(
      "Fcontext",
      input.blockingPredicate ?? "context_stale",
      "agent_contract",
      "medium",
      "Context was stale, missing, or not acknowledged."
    );
  } else if (
    text.includes("schema") ||
    text.includes("manifest") ||
    text.includes("trace") ||
    text.includes("episode")
  ) {
    primary = candidate(
      "Fobservability",
      input.blockingPredicate ?? "observability_invalid",
      "episode_packager",
      "medium",
      "Trace, schema, or episode observability artifact was missing or malformed."
    );
  } else if (
    text.includes("component") ||
    text.includes("policy drift") ||
    text.includes("tier label")
  ) {
    primary = candidate(
      "Fentropy",
      input.blockingPredicate ?? "harness_drift",
      "component_registry",
      "medium",
      "Harness metadata or policy drift signal was detected."
    );
  } else {
    primary = candidate(
      "Funknown",
      input.blockingPredicate ?? "unknown_failure",
      "unknown",
      "low",
      "No deterministic attribution rule matched the episode data."
    );
  }

  return {
    schema_version: "1",
    episode_id: input.episodeId,
    task_id: input.taskId,
    created_at: input.createdAt,
    verdict,
    primary,
    candidates: [primary],
    unknown_rate_signal: {
      is_unknown: primary.taxonomy === "Funknown",
      reason:
        primary.taxonomy === "Funknown"
          ? "no attribution rule matched"
          : "deterministic attribution rule matched",
    },
    admission_authority: false,
  };
}

export async function validateAttribution(
  attribution: FailureAttribution
): Promise<{ ok: boolean; errors: string[] }> {
  const schema = await loadSchema("attribution");
  const validate = compileSchema(schema);
  if (validate(attribution)) return { ok: true, errors: [] };
  return {
    ok: false,
    errors: (validate.errors ?? []).map(
      (err) => `${err.instancePath || "/"} ${err.message ?? "invalid"}`
    ),
  };
}

export async function loadOrCreateAttribution(
  episodeDir: string
): Promise<FailureAttribution> {
  const attributionPath = path.join(episodeDir, "failure-attribution.json");
  if (await fs.pathExists(attributionPath)) {
    return (await fs.readJson(attributionPath)) as FailureAttribution;
  }

  const manifest = (await fs.readJson(
    path.join(episodeDir, "manifest.json")
  )) as EpisodeManifest;
  const trace = await readTraceFromFile(path.join(episodeDir, "trace.jsonl"));
  const latest = trace[trace.length - 1] as
    | {
        errors?: string[];
        notes?: string[];
      }
    | undefined;

  return createFailureAttribution({
    episodeId: manifest.episode_id,
    taskId: manifest.task_id,
    createdAt: manifest.created_at,
    admissionOutcome: manifest.verdict.admission_outcome,
    acceptanceStatus: manifest.verdict.acceptance_status,
    blockingPredicate: manifest.verdict.blocking_predicate,
    errors: latest?.errors ?? [],
    notes: latest?.notes ?? [],
  });
}

export async function writeAttribution(
  episodeDir: string,
  attribution: FailureAttribution
): Promise<void> {
  const validation = await validateAttribution(attribution);
  if (!validation.ok) {
    throw new Error(
      `attribution validation failed: ${validation.errors.join("; ")}`
    );
  }
  await fs.writeJson(
    path.join(episodeDir, "failure-attribution.json"),
    attribution,
    {
      spaces: 2,
    }
  );
}

export async function listAttributions(
  episodesDir: string
): Promise<FailureAttribution[]> {
  if (!(await fs.pathExists(episodesDir))) return [];
  const entries = await fs.readdir(episodesDir, { withFileTypes: true });
  const attributions: FailureAttribution[] = [];
  for (const entry of entries) {
    if (!entry.isDirectory() || !entry.name.startsWith("ep_")) continue;
    const dir = path.join(episodesDir, entry.name);
    if (!(await fs.pathExists(path.join(dir, "manifest.json")))) continue;
    attributions.push(await loadOrCreateAttribution(dir));
  }
  return attributions.sort(
    (a, b) =>
      new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
  );
}

export function buildAttributionReport(
  attributions: FailureAttribution[],
  groupBy: "predicate" | "taxonomy" | "component"
): AttributionReport {
  const withheld = attributions.filter(
    (item) => item.verdict.acceptance_status === "withheld"
  );
  const unknownCount = withheld.filter(
    (item) => item.primary?.taxonomy === "Funknown"
  ).length;
  const groups = new Map<
    string,
    {
      count: number;
      episode_ids: Set<string>;
      predicates: Set<string>;
      taxonomies: Set<string>;
      components: Set<string>;
    }
  >();

  for (const item of withheld) {
    const primary = item.primary;
    const key =
      groupBy === "predicate"
        ? (primary?.predicate ?? "none")
        : groupBy === "taxonomy"
          ? (primary?.taxonomy ?? "none")
          : (primary?.component_id ?? "none");
    const group = groups.get(key) ?? {
      count: 0,
      episode_ids: new Set<string>(),
      predicates: new Set<string>(),
      taxonomies: new Set<string>(),
      components: new Set<string>(),
    };
    group.count += 1;
    group.episode_ids.add(item.episode_id);
    if (primary) {
      group.predicates.add(primary.predicate);
      group.taxonomies.add(primary.taxonomy);
      group.components.add(primary.component_id);
    }
    groups.set(key, group);
  }

  const unknownRate =
    withheld.length === 0 ? 0 : unknownCount / withheld.length;
  return {
    ok: true,
    group_by: groupBy,
    total_episodes: attributions.length,
    withheld_episodes: withheld.length,
    unknown_count: unknownCount,
    unknown_rate: Number(unknownRate.toFixed(4)),
    groups: [...groups.entries()]
      .map(([key, group]) => ({
        key,
        count: group.count,
        episode_ids: [...group.episode_ids].sort(),
        predicates: [...group.predicates].sort(),
        taxonomies: [...group.taxonomies].sort(),
        components: [...group.components].sort(),
      }))
      .sort((a, b) => b.count - a.count || a.key.localeCompare(b.key)),
    entropy_warning:
      withheld.length > 0 && unknownRate >= 0.5
        ? "high Funknown attribution rate; inspect failure taxonomy and episode observability"
        : null,
  };
}
