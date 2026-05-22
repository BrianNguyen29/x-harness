import { Command } from "commander";
import fs from "fs-extra";
import * as path from "node:path";
import YAML from "yaml";

export function addCommand(): Command {
  return new Command("add")
    .description("Add a module file (claim, evidence, completion-card)")
    .argument("<module>", "Module type: claim, evidence, completion-card")
    .argument("[values]", "Comma-separated key=value pairs")
    .option("--out <path>", "Output file path")
    .action(
      async (
        module: string,
        values: string | undefined,
        opts: { out?: string }
      ) => {
        const data: Record<string, unknown> = {
          id: `${module.toUpperCase()}-${Date.now()}`,
          created_at: new Date().toISOString(),
        };

        if (values) {
          for (const pair of values.split(",")) {
            const [k, v] = pair.split("=");
            if (k && v !== undefined) {
              data[k] = v;
            }
          }
        }

        const ext = module === "completion-card" ? "yaml" : "yaml";
        const outPath = opts.out ?? `${module}.${ext}`;

        const yamlContent = YAML.stringify(data);

        await fs.writeFile(path.resolve(outPath), yamlContent, "utf-8");
        console.log(`Added ${module} -> ${outPath}`);
      }
    );
}
