import { Command } from "commander";
import * as path from "node:path";
import {
  checkEvolutionConstitution,
  evaluateEvolutionBudget,
  renderChangeRequest,
  writeChangeRequest,
} from "../core/evolution.js";
import { CliError } from "../core/exit.js";

interface RootJsonOptions {
  root?: string;
  json?: boolean;
}

interface CandidateOptions extends RootJsonOptions {
  candidate?: string;
  constitution?: string;
  out?: string;
}

interface ProposeOptions extends RootJsonOptions {
  component?: string;
  out?: string;
  write?: boolean;
}

function printJson(data: unknown): void {
  console.log(JSON.stringify(data, null, 2));
}

export function evolveCommand(): Command {
  const evolve = new Command("evolve").description(
    "Evaluate experimental evolution candidates without mutating source"
  );

  evolve
    .command("evaluate")
    .description("Evaluate the local evolution budget")
    .option("--root <path>", "Repository root", process.cwd())
    .option("--json", "Output JSON instead of text", false)
    .action(async (opts: RootJsonOptions) => {
      const result = await evaluateEvolutionBudget(
        path.resolve(opts.root ?? process.cwd())
      );
      if (opts.json) {
        printJson(result);
      } else {
        console.log(result.message);
      }
    });

  evolve
    .command("analyze")
    .description("Create an advisory analysis request for an evolution run")
    .requiredOption("--run <run-id>", "Run id to analyze")
    .option("--root <path>", "Repository root", process.cwd())
    .option("--out <path>", "Write analysis request to this path")
    .option("--json", "Output JSON instead of text", false)
    .action(async (opts: RootJsonOptions & { run: string; out?: string }) => {
      const root = path.resolve(opts.root ?? process.cwd());
      const content = renderChangeRequest({
        kind: "analysis",
        summary: `Analyze evolution run ${opts.run}`,
      });
      const out = opts.out
        ? await writeChangeRequest(root, content, opts.out)
        : null;
      const result = {
        ok: true,
        status: out ? "written" : "proposed",
        path: out,
        run_id: opts.run,
        admission_authority: false,
      };
      if (opts.json) printJson(result);
      else console.log(out ? `analysis request written: ${out}` : content);
    });

  evolve
    .command("propose")
    .description(
      "Create a change request for a component; does not edit source"
    )
    .requiredOption("--component <id>", "Component id")
    .option("--root <path>", "Repository root", process.cwd())
    .option("--out <path>", "Write change request to this path")
    .option("--write", "Write the change request", false)
    .option("--json", "Output JSON instead of text", false)
    .action(async (opts: ProposeOptions) => {
      const root = path.resolve(opts.root ?? process.cwd());
      const content = renderChangeRequest({
        kind: "proposal",
        component: opts.component,
        summary: `Propose a candidate for ${opts.component}`,
      });
      const out =
        opts.write || opts.out
          ? await writeChangeRequest(root, content, opts.out)
          : null;
      const result = {
        ok: true,
        status: out ? "written" : "proposed",
        path: out,
        component: opts.component,
        admission_authority: false,
      };
      if (opts.json) printJson(result);
      else console.log(out ? `change request written: ${out}` : content);
    });

  evolve
    .command("constitution-check")
    .description("Check an evolution candidate against the constitution")
    .requiredOption("--candidate <path-or-id>", "Candidate manifest path or id")
    .option("--constitution <path>", "Constitution path")
    .option("--root <path>", "Repository root", process.cwd())
    .option("--json", "Output JSON instead of text", false)
    .action(async (opts: CandidateOptions) => {
      const result = await checkEvolutionConstitution({
        root: path.resolve(opts.root ?? process.cwd()),
        candidate: opts.candidate as string,
        constitutionPath: opts.constitution,
      });
      if (opts.json) {
        printJson(result);
      } else if (result.ok) {
        console.log(`constitution passed: ${result.candidate_id}`);
      } else {
        console.log(`constitution failed: ${result.candidate_id}`);
        for (const violation of result.violations) {
          console.log(`- ${violation}`);
        }
      }
      if (!result.ok) {
        throw new CliError("constitution check failed", 1);
      }
    });

  evolve
    .command("compare")
    .description("Compare candidate metrics against baseline")
    .requiredOption("--candidate <path-or-id>", "Candidate manifest path or id")
    .option("--root <path>", "Repository root", process.cwd())
    .option("--json", "Output JSON instead of text", false)
    .action(async (opts: CandidateOptions) => {
      const result = await checkEvolutionConstitution({
        root: path.resolve(opts.root ?? process.cwd()),
        candidate: opts.candidate as string,
      });
      const output = {
        ok: result.ok,
        candidate_id: result.candidate_id,
        constitution_status: result.status,
        false_accept_regression: result.violations.some((item) =>
          item.includes("false_accept")
        ),
        admission_authority: false,
      };
      if (opts.json) printJson(output);
      else console.log(JSON.stringify(output, null, 2));
      if (!output.ok) throw new CliError("candidate comparison failed", 1);
    });

  evolve
    .command("promote")
    .description(
      "Generate a promotion request; does not merge or mutate policy"
    )
    .requiredOption("--candidate <path-or-id>", "Candidate manifest path or id")
    .option("--root <path>", "Repository root", process.cwd())
    .option("--out <path>", "Write promotion request to this path")
    .option("--json", "Output JSON instead of text", false)
    .action(async (opts: CandidateOptions) => {
      const root = path.resolve(opts.root ?? process.cwd());
      const constitution = await checkEvolutionConstitution({
        root,
        candidate: opts.candidate as string,
        constitutionPath: opts.constitution,
      });
      if (!constitution.ok) {
        if (opts.json) printJson(constitution);
        throw new CliError("promotion blocked by constitution", 1);
      }
      const content = renderChangeRequest({
        kind: "promotion",
        candidateId: constitution.candidate_id,
        summary:
          "Promotion requires human review and explicit merge outside x-harness.",
        constitution,
      });
      const out = await writeChangeRequest(root, content, opts.out);
      const result = {
        ok: true,
        status: "written",
        path: out,
        candidate_id: constitution.candidate_id,
        admission_authority: false,
      };
      if (opts.json) printJson(result);
      else console.log(`promotion request written: ${out}`);
    });

  evolve
    .command("rollback")
    .description("Generate a rollback request; does not run git")
    .requiredOption("--candidate <path-or-id>", "Candidate manifest path or id")
    .option("--root <path>", "Repository root", process.cwd())
    .option("--out <path>", "Write rollback request to this path")
    .option("--json", "Output JSON instead of text", false)
    .action(async (opts: CandidateOptions) => {
      const root = path.resolve(opts.root ?? process.cwd());
      const constitution = await checkEvolutionConstitution({
        root,
        candidate: opts.candidate as string,
        constitutionPath: opts.constitution,
      });
      const content = renderChangeRequest({
        kind: "rollback",
        candidateId: constitution.candidate_id,
        summary:
          "Rollback requires human review and explicit git operation outside x-harness.",
        constitution,
      });
      const out = await writeChangeRequest(root, content, opts.out);
      const result = {
        ok: true,
        status: "written",
        path: out,
        candidate_id: constitution.candidate_id,
        admission_authority: false,
      };
      if (opts.json) printJson(result);
      else console.log(`rollback request written: ${out}`);
    });

  return evolve;
}
