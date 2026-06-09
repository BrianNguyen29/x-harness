import { Command } from "commander";
import * as path from "node:path";
import {
  type Lang,
  resolveLang,
  startTitle,
  startStepLabel,
  startNextStepsTitle,
  startFirstVerification,
  startReadDocs,
} from "../i18n.js";

interface StartStep {
  name: string;
  status: string;
  note?: string;
}

interface StartResult {
  ok: boolean;
  steps: StartStep[];
  next_steps: string[];
}

/**
 * TS start is a preview/planner: it lists planned steps without executing them.
 * The Go CLI executes doctor, examples verify, and init wizard directly.
 */
export function startCommand(): Command {
  return new Command("start")
    .description(
      "Guided onboarding: doctor, examples verify, init wizard, next steps"
    )
    .option("--root <path>", "Repository root", process.cwd())
    .option(
      "--profile <profile>",
      "Install profile (minimal, standard, deep)",
      "minimal"
    )
    .option("--apply", "Apply changes (init wizard without dry-run)", false)
    .option("--skip-doctor", "Skip doctor step", false)
    .option("--skip-examples", "Skip examples verify step", false)
    .option("--json", "Output JSON instead of text", false)
    .option(
      "--wizard-with-card <task_id>",
      "Scaffold a completion card on apply"
    )
    .option("--lang <code>", "Language", "en")
    .action(
      async (
        opts: {
          root: string;
          profile: string;
          apply: boolean;
          skipDoctor: boolean;
          skipExamples: boolean;
          json: boolean;
          wizardWithCard?: string;
          lang?: string;
        },
        cmd: Command
      ) => {
        const validProfiles = ["minimal", "standard", "full", "deep"];
        if (!validProfiles.includes(opts.profile)) {
          console.error(
            "usage: xh start [--root <path>] [--profile minimal|standard|full|deep] [--apply] [--skip-doctor] [--skip-examples] [--json] [--wizard-with-card <task_id>]"
          );
          console.error(`invalid profile: ${opts.profile}`);
          process.exit(2);
        }

        const lang: Lang = resolveLang(opts, cmd.parent?.opts() ?? {});

        const steps: StartStep[] = [];

        // Step 1: Doctor
        if (!opts.skipDoctor) {
          steps.push({
            name: "doctor",
            status: "planned",
            note: `xh doctor --root ${path.resolve(opts.root)} --json`,
          });
        }

        // Step 2: Examples verify
        if (!opts.skipExamples) {
          steps.push({
            name: "examples_verify",
            status: "planned",
            note: "xh examples verify --json",
          });
        }

        // Step 3: Init wizard
        const mode = opts.apply ? "apply" : "preview";
        const initNote = `xh init ${path.resolve(opts.root)} --wizard --wizard-profile ${opts.profile}${
          opts.apply ? "" : " --wizard-dry-run"
        }${opts.wizardWithCard ? ` --wizard-with-card ${opts.wizardWithCard}` : ""}`;
        steps.push({
          name: "init_wizard",
          status: "planned",
          note: initNote,
        });

        const nextStepsText = [
          startFirstVerification(lang),
          startReadDocs(lang),
        ];
        const nextStepsJSON = [
          "Run your first verification: xh check --card completion-card.yaml",
          "Read the docs: docs/GETTING_STARTED.md",
        ];

        if (opts.json) {
          const result: StartResult = {
            ok: true,
            steps,
            next_steps: nextStepsJSON,
          };
          console.log(JSON.stringify(result, null, 2));
        } else {
          console.log(`# ${startTitle(lang)}`);
          console.log("");
          if (!opts.skipDoctor) {
            console.log(`Step 1/4: ${startStepLabel("doctor", lang)}`);
            console.log(`  -> ${steps[0].note}`);
            console.log("");
          }
          if (!opts.skipExamples) {
            const idx = opts.skipDoctor ? 0 : 1;
            console.log(
              `Step ${idx + 1}/4: ${startStepLabel("examples_verify", lang)}`
            );
            console.log(`  -> ${steps[idx].note}`);
            console.log("");
          }
          const initIdx = steps.findIndex((s) => s.name === "init_wizard");
          console.log(
            `Step ${initIdx + 1}/4: ${startStepLabel("init_wizard", lang)} (${mode})`
          );
          console.log(`  -> ${steps[initIdx].note}`);
          console.log("");
          console.log(startNextStepsTitle(lang));
          for (const s of nextStepsText) {
            console.log(`  - ${s}`);
          }
        }
      }
    );
}
