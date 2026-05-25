import { checkGovernance } from "./authority.js";
import {
  defaultChangedFilesSource,
  resolveChangedFiles,
  type ChangedFilesResolution,
} from "./changed-files.js";

export interface VerifyGovernanceOptions {
  strict?: boolean;
  governanceEnforced?: boolean;
  diff?: string;
  changedFilesSource?: string;
}

export interface VerifyGovernanceResult {
  changedFiles?: ChangedFilesResolution;
  errors: string[];
  notes: string[];
}

function filesChangedFromCard(card?: Record<string, unknown>): string[] {
  const evidence = card?.evidence as Record<string, unknown> | undefined;
  const filesChanged = evidence?.files_changed;
  return Array.isArray(filesChanged)
    ? filesChanged.filter((item): item is string => typeof item === "string")
    : [];
}

export async function runVerifyGovernance(input: {
  card?: Record<string, unknown>;
  cwd: string;
  opts: VerifyGovernanceOptions;
}): Promise<VerifyGovernanceResult> {
  const errors: string[] = [];
  const notes: string[] = [];
  if (!input.card) return { errors, notes };

  try {
    const changedFiles = await resolveChangedFiles({
      cardFiles: filesChangedFromCard(input.card),
      diffRef: input.opts.diff,
      root: input.cwd,
      source: defaultChangedFilesSource({
        explicit: input.opts.changedFilesSource,
        diffRef: input.opts.diff,
        strict: input.opts.strict,
      }),
    });
    errors.push(...changedFiles.errors);
    notes.push(...changedFiles.notes);

    const governanceResult = await checkGovernance(
      changedFiles.files,
      input.cwd,
      {
        enforce:
          input.opts.governanceEnforced === true || input.opts.strict === true,
        governance: input.card.governance as
          | Record<string, unknown>
          | undefined,
      }
    );
    for (const warning of governanceResult.warnings) {
      notes.push(
        `governance warning: ${warning.authority} path ${warning.path}: ${warning.approval_note ?? warning.rationale}`
      );
    }
    if (governanceResult.enforced) {
      notes.push("governance enforced mode enabled");
      for (const violation of governanceResult.violations) {
        errors.push(
          `governance permission violation: ${violation.authority} path ${violation.path}: ${violation.approval_note ?? violation.rationale}`
        );
      }
    }
    return { changedFiles, errors, notes };
  } catch (err) {
    if (
      input.opts.governanceEnforced === true ||
      Boolean(input.opts.diff) ||
      Boolean(input.opts.changedFilesSource)
    ) {
      errors.push(
        `governance enforcement error: ${err instanceof Error ? err.message : String(err)}`
      );
    } else {
      notes.push(
        `governance check skipped: ${err instanceof Error ? err.message : String(err)}`
      );
    }
    return { errors, notes };
  }
}
