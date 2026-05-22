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
      if (opts.archiveSuccess) {
        const cardPath = path.join(cwd, "completion-card.yaml");
        if (await fs.pathExists(cardPath)) {
          try {
            const content = await fs.readFile(cardPath, "utf-8");
            const YAML = await import("yaml");
            const data = YAML.parse(content) as Record<string, unknown>;
            if (
              data.acceptance_status === "accepted" &&
              (data.admission as Record<string, unknown>)?.outcome === "success"
            ) {
              const archiveDir = path.join(cwd, ".x-harness", "archive");
              const archiveName = `completion-card-${Date.now()}.yaml`;
              const archivePath = path.join(archiveDir, archiveName);
              actions.push({
                type: "move",
                path: `${cardPath} -> ${archivePath}`,
                note: "archive successful card",
              });
            } else {
              console.log(
                "Current completion card is not accepted; skipping archive."
              );
            }
          } catch {
            console.log(
              "Could not parse completion-card.yaml; skipping archive."
            );
          }
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
        console.log("# x-harness clean (dry-run)");
        for (const a of safeActions) {
          console.log(`would ${a.type}: ${a.path} (${a.note})`);
        }
        console.log("\nTo apply, run again with --force");
        process.exit(0);
      }

      // Execute mutations
      console.log("# x-harness clean (applying)");
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
