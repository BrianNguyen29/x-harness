import { Command } from "commander";
import {
  buildAgentProfile,
  defaultAgentProfilePath,
  readAgentProfile,
  writeAgentProfile,
} from "../core/agent-profile.js";
import { CliError } from "../core/exit.js";

interface AgentProfileOptions {
  agent?: string;
  fromBenchmark?: string;
  out?: string;
  profile?: string;
  root?: string;
  json?: boolean;
}

export function agentProfileCommand(): Command {
  const cmd = new Command("agent-profile").description(
    "Build advisory agent capability profiles from benchmark reports"
  );

  cmd
    .command("update")
    .description("Create or update an advisory profile from benchmark output")
    .requiredOption("--agent <id>", "Agent id")
    .option("--from-benchmark <path>", "Benchmark report JSON/YAML path")
    .option("--out <path>", "Profile output path")
    .option("--root <path>", "Repository root", process.cwd())
    .option("--json", "Output JSON instead of text", false)
    .action(async (opts: AgentProfileOptions) => {
      const profile = await buildAgentProfile({
        agentId: opts.agent as string,
        benchmarkReportPath: opts.fromBenchmark,
      });
      const outPath = await writeAgentProfile(
        profile,
        opts.out ??
          defaultAgentProfilePath(opts.root ?? process.cwd(), profile.agent_id)
      );
      const output = { ok: true, path: outPath, profile };
      if (opts.json) console.log(JSON.stringify(output, null, 2));
      else {
        console.log(`# x-harness Agent Profile: ${profile.agent_id}`);
        console.log(
          `- observed_failure_modes: ${profile.observed_failure_modes.length}`
        );
        console.log(
          `- required_extra_checks: ${profile.required_extra_checks.join(", ")}`
        );
        if (outPath) console.log(`- path: ${outPath}`);
      }
    });

  cmd
    .command("report")
    .description("Read and validate an advisory agent profile")
    .option("--profile <path>", "Profile JSON/YAML path")
    .option("--agent <id>", "Agent id for the default profile path")
    .option("--root <path>", "Repository root", process.cwd())
    .option("--json", "Output JSON instead of text", false)
    .action(async (opts: AgentProfileOptions) => {
      const profilePath =
        opts.profile ??
        (opts.agent
          ? defaultAgentProfilePath(opts.root ?? process.cwd(), opts.agent)
          : null);
      if (!profilePath) {
        throw new CliError(
          "agent-profile report requires --profile or --agent",
          2
        );
      }
      const profile = await readAgentProfile(profilePath);
      if (opts.json) console.log(JSON.stringify(profile, null, 2));
      else {
        console.log(`# x-harness Agent Profile: ${profile.agent_id}`);
        console.log(`- advisory_only: ${profile.advisory_only}`);
        console.log(`- admission_authority: ${profile.admission_authority}`);
      }
    });

  return cmd;
}
