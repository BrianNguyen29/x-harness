import { Command } from "commander";
import * as path from "node:path";
import fs from "fs-extra";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

interface InitOptions {
  minimal?: boolean;
  standard?: boolean;
  full?: boolean;
  dryRun?: boolean;
  merge?: boolean;
  force?: boolean;
  adapters?: string;
}

const MODE_ASSETS: Record<string, string[]> = {
  minimal: ["examples/00-minimal"],
  standard: ["examples/01-light-task", "examples/02-standard-feature", "schemas", "policies"],
  full: ["examples", "schemas", "policies", "templates", "adapters"],
};

export function initCommand(): Command {
  return new Command("init")
    .description("Initialize ClaimGate files")
    .option("--minimal", "Minimal mode")
    .option("--standard", "Standard mode (default)")
    .option("--full", "Full mode")
    .option("--dry-run", "Show what would be copied")
    .option("--merge", "Merge with existing files")
    .option("--force", "Overwrite existing files")
    .option("--adapters <list>", "Comma-separated adapter list")
    .argument("[target]", "Target directory", ".")
    .action(async (target: string, opts: InitOptions) => {
      const mode = opts.full ? "full" : opts.minimal ? "minimal" : "standard";
      const targetDir = path.resolve(target);
      const rootDir = path.resolve(path.join(__dirname, "..", "..", "..", ".."));
      const assets = MODE_ASSETS[mode];

      const plan: { src: string; dest: string }[] = [];

      for (const asset of assets) {
        const src = path.join(rootDir, asset);
        if (!(await fs.pathExists(src))) continue;
        const dest = path.join(targetDir, path.basename(asset));
        plan.push({ src, dest });
      }

      // Copy adapter-specific files if requested
      if (opts.adapters) {
        for (const adapter of opts.adapters.split(",").map((a) => a.trim())) {
          const src = path.join(rootDir, "adapters", adapter);
          if (await fs.pathExists(src)) {
            plan.push({ src, dest: path.join(targetDir, "adapters", adapter) });
          }
        }
      }

      if (opts.dryRun) {
        console.log(`# ClaimGate init (${mode}) - dry run`);
        for (const p of plan) {
          console.log(`would copy: ${p.src} -> ${p.dest}`);
        }
        return;
      }

      for (const p of plan) {
        if (await fs.pathExists(p.dest)) {
          if (!opts.force && !opts.merge) {
            console.log(`skip (exists): ${p.dest}`);
            continue;
          }
          if (opts.force) {
            await fs.remove(p.dest);
          }
        }
        await fs.copy(p.src, p.dest);
        console.log(`copied: ${p.dest}`);
      }

      console.log(`ClaimGate init (${mode}) complete: ${plan.length} assets copied to ${targetDir}`);
    });
}
