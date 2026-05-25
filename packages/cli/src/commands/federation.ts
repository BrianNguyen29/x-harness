import { Command } from "commander";
import * as path from "node:path";
import {
  exportFederationPatterns,
  importFederationPatterns,
  validateFederationPatternFile,
} from "../core/federation.js";
import { CliError } from "../core/exit.js";

interface ExportOptions {
  root?: string;
  index?: string;
  out?: string;
  tenant?: string;
  source?: string;
  optIn?: boolean;
  redacted?: boolean;
  benchmarkReport?: string;
  policy?: string;
  json?: boolean;
}

interface ImportOptions {
  root?: string;
  target?: string;
  dryRun?: boolean;
  merge?: boolean;
  force?: boolean;
  json?: boolean;
}

function printJson(data: unknown): void {
  console.log(JSON.stringify(data, null, 2));
}

export function federationCommand(): Command {
  const federation = new Command("federation").description(
    "Export and import anonymized federation patterns; disabled by default"
  );

  federation
    .command("export-patterns")
    .description("Export anonymized failure patterns from an evidence index")
    .option("--root <path>", "Repository root", process.cwd())
    .option("--index <path>", "Evidence index path", "evidence/index.jsonl")
    .requiredOption("--out <path>", "Output JSONL pattern file")
    .requiredOption("--tenant <id>", "Tenant boundary id used for hashing")
    .option("--source <id>", "Source repository id used for hashing", "local")
    .option("--opt-in", "Explicitly opt in to local federation export", false)
    .option("--redacted", "Require redacted/anonymized output", false)
    .option("--benchmark-report <path>", "Optional benchmark report JSON")
    .option("--policy <path>", "Federation policy path")
    .option("--json", "Output JSON summary", false)
    .action(async (opts: ExportOptions) => {
      try {
        const result = await exportFederationPatterns({
          root: path.resolve(opts.root ?? process.cwd()),
          indexPath: opts.index ?? "evidence/index.jsonl",
          outPath: opts.out as string,
          tenant: opts.tenant as string,
          source: opts.source ?? "local",
          optIn: Boolean(opts.optIn),
          redacted: Boolean(opts.redacted),
          benchmarkReportPath: opts.benchmarkReport,
          policyPath: opts.policy,
        });
        if (opts.json) printJson(result);
        else {
          console.log(`federation patterns written: ${result.out_path}`);
          console.log(`records: ${result.record_count}`);
        }
      } catch (err) {
        throw new CliError(err instanceof Error ? err.message : String(err), 2);
      }
    });

  federation
    .command("import-patterns")
    .description("Validate and optionally store anonymized federation patterns")
    .argument("<patterns>", "Federation pattern JSONL or JSON file")
    .option("--root <path>", "Repository root", process.cwd())
    .option(
      "--target <path>",
      "Imported pattern store path",
      ".x-harness/federation/imported-patterns.jsonl"
    )
    .option("--dry-run", "Preview import without writing", true)
    .option("--merge", "Merge records into the target store", false)
    .option("--force", "Overwrite the target store", false)
    .option("--json", "Output JSON summary", false)
    .action(async (patterns: string, opts: ImportOptions) => {
      const dryRun = !opts.merge && !opts.force;
      const result = await importFederationPatterns({
        root: path.resolve(opts.root ?? process.cwd()),
        patternsPath: patterns,
        targetPath:
          opts.target ?? ".x-harness/federation/imported-patterns.jsonl",
        dryRun,
        merge: opts.merge,
        force: opts.force,
      });
      if (opts.json) printJson(result);
      else if (result.ok) {
        console.log(
          dryRun
            ? `federation import dry-run: ${result.planned_count} record(s)`
            : `federation import wrote ${result.written_count} record(s)`
        );
      } else {
        console.log("federation import failed:");
        for (const error of result.errors) console.log(`- ${error}`);
      }
      if (!result.ok) {
        throw new CliError("federation import failed", 1);
      }
    });

  federation
    .command("validate")
    .description("Validate a federation pattern file")
    .argument("<patterns>", "Federation pattern JSONL or JSON file")
    .option("--json", "Output JSON summary", false)
    .action(async (patterns: string, opts: { json?: boolean }) => {
      const result = await validateFederationPatternFile(patterns);
      const output = {
        ok: result.ok,
        record_count: result.patterns.length,
        errors: result.errors,
        admission_authority: false,
      };
      if (opts.json) printJson(output);
      else if (result.ok) {
        console.log(`federation patterns valid: ${result.patterns.length}`);
      } else {
        console.log("federation patterns invalid:");
        for (const error of result.errors) console.log(`- ${error}`);
      }
      if (!result.ok) {
        throw new CliError("federation validation failed", 1);
      }
    });

  return federation;
}
