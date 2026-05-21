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
  standard: ["examples/01-light-task", "examples/02-standard-task", "schemas", "policies"],
  full: ["examples", "schemas", "policies", "templates", "adapters"],
};

export function initCommand(): Command {
  return new Command("init")
    .description("Initialize x-harness files")
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

      const plan: { src: string; dest: string }[] = [];

      if (mode === "minimal") {
        const minimalFiles = [
          { src: "AGENTS.md", dest: "AGENTS.md" },
          { src: "X_HARNESS.md", dest: "X_HARNESS.md" },
          { src: "docs/VERIFY_GATE.md", dest: "docs/VERIFY_GATE.md" },
          { src: "docs/RUNTIME_CONTRACT.md", dest: "docs/RUNTIME_CONTRACT.md" },
          { src: "templates/SUBAGENT_TASK_light.md", dest: "templates/SUBAGENT_TASK_light.md" },
          { src: "templates/SUBAGENT_TASK_standard.md", dest: "templates/SUBAGENT_TASK_standard.md" },
          { src: "templates/SUBAGENT_TASK_deep.md", dest: "templates/SUBAGENT_TASK_deep.md" },
          { src: "templates/COMPLETION_CARD.md", dest: "templates/COMPLETION_CARD.md" },
          { src: "policies/admission.yaml", dest: "policies/admission.yaml" },
        ];
        for (const f of minimalFiles) {
          const src = path.join(rootDir, f.src);
          if (await fs.pathExists(src)) {
            plan.push({ src, dest: path.join(targetDir, f.dest) });
          }
        }
      } else {
        const assets = MODE_ASSETS[mode];
        for (const asset of assets) {
          const src = path.join(rootDir, asset);
          if (!(await fs.pathExists(src))) continue;
          const dest = path.join(targetDir, path.basename(asset));
          plan.push({ src, dest });
        }
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
        console.log(`# x-harness init (${mode}) - dry run`);
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

      console.log(`x-harness init (${mode}) complete: ${plan.length} assets copied to ${targetDir}`);
    });
}
