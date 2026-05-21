import { Command } from "commander";
import * as path from "node:path";
import fs from "fs-extra";

const CRITICAL_ASSETS = [
  "AGENTS.md",
  "X_HARNESS.md",
  "schemas/completion-card.schema.json",
  "schemas/subagent-return.schema.json",
  "schemas/verify-event.schema.json",
  "schemas/pgv-advice.schema.json",
  "policies/admission.yaml",
];

export function doctorCommand(): Command {
  return new Command("doctor")
    .description("Check required files, schemas, policies, templates, and adapters")
    .option("--root <path>", "Repository root", process.cwd())
    .action(async (opts: { root: string }) => {
      const root = path.resolve(opts.root);
      const missing: string[] = [];
      const present: string[] = [];
      const notes: string[] = [];

      for (const asset of CRITICAL_ASSETS) {
        const assetPath = path.join(root, asset);
        if (await fs.pathExists(assetPath)) {
          present.push(asset);
        } else {
          missing.push(asset);
        }
      }

      // Check templates
      const templatesDir = path.join(root, "templates");
      if (await fs.pathExists(templatesDir)) {
        present.push("templates/");
      } else {
        notes.push("templates/ directory not found (optional)");
      }

      // Check examples
      const examplesDir = path.join(root, "examples");
      if (await fs.pathExists(examplesDir)) {
        present.push("examples/");
      } else {
        notes.push("examples/ directory not found (optional)");
      }

      // Check no core Python in packages/cli (file-first philosophy)
      const cliSrc = path.join(root, "packages", "cli", "src");
      let pythonFiles = 0;
      if (await fs.pathExists(cliSrc)) {
        const walk = async (dir: string) => {
          const entries = await fs.readdir(dir, { withFileTypes: true });
          for (const entry of entries) {
            const fullPath = path.join(dir, entry.name);
            if (entry.isDirectory()) {
              await walk(fullPath);
            } else if (entry.name.endsWith(".py")) {
              pythonFiles++;
            }
          }
        };
        await walk(cliSrc);
      }

      if (pythonFiles > 0) {
        missing.push(`python files in packages/cli/src (${pythonFiles} found)`);
      } else {
        notes.push("no Python files in packages/cli/src");
      }

      const healthy = missing.length === 0;

      const report = {
        healthy,
        present_count: present.length,
        missing_count: missing.length,
        present,
        missing,
        notes,
      };

      console.log(JSON.stringify(report, null, 2));
      process.exit(healthy ? 0 : 1);
    });
}
