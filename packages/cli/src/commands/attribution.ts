import { Command } from "commander";
import {
  buildAttributionReport,
  listAttributions,
  loadOrCreateAttribution,
} from "../core/attribution.js";
import { withEpisodeDirectory } from "../core/episode.js";
import { CliError } from "../core/exit.js";

type GroupBy = "predicate" | "taxonomy" | "component";

function parseSinceMs(since?: string): number | null {
  if (!since) return null;
  const match = /^(\d+)([dh])$/.exec(since);
  if (!match) return null;
  const value = Number(match[1]);
  return match[2] === "d"
    ? value * 24 * 60 * 60 * 1000
    : value * 60 * 60 * 1000;
}

function filterSince<T extends { created_at: string }>(
  items: T[],
  since?: string
): T[] {
  const sinceMs = parseSinceMs(since);
  if (!sinceMs) return items;
  const cutoff = Date.now() - sinceMs;
  return items.filter((item) => new Date(item.created_at).getTime() >= cutoff);
}

export function attributionCommand(): Command {
  const attribution = new Command("attribution").description(
    "Explain and report deterministic failure attribution from episodes"
  );

  attribution
    .command("explain")
    .description("Explain failure attribution for one episode")
    .requiredOption("--episode <path>", "Episode directory or bundle")
    .option("--json", "Output JSON instead of text", false)
    .action(async (opts: { episode: string; json?: boolean }) => {
      const result = await withEpisodeDirectory(opts.episode, (episodeDir) =>
        loadOrCreateAttribution(episodeDir)
      );
      if (opts.json) {
        console.log(JSON.stringify(result, null, 2));
      } else {
        console.log("# x-harness Failure Attribution");
        console.log(`- episode_id: ${result.episode_id}`);
        console.log(`- task_id: ${result.task_id}`);
        console.log(`- acceptance_status: ${result.verdict.acceptance_status}`);
        console.log(`- admission_outcome: ${result.verdict.admission_outcome}`);
        if (result.primary) {
          console.log(`- taxonomy: ${result.primary.taxonomy}`);
          console.log(`- predicate: ${result.primary.predicate}`);
          console.log(`- component_id: ${result.primary.component_id}`);
          console.log(`- confidence: ${result.primary.confidence}`);
          console.log(`- rationale: ${result.primary.rationale}`);
        } else {
          console.log("- taxonomy: none");
          console.log("- predicate: none");
        }
      }
    });

  attribution
    .command("report")
    .description("Aggregate failure attribution across episodes")
    .option("--episodes-dir <dir>", "Episodes directory", ".x-harness/episodes")
    .option(
      "--group-by <field>",
      "Group by predicate, taxonomy, or component",
      "predicate"
    )
    .option("--since <duration>", "Filter by age, e.g. 7d or 30d")
    .option("--json", "Output JSON instead of text", false)
    .action(
      async (opts: {
        episodesDir?: string;
        groupBy?: string;
        since?: string;
        json?: boolean;
      }) => {
        if (
          opts.groupBy !== "predicate" &&
          opts.groupBy !== "taxonomy" &&
          opts.groupBy !== "component"
        ) {
          throw new CliError(
            "--group-by must be predicate, taxonomy, or component",
            2
          );
        }
        const attributions = filterSince(
          await listAttributions(opts.episodesDir ?? ".x-harness/episodes"),
          opts.since
        );
        const report = buildAttributionReport(
          attributions,
          opts.groupBy as GroupBy
        );
        if (opts.json) {
          console.log(JSON.stringify(report, null, 2));
        } else {
          console.log("# x-harness Attribution Report");
          console.log(`- group_by: ${report.group_by}`);
          console.log(`- total_episodes: ${report.total_episodes}`);
          console.log(`- withheld_episodes: ${report.withheld_episodes}`);
          console.log(`- unknown_rate: ${report.unknown_rate}`);
          if (report.entropy_warning) {
            console.log(`- entropy_warning: ${report.entropy_warning}`);
          }
          console.log("");
          console.log("## Groups");
          if (report.groups.length === 0) {
            console.log("None.");
          } else {
            for (const group of report.groups) {
              console.log(`- ${group.key}: ${group.count}`);
            }
          }
        }
      }
    );

  return attribution;
}
