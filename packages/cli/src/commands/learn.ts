import { Command } from "commander";
import {
  type Lang,
  resolveLang,
  learnTitle,
  learnOverview,
  learnCoreConcepts,
  learnTiersAndEvidence,
  learnNextStepsLabel,
  learnOverviewBody,
  learnCoreConceptsBody,
  learnTiersAndEvidenceBody,
  learnNextSteps,
} from "../i18n.js";

interface LearnSection {
  title: string;
  body: string;
}

interface LearnResult {
  sections: LearnSection[];
  next_steps: string[];
}

export function learnCommand(): Command {
  return new Command("learn")
    .description("Read-only concept tour for beginners")
    .option("--json", "Output JSON instead of text", false)
    .option("--lang <code>", "Language", "en")
    .action(async (opts: { json: boolean; lang: string }, cmd: Command) => {
      const lang: Lang = resolveLang(opts, cmd.parent?.opts() ?? {});

      const sections: LearnSection[] = [
        {
          title: "Overview",
          body: "x-harness is a lightweight verify-gated harness for AI-agent workflows. It enforces that completion is admitted, not claimed, via a read-only verifier.",
        },
        {
          title: "Core concepts",
          body: `Completion is admitted, not claimed — only the verify gate can accept work.
Verifier is read-only — it inspects evidence but never edits source files.
Success is the only accepted outcome — all non-success results are withheld.
Canonical tiers are light, standard, and deep — each with increasing evidence requirements.
PGV (pre-gate validation) is advisory-only — it never overrides the verify gate.`,
        },
        {
          title: "Tiers and evidence",
          body: `light: files_changed plus command evidence or manual rationale.
standard: adds done_checklist and prediction.
deep: adds evidence scope declaration, untested regions, remaining risks, execution controls, rollback policy, read/write sets, and verification artifacts.`,
        },
      ];

      const nextSteps = [
        "Run xh start for guided onboarding",
        "Run xh check --card <card> to verify a completion card",
        "Run xh actions to see beginner-friendly commands",
        "Read docs/GETTING_STARTED.md",
      ];

      if (opts.json) {
        const result: LearnResult = {
          sections,
          next_steps: nextSteps,
        };
        console.log(JSON.stringify(result, null, 2));
      } else {
        console.log(`# ${learnTitle(lang)}`);
        console.log("");
        for (let i = 0; i < sections.length; i++) {
          const sec = sections[i];
          let title = sec.title;
          let body = sec.body;
          if (lang === "vi") {
            if (i === 0) {
              title = learnOverview(lang);
              body = learnOverviewBody(lang);
            } else if (i === 1) {
              title = learnCoreConcepts(lang);
              body = learnCoreConceptsBody(lang);
            } else if (i === 2) {
              title = learnTiersAndEvidence(lang);
              body = learnTiersAndEvidenceBody(lang);
            }
          }
          console.log(`## ${title}`);
          console.log("");
          console.log(body);
          console.log("");
        }
        console.log(learnNextStepsLabel(lang));
        for (const s of learnNextSteps(lang)) {
          console.log(`  - ${s}`);
        }
      }
    });
}
