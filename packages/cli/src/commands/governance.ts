import { Command } from "commander";
import * as path from "node:path";
import fs from "fs-extra";
import { readYamlOrJson } from "../core/schema.js";
import {
  loadAuthorityPolicy,
  explainPath,
  getProtectedPaths,
  checkGovernance,
} from "../core/authority.js";
import {
  defaultChangedFilesSource,
  getGitDiffFiles,
  resolveChangedFiles,
} from "../core/changed-files.js";
import { validate as validateIntervention } from "../validators/intervention.js";

interface GovernanceCheckOptions {
  card?: string;
  diff?: string;
  changedFilesSource?: string;
  enforce?: boolean;
  json?: boolean;
  root?: string;
}

interface GovernanceExplainOptions {
  path?: string;
  json?: boolean;
  root?: string;
}

interface GovernanceListProtectedOptions {
  json?: boolean;
  root?: string;
}

interface InterventionValidateOptions {
  intervention?: string;
  json?: boolean;
}

async function loadCardGovernanceData(
  cardPath: string
): Promise<{ files: string[]; governance?: Record<string, unknown> }> {
  try {
    const data = await readYamlOrJson(cardPath);
    const card = data as Record<string, unknown>;
    const governance = card.governance as Record<string, unknown> | undefined;

    // Extract files_changed from evidence
    const evidence = card.evidence as Record<string, unknown> | undefined;
    if (evidence) {
      const filesChanged = evidence.files_changed as string[] | undefined;
      if (filesChanged && Array.isArray(filesChanged)) {
        return { files: filesChanged, governance };
      }
    }

    return { files: [], governance };
  } catch {
    return { files: [] };
  }
}

export async function governanceCheckAction(
  opts: GovernanceCheckOptions
): Promise<void> {
  const root = path.resolve(opts.root ?? process.cwd());
  let files: string[] = [];
  let cardFiles: string[] = [];
  let gitFiles: string[] = [];
  let changedFilesSource = "card";
  const changedFileErrors: string[] = [];
  const changedFileNotes: string[] = [];
  let governance: Record<string, unknown> | undefined;

  if (opts.card) {
    const cardPath = path.resolve(root, opts.card);
    if (!(await fs.pathExists(cardPath))) {
      console.error(`Error: Card not found: ${cardPath}`);
      process.exit(2);
    }
    const cardData = await loadCardGovernanceData(cardPath);
    cardFiles = cardData.files;
    governance = cardData.governance;
  }

  if (opts.diff && !opts.card) {
    gitFiles = await getGitDiffFiles(opts.diff, root);
    files = gitFiles;
    changedFilesSource = "git";
  } else if (opts.card) {
    const resolved = await resolveChangedFiles({
      cardFiles,
      diffRef: opts.diff,
      root,
      source: defaultChangedFilesSource({
        explicit: opts.changedFilesSource,
        diffRef: opts.diff,
      }),
    });
    files = resolved.files;
    gitFiles = resolved.git_files;
    changedFilesSource = resolved.source;
    changedFileErrors.push(...resolved.errors);
    changedFileNotes.push(...resolved.notes);
  }

  if (changedFileErrors.length > 0) {
    if (opts.json) {
      console.log(
        JSON.stringify(
          {
            ok: false,
            violations: [],
            warnings: [],
            total_violations: 0,
            total_warnings: 0,
            changed_files: {
              source: changedFilesSource,
              card_files: cardFiles,
              git_files: gitFiles,
              files,
            },
            errors: changedFileErrors,
          },
          null,
          2
        )
      );
    } else {
      for (const error of changedFileErrors) console.error(error);
    }
    process.exit(1);
  }

  if (files.length === 0) {
    if (opts.json) {
      console.log(
        JSON.stringify(
          {
            ok: true,
            violations: [],
            warnings: [],
            total_violations: 0,
            total_warnings: 0,
            changed_files: {
              source: changedFilesSource,
              card_files: cardFiles,
              git_files: gitFiles,
              files,
            },
            message: "No files to check",
          },
          null,
          2
        )
      );
    } else {
      console.log("No files to check for governance violations.");
    }
    process.exit(0);
  }

  const result = await checkGovernance(files, root, {
    enforce: opts.enforce === true,
    governance,
  });

  if (opts.json) {
    console.log(
      JSON.stringify(
        {
          ok: result.total_violations === 0 && result.total_warnings === 0,
          violations: result.violations,
          warnings: result.warnings,
          total_violations: result.total_violations,
          total_warnings: result.total_warnings,
          report_only: result.report_only,
          enforced: result.enforced,
          changed_files: {
            source: changedFilesSource,
            card_files: cardFiles,
            git_files: gitFiles,
            files,
          },
          notes: changedFileNotes,
        },
        null,
        2
      )
    );
  } else {
    if (result.violations.length > 0) {
      console.log("Authority violations (enforced mode):");
      for (const violation of result.violations) {
        console.log(`  [${violation.authority}] ${violation.path}`);
        console.log(`    ${violation.rationale}`);
        if (violation.approval_note) {
          console.log(`    ${violation.approval_note}`);
        }
      }
      console.log("");
      console.log(`Total: ${result.total_violations} violation(s)`);
    } else if (result.warnings.length > 0) {
      console.log("Authority warnings (report-only, no admission block):");
      for (const warning of result.warnings) {
        console.log(`  [${warning.authority}] ${warning.path}`);
        console.log(`    ${warning.rationale}`);
      }
      console.log("");
      console.log(
        `Total: ${result.total_warnings} warning(s) - admission NOT blocked (PR2 report-only)`
      );
    } else {
      console.log("No governance violations found.");
    }
  }

  process.exit(result.total_violations > 0 ? 1 : 0);
}

