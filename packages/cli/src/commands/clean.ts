import { Command } from "commander";
import * as path from "node:path";
import fs from "fs-extra";

interface CleanOptions {
  dryRun?: boolean;
  tmp?: boolean;
  resetCard?: boolean;
  archiveSuccess?: boolean;
  force?: boolean;
}

export async function cleanTmpAction(): Promise<void> {
  const cwd = process.cwd();
  console.log("# xh clean --tmp --force");
  for (const dir of [".x-harness/tmp", ".x-harness/cache"]) {
    const fullPath = path.join(cwd, dir);
    if (await fs.pathExists(fullPath)) {
      await fs.remove(fullPath);
      console.log(`deleted: ${dir}/`);
    } else {
      console.log(`not found (skipping): ${dir}/`);
    }
  }
  console.log("\nreset complete.");
}

const PROTECTED_PATHS = [
  "AGENTS.md",
  "X_HARNESS.md",
  "README.md",
  "templates",
  "schemas",
  "policies",
  "docs",
  "adapters",
  "examples",
  "packages",
  ".git",
];

function isProtected(targetPath: string, cwd: string): boolean {
  const rel = path.relative(cwd, targetPath);
  const parts = rel.split(path.sep);
  for (const part of parts) {
    if (PROTECTED_PATHS.includes(part)) return true;
  }
  return false;
}

export function cleanCommand(): Command {
  return new Command("clean")
    .description("Clean x-harness artifacts safely (default: dry-run)")
    .option("--dry-run", "Show what would be cleaned without deleting", true)
    .option("--tmp", "Clean .x-harness/tmp/ and .x-harness/cache/")
    .option("--reset-card", "Reset completion-card.yaml by renaming to backup")
    .option(
      "--archive-success",
      "Move accepted completion cards to .x-harness/archive/"
    )
    .option(
      "--force",
      "Actually perform mutations (required when not using --dry-run)"
    )
    .action(async (opts: CleanOptions) => {
      const cwd = process.cwd();
      const actions: { type: string; path: string; note: string }[] = [];
      const wouldMutate = opts.tmp || opts.resetCard || opts.archiveSuccess;
      const dryRun = opts.dryRun !== false && (!opts.force || !wouldMutate);

      // --tmp: clean tmp/cache
      if (opts.tmp) {
        for (const dir of [".x-harness/tmp", ".x-harness/cache"]) {
          const fullPath = path.join(cwd, dir);
          if (await fs.pathExists(fullPath)) {
            actions.push({
              type: "delete",
              path: fullPath,
              note: `remove ${dir}`,
            });
          }
        }
      }

      // --reset-card: rename completion-card.yaml to backup
      if (opts.resetCard) {
        const cardPath = path.join(cwd, "completion-card.yaml");
        if (await fs.pathExists(cardPath)) {
          const backupPath = path.join(
            cwd,
            `completion-card.yaml.bak.${Date.now()}`
          );
          actions.push({
            type: "rename",
            path: `${cardPath} -> ${backupPath}`,
            note: "reset completion card",
          });
        } else {
          console.log("No completion-card.yaml found to reset.");
        }
      }

      // --archive-success: move accepted cards to archive
      // Note: content check is deferred to execution time to avoid TOCTOU window
      if (opts.archiveSuccess) {
        const cardPath = path.join(cwd, "completion-card.yaml");
        if (await fs.pathExists(cardPath)) {
          actions.push({
            type: "archive-accept",
            path: cardPath,
            note: "archive accepted card",
          });
        } else {
          console.log("No completion-card.yaml found to archive.");
        }
      }

      // Safety: filter out protected paths
      const safeActions = actions.filter((a) => {
        const target = a.path.split(" -> ")[0];
        if (isProtected(target, cwd)) {
          console.log(`SKIPPED (protected): ${a.path}`);
          return false;
        }
        return true;
      });

      if (safeActions.length === 0) {
        console.log("Nothing to clean.");
        console.log(
          "Use --tmp, --reset-card, or --archive-success to specify what to clean."
        );
        process.exit(0);
      }

      if (dryRun) {
        console.log("# xh clean (dry-run)");
        for (const a of safeActions) {
          console.log(`would ${a.type}: ${a.path} (${a.note})`);
        }
        console.log("\nTo apply, run again with --force");
        process.exit(0);
      }

      // Execute mutations
      console.log("# xh clean (applying)");
      for (const a of safeActions) {
        try {
          if (a.type === "delete") {
            await fs.remove(a.path);
            console.log(`deleted: ${a.path}`);
          } else if (a.type === "rename") {
            const [src, dest] = a.path.split(" -> ");
            await fs.ensureDir(path.dirname(dest));
            await fs.move(src.trim(), dest.trim());
            console.log(`renamed: ${a.path}`);
          } else if (a.type === "move") {
            const [src, dest] = a.path.split(" -> ");
            await fs.ensureDir(path.dirname(dest.trim()));
            await fs.move(src.trim(), dest.trim());
            console.log(`moved: ${a.path}`);
          } else if (a.type === "archive-accept") {
            // Re-verify content at move time to avoid TOCTOU window
            const cardPath = a.path;
            const content = await fs.readFile(cardPath, "utf-8");
            const YAML = await import("yaml");
            const data = YAML.parse(content) as Record<string, unknown>;
            if (
              data.acceptance_status !== "accepted" ||
              (data.admission as Record<string, unknown>)?.outcome !== "success"
            ) {
              console.log(
                "Current completion card is not accepted; skipping archive."
              );
              continue;
            }
            const archiveDir = path.join(cwd, ".x-harness", "archive");
            const archiveName = `completion-card-${Date.now()}.yaml`;
            const archivePath = path.join(archiveDir, archiveName);
            await fs.ensureDir(archiveDir);
            await fs.move(cardPath, archivePath);
            console.log(`archived: ${cardPath} -> ${archivePath}`);
          }
        } catch (err) {
          console.error(
            `failed: ${a.path} - ${err instanceof Error ? err.message : String(err)}`
          );
        }
      }
      console.log("\nclean complete.");
    });
}
