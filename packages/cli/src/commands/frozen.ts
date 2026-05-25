import { Command } from "commander";
import * as path from "node:path";
import {
  exportFrozenBundle,
  importFrozenBundle,
  verifyFrozenBundle,
} from "../core/frozen.js";
import { CliError } from "../core/exit.js";

interface FrozenExportOptions {
  frozen?: boolean;
  root?: string;
  out?: string;
  json?: boolean;
}

interface FrozenImportOptions {
  frozen?: boolean;
  target?: string;
  dryRun?: boolean;
  merge?: boolean;
  force?: boolean;
  json?: boolean;
}

interface FrozenVerifyOptions {
  json?: boolean;
}

function printJson(data: unknown): void {
  console.log(JSON.stringify(data, null, 2));
}

export function frozenExportCommand(): Command {
  return new Command("export")
    .description("Export a frozen x-harness bundle")
    .option("--frozen", "Export frozen harness bundle", false)
    .requiredOption("--out <path>", "Output bundle path")
    .option("--root <path>", "Repository root", process.cwd())
    .option("--json", "Output JSON instead of text", false)
    .action(async (opts: FrozenExportOptions) => {
      if (!opts.frozen) {
        throw new CliError("export currently requires --frozen", 2);
      }
      const result = await exportFrozenBundle({
        root: path.resolve(opts.root ?? process.cwd()),
        out: opts.out as string,
      });
      if (opts.json) printJson(result);
      else {
        console.log(`frozen bundle written: ${result.out}`);
        console.log(`files: ${result.file_count}`);
      }
    });
}

export function frozenImportCommand(): Command {
  return new Command("import")
    .description("Import a frozen x-harness bundle")
    .argument("<bundle>", "Frozen bundle path")
    .option("--frozen", "Import frozen harness bundle", false)
    .requiredOption("--target <path>", "Target repository path")
    .option("--dry-run", "Preview import without writing", true)
    .option("--merge", "Write missing files but preserve existing files", false)
    .option("--force", "Overwrite existing files", false)
    .option("--json", "Output JSON instead of text", false)
    .action(async (bundle: string, opts: FrozenImportOptions) => {
      if (!opts.frozen) {
        throw new CliError("import currently requires --frozen", 2);
      }
      const dryRun = !opts.merge && !opts.force;
      const result = await importFrozenBundle({
        bundlePath: bundle,
        target: opts.target as string,
        dryRun,
        merge: opts.merge,
        force: opts.force,
      });
      if (opts.json) printJson(result);
      else {
        console.log(
          dryRun
            ? `frozen import dry-run: ${result.planned.length} file(s)`
            : `frozen import wrote ${result.written.length} file(s)`
        );
        for (const conflict of result.conflicts) {
          console.log(`conflict: ${conflict}`);
        }
      }
      if (!result.ok) {
        throw new CliError("frozen import failed", 1);
      }
    });
}

export function frozenCommand(): Command {
  const frozen = new Command("frozen").description(
    "Verify frozen x-harness transfer bundles"
  );
  frozen
    .command("verify")
    .description("Verify frozen bundle manifest and checksums")
    .argument("<bundle>", "Frozen bundle path")
    .option("--json", "Output JSON instead of text", false)
    .action(async (bundle: string, opts: FrozenVerifyOptions) => {
      const result = await verifyFrozenBundle(bundle);
      if (opts.json) printJson(result);
      else if (result.ok) {
        console.log(`frozen bundle valid: ${result.bundle_path}`);
        console.log(`files: ${result.file_count}`);
      } else {
        console.log("frozen bundle invalid:");
        for (const error of result.errors) console.log(`- ${error}`);
      }
      if (!result.ok) {
        throw new CliError("frozen verify failed", 1);
      }
    });
  return frozen;
}