export async function governanceExplainAction(
  opts: GovernanceExplainOptions
): Promise<void> {
  if (!opts.path) {
    console.error("Error: --path is required");
    process.exit(2);
  }

  const root = path.resolve(opts.root ?? process.cwd());
  const targetPath = path.isAbsolute(opts.path)
    ? opts.path
    : path.resolve(root, opts.path);

  try {
    const result = await explainPath(targetPath, root);

    if (opts.json) {
      console.log(
        JSON.stringify(
          {
            path: result.path,
            authority: result.authority,
            rationale: result.rationale,
          },
          null,
          2
        )
      );
    } else {
      console.log(`Path: ${result.path}`);
      console.log(`Authority: ${result.authority}`);
      console.log(`Rationale: ${result.rationale}`);
    }
  } catch (err) {
    console.error(`Error: ${err instanceof Error ? err.message : String(err)}`);
    process.exit(2);
  }
}

export async function governanceListProtectedAction(
  opts: GovernanceListProtectedOptions
): Promise<void> {
  const root = path.resolve(opts.root ?? process.cwd());

  try {
    const policy = await loadAuthorityPolicy(root);
    const protectedPaths = getProtectedPaths(policy);

    if (opts.json) {
      console.log(
        JSON.stringify(
          {
            authority_classes: policy.authority_classes,
            protected_paths: protectedPaths,
          },
          null,
          2
        )
      );
    } else {
      console.log("Authority classes:");
      for (const [name, cls] of Object.entries(policy.authority_classes)) {
        console.log(`  ${name}: ${cls.description}`);
      }
      console.log("");
      console.log("Protected paths:");
      for (const pp of protectedPaths) {
        console.log(`  ${pp.path} -> ${pp.authority}`);
        console.log(`    ${pp.rationale}`);
      }
    }
  } catch (err) {
    console.error(`Error: ${err instanceof Error ? err.message : String(err)}`);
    process.exit(2);
  }
}

export async function interventionValidateAction(
  opts: InterventionValidateOptions
): Promise<void> {
  if (!opts.intervention) {
    console.error("Error: --intervention <path> is required");
    process.exit(2);
  }

  const interventionPath = path.resolve(opts.intervention);

  if (!(await fs.pathExists(interventionPath))) {
    console.error(`Error: Intervention file not found: ${interventionPath}`);
    process.exit(2);
  }

  try {
    const data = await readYamlOrJson(interventionPath);
    const result = await validateIntervention(data);

    if (opts.json) {
      console.log(
        JSON.stringify(
          {
            valid: result.valid,
            errors: result.errors,
          },
          null,
          2
        )
      );
    } else {
      if (result.valid) {
        console.log("Intervention is valid.");
      } else {
        console.log("Intervention validation failed:");
        for (const error of result.errors) {
          console.log(`  - ${error}`);
        }
      }
    }

    process.exit(result.valid ? 0 : 1);
  } catch (err) {
    console.error(`Error: ${err instanceof Error ? err.message : String(err)}`);
    process.exit(2);
  }
}

export function governanceCommand(): Command {
  const cmd = new Command("governance").description(
    "Governance boundary and authority checking (report-only, no admission block)"
  );

  cmd
    .command("check")
    .description(
      "Check files against authority boundary (report-only warnings, no admission block)"
    )
    .option("--card <path>", "Path to completion card to check files from")
    .option(
      "--diff <ref>",
      "Git ref to diff against (e.g., HEAD) to get changed files"
    )
    .option(
      "--changed-files-source <mode>",
      "Changed files source when --card and --diff are both set: card, git, union, strict"
    )
    .option(
      "--enforce",
      "Exit non-zero for unapproved authority violations",
      false
    )
    .option("--json", "Output JSON instead of human-readable text", false)
    .option("--root <path>", "Repository root", process.cwd())
    .action(governanceCheckAction);

  cmd
    .command("explain")
    .description("Explain authority class for a given path")
    .option("--path <path>", "Path to explain authority for")
    .option("--json", "Output JSON instead of human-readable text", false)
    .option("--root <path>", "Repository root", process.cwd())
    .action(governanceExplainAction);

  cmd
    .command("list-protected")
    .description("List all protected paths and their authority classes")
    .option("--json", "Output JSON instead of human-readable text", false)
    .option("--root <path>", "Repository root", process.cwd())
    .action(governanceListProtectedAction);

  return cmd;
}

export function interventionCommand(): Command {
  const cmd = new Command("intervention").description(
    "Intervention artifact validation"
  );

  cmd
    .command("validate")
    .description("Validate an intervention artifact against the schema")
    .option("--intervention <path>", "Path to intervention YAML/JSON file")
    .option("--json", "Output JSON instead of human-readable text", false)
    .action(interventionValidateAction);

  return cmd;
}
