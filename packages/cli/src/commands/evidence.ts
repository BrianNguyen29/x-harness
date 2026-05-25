import { Command } from "commander";
import * as path from "node:path";
import {
  buildEvidenceDigest,
  createEvidenceIndex,
  readEvidenceIndex,
  renderEvidenceDigestMarkdown,
  toJsonl,
  validateEvidenceIndex,
} from "../core/evidence-corpus.js";
import { CliError } from "../core/exit.js";

interface EvidenceIndexOptions {
  root?: string;
  episode?: string;
  card?: string;
  taskId?: string;
  out?: string;
  redact?: boolean;
  redactedDir?: string;
  json?: boolean;
}

interface EvidenceValidateOptions {
  index?: string;
  json?: boolean;
}

interface EvidenceGrepOptions {
  index?: string;
  predicate?: string;
  json?: boolean;
}

export function evidenceCommand(): Command {
  const evidence = new Command("evidence").description(
    "Index, validate, and query replayable evidence records"
  );

  evidence
    .command("index")
    .description(
      "Build a JSONL evidence index from an episode directory or card"
    )
    .option("--root <path>", "Repository root", process.cwd())
    .option("--episode <dir>", "Episode directory to scan")
    .option("--card <path>", "Completion card to index")
    .option("--task-id <id>", "Task id when it cannot be inferred")
    .option(
      "--out <path>",
      "Evidence index JSONL output path",
      "evidence/index.jsonl"
    )
    .option(
      "--redact",
      "Write redacted text artifacts for known secrets",
      false
    )
    .option("--redacted-dir <dir>", "Redacted evidence output directory")
    .option("--json", "Output JSON summary", false)
    .action(async (opts: EvidenceIndexOptions) => {
      if (!opts.episode && !opts.card) {
        throw new CliError("evidence index requires --episode or --card", 2);
      }
      const root = path.resolve(opts.root ?? process.cwd());
      const result = await createEvidenceIndex({
        root,
        episodeDir: opts.episode,
        cardPath: opts.card,
        taskId: opts.taskId,
        outPath: opts.out,
        redact: Boolean(opts.redact),
        redactedDir: opts.redactedDir,
      });
      const validation = await validateEvidenceIndex(result.entries);
      if (!validation.ok) {
        if (opts.json) {
          console.log(
            JSON.stringify(
              {
                ok: false,
                errors: validation.errors,
                ...result,
              },
              null,
              2
            )
          );
        }
        throw new CliError(
          `evidence index validation failed: ${validation.errors.join("; ")}`,
          1
        );
      }

      if (opts.json) {
        console.log(
          JSON.stringify(
            {
              ok: true,
              ...result,
            },
            null,
            2
          )
        );
      } else {
        console.log("Evidence index written.");
        console.log(`- task_id: ${result.task_id}`);
        console.log(`- entries: ${result.entry_count}`);
        console.log(`- index_hash: ${result.index_hash}`);
        if (result.out_path) console.log(`- out: ${result.out_path}`);
        if (result.redacted_dir) {
          console.log(`- redacted_dir: ${result.redacted_dir}`);
        }
        for (const warning of result.warnings) {
          console.log(`warning: ${warning}`);
        }
      }
    });

  evidence
    .command("validate")
    .description("Validate an evidence index JSONL or JSON envelope")
    .option("--index <path>", "Evidence index path", "evidence/index.jsonl")
    .option("--json", "Output JSON instead of text", false)
    .action(async (opts: EvidenceValidateOptions) => {
      const entries = await readEvidenceIndex(
        opts.index ?? "evidence/index.jsonl"
      );
      const result = await validateEvidenceIndex(entries);
      if (opts.json) {
        console.log(
          JSON.stringify(
            {
              ok: result.ok,
              errors: result.errors,
              entry_count: entries.length,
            },
            null,
            2
          )
        );
      } else if (result.ok) {
        console.log(`Evidence index valid (${entries.length} entries).`);
      } else {
        console.log("Evidence index invalid:");
        for (const error of result.errors) console.log(`- ${error}`);
      }
      if (!result.ok) {
        throw new CliError("evidence index validation failed", 1);
      }
    });

  evidence
    .command("grep")
    .description("Filter evidence index entries by predicate")
    .requiredOption("--predicate <predicate>", "Predicate value to match")
    .option("--index <path>", "Evidence index path", "evidence/index.jsonl")
    .option("--json", "Output JSON instead of text", false)
    .action(async (opts: EvidenceGrepOptions) => {
      const entries = await readEvidenceIndex(
        opts.index ?? "evidence/index.jsonl"
      );
      const matches = entries.filter(
        (entry) => entry.predicate === opts.predicate
      );
      if (opts.json) {
        console.log(
          JSON.stringify(
            {
              predicate: opts.predicate,
              count: matches.length,
              entries: matches,
            },
            null,
            2
          )
        );
      } else {
        console.log(`# evidence grep: ${opts.predicate}`);
        if (matches.length === 0) {
          console.log("No matching entries.");
        } else {
          for (const entry of matches) {
            console.log(`- ${entry.evidence_id} ${entry.kind} ${entry.path}`);
          }
        }
      }
    });

  evidence
    .command("digest")
    .description("Render a deterministic digest from an evidence index")
    .requiredOption("--task-id <id>", "Task id to summarize")
    .option("--index <path>", "Evidence index path", "evidence/index.jsonl")
    .option("--json", "Output JSON instead of Markdown", false)
    .action(
      async (opts: { taskId: string; index?: string; json?: boolean }) => {
        const entries = await readEvidenceIndex(
          opts.index ?? "evidence/index.jsonl"
        );
        const digest = buildEvidenceDigest({
          taskId: opts.taskId,
          entries,
        });
        if (opts.json) {
          console.log(JSON.stringify(digest, null, 2));
        } else {
          console.log(renderEvidenceDigestMarkdown(digest));
        }
      }
    );

  evidence
    .command("print")
    .description("Print an evidence index JSONL file after parsing")
    .option("--index <path>", "Evidence index path", "evidence/index.jsonl")
    .action(async (opts: { index?: string }) => {
      const entries = await readEvidenceIndex(
        opts.index ?? "evidence/index.jsonl"
      );
      process.stdout.write(toJsonl(entries));
    });

  return evidence;
}
