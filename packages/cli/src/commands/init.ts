import { Command } from "commander";
import * as path from "node:path";
import fs from "fs-extra";
import { resolveAssetRoot } from "../core/assets.js";

interface InitOptions {
  minimal?: boolean;
  standard?: boolean;
  full?: boolean;
  dryRun?: boolean;
  preview?: boolean;
  apply?: boolean;
  merge?: boolean;
  force?: boolean;
  adapters?: string;
  profile?: string;
}

const MODE_ASSETS: Record<string, string[]> = {
  minimal: ["examples/00-minimal"],
  standard: [
    "examples/01-solo-agent",
    "examples/02-assisted-agent",
    "schemas",
    "policies",
  ],
  full: ["examples", "schemas", "policies", "templates", "adapters"],
};

const FULL_PROFILE_CORE_ASSETS = [
  "AGENTS.md",
  "X_HARNESS.md",
  ".github/workflows/x-harness-verify.yml",
  ".x-harness/managed-blocks.yaml",
  "docs",
  "components",
  "tools",
];

export function initCommand(): Command {
  return new Command("init")
    .description("Initialize x-harness files")
    .option("--minimal", "Minimal mode (default)")
    .option("--standard", "Standard mode")
    .option("--full", "Full mode")
    .option("--profile <profile>", "Install profile (minimal, standard, deep)")
    .option("--dry-run", "Show what would be copied")
    .option("--preview", "Preview changes (alias for --dry-run)")
    .option("--apply", "Apply changes explicitly")
    .option("--merge", "Merge with existing files")
    .option("--force", "Overwrite existing files")
    .option("--adapters <list>", "Comma-separated adapter list")
    .argument("[target]", "Target directory", ".")
    .action(async (target: string, opts: InitOptions) => {
      let mode = opts.full ? "full" : opts.standard ? "standard" : "minimal";
      const legacyModeSet = !!(opts.minimal || opts.standard || opts.full);

      if (opts.profile) {
        if (legacyModeSet) {
          console.error("usage: init [target] [options]");
          console.error(
            "cannot use --profile with --minimal, --standard, or --full"
          );
          process.exit(1);
        }
        switch (opts.profile) {
          case "minimal":
          case "standard":
            mode = opts.profile;
            break;
          case "deep":
            mode = "full";
            break;
          default:
            console.error("usage: init [target] [options]");
            console.error(`invalid profile: ${opts.profile}`);
            process.exit(1);
        }
      }

      const displayMode = opts.profile === "deep" ? "deep" : mode;
      const dryRun = opts.dryRun || opts.preview || false;
      const targetDir = path.resolve(target);
      const assetRoot = await resolveAssetRoot();

      const plan: { src: string; dest: string }[] = [];

      if (mode === "minimal") {
        const minimalFiles = [
          { src: "AGENTS.md", dest: "AGENTS.md" },
          { src: "X_HARNESS.md", dest: "X_HARNESS.md" },
          { src: "docs/VERIFY_GATE.md", dest: "docs/VERIFY_GATE.md" },
          { src: "docs/RUNTIME_CONTRACT.md", dest: "docs/RUNTIME_CONTRACT.md" },
          {
            src: "templates/SUBAGENT_TASK_light.md",
            dest: "templates/SUBAGENT_TASK_light.md",
          },
          {
            src: "templates/SUBAGENT_TASK_standard.md",
            dest: "templates/SUBAGENT_TASK_standard.md",
          },
          {
            src: "templates/SUBAGENT_TASK_deep.md",
            dest: "templates/SUBAGENT_TASK_deep.md",
          },
          {
            src: "templates/COMPLETION_CARD.md",
            dest: "templates/COMPLETION_CARD.md",
          },
          { src: "policies/admission.yaml", dest: "policies/admission.yaml" },
        ];
        for (const f of minimalFiles) {
          const src = path.join(assetRoot, f.src);
          if (await fs.pathExists(src)) {
            plan.push({ src, dest: path.join(targetDir, f.dest) });
          }
        }
      } else {
        const assets = MODE_ASSETS[mode];
        for (const asset of assets) {
          const src = path.join(assetRoot, asset);
          if (!(await fs.pathExists(src))) continue;
          const dest = path.join(targetDir, path.basename(asset));
          plan.push({ src, dest });
        }

        if (mode === "full") {
          for (const asset of FULL_PROFILE_CORE_ASSETS) {
            const src = path.join(assetRoot, asset);
            if (!(await fs.pathExists(src))) continue;
            plan.push({ src, dest: path.join(targetDir, asset) });
          }
        }

        // Include adapter guidance in standard mode
        if (mode === "standard") {
          const adaptersDocSrc = path.join(assetRoot, "docs/ADAPTERS.md");
          if (await fs.pathExists(adaptersDocSrc)) {
            plan.push({
              src: adaptersDocSrc,
              dest: path.join(targetDir, "docs/ADAPTERS.md"),
            });
          }
        }
      }

      // Copy adapter-specific files if requested
      if (opts.adapters) {
        for (const adapter of opts.adapters.split(",").map((a) => a.trim())) {
          const src = path.join(assetRoot, "adapters", adapter);
          if (await fs.pathExists(src)) {
            plan.push({ src, dest: path.join(targetDir, "adapters", adapter) });
          }
        }

        // Include adapter guidance when adapters are requested
        const adaptersDocSrc = path.join(assetRoot, "docs/ADAPTERS.md");
        const adaptersDocDest = path.join(targetDir, "docs/ADAPTERS.md");
        if (await fs.pathExists(adaptersDocSrc)) {
          const alreadyPlanned = plan.some((p) => p.dest === adaptersDocDest);
          if (!alreadyPlanned) {
            plan.push({ src: adaptersDocSrc, dest: adaptersDocDest });
          }
        }
      }

      if (dryRun) {
        console.log(`# x-harness init (${displayMode}) - dry run`);
        for (const p of plan) {
          console.log(`would copy: ${p.src} -> ${p.dest}`);
        }
        return;
      }

      const conflicts: string[] = [];
      const copied: string[] = [];

      for (const p of plan) {
        if (await fs.pathExists(p.dest)) {
          if (!opts.force && !opts.merge) {
            conflicts.push(p.dest);
            continue;
          }
          if (opts.force) {
            await fs.remove(p.dest);
          } else if (opts.merge) {
            // Merge mode: preserve existing files, copy only missing ones
            continue;
          }
        }
        await fs.copy(p.src, p.dest);
        copied.push(p.dest);
        console.log(`copied: ${p.dest}`);
      }

      if (conflicts.length > 0) {
        console.error(
          `\n# x-harness init (${displayMode}) blocked: ${conflicts.length} file(s) already exist`
        );
        for (const c of conflicts) {
          console.error(`conflict: ${c}`);
        }
        console.error(
          "\nUse --force to overwrite or --merge to merge with existing files."
        );
        process.exit(1);
      }

      console.log(
        `x-harness init (${displayMode}) complete: ${copied.length} assets copied to ${targetDir}`
      );
    });
}
