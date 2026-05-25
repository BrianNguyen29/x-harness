import { Command } from "commander";
import { inspectEpisode, verifyEpisodeChain } from "../core/episode.js";
import { CliError } from "../core/exit.js";

export function episodeCommand(): Command {
  const episode = new Command("episode").description(
    "Inspect and verify x-harness episode packages"
  );

  episode
    .command("inspect")
    .description("Inspect an episode directory or .tar.gz bundle")
    .argument("<path>", "Episode directory or bundle path")
    .option("--json", "Output JSON instead of text", false)
    .action(async (episodePath: string, opts: { json?: boolean }) => {
      const result = await inspectEpisode(episodePath);
      if (opts.json) {
        console.log(JSON.stringify(result, null, 2));
      } else {
        console.log("# x-harness Episode Inspect");
        console.log(`- ok: ${result.ok}`);
        console.log(`- episode_id: ${result.episode_id ?? "unknown"}`);
        console.log(`- task_id: ${result.task_id ?? "unknown"}`);
        console.log(`- file_count: ${result.file_count}`);
        if (result.errors.length > 0) {
          console.log("");
          console.log("## Errors");
          for (const error of result.errors) console.log(`- ${error}`);
        }
        if (result.warnings.length > 0) {
          console.log("");
          console.log("## Warnings");
          for (const warning of result.warnings) console.log(`- ${warning}`);
        }
      }
      if (!result.ok) {
        throw new CliError("episode validation failed", 1);
      }
    });

  episode
    .command("verify-chain")
    .description("Verify episode chain integrity for a task")
    .requiredOption("--task-id <id>", "Task id")
    .option("--episodes-dir <dir>", "Episodes directory", ".x-harness/episodes")
    .option("--json", "Output JSON instead of text", false)
    .action(
      async (opts: {
        taskId: string;
        episodesDir?: string;
        json?: boolean;
      }) => {
        const result = await verifyEpisodeChain(opts.taskId, opts.episodesDir);
        if (opts.json) {
          console.log(JSON.stringify(result, null, 2));
        } else if (result.ok) {
          console.log(
            `episode chain valid: ${result.episodes_checked} episode(s) checked`
          );
          for (const id of result.episode_ids) console.log(`- ${id}`);
        } else {
          console.log("episode chain invalid:");
          for (const error of result.errors) console.log(`- ${error}`);
        }
        if (!result.ok) {
          throw new CliError("episode chain validation failed", 1);
        }
      }
    );

  episode
    .command("sign")
    .description("Signing skeleton for future release hardening")
    .option("--episode <path>", "Episode directory or bundle")
    .option("--mode <mode>", "Signing mode", "unsigned")
    .option("--json", "Output JSON instead of text", false)
    .action((opts: { episode?: string; mode?: string; json?: boolean }) => {
      const result = {
        implemented: false,
        mode: opts.mode ?? "unsigned",
        episode: opts.episode ?? null,
        note: "local MVP supports unsigned episode packages only; signing is reserved for release hardening",
      };
      if (opts.json) {
        console.log(JSON.stringify(result, null, 2));
      } else {
        console.log("episode sign is not implemented in local MVP.");
        console.log(result.note);
      }
    });

  return episode;
}
