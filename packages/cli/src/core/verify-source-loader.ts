import * as path from "node:path";
import fs from "fs-extra";
import { sha256File, sha256String } from "./hash.js";
import { readYamlOrJson } from "./schema.js";
import { validate as validateClaim } from "../validators/claim.js";
import { validate as validateEvidence } from "../validators/evidence.js";
import { validate as validateSubagentReturn } from "../validators/subagentReturn.js";
import { validate as validateCompletionCard } from "../validators/completionCard.js";

export interface VerifySourceOptions {
  card?: string;
  claim?: string;
  evidence?: string;
  subagentReturn?: string;
}

export interface VerifyLoadedSources {
  errors: string[];
  notes: string[];
  claim?: Record<string, unknown>;
  evidence?: Record<string, unknown>;
  subagentReturn?: Record<string, unknown>;
  card?: Record<string, unknown>;
  cardPath?: string;
  inputCardHash: string | null;
  policyHash: string | null;
}

export class VerifyInputError extends Error {
  constructor(
    message: string,
    readonly details: string[] = [],
    readonly exitCode = 2
  ) {
    super(message);
    this.name = "VerifyInputError";
  }
}

const DEFAULT_CARD_PATHS = [
  "completion-card.yaml",
  "completion-card.yml",
  ".x-harness/completion-card.yaml",
];

export async function resolveCardPath(
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

export async function loadVerifySources(
  cwd: string,
  opts: VerifySourceOptions
): Promise<VerifyLoadedSources> {
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
    cardPath = await resolveCardPath(cwd, opts.card);
    if (!cardPath) {
      throw new VerifyInputError("No completion card found.", [
        "Searched: " + DEFAULT_CARD_PATHS.join(", "),
        "Provide --card <path> or use --claim/--evidence/--subagent-return for compatibility mode.",
      ]);
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
        notes.push(`completion card valid: ${path.relative(cwd, cardPath)}`);
      }
      inputCardHash = sha256String(JSON.stringify(data));
    } catch (err) {
      errors.push(
        `completion card load error: ${err instanceof Error ? err.message : String(err)}`
      );
    }
  }

  const policyPath = path.resolve(cwd, "policies", "admission.yaml");
  try {
    policyHash = await sha256File(policyPath);
  } catch (err) {
    errors.push(
      `policy hash error: could not read ${policyPath} - ${err instanceof Error ? err.message : String(err)}`
    );
  }

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
      errors.push(
        `evidence load error: ${err instanceof Error ? err.message : String(err)}`
      );
    }
  }

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

  return {
    errors,
    notes,
    claim,
    evidence,
    subagentReturn,
    card,
    cardPath,
    inputCardHash,
    policyHash,
  };
}
