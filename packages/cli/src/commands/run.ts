import { Command } from "commander";
import * as path from "node:path";

interface RunStep {
  name: string;
  status: string;
  note?: string;
}

interface RunResult {
  recipe: string;
  ok: boolean;
  steps: RunStep[];
}

/**
 * TS run is a preview/planner: it lists planned steps without executing them.
 * The Go CLI executes built-in CI steps directly.
 */
export function runCommand(): Command {
  return new Command("run")
    .description("Run a built-in workflow recipe")
    .option("--list", "List available recipes", false)
    .option("--dry-run", "Print planned steps without executing", false)
    .option("--json", "Output JSON instead of text", false)
    .argument("[recipe]", "Recipe name (e.g. builtin:ci)")
    .action(
      async (
        recipe: string | undefined,
        opts: { list: boolean; dryRun: boolean; json: boolean }
      ) => {
        if (opts.list) {
          if (opts.json) {
            console.log(JSON.stringify({ recipes: ["builtin:ci"] }, null, 2));
          } else {
            console.log("Available recipes:");
            console.log("  builtin:ci");
          }
          return;
        }

        if (!recipe) {
          console.error(
            "usage: xh run [--list] [<recipe>] [--dry-run] [--json]"
          );
          process.exit(2);
        }

        if (recipe !== "builtin:ci") {
          console.error(`unknown recipe: ${recipe}`);
          console.error("run `xh run --list` for available recipes");
          process.exit(2);
        }

        const root = process.cwd();
        const steps: RunStep[] = [
          {
            name: "doctor",
            status: "planned",
            note: `xh doctor --root ${path.resolve(root)} --json`,
          },
          {
            name: "doctor_docs_drift",
            status: "planned",
            note: `xh doctor --docs-drift --root ${path.resolve(root)} --json`,
          },
          {
            name: "examples_verify",
            status: "planned",
            note: "xh examples verify --json",
          },
          {
            name: "verify_ci_standard",
            status: "planned",
            note: `xh verify --profile ci-standard --card examples/ci/strict-verify/completion-card.yaml --json`,
          },
        ];

        const ok = true;

        if (opts.json) {
          const result: RunResult = {
            recipe,
            ok,
            steps,
          };
          console.log(JSON.stringify(result, null, 2));
        } else {
          if (opts.dryRun) {
            console.log(`# xh run ${recipe} --dry-run`);
          } else {
            console.log(`# xh run ${recipe}`);
          }
          console.log("");
          for (const s of steps) {
            console.log(`step: ${s.name}`);
            console.log(`  status: ${s.status}`);
            if (s.note) {
              console.log(`  note: ${s.note}`);
            }
          }
          console.log("");
          console.log("Result: ok");
        }
      }
    );
}
