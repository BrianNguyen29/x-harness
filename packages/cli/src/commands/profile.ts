import { Command } from "commander";

interface ProfileRecommendOptions {
  goal?: string;
  json?: boolean;
}

interface ProfileRecommendation {
  recommended_profile: string;
  reason: string;
  required_commands: string[];
  recommended_checks: string[];
  not_needed: string[];
}

function recommendProfile(goal: string): ProfileRecommendation {
  const goalLower = goal.toLowerCase();

  const deepKeywords = [
    "release",
    "security",
    "deep",
    "governance",
    "approval",
  ];
  const standardKeywords = ["pr", "ci", "team", "verification"];
  const minimalKeywords = ["local", "basic", "quick", "single-agent"];

  for (const kw of deepKeywords) {
    if (goalLower.includes(kw)) {
      return {
        recommended_profile: "deep",
        reason: `Goal "${goal}" involves release, security, governance, or approval concerns; deep profile provides full evidence floor, rollback policy, and release readiness.`,
        required_commands: [
          "x-harness verify --strict",
          "x-harness report --format json",
          "x-harness conformance run --profile minimal",
        ],
        recommended_checks: [
          "mutation_guard",
          "evidence_provenance",
          "denominator_contract",
          "approval_receipt",
          "packet_chain",
        ],
        not_needed: [],
      };
    }
  }

  for (const kw of standardKeywords) {
    if (goalLower.includes(kw)) {
      return {
        recommended_profile: "standard",
        reason: `Goal "${goal}" involves PR/CI/team verification; standard profile provides mutation guard, trace, and report config.`,
        required_commands: [
          "x-harness verify",
          "x-harness report --format json",
        ],
        recommended_checks: ["mutation_guard", "evidence_provenance"],
        not_needed: [
          "packet_chain",
          "release_evidence_bundle",
          "approval_receipt",
        ],
      };
    }
  }

  for (const kw of minimalKeywords) {
    if (goalLower.includes(kw)) {
      return {
        recommended_profile: "minimal",
        reason: `Goal "${goal}" is local/basic/quick; minimal profile provides core verify contract and templates.`,
        required_commands: ["x-harness verify"],
        recommended_checks: ["standard_verify_gate"],
        not_needed: [
          "mutation_guard",
          "packet_chain",
          "release_evidence_bundle",
          "approval_receipt",
        ],
      };
    }
  }

  // Default to standard for unknown goals
  return {
    recommended_profile: "standard",
    reason: `Goal "${goal}" does not match a specific pattern; defaulting to standard profile for general verification.`,
    required_commands: ["x-harness verify", "x-harness report --format json"],
    recommended_checks: ["mutation_guard", "evidence_provenance"],
    not_needed: ["packet_chain", "release_evidence_bundle", "approval_receipt"],
  };
}

export function profileCommand(): Command {
  const cmd = new Command("profile").description(
    "Recommend installation profiles based on goals"
  );

  cmd
    .command("recommend")
    .description("Recommend a profile for a given goal")
    .requiredOption("--goal <goal>", "Goal description to match against")
    .option("--json", "Output JSON instead of text", false)
    .action((opts: ProfileRecommendOptions) => {
      const goal = opts.goal ?? "";
      if (!goal) {
        console.error(
          "usage: x-harness profile recommend --goal <goal> [--json]"
        );
        process.exit(2);
      }
      const rec = recommendProfile(goal);
      if (opts.json) {
        console.log(JSON.stringify(rec, null, 2));
      } else {
        console.log(`Recommended profile: ${rec.recommended_profile}`);
        console.log(`Reason: ${rec.reason}`);
        console.log("");
        console.log("Required commands:");
        for (const cmd of rec.required_commands) {
          console.log(`  - ${cmd}`);
        }
        console.log("");
        console.log("Recommended checks:");
        for (const check of rec.recommended_checks) {
          console.log(`  - ${check}`);
        }
        if (rec.not_needed.length > 0) {
          console.log("");
          console.log("Not needed:");
          for (const item of rec.not_needed) {
            console.log(`  - ${item}`);
          }
        }
      }
    });

  return cmd;
}
