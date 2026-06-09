import { Command } from "commander";
import * as fs from "node:fs";
import * as path from "node:path";
import {
  type Lang,
  resolveLang,
  quickTitle,
  quickRootLabel,
  quickRecommendationLabel,
  quickReasonLabel,
  quickDetectedSignalsLabel,
  quickNoneLabel,
  quickNextStepsLabel,
} from "../i18n.js";

interface QuickResult {
  root: string;
  recommendation: string;
  reason: string;
  next_steps: string[];
  detected_signals: string[];
}

const harnessMarkers = ["AGENTS.md", "X_HARNESS.md", ".x-harness"];
const cardNames = [
  "completion-card.yaml",
  "completion-card.yml",
  "completion-card.json",
];

function detectSignals(root: string): string[] {
  const signals: string[] = [];
  for (const marker of harnessMarkers) {
    const p = path.join(root, marker);
    if (fs.existsSync(p)) {
      signals.push(`harness_marker:${marker}`);
    }
  }
  // Walk up to depth 4 looking for completion cards
  function walk(dir: string, depth: number) {
    if (depth > 4) return;
    let entries: fs.Dirent[];
    try {
      entries = fs.readdirSync(dir, { withFileTypes: true });
    } catch {
      return;
    }
    for (const entry of entries) {
      if (entry.isDirectory()) {
        const name = entry.name;
        if (
          name === "node_modules" ||
          name === ".git" ||
          name === "vendor" ||
          name === "dist" ||
          name === "coverage"
        ) {
          continue;
        }
        // Skip generated harness state directories
        if (name === "tmp" || name === "cache") {
          if (path.basename(dir) === ".x-harness") {
            continue;
          }
        }
        walk(path.join(dir, name), depth + 1);
      } else {
        for (const name of cardNames) {
          if (entry.name === name) {
            const rel = path.relative(root, path.join(dir, entry.name));
            signals.push(`completion_card:${rel}`);
            break;
          }
        }
      }
    }
  }
  walk(root, 0);
  return signals;
}

function buildRecommendation(
  root: string,
  signals: string[]
): { recommendation: string; reason: string; nextSteps: string[] } {
  let hasHarness = false;
  const cardPaths: string[] = [];
  for (const s of signals) {
    if (s.startsWith("harness_marker:")) {
      hasHarness = true;
    }
    if (s.startsWith("completion_card:")) {
      cardPaths.push(s.slice("completion_card:".length));
    }
  }

  let recommendation: string;
  let reason: string;
  const nextSteps: string[] = [];

  if (!hasHarness) {
    recommendation = "xh start";
    reason =
      "No harness markers found under root. Begin with guided onboarding.";
    nextSteps.push("xh start");
    nextSteps.push("xh init");
  } else if (cardPaths.length > 0) {
    recommendation = `xh check --card ${cardPaths[0]}`;
    reason = "A completion card was found. Verify it as the next step.";
    nextSteps.push(`xh check --card ${cardPaths[0]}`);
  } else {
    recommendation = `xh doctor --root ${root} --json`;
    reason =
      "Harness is present but no completion card found yet. Check workspace health first.";
    nextSteps.push(`xh doctor --root ${root} --json`);
  }

  nextSteps.push("xh run builtin:ci --dry-run");
  nextSteps.push("xh learn");

  return { recommendation, reason, nextSteps };
}

export function quickCommand(): Command {
  return new Command("quick")
    .description("Read-only next-action recommender for newcomers")
    .option("--root <path>", "Repository root", process.cwd())
    .option("--json", "Output JSON instead of text", false)
    .option("--lang <code>", "Language", "en")
    .action(
      async (
        opts: { root: string; json: boolean; lang: string },
        cmd: Command
      ) => {
        const lang: Lang = resolveLang(opts, cmd.parent?.opts() ?? {});
        const root = path.resolve(opts.root);
        const signals = detectSignals(root);
        const { recommendation, reason, nextSteps } = buildRecommendation(
          root,
          signals
        );

        const result: QuickResult = {
          root,
          recommendation,
          reason,
          next_steps: nextSteps,
          detected_signals: signals,
        };

        if (opts.json) {
          console.log(JSON.stringify(result, null, 2));
        } else {
          console.log(`# ${quickTitle(lang)}`);
          console.log("");
          console.log(`${quickRootLabel(lang)}: ${result.root}`);
          console.log(
            `${quickRecommendationLabel(lang)}: ${result.recommendation}`
          );
          console.log(`${quickReasonLabel(lang)}: ${result.reason}`);
          console.log("");
          console.log(quickDetectedSignalsLabel(lang));
          if (result.detected_signals.length === 0) {
            console.log(quickNoneLabel(lang));
          } else {
            for (const s of result.detected_signals) {
              console.log(`  - ${s}`);
            }
          }
          console.log("");
          console.log(quickNextStepsLabel(lang));
          for (const s of result.next_steps) {
            console.log(`  - ${s}`);
          }
        }
      }
    );
}
