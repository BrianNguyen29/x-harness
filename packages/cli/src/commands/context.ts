import { Command } from "commander";
import * as path from "node:path";
import fs from "fs-extra";
import {
  getCanonicalContext,
  getContextHash,
  generateManagedBlock,
  injectManagedBlock,
  validateManagedBlock,
} from "../core/context.js";

interface ContextOptions {
  verbose?: boolean;
  json?: boolean;
  refresh?: boolean;
  root?: string;
}

export function contextCommand(): Command {
  return new Command("context")
    .description("Show x-harness canonical context and refresh AGENTS.md")
    .option("--verbose", "Show full verbose context", false)
    .option("--json", "Output context as JSON", false)
    .option("--refresh", "Refresh AGENTS.md managed context block", false)
    .option("--root <path>", "Repository root", process.cwd())
    .action(async (opts: ContextOptions) => {
      const root = path.resolve(opts.root ?? process.cwd());

      if (opts.refresh) {
        const agentsPath = path.join(root, "AGENTS.md");
        if (!(await fs.pathExists(agentsPath))) {
          console.error(`Error: AGENTS.md not found at ${agentsPath}`);
          process.exit(2);
        }
        const agentsContent = await fs.readFile(agentsPath, "utf-8");
        const block = generateManagedBlock();
        const updated = injectManagedBlock(agentsContent, block);
        await fs.writeFile(agentsPath, updated, "utf-8");
        const hashMatch = block.match(/<!-- context-hash: ([a-f0-9]+) -->/);
        const hash = hashMatch ? hashMatch[1] : "unknown";
        console.log(`AGENTS.md refreshed (context-hash: ${hash})`);
        return;
      }

      const context = getCanonicalContext(opts.verbose);
      const hash = getContextHash(context);

      if (opts.json) {
        const agentsPath = path.join(root, "AGENTS.md");
        const agentsFresh = (await fs.pathExists(agentsPath))
          ? validateManagedBlock(await fs.readFile(agentsPath, "utf-8"))
          : { valid: false, note: "AGENTS.md not found" };

        console.log(
          JSON.stringify(
            {
              context,
              hash,
              mode: opts.verbose ? "verbose" : "compact",
              agents_fresh: agentsFresh.valid,
              agents_note: agentsFresh.note,
            },
            null,
            2
          )
        );
        return;
      }

      console.log(context);
      if (opts.verbose) {
        console.log("");
        console.log(`context-hash: ${hash}`);
      }
    });
}
